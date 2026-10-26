package write

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func restoreContractFixture(t *testing.T) (*Service, *fakeProvider, []api.Plan) {
	t.Helper()
	s, p, requests := backupReceiptHistory(t, 24)
	put(t, p, ".kb/config.toml", append([]byte("# Preserve this synthetic configuration comment.\n"), get(t, p, ".kb/config.toml")...))
	put(t, p, "nested/evidence.md", []byte("---\n{\"type\":\"Reference\",\"x-count\":9007199254740993,\"x-decimal\":0.1234567890123456789}\n---\n<!-- Preserve this synthetic comment. -->\r\nOriginal evidence.\r\n"))
	put(t, p, "raw/source.bin", []byte{0, 255, 128, 1})
	return s, p, requests
}

// Ignore only disposable state and operational outputs, not config or receipts.
func restoreContractFiles(t *testing.T, root string) map[string][]byte {
	t.Helper()
	fixture, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := fixture.Close(); err != nil {
			t.Error(err)
		}
	}()
	files := map[string][]byte{}
	err = filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		switch relative {
		case ".kb/backups", ".kb/restores", ".kb/cache":
			return fs.SkipDir
		case ".kb/state/lock":
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			t.Fatalf("unexpected nonregular authoritative file %s", relative)
		}
		data, err := fixture.ReadFile(relative)
		if err != nil {
			return err
		}
		files[relative] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestRestoreContractPrivateOutputsRemainUsable(t *testing.T) {
	s, p, requests := restoreContractFixture(t)
	want := restoreContractFiles(t, p.root)
	archive := api.BackupInput{Archive: ".kb/backups/private.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	archiveBytes := get(t, p, archive.Archive)
	destination := ".kb/restores/private"
	if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: destination}); err != nil {
		t.Fatal(err)
	}
	copied := *p
	copied.root = filepath.Join(p.root, destination)
	if got := restoreContractFiles(t, copied.root); !reflect.DeepEqual(got, want) {
		t.Fatal("private restore changed authoritative paths, metadata, comments, config or receipts")
	}
	if err := filepath.WalkDir(copied.root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		// A restrictive umask may remove owner bits; it must never grant extra ones.
		allowed := os.FileMode(0600)
		if entry.IsDir() {
			allowed = 0700
		} else if !info.Mode().IsRegular() {
			t.Fatalf("restored nonregular file %s", name)
		}
		if info.Mode().Perm() & ^allowed != 0 || info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
			t.Fatalf("restored path has nonprivate permissions: %s %v", name, info.Mode())
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, len(requests) / 2, len(requests) - 1} {
		original, err := s.Apply(context.Background(), requests[index])
		if err != nil {
			t.Fatal(err)
		}
		replayed, err := New(&copied).Apply(context.Background(), requests[index])
		if err != nil || !original.Replayed || !reflect.DeepEqual(original, replayed) {
			t.Fatalf("private restored receipt could not replay exactly: %v", err)
		}
	}
	if got := restoreContractFiles(t, copied.root); !reflect.DeepEqual(got, want) {
		t.Fatal("replaying restored requests changed authoritative bytes")
	}
	continued := create("nested/continued.md")
	continued.Operations[0].Body = body("Continued write after private restore\n")
	if _, err := New(&copied).Apply(context.Background(), continued); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(get(t, &copied, "nested/continued.md"), []byte("Continued write after private restore\n")) {
		t.Fatal("private restored root did not persist a normal write")
	}
	if !bytes.Equal(get(t, &copied, "nested/evidence.md"), want["nested/evidence.md"]) || !bytes.Equal(get(t, &copied, ".kb/config.toml"), want[".kb/config.toml"]) {
		t.Fatal("continued writes changed preserved metadata, comments or configuration")
	}
	if got := restoreContractFiles(t, p.root); !reflect.DeepEqual(got, want) || !bytes.Equal(get(t, p, archive.Archive), archiveBytes) {
		t.Fatal("restored replay or writes changed original authority or archive")
	}
}

