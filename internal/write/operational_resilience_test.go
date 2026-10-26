package write

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
)

func TestBoundedMixedOperationalWorkload(t *testing.T) {
	s, p := backupSetup(t)
	ctx := context.Background()
	source := get(t, p, "raw/source.bin")
	for i := range 24 {
		name := fmt.Sprintf("mixed/note-%02d.md", i)
		createRequest := create(name)
		createRequest.IdempotencyKey = fmt.Sprintf("mixed/create/%d", i)
		if _, err := s.Apply(ctx, createRequest); err != nil {
			t.Fatal(err)
		}
		patch := plan(api.Operation{Kind: "patch", Path: name, ExpectedRevision: api.Revision(get(t, p, name)), Body: body(fmt.Sprintf("Iteration %d\n", i))})
		patch.IdempotencyKey = fmt.Sprintf("mixed/patch/%d", i)
		patch.DryRun = true
		before := inventory(t, p)
		if _, err := s.Apply(ctx, patch); err != nil {
			t.Fatal(err)
		}
		equalInventory(t, before, inventory(t, p))
		patch.DryRun = false
		if _, err := s.Apply(ctx, patch); err != nil {
			t.Fatal(err)
		}
		if replay, err := s.Apply(ctx, patch); err != nil || !replay.Replayed {
			t.Fatalf("replay: %v %+v", err, replay)
		}
		conflict := patch
		conflict.IdempotencyKey += "/stale"
		conflict.Operations = append([]api.Operation(nil), patch.Operations...)
		conflict.Operations[0].Body = body("stale overwrite")
		before = inventory(t, p)
		_, err := s.Apply(ctx, conflict)
		errorCode(t, err, "REVISION_CONFLICT")
		equalInventory(t, before, inventory(t, p))
		move := plan(api.Operation{Kind: "move", Path: name, To: fmt.Sprintf("archive/note-%02d.md", i), ExpectedRevision: api.Revision(get(t, p, name))})
		move.IdempotencyKey = fmt.Sprintf("mixed/move/%d", i)
		if _, err := s.Apply(ctx, move); err != nil {
			t.Fatal(err)
		}
		if i%6 == 5 {
			archive := api.BackupInput{Archive: fmt.Sprintf(".kb/backups/mixed-%d.tar", i)}
			if _, err := s.Backup(ctx, archive); err != nil {
				t.Fatal(err)
			}
			dest := fmt.Sprintf(".kb/restores/mixed-%d", i)
			if _, err := s.Restore(ctx, api.RestoreInput{Archive: archive.Archive, Destination: dest}); err != nil {
				t.Fatal(err)
			}
			copied := *p
			copied.root = filepath.Join(p.root, dest)
			equalInventory(t, inventory(t, p), inventory(t, &copied))
			if receipt, err := New(&copied).Apply(ctx, move); err != nil || !receipt.Replayed {
				t.Fatalf("restored replay: %v %+v", err, receipt)
			}
		}
	}
	if !reflect.DeepEqual(source, get(t, p, "raw/source.bin")) {
		t.Fatal("mixed workload changed original asset")
	}
}

// This helper runs in a real child process. The parent sends SIGKILL only after
// a durable boundary has been observed, not after an arbitrary timing delay.
func TestOperationalInterruptionChild(t *testing.T) {
	root := os.Getenv("KB_TEST_INTERRUPT_ROOT")
	if root == "" {
		return
	}
	p := &fakeProvider{root: root}
	p.config.Write.Enabled = true
	p.config.Write.MaintainIndexes = true
	p.config.Write.MaintainLog = true
	s := New(p)
	stage := os.Getenv("KB_TEST_INTERRUPT_STAGE")
	position := 1
	if value := os.Getenv("KB_TEST_INTERRUPT_POSITION"); value != "" {
		var err error
		position, err = strconv.Atoi(value)
		if err != nil || position < 1 {
			t.Fatalf("invalid boundary position: %q", value)
		}
	}
	seen := 0
	s.fault = func(at, name string) error {
		if at != stage {
			return nil
		}
		seen++
		if seen == position {
			handle, err := os.OpenRoot(root)
			if err != nil {
				return err
			}
			err = errors.Join(handle.WriteFile(".kb/boundary-ready", []byte(at+":"+name), 0600), handle.Close())
			if err != nil {
				return err
			}
			for {
				time.Sleep(time.Hour)
			}
		}
		return nil
	}
	var err error
	switch {
	case strings.HasPrefix(stage, "backup-"):
		_, err = s.Backup(context.Background(), api.BackupInput{Archive: ".kb/backups/interrupted.tar"})
	case strings.HasPrefix(stage, "restore-"):
		_, err = s.Restore(context.Background(), api.RestoreInput{Archive: ".kb/backups/interrupted.tar", Destination: ".kb/restores/interrupted"})
	case stage == "recovered":
		_, err = s.Recover(context.Background(), false)
	case os.Getenv("KB_TEST_INTERRUPT_MULTI") == "1":
		_, err = s.Apply(context.Background(), interruptionPlan())
	default:
		_, err = s.Apply(context.Background(), create("interrupted/new.md"))
	}
	t.Fatalf("boundary was not reached: %v", err)
}

