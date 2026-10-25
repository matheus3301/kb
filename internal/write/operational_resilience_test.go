package write

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
	s.fault = func(at, _ string) error {
		if at == stage {
			handle, err := os.OpenRoot(root)
			if err != nil {
				return err
			}
			err = errors.Join(handle.WriteFile(".kb/boundary-ready", []byte(at), 0600), handle.Close())
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
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			child := exec.Command(executable, "-test.run=^TestOperationalInterruptionChild$", "-test.timeout=30s") // #nosec G204 -- Re-executes only this test executable with fixed test arguments.
			child.Env = append(os.Environ(), "KB_TEST_INTERRUPT_ROOT="+p.root, "KB_TEST_INTERRUPT_STAGE="+stage)
			output := &strings.Builder{}
			child.Stdout = output
			child.Stderr = output
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			finished := make(chan error, 1)
			go func() { finished <- child.Wait() }()
			deadline := time.NewTimer(15 * time.Second)
			defer deadline.Stop()
			ticker := time.NewTicker(5 * time.Millisecond)
			defer ticker.Stop()
			ready := false
			for !ready {
				select {
				case err := <-finished:
					t.Fatalf("child exited before boundary: %v %s", err, output.String())
				case <-deadline.C:
					_ = child.Process.Kill()
					waitErr := <-finished
					t.Fatalf("child did not reach boundary: %v %s", waitErr, output.String())
				case <-ticker.C:
					_, err := os.Stat(filepath.Join(p.root, ".kb/boundary-ready"))
					ready = err == nil
				}
			}
			if err := child.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			if err := <-finished; err == nil {
				t.Fatal("child was not interrupted")
			}
			if err := os.Remove(filepath.Join(p.root, ".kb/boundary-ready")); err != nil {
				t.Fatal(err)
			}
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
