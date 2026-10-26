package write

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

// Produce every receipt through the transaction service, not by copying or
// synthesizing receipt JSON. Only the small note corpus is reused between writes.
func backupReceiptHistory(t testing.TB, count int) (*Service, *fakeProvider, []api.Plan) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{root: root}
	p.config.Write.Enabled = true
	p.config.Write.MaintainIndexes = true
	s := New(p)
	if err := os.MkdirAll(filepath.Join(root, ".kb"), 0700); err != nil {
		t.Fatal(err)
	}
	config := []byte("version = 1\n[write]\nenabled = true\nmaintain_indexes = true\nmaintain_log = false\n")
	if err := os.WriteFile(filepath.Join(root, ".kb/config.toml"), config, 0600); err != nil {
		t.Fatal(err)
	}
	requests := make([]api.Plan, 0, count)
	var revision string
	for i := range count {
		op := api.Operation{Kind: "patch", Path: "note.md", ExpectedRevision: revision, Body: body(fmt.Sprintf("Synthetic receipt effect %06d\n", i))}
		if i == 0 {
			op.Kind, op.Type = "create", "Reference"
		}
		request := plan(op)
		request.IdempotencyKey = fmt.Sprintf("backup-history/%d", i)
		receipt, err := s.Apply(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		for _, change := range receipt.Changes {
			if change.Path == op.Path {
				revision = change.AfterRevision
			}
		}
		requests = append(requests, request)
	}
	return s, p, requests
}

func TestBackupReceiptHistoryPreservesEveryConfirmedReceipt(t *testing.T) {
	s, p, requests := backupReceiptHistory(t, 128)
	archive := api.BackupInput{Archive: ".kb/backups/history.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	destination := ".kb/restores/history"
	if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: destination}); err != nil {
		t.Fatal(err)
	}
	copied := *p
	copied.root = filepath.Join(p.root, destination)
	for _, request := range requests {
		name := receiptPath(request.IdempotencyKey)
		if !bytes.Equal(get(t, p, name), get(t, &copied, name)) {
			t.Fatal("restoration changed a confirmed durable receipt")
		}
		replayed, err := New(&copied).Apply(context.Background(), request)
		if err != nil || !replayed.Replayed {
			t.Fatalf("confirmed request failed to replay: %v", err)
		}
	}
	if _, err := New(&copied).Apply(context.Background(), create("continued.md")); err != nil {
		t.Fatal(err)
	}
}

func TestBackupPublicationRevalidatesReceiptBytesAndSpellings(t *testing.T) {
	for _, change := range []string{"same-size-same-mtime", "renamed-leaf-case", "renamed-parent-case", "symlink-parent", "hardlink", "added-receipt", "removed-receipt", "added-file"} {
		t.Run(change, func(t *testing.T) {
			s, p, requests := backupReceiptHistory(t, 32)
			name := receiptPath(requests[0].IdempotencyKey)
			injected := false
			s.fault = func(stage, _ string) error {
				if stage != "backup-synced" {
					return nil
				}
				err := changeBackupReceipt(p.root, name, change)
				injected = err == nil
				return err
			}
			_, err := s.Backup(context.Background(), api.BackupInput{Archive: ".kb/backups/raced.tar"})
			if !injected {
				t.Fatalf("synthetic source change failed: %v", err)
			}
			if err == nil {
				t.Fatal("backup published a stale receipt inventory")
			}
			if _, err := os.Lstat(filepath.Join(p.root, ".kb/backups/raced.tar")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed revalidation published an archive: %v", err)
			}
		})
	}
}

func readBackupFixture(root, name string) (data []byte, err error) {
	handle, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, handle.Close()) }()
	return handle.ReadFile(name)
}