func TestRealProcessInterruptionRecoveryAndOperationalArchives(t *testing.T) {
	for _, stage := range []string{"prepared", "published", "committed", "receipt", "removed", "recovered", "backup-synced", "backup-published", "restore-file", "restore-validated", "restore-published"} {
		t.Run(stage, func(t *testing.T) {
			s, p := backupSetup(t)
			if strings.HasPrefix(stage, "restore-") {
				if _, err := s.Backup(context.Background(), api.BackupInput{Archive: ".kb/backups/interrupted.tar"}); err != nil {
					t.Fatal(err)
				}
			}
			if stage == "recovered" {
				s.fault = func(at, _ string) error {
					if at == "published" {
						return errors.New("prepare recovery interruption")
					}
					return nil
				}
				if _, err := s.Apply(context.Background(), create("interrupted/new.md")); err == nil {
					t.Fatal("expected pending journal")
				}
				s.fault = nil
			}
			killAtOperationalBoundary(t, p.root, stage, 1, false)
			switch {
			case strings.HasPrefix(stage, "backup-"):
				if stage == "backup-synced" {
					if _, err := s.Backup(context.Background(), api.BackupInput{Archive: ".kb/backups/interrupted.tar"}); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: ".kb/backups/interrupted.tar", Destination: ".kb/restores/after-kill"}); err != nil {
					t.Fatal(err)
				}
			case strings.HasPrefix(stage, "restore-"):
				_, err := os.Stat(filepath.Join(p.root, ".kb/restores/interrupted/.kb/config.toml"))
				if stage == "restore-published" {
					if err != nil {
						t.Fatal(err)
					}
				} else if !errors.Is(err, os.ErrNotExist) {
					t.Fatal("incomplete restore exposed configuration")
				}
				_, err = s.Restore(context.Background(), api.RestoreInput{Archive: ".kb/backups/interrupted.tar", Destination: ".kb/restores/interrupted"})
				errorCode(t, err, "DESTINATION_EXISTS")
				if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: ".kb/backups/interrupted.tar", Destination: ".kb/restores/retry-after-kill"}); err != nil {
					t.Fatal(err)
				}
			default:
				if _, err := s.Recover(context.Background(), false); err != nil {
					t.Fatal(err)
				}
				replay, err := s.Apply(context.Background(), create("interrupted/new.md"))
				if err != nil {
					t.Fatal(err)
				}
				wantReplay := stage == "committed" || stage == "receipt" || stage == "removed"
				if replay.Replayed != wantReplay {
					t.Fatalf("replay=%v want %v", replay.Replayed, wantReplay)
				}
				if _, err := s.Apply(context.Background(), create("interrupted/new.md")); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(get(t, p, "raw/source.bin"), []byte{0, 1, 2, 255}) {
				t.Fatal("source changed by interruption")
			}
		})
	}
}