func TestRestoreContractLateCancellationCleansOwnedState(t *testing.T) {
	s, p, _ := restoreContractFixture(t)
	want := restoreContractFiles(t, p.root)
	archive := api.BackupInput{Archive: ".kb/backups/cancel.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	archiveBytes := get(t, p, archive.Archive)
	destination := ".kb/restores/cancelled"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	extracted := 0
	s.fault = func(stage, name string) error {
		if stage == "restore-file" && strings.HasPrefix(name, ".kb/state/receipts/") {
			extracted++
			if extracted == 18 {
				if !bytes.Equal(get(t, p, destination+"/"+name), want[name]) {
					t.Fatal("late cancellation did not observe the extracted receipt bytes")
				}
				if _, err := os.Lstat(filepath.Join(p.root, destination, ".kb/config.toml")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("configuration was exposed before cancellation: %v", err)
				}
				cancel()
			}
		}
		return nil
	}
	_, err := s.Restore(ctx, api.RestoreInput{Archive: archive.Archive, Destination: destination})
	s.fault = nil
	if !errors.Is(err, context.Canceled) || extracted != 18 {
		t.Fatalf("late extraction cancellation was not propagated: receipts=%d err=%v", extracted, err)
	}
	if _, err := os.Lstat(filepath.Join(p.root, destination, ".kb/config.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled restore exposed configuration: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(p.root, destination)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled restore retained its owned partial destination: %v", err)
	}
	if got := restoreContractFiles(t, p.root); !reflect.DeepEqual(got, want) || !bytes.Equal(get(t, p, archive.Archive), archiveBytes) {
		t.Fatal("cancellation changed original authority or archive")
	}
	// Nonblocking acquisition proves release without a wall-time performance gate.
	lock, err := os.OpenFile(filepath.Join(p.root, ".kb/state/lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lock.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("cancelled restore retained the source lock: %v", err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	retry := ".kb/restores/retry-after-cancel"
	if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: retry}); err != nil {
		t.Fatal(err)
	}
	if got := restoreContractFiles(t, filepath.Join(p.root, retry)); !reflect.DeepEqual(got, want) {
		t.Fatal("fresh restore after cancellation lost authoritative bytes")
	}
	if got := restoreContractFiles(t, p.root); !reflect.DeepEqual(got, want) || !bytes.Equal(get(t, p, archive.Archive), archiveBytes) {
		t.Fatal("fresh retry changed original authority or archive")
	}
}

func TestRestoreContractRejectsOutsideOnlyHardlink(t *testing.T) {
	s, p, requests := restoreContractFixture(t)
	want := restoreContractFiles(t, p.root)
	archive := api.BackupInput{Archive: ".kb/backups/hardlink.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	archiveBytes := get(t, p, archive.Archive)
	destination := ".kb/restores/hardlinked"
	name := receiptPath(requests[len(requests)/2].IdempotencyKey)
	outside := t.TempDir()
	alias := filepath.Join(outside, "receipt-alias.json")
	injected := false
	s.fault = func(stage, _ string) error {
		if stage != "restore-validated" {
			return nil
		}
		copied := filepath.Join(p.root, destination)
		before := restoreContractFiles(t, copied)
		expected := make(map[string][]byte, len(want)-1)
		for path, raw := range want {
			if path != ".kb/config.toml" {
				expected[path] = raw
			}
		}
		if !reflect.DeepEqual(before, expected) {
			t.Fatal("hardlink injection did not reach an otherwise exact prepublication inventory")
		}
		if err := os.Link(filepath.Join(copied, name), alias); err != nil {
			return err
		}
		injected = true
		if got := restoreContractFiles(t, copied); !reflect.DeepEqual(got, before) {
			t.Fatal("outside-only hardlink unexpectedly changed the restored path/byte inventory")
		}
		return nil
	}
	_, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: destination})
	s.fault = nil
	if !injected {
		t.Fatalf("outside hardlink injection was not reached: %v", err)
	}
	errorCode(t, err, "INVALID_BACKUP")
	if _, err := os.Lstat(filepath.Join(p.root, destination, ".kb/config.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("hardlinked restore exposed configuration: %v", err)
	}
	outsideFiles := restoreContractFiles(t, outside)
	if !reflect.DeepEqual(outsideFiles, map[string][]byte{"receipt-alias.json": want[name]}) {
		t.Fatal("restore rejection or cleanup changed the outside hardlink alias")
	}
	if got := restoreContractFiles(t, p.root); !reflect.DeepEqual(got, want) || !bytes.Equal(get(t, p, archive.Archive), archiveBytes) {
		t.Fatal("hardlink rejection or cleanup changed original authority or archive")
	}
}
