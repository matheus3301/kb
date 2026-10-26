package write

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func TestRestoreRejectsReceiptParentSymlinkDuringExtraction(t *testing.T) {
	s, p, _ := backupReceiptHistory(t, 32)
	archive := api.BackupInput{Archive: ".kb/backups/history.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "sentinel"), []byte("unrelated external bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	injected := false
	s.fault = func(stage, name string) error {
		if injected || stage != "restore-file" || !strings.HasPrefix(name, ".kb/state/receipts/") {
			return nil
		}
		injected = true
		parent := filepath.Join(p.root, ".kb/restores/raced/.kb/state/receipts")
		if err := os.Rename(parent, parent+"-kept"); err != nil {
			return err
		}
		return os.Symlink(outside, parent)
	}
	_, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/raced"})
	if err == nil || !injected {
		t.Fatal("receipt parent replacement did not interrupt extraction")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel" {
		t.Fatalf("restore or cleanup wrote outside its destination: %v", err)
	}
	data, err := readBackupFixture(outside, "sentinel")
	if err != nil || string(data) != "unrelated external bytes" {
		t.Fatal("restore or cleanup changed external data")
	}
	if _, err := os.Lstat(filepath.Join(p.root, ".kb/restores/raced/.kb/config.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed extraction published configuration: %v", err)
	}
}

func TestRestoreRejectsInjectedConfigurationCaseAlias(t *testing.T) {
	s, p, _ := backupReceiptHistory(t, 16)
	archive := api.BackupInput{Archive: ".kb/backups/history.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(p.root, ".kb/restores/raced/.kb/CONFIG.toml")
	s.fault = func(stage, _ string) error {
		if stage == "restore-validated" {
			return os.WriteFile(foreign, []byte("external configuration bytes"), 0600)
		}
		return nil
	}
	if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/raced"}); err == nil {
		t.Fatal("case-aliased external configuration was overwritten or published")
	}
	data, err := readBackupFixture(filepath.Join(p.root, ".kb/restores/raced"), ".kb/CONFIG.toml")
	if err != nil || string(data) != "external configuration bytes" {
		t.Fatal("failed restore erased an external case alias")
	}
}

func TestBackupParentRechecksIdentityAfterUse(t *testing.T) {
	for _, replacement := range []string{"same-byte-directory", "in-root-symlink", "outside-symlink"} {
		t.Run(replacement, func(t *testing.T) {
			_, p, requests := backupReceiptHistory(t, 2)
			name := receiptPath(requests[0].IdempotencyKey)
			root, err := os.OpenRoot(p.root)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = root.Close() }()
			opened, err := backupParent(root, name, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = opened.Close() }()
			old := get(t, p, name)
			parent := filepath.Join(p.root, ".kb/state/receipts")
			if err := os.Rename(parent, parent+"-kept"); err != nil {
				t.Fatal(err)
			}
			switch replacement {
			case "same-byte-directory":
				if err := os.Mkdir(parent, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(parent, filepath.Base(name)), old, 0600); err != nil {
					t.Fatal(err)
				}
			case "in-root-symlink":
				if err := os.Symlink("receipts-kept", parent); err != nil {
					t.Fatal(err)
				}
			case "outside-symlink":
				if err := os.Symlink(t.TempDir(), parent); err != nil {
					t.Fatal(err)
				}
			}
			if err := backupParentIdentity(root, name, opened); err == nil {
				t.Fatal("a detached receipt-directory handle remained trusted")
			}
			if replacement != "same-byte-directory" {
				if _, err := backupFile(root, name); err == nil {
					t.Fatal("a fresh read followed a forbidden parent symlink")
				}
			} else {
				fresh, err := backupFile(root, name)
				if err != nil || string(fresh) != string(old) {
					t.Fatal("a new request inherited stale parent identity")
				}
			}
		})
	}
}

func TestBackupInventoryDetectsSameByteDirectoryReplacement(t *testing.T) {
	_, p, requests := backupReceiptHistory(t, 2)
	root, err := os.OpenRoot(p.root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	name := receiptPath(requests[0].IdempotencyKey)
	injected := false
	err = walkBackupInventory(context.Background(), root, func(dir *os.Root, current string, entry fs.DirEntry) error {
		if injected || current != name {
			return nil
		}
		raw, err := backupFile(dir, entry.Name())
		if err != nil {
			return err
		}
		parent := filepath.Join(p.root, ".kb/state/receipts")
		if err := os.Rename(parent, parent+"-kept"); err != nil {
			return err
		}
		if err := os.Mkdir(parent, 0700); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(parent, entry.Name()), raw, 0600); err != nil {
			return err
		}
		injected = true
		return nil
	})
	if !injected || err == nil {
		t.Fatalf("same-byte replacement during anchored traversal was accepted: %v", err)
	}
}

func TestRestoreCleanupPreservesSameByteForeignReplacement(t *testing.T) {
	s, p, requests := backupReceiptHistory(t, 8)
	archive := api.BackupInput{Archive: ".kb/backups/history.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	name := receiptPath(requests[0].IdempotencyKey)
	raw := get(t, p, name)
	destination := ".kb/restores/raced"
	foreign := filepath.Join(p.root, destination, name)
	saved := filepath.Join(t.TempDir(), "owned-receipt.json")
	fault := errors.New("stop before publication after foreign replacement")
	s.fault = func(stage, _ string) error {
		if stage != "restore-validated" {
			return nil
		}
		if err := os.Rename(foreign, saved); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(foreign, raw, 0600); err != nil {
			t.Fatal(err)
		}
		return fault
	}
	_, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: destination})
	if !errors.Is(err, fault) {
		t.Fatalf("controlled pre-publication fault was not reached: %v", err)
	}
	data, err := readBackupFixture(filepath.Join(p.root, destination), name)
	if err != nil || string(data) != string(raw) {
		t.Fatal("cleanup erased a same-byte file with foreign identity")
	}
	if _, err := os.Lstat(filepath.Join(p.root, destination, ".kb/config.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed restore exposed configuration: %v", err)
	}
}