func changeBackupReceipt(root, name, change string) (err error) {
	handle, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, handle.Close()) }()
	file := filepath.Join(root, name)
	switch change {
	case "same-size-same-mtime":
		info, err := handle.Stat(name)
		if err != nil {
			return err
		}
		raw, err := handle.ReadFile(name)
		if err != nil {
			return err
		}
		raw[len(raw)-1] ^= 1
		if err := handle.WriteFile(name, raw, 0600); err != nil {
			return err
		}
		return handle.Chtimes(name, info.ModTime(), info.ModTime())
	case "renamed-leaf-case":
		return os.Rename(file, strings.TrimSuffix(file, ".json")+".JSON")
	case "renamed-parent-case":
		parent := filepath.Dir(file)
		return os.Rename(parent, filepath.Join(filepath.Dir(parent), "Receipts"))
	case "symlink-parent":
		parent := filepath.Dir(file)
		if err := os.Rename(parent, parent+"-moved"); err != nil {
			return err
		}
		return os.Symlink("receipts-moved", parent)
	case "hardlink":
		return os.Link(file, filepath.Join(root, "external-receipt-link.bin"))
	case "added-receipt":
		return os.WriteFile(filepath.Join(root, receiptPath("external-receipt")), []byte("external bytes"), 0600)
	case "removed-receipt":
		return os.Remove(file)
	case "added-file":
		return os.WriteFile(filepath.Join(root, "external-new.bin"), []byte("external source"), 0600)
	default:
		return errors.New("unknown synthetic receipt change")
	}
}

func TestRestoreBatchedValidationPreservesExternalReceiptChanges(t *testing.T) {
	for _, change := range []string{"same-size-same-mtime", "renamed-leaf-case", "renamed-parent-case", "symlink-parent", "hardlink", "added-receipt", "removed-receipt", "added-file"} {
		t.Run(change, func(t *testing.T) {
			s, p, requests := backupReceiptHistory(t, 32)
			archive := api.BackupInput{Archive: ".kb/backups/history.tar"}
			if _, err := s.Backup(context.Background(), archive); err != nil {
				t.Fatal(err)
			}
			original := get(t, p, receiptPath(requests[0].IdempotencyKey))
			destination := ".kb/restores/raced"
			copied := filepath.Join(p.root, destination)
			injected := false
			s.fault = func(stage, _ string) error {
				if stage == "restore-validated" {
					err := changeBackupReceipt(copied, receiptPath(requests[0].IdempotencyKey), change)
					injected = err == nil
					return err
				}
				return nil
			}
			_, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: destination})
			if !injected {
				t.Fatalf("synthetic destination change failed: %v", err)
			}
			if err == nil {
				t.Fatal("restore published a changed receipt inventory")
			}
			if _, err := os.Lstat(filepath.Join(copied, ".kb/config.toml")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed restoration exposed configuration: %v", err)
			}
			if !bytes.Equal(original, get(t, p, receiptPath(requests[0].IdempotencyKey))) {
				t.Fatal("restoration changed the source receipt")
			}
			if change == "same-size-same-mtime" {
				want := append([]byte(nil), original...)
				want[len(want)-1] ^= 1
				if got, err := readBackupFixture(copied, receiptPath(requests[0].IdempotencyKey)); err != nil || !bytes.Equal(got, want) {
					t.Fatal("cleanup erased external receipt bytes")
				}
			}
		})
	}
}

func TestRestoreReceiptHistoryCleansOwnedFilesAfterFault(t *testing.T) {
	s, p, _ := backupReceiptHistory(t, 128)
	archive := api.BackupInput{Archive: ".kb/backups/history.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	seen := 0
	fault := errors.New("interrupt receipt extraction")
	s.fault = func(stage, name string) error {
		if stage == "restore-file" && strings.HasPrefix(name, ".kb/state/receipts/") {
			seen++
			if seen == 96 {
				return fault
			}
		}
		return nil
	}
	_, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/failed"})
	if !errors.Is(err, fault) {
		t.Fatalf("extraction boundary was not reached: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(p.root, ".kb/restores/failed")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned partial destination survived ordinary failure: %v", err)
	}
}

func BenchmarkBackupReceiptHistory(b *testing.B) {
	for _, count := range []int{250, 2000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			s, p, _ := backupReceiptHistory(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				archive := fmt.Sprintf(".kb/backups/bench-%d.tar", i)
				destination := fmt.Sprintf(".kb/restores/bench-%d", i)
				backed, err := s.Backup(context.Background(), api.BackupInput{Archive: archive})
				if err != nil {
					b.Fatal(err)
				}
				restored, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive, Destination: destination})
				if err != nil || backed.ManifestRevision != restored.ManifestRevision {
					b.Fatalf("restore differs: %v", err)
				}
				b.StopTimer()
				if err := os.RemoveAll(filepath.Join(p.root, destination)); err != nil {
					b.Fatal(err)
				}
				if err := os.Remove(filepath.Join(p.root, archive)); err != nil {
					b.Fatal(err)
				}
				b.StartTimer()
			}
		})
	}
}