func killAtOperationalBoundary(t *testing.T, root, stage string, position int, multi bool) string {
	t.Helper()
	handle, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := handle.Close(); err != nil {
			t.Error(err)
		}
	})
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(executable, "-test.run=^TestOperationalInterruptionChild$", "-test.timeout=30s") // #nosec G204 -- Re-executes this test binary with fixed test arguments.
	child.Env = append(os.Environ(), "KB_TEST_INTERRUPT_ROOT="+root, "KB_TEST_INTERRUPT_STAGE="+stage,
		"KB_TEST_INTERRUPT_POSITION="+strconv.Itoa(position), "KB_TEST_INTERRUPT_MULTI="+strconv.FormatBool(multi))
	if multi {
		child.Env[len(child.Env)-1] = "KB_TEST_INTERRUPT_MULTI=1"
	}
	output := &strings.Builder{}
	child.Stdout, child.Stderr = output, output
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	go func() { finished <- child.Wait() }()
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	marker := filepath.Join(root, ".kb/boundary-ready")
	var observed string
	for observed == "" {
		select {
		case err := <-finished:
			t.Fatalf("child exited before %s/%d: %v %s", stage, position, err, output.String())
		case <-deadline.C:
			_ = child.Process.Kill()
			waitErr := <-finished
			t.Fatalf("child did not reach %s/%d: %v %s", stage, position, waitErr, output.String())
		case <-ticker.C:
			raw, err := handle.ReadFile(".kb/boundary-ready")
			if err == nil && strings.HasPrefix(string(raw), stage+":") {
				observed = strings.TrimPrefix(string(raw), stage+":")
			}
		}
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := <-finished; err == nil {
		t.Fatal("child was not interrupted")
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	return observed
}

func interruptionPlan() api.Plan {
	return api.Plan{Actor: "process:interruption-test", Summary: "Publish coordinated synthetic evidence", IdempotencyKey: "multi-interruption",
		Operations: []api.Operation{
			{OperationID: "a", Kind: "create", Path: "interrupted/a.md", Type: "Concept", Body: body("Confirmed A\n")},
			{OperationID: "b", Kind: "create", Path: "interrupted/b.md", Type: "Concept", Body: body("Confirmed B\n")},
			{OperationID: "c", Kind: "create", Path: "interrupted/c.md", Type: "Concept", Body: body("Confirmed C\n")},
		}}
}

func TestRealProcessInterruptionEveryPublicationAndRecoveryPosition(t *testing.T) {
	probe, _ := backupSetup(t)
	preview := interruptionPlan()
	preview.DryRun = true
	planned, err := probe.Apply(context.Background(), preview)
	if err != nil {
		t.Fatal(err)
	}
	if len(planned.Changes) < 6 {
		t.Fatalf("fixture needs multiple notes, navigation and log: %+v", planned.Changes)
	}
	for _, mode := range []string{"publication", "rollback", "rollforward"} {
		for position := 1; position <= len(planned.Changes); position++ {
			t.Run(fmt.Sprintf("%s/%d", mode, position), func(t *testing.T) {
				s, p := backupSetup(t)
				acknowledged := create("acknowledged.md")
				acknowledged.IdempotencyKey = "prior-acknowledged"
				ack, err := s.Apply(context.Background(), acknowledged)
				if err != nil {
					t.Fatal(err)
				}
				ackBytes := get(t, p, receiptPath(acknowledged.IdempotencyKey))
				before := inventory(t, p)
				stage := "published"
				if mode != "publication" {
					failAt := "published"
					if mode == "rollforward" {
						failAt = "committed"
					}
					s.fault = func(at, _ string) error {
						if at == failAt {
							return errors.New("prepare interrupted recovery")
						}
						return nil
					}
					if _, err := s.Apply(context.Background(), interruptionPlan()); err == nil {
						t.Fatal("expected interrupted transaction")
					}
					s.fault = nil
					stage = "recovered"
				}
				name := killAtOperationalBoundary(t, p.root, stage, position, true)
				if name != planned.Changes[position-1].Path {
					t.Fatalf("wrong observed boundary: got %s want %s", name, planned.Changes[position-1].Path)
				}
				if _, err := s.Recover(context.Background(), false); err != nil {
					t.Fatal(err)
				}
				if mode != "rollforward" {
					equalInventory(t, before, inventory(t, p))
				}
				result, err := s.Apply(context.Background(), interruptionPlan())
				if err != nil || result.Replayed != (mode == "rollforward") {
					t.Fatalf("retry after %s: %v %+v", mode, err, result)
				}
				for _, change := range result.Changes {
					if got := api.Revision(get(t, p, change.Path)); got != change.AfterRevision {
						t.Fatalf("partial confirmed effect at %s: %s != %s", change.Path, got, change.AfterRevision)
					}
				}
				for _, op := range interruptionPlan().Operations {
					if !strings.HasSuffix(string(get(t, p, op.Path)), *op.Body) {
						t.Fatalf("missing coordinated body at %s", op.Path)
					}
				}
				if strings.Count(string(get(t, p, "log.md")), interruptionPlan().Summary) != 1 {
					t.Fatal("missing or duplicated transaction log effect")
				}
				settled := inventory(t, p)
				replay, err := s.Apply(context.Background(), interruptionPlan())
				result.Replayed = true
				if err != nil || !reflect.DeepEqual(result, replay) {
					t.Fatalf("retry receipt changed: %v %+v", err, replay)
				}
				prior, err := s.Apply(context.Background(), acknowledged)
				ack.Replayed = true
				if err != nil || !reflect.DeepEqual(ack, prior) || !reflect.DeepEqual(ackBytes, get(t, p, receiptPath(acknowledged.IdempotencyKey))) {
					t.Fatalf("lost prior acknowledged receipt: %v %+v", err, prior)
				}
				equalInventory(t, settled, inventory(t, p))
				if !reflect.DeepEqual(get(t, p, "raw/source.bin"), []byte{0, 1, 2, 255}) {
					t.Fatal("interruption changed immutable source")
				}
			})
		}
	}
}

func TestRealProcessInterruptionEveryRestoreFilePosition(t *testing.T) {
	fixture := func(t *testing.T) (*Service, *fakeProvider, api.Plan, api.Receipt) {
		t.Helper()
		s, p := backupSetup(t)
		request := plan(api.Operation{Kind: "patch", Path: "note.md",
			ExpectedRevision: api.Revision(get(t, p, "note.md")), Body: body("Acknowledged before restore\n")})
		ack, err := s.Apply(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		return s, p, request, ack
	}
	probe, _, _, _ := fixture(t)
	probeArchive, err := probe.Backup(context.Background(), api.BackupInput{Archive: ".kb/backups/probe.tar"})
	if err != nil {
		t.Fatal(err)
	}
	// Configuration is deferred until restore-published; all other included files
	// expose a restore-file handshake, including receipt and source bytes.
	seen := map[string]bool{}
	for position := 1; position < probeArchive.Files; position++ {
		t.Run(strconv.Itoa(position), func(t *testing.T) {
			s, p, request, ack := fixture(t)
			before := inventory(t, p)
			receiptBytes := get(t, p, receiptPath(request.IdempotencyKey))
			if _, err := s.Backup(context.Background(), api.BackupInput{Archive: ".kb/backups/interrupted.tar"}); err != nil {
				t.Fatal(err)
			}
			name := killAtOperationalBoundary(t, p.root, "restore-file", position, false)
			if seen[name] {
				t.Fatalf("duplicate restore position: %s", name)
			}
			seen[name] = true
			if _, err := os.Stat(filepath.Join(p.root, ".kb/restores/interrupted/.kb/config.toml")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("partial restore exposed configuration")
			}
			dest := ".kb/restores/retry-after-kill"
			if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: ".kb/backups/interrupted.tar", Destination: dest}); err != nil {
				t.Fatal(err)
			}
			copied := *p
			copied.root = filepath.Join(p.root, dest)
			equalInventory(t, before, inventory(t, &copied))
			ack.Replayed = true
			replay, err := New(&copied).Apply(context.Background(), request)
			if err != nil || !reflect.DeepEqual(ack, replay) || !reflect.DeepEqual(receiptBytes, get(t, &copied, receiptPath(request.IdempotencyKey))) {
				t.Fatalf("restore lost acknowledged receipt: %v %+v", err, replay)
			}
			equalInventory(t, before, inventory(t, p))
			for _, provider := range []*fakeProvider{p, &copied} {
				if !reflect.DeepEqual(get(t, provider, "raw/source.bin"), []byte{0, 1, 2, 255}) {
					t.Fatal("restore interruption changed source")
				}
				if !reflect.DeepEqual(receiptBytes, get(t, provider, receiptPath(request.IdempotencyKey))) {
					t.Fatal("restore interruption changed acknowledged receipt")
				}
			}
		})
	}
	if !seen["raw/source.bin"] || !seen["note.md"] {
		t.Fatalf("missing restore positions: %v", seen)
	}
	receiptSeen := false
	for name := range seen {
		receiptSeen = receiptSeen || strings.HasPrefix(name, ".kb/state/receipts/")
	}
	if !receiptSeen {
		t.Fatal("no acknowledged receipt restore position was interrupted")
	}
}
