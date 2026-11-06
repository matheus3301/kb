package setup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
)

type journal struct {
	Schema      int               `json:"schema_version"`
	Changes     []Change          `json:"changes"`
	Stages      map[string]string `json:"stages"`
	Directories []string          `json:"directories,omitempty"`
	Applied     int               `json:"applied"`
	Committed   bool              `json:"committed"`
}

func mkdirSafe(path string) error {
	if err := safePath(path); err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	return safePath(path)
}

func syncDirectory(path string) error {
	f, err := os.Open(path) // #nosec G304 -- checked canonical setup destination directory.
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func replace(path string, s Snapshot) error {
	if err := safePath(path); err != nil {
		return err
	}
	parent := filepath.Dir(path)
	if err := mkdirSafe(parent); err != nil {
		return err
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	base := filepath.Base(path)
	if !s.Exists {
		err := root.Remove(base)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		return syncDirectory(parent)
	}
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	name := ".kb-setup-" + hex.EncodeToString(random[:]) + ".tmp"
	f, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, os.FileMode(s.Mode))
	if err != nil {
		return err
	}
	defer func() { _ = root.Remove(name) }()
	if _, err = f.Write(s.Content); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Chmod(os.FileMode(s.Mode)); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = root.Rename(name, base); err != nil {
		return err
	}
	return syncDirectory(parent)
}

func (e *Engine) lock(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := mkdirSafe(e.stateDir); err != nil {
		return nil, err
	}
	// Shared config before-images can contain credentials. Never place private
	// journals under an accessible state directory supplied by another program.
	info, err := os.Stat(e.stateDir)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, conflict("setup state directory must be private (0700): " + e.stateDir)
	}
	path := filepath.Join(e.stateDir, "lock")
	if err := safePath(path); err != nil {
		return nil, err
	}
	directory, err := os.OpenRoot(e.stateDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = directory.Close() }()
	f, err := directory.OpenFile("lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, conflict("another setup operation holds the setup lock")
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}
func (e *Engine) saveJournal(j journal) error {
	raw, err := encode(j)
	if err != nil {
		return err
	}
	return replace(e.pendingPath(), Snapshot{Exists: true, Content: raw, Mode: 0600})
}
func (e *Engine) fail(boundary string) error {
	if e.options.Failpoint != nil {
		return e.options.Failpoint(boundary)
	}
	return nil
}
func checkPreconditions(pre map[string]Snapshot) error {
	for _, path := range sortedKeys(pre) {
		now, err := readSnapshot(path)
		if err != nil {
			return err
		}
		if !now.equal(pre[path]) {
			return conflict("setup precondition changed: " + path)
		}
	}
	return nil
}

// Apply uses one private user-wide lock and compare-before-publish checks.
// Each rename is atomic; the collection of renames is deliberately not claimed
// to be globally atomic. Pending state blocks resolution until explicit recovery.
func (e *Engine) Apply(ctx context.Context, p *Plan) (Result, error) {
	if p == nil || p.stateDir != e.stateDir {
		return Result{}, invalid("plan does not belong to this setup engine")
	}
	if len(p.Changes) == 0 {
		return p.Result(false), nil
	}
	unlock, err := e.lock(ctx)
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	if pending, err := readSnapshot(e.pendingPath()); err != nil {
		return Result{}, err
	} else if pending.Exists {
		return Result{}, recoveryRequired()
	}
	if err := checkPreconditions(p.Preconditions); err != nil {
		return Result{}, err
	}
	j := journal{Schema: SchemaVersion, Changes: p.Changes, Stages: map[string]string{}}
	for _, change := range j.Changes {
		if !change.After.Exists {
			continue
		}
		for dir := filepath.Dir(change.Path); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
				j.Directories = append(j.Directories, dir)
			} else if err != nil {
				return Result{}, err
			} else {
				break
			}
		}
		var random [12]byte
		if _, err := rand.Read(random[:]); err != nil {
			return Result{}, err
		}
		j.Stages[change.Path] = filepath.Join(filepath.Dir(change.Path), ".kb-stage-"+hex.EncodeToString(random[:]))
	}
	if err := e.saveJournal(j); err != nil {
		return Result{}, err
	}
	abort := func(cause error) (Result, error) {
		rollbackErr := e.rollback(j)
		if rollbackErr != nil {
			return Result{Schema: SchemaVersion, Status: "partial"}, errors.Join(cause, rollbackErr, recoveryRequired())
		}
		return Result{Schema: SchemaVersion, Status: "rolled_back"}, cause
	}
	if err := e.fail("prepared"); err != nil {
		return abort(err)
	}
	// Stage every after-image beside its final destination before publication.
	for _, change := range j.Changes {
		if !change.After.Exists {
			continue
		}
		if err := ctx.Err(); err != nil {
			return abort(err)
		}
		stage := j.Stages[change.Path]
		if err := mkdirSafe(filepath.Dir(stage)); err != nil {
			return abort(err)
		}
		if err := replace(stage, change.After); err != nil {
			return abort(err)
		}
	}
	if err := checkPreconditions(p.Preconditions); err != nil {
		return abort(err)
	}
	for i, change := range j.Changes {
		if err := ctx.Err(); err != nil {
			return abort(err)
		}
		current, err := readSnapshot(change.Path)
		if err != nil {
			return abort(err)
		}
		if !current.equal(change.Before) {
			return abort(conflict("destination changed during publication: " + change.Path))
		}
		if change.After.Exists {
			root, err := os.OpenRoot(filepath.Dir(change.Path))
			if err != nil {
				return abort(err)
			}
			err = root.Rename(filepath.Base(j.Stages[change.Path]), filepath.Base(change.Path))
			closeErr := root.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				err = syncDirectory(filepath.Dir(change.Path))
			}
			if err != nil {
				return abort(err)
			}
		} else if err := replace(change.Path, change.After); err != nil {
			return abort(err)
		}
		j.Applied = i + 1
		if err := e.saveJournal(j); err != nil {
			return abort(err)
		}
		if err := e.fail(fmt.Sprintf("published:%d", i)); err != nil {
			return abort(err)
		}
	}
	for _, change := range j.Changes {
		now, err := readSnapshot(change.Path)
		if err != nil {
			return abort(err)
		}
		if !now.equal(change.After) {
			return abort(conflict("published destination changed: " + change.Path))
		}
	}
	j.Committed = true
	if err := e.saveJournal(j); err != nil {
		return abort(err)
	}
	if err := e.fail("committed"); err != nil {
		return Result{Schema: SchemaVersion, Status: "partial"}, errors.Join(err, recoveryRequired())
	}
	if err := e.finish(j); err != nil {
		return Result{Schema: SchemaVersion, Status: "partial"}, errors.Join(err, recoveryRequired())
	}
	return p.Result(false), nil
}

func (e *Engine) cleanupStages(j journal) error {
	var errs []error
	for _, path := range sortedKeys(j.Stages) {
		stage := j.Stages[path]
		if err := safePath(stage); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := os.Remove(stage); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
func (e *Engine) finish(j journal) error {
	if err := e.cleanupStages(j); err != nil {
		return err
	}
	if err := os.Remove(e.pendingPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return syncDirectory(e.stateDir)
}
func (e *Engine) rollback(j journal) error {
	var errs []error
	// Compare every boundary, not only Applied: a crash may occur between a
	// successful rename and the subsequent durable journal progress update.
	for i := len(j.Changes) - 1; i >= 0; i-- {
		change := j.Changes[i]
		now, err := readSnapshot(change.Path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if now.equal(change.Before) {
			continue
		}
		if !now.equal(change.After) {
			errs = append(errs, conflict("recovery preserves externally modified destination: "+change.Path))
			continue
		}
		if err := replace(change.Path, change.Before); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	if err := e.cleanupStages(j); err != nil {
		return err
	}
	sort.Slice(j.Directories, func(i, k int) bool { return len(j.Directories[i]) > len(j.Directories[k]) })
	for _, dir := range j.Directories {
		if err := os.Remove(dir); err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTEMPTY) {
			return err
		}
	}
	return e.finish(j)
}

// Recover rolls back an uncommitted journal, or verifies/finishes a committed
// journal. It never overwrites a third-party edit with an old before-image.
func (e *Engine) Recover(ctx context.Context, dry bool) (Result, error) {
	snap, err := readSnapshot(e.pendingPath())
	if err != nil {
		return Result{}, err
	}
	if !snap.Exists {
		return Result{Schema: SchemaVersion, Status: "no_pending_transaction", DryRun: dry}, nil
	}
	var j journal
	if err := json.Unmarshal(snap.Content, &j); err != nil {
		return Result{}, conflict("invalid setup recovery journal")
	}
	if j.Schema != SchemaVersion {
		return Result{}, conflict("unsupported setup recovery journal version")
	}
	for _, c := range j.Changes {
		if err := safePath(c.Path); err != nil {
			return Result{}, err
		}
	}
	if dry {
		return Result{Schema: SchemaVersion, Status: "recovery_required", DryRun: true}, nil
	}
	unlock, err := e.lock(ctx)
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	fresh, err := readSnapshot(e.pendingPath())
	if err != nil {
		return Result{}, err
	}
	if !fresh.equal(snap) {
		return Result{}, conflict("recovery journal changed")
	}
	if j.Committed {
		for _, c := range j.Changes {
			now, err := readSnapshot(c.Path)
			if err != nil {
				return Result{}, err
			}
			if !now.equal(c.After) {
				return Result{}, conflict("committed setup destination changed: " + c.Path)
			}
		}
		err = e.finish(j)
	} else {
		err = e.rollback(j)
	}
	if err != nil {
		return Result{Schema: SchemaVersion, Status: "partial"}, err
	}
	return Result{Schema: SchemaVersion, Status: "recovered"}, nil
}
