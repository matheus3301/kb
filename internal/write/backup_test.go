package write

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/internal/filelock"
)

func backupSetup(t *testing.T) (*Service, *fakeProvider) {
	t.Helper()
	s, p := setup(t)
	put(t, p, ".kb/config.toml", []byte("version = 1\n# preserved comment\n"))
	put(t, p, "raw/source.bin", []byte{0, 1, 2, 255})
	put(t, p, "note.md", document("Concept", "exact source\r\n"))
	return s, p
}

func TestOperationalBackupPreservesBytesAndReceipts(t *testing.T) {
	s, p := backupSetup(t)
	request := create("topic/new.md")
	original, err := s.Apply(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	put(t, p, ".kb/cache/disposable", []byte("cache"))
	put(t, p, ".git/config", []byte("excluded synthetic git"))
	source := get(t, p, "raw/source.bin")
	archive := api.BackupInput{Archive: ".kb/backups/snapshot.tar"}
	result, err := s.Backup(context.Background(), archive)
	if err != nil {
		t.Fatal(err)
	}
	if result.ManifestRevision == "" {
		t.Fatal("missing manifest revision")
	}
	restore := api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/roundtrip"}
	restored, err := s.Restore(context.Background(), restore)
	if err != nil {
		t.Fatal(err)
	}
	if result.ManifestRevision != restored.ManifestRevision {
		t.Fatal("manifest changed")
	}
	for _, name := range []string{"note.md", "topic/new.md", "topic/index.md", "index.md", "log.md", "raw/source.bin", ".kb/config.toml", receiptPath(request.IdempotencyKey)} {
		got := get(t, &fakeProvider{root: filepath.Join(p.root, restore.Destination)}, name)
		if !bytes.Equal(got, get(t, p, name)) {
			t.Fatalf("%s not byte preserved", name)
		}
	}
	for _, name := range []string{".kb/cache/disposable", ".git/config", ".kb/state/pending.json", ".kb/state/lock"} {
		if _, err := os.Lstat(filepath.Join(p.root, restore.Destination, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("excluded %s exists: %v", name, err)
		}
	}
	copyProvider := *p
	copyProvider.root = filepath.Join(p.root, restore.Destination)
	replay, err := New(&copyProvider).Apply(context.Background(), request)
	original.Replayed = true
	if err != nil || !reflect.DeepEqual(original, replay) {
		t.Fatalf("receipt replay differs: %v %+v", err, replay)
	}
	if _, err := New(&copyProvider).Apply(context.Background(), api.Plan{Operations: []api.Operation{{Kind: "create", Path: "after.md", Type: "Concept"}}, Actor: "test", Summary: "Continue after restore", IdempotencyKey: "after-restore"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(source, get(t, p, "raw/source.bin")) {
		t.Fatal("source changed")
	}
	_, err = s.Backup(context.Background(), archive)
	errorCode(t, err, "DESTINATION_EXISTS")
	_, err = s.Restore(context.Background(), restore)
	errorCode(t, err, "DESTINATION_EXISTS")
}

func TestBackupRefusesRecoveryAndUnsafeState(t *testing.T) {
	for _, kind := range []string{"pending", "symlink", "hardlink", "unknown-state", "readonly", "unsafe-storage", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			s, p := backupSetup(t)
			expected := "INVALID_BACKUP"
			switch kind {
			case "pending":
				put(t, p, pendingPath, []byte("pending evidence"))
				expected = "RECOVERY_REQUIRED"
			case "symlink":
				if err := os.Symlink("note.md", filepath.Join(p.root, "alias.md")); err != nil {
					t.Fatal(err)
				}
				expected = "PATH_OUTSIDE_ROOT"
			case "hardlink":
				if err := os.Link(filepath.Join(p.root, "note.md"), filepath.Join(p.root, "alias.md")); err != nil {
					t.Fatal(err)
				}
			case "unknown-state":
				put(t, p, ".kb/state/future.json", []byte("do not discard"))
			case "readonly":
				p.config.Write.Enabled = false
				expected = "READ_ONLY"
			case "unsafe-storage":
				if err := os.Mkdir(filepath.Join(p.root, ".kb/backups"), 0755); err != nil { // #nosec G301 -- Intentionally unsafe permissions exercise storage rejection.
					t.Fatal(err)
				}
				expected = "ACCESS_DENIED"
			case "oversized":
				f, err := os.Create(filepath.Join(p.root, "huge.bin"))
				if err != nil {
					t.Fatal(err)
				}
				if err := f.Truncate(maxBackupFileBytes + 1); err != nil {
					t.Fatal(err)
				}
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
				expected = "INPUT_TOO_LARGE"
			}
			_, err := s.Backup(context.Background(), api.BackupInput{Archive: ".kb/backups/test.tar"})
			errorCode(t, err, expected)
			if kind == "pending" && string(get(t, p, pendingPath)) != "pending evidence" {
				t.Fatal("pending state modified")
			}
		})
	}
}

func TestRestoreRequiresExistingPrivateSourceStorageWithoutMutation(t *testing.T) {
	for _, kind := range []string{"public-source", "group-source", "other-source", "missing-storage", "missing-archive", "source-file", "source-symlink"} {
		t.Run(kind, func(t *testing.T) {
			s, p := backupSetup(t)
			ctx := context.Background()
			archive := api.BackupInput{Archive: ".kb/backups/snapshot.tar"}
			if kind != "missing-storage" && kind != "source-file" {
				if _, err := s.Backup(ctx, archive); err != nil {
					t.Fatal(err)
				}
			}
			storage := filepath.Join(p.root, ".kb/backups")
			switch kind {
			case "public-source", "group-source", "other-source":
				mode := map[string]os.FileMode{"public-source": 0755, "group-source": 0710, "other-source": 0704}[kind]
				if err := os.Chmod(storage, mode); err != nil { // #nosec G302 -- Deliberately unsafe modes verify rejection without silently fixing permissions.
					t.Fatal(err)
				}
			case "missing-archive":
				if err := os.Remove(filepath.Join(p.root, archive.Archive)); err != nil {
					t.Fatal(err)
				}
			case "source-file":
				put(t, p, ".kb/backups", []byte("not a directory"))
			case "source-symlink":
				if err := os.Rename(storage, storage+"-kept"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("backups-kept", storage); err != nil {
					t.Fatal(err)
				}
			}
			// Create the ordinary advisory lock before measuring rejected-restore purity.
			if _, err := s.Recover(ctx, true); err != nil {
				t.Fatal(err)
			}
			type entry struct {
				mode os.FileMode
				data string
			}
			snapshot := func() map[string]entry {
				t.Helper()
				handle, err := os.OpenRoot(p.root)
				if err != nil {
					t.Fatal(err)
				}
				files := map[string]entry{}
				err = filepath.Walk(p.root, func(name string, info os.FileInfo, walkErr error) error {
					if walkErr != nil {
						return walkErr
					}
					relative, err := filepath.Rel(p.root, name)
					if err != nil {
						return err
					}
					var data []byte
					if info.Mode().IsRegular() {
						data, err = handle.ReadFile(relative)
					} else if info.Mode()&os.ModeSymlink != 0 {
						var target string
						target, err = handle.Readlink(relative)
						data = []byte(target)
					}
					files[name] = entry{info.Mode(), string(data)}
					return err
				})
				if err := errors.Join(err, handle.Close()); err != nil {
					t.Fatal(err)
				}
				return files
			}
			before := snapshot()
			_, err := s.Restore(ctx, api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/rejected"})
			switch kind {
			case "missing-storage", "missing-archive":
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("missing archive error changed: %v", err)
				}
			case "source-symlink":
				errorCode(t, err, "PATH_OUTSIDE_ROOT")
			default:
				errorCode(t, err, "ACCESS_DENIED")
			}
			if !reflect.DeepEqual(before, snapshot()) {
				t.Fatal("rejected restore changed paths, permissions, links or file bytes")
			}
			if _, err := os.Lstat(filepath.Join(p.root, ".kb/restores")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("rejected restore created destination storage: %v", err)
			}
		})
	}
}

func TestOperationalPublicationFailuresAndRetry(t *testing.T) {
	for _, stage := range []string{"backup-synced", "backup-published", "restore-file", "restore-validated", "restore-published"} {
		t.Run(stage, func(t *testing.T) {
			s, p := backupSetup(t)
			archive := api.BackupInput{Archive: ".kb/backups/test.tar"}
			if stage[0] == 'r' {
				if _, err := s.Backup(context.Background(), archive); err != nil {
					t.Fatal(err)
				}
			}
			injected := errors.New("injected boundary failure")
			s.fault = func(at, _ string) error {
				if at == stage {
					return injected
				}
				return nil
			}
			restore := api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/retry"}
			var err error
			if stage[0] == 'b' {
				_, err = s.Backup(context.Background(), archive)
			} else {
				_, err = s.Restore(context.Background(), restore)
			}
			if !errors.Is(err, injected) {
				t.Fatalf("boundary not exercised: %v", err)
			}
			s.fault = nil
			if stage == "backup-published" {
				_, err = s.Backup(context.Background(), archive)
				errorCode(t, err, "DESTINATION_EXISTS")
				return
			}
			if stage == "restore-published" {
				_, err = s.Restore(context.Background(), restore)
				errorCode(t, err, "DESTINATION_EXISTS")
				return
			}
			if stage[0] == 'b' {
				_, err = s.Backup(context.Background(), archive)
			} else {
				_, err = s.Restore(context.Background(), restore)
			}
			if err != nil {
				t.Fatalf("retry failed: %v", err)
			}
			if !bytes.Equal(get(t, p, "raw/source.bin"), []byte{0, 1, 2, 255}) {
				t.Fatal("source changed")
			}
		})
	}
}

func TestBackupDetectsEditorAndRestoreDestinationRaces(t *testing.T) {
	for _, kind := range []string{"backup-edit", "restore-edit", "restore-swap", "restore-symlink"} {
		t.Run(kind, func(t *testing.T) {
			s, p := backupSetup(t)
			archive := api.BackupInput{Archive: ".kb/backups/test.tar"}
			if kind != "backup-edit" {
				if _, err := s.Backup(context.Background(), archive); err != nil {
					t.Fatal(err)
				}
			}
			s.fault = func(stage, _ string) error {
				if kind == "backup-edit" && stage == "backup-synced" {
					put(t, p, "note.md", document("Concept", "external edit"))
				}
				if stage == "restore-validated" {
					target := filepath.Join(p.root, ".kb/restores/raced")
					switch kind {
					case "restore-edit":
						if err := os.WriteFile(filepath.Join(target, "note.md"), []byte("external edit"), 0600); err != nil {
							return err
						}
					case "restore-swap", "restore-symlink":
						if err := os.Rename(target, target+"-moved"); err != nil {
							return err
						}
						if kind == "restore-symlink" {
							return os.Symlink(target+"-moved", target)
						}
						return os.Mkdir(target, 0700)
					}
				}
				return nil
			}
			var err error
			if kind == "backup-edit" {
				_, err = s.Backup(context.Background(), archive)
				errorCode(t, err, "REVISION_CONFLICT")
			} else {
				_, err = s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/raced"})
				if err == nil {
					t.Fatal("destination race accepted")
				}
				if kind == "restore-edit" {
					if data := get(t, p, ".kb/restores/raced/note.md"); string(data) != "external edit" {
						t.Fatal("cleanup erased external edit")
					}
				}
			}
		})
	}
}

func TestConcurrentOperationalBackupAndRestore(t *testing.T) {
	s, p := backupSetup(t)
	lock, err := filelock.Acquire(context.Background(), p.root, true)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Backup(ctx, api.BackupInput{Archive: ".kb/backups/canceled.tar"})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	outcomes := make(chan error, 8)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Backup(context.Background(), api.BackupInput{Archive: ".kb/backups/shared.tar"})
			outcomes <- err
		}()
	}
	wg.Wait()
	close(outcomes)
	successes := 0
	for err := range outcomes {
		if err == nil {
			successes++
		} else {
			errorCode(t, err, "DESTINATION_EXISTS")
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent publication successes: %d", successes)
	}
	outcomes = make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Restore(context.Background(), api.RestoreInput{Archive: ".kb/backups/shared.tar", Destination: ".kb/restores/shared"})
			outcomes <- err
		}()
	}
	wg.Wait()
	close(outcomes)
	successes = 0
	for err := range outcomes {
		if err == nil {
			successes++
		} else {
			errorCode(t, err, "DESTINATION_EXISTS")
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent restore successes: %d", successes)
	}
}

func syntheticArchive(t testing.TB, alter func(*backupManifest), headers func(int, *tar.Header), payload func(int, []byte) []byte) []byte {
	t.Helper()
	entries := []struct {
		name string
		data []byte
	}{{".kb/config.toml", []byte("version = 1\n")}, {"note.md", []byte("---\ntype: Concept\n---\nExact bytes\n")}}
	m := backupManifest{Version: 1, Kind: "kb-operational-backup", Excluded: []string{}}
	for _, e := range entries {
		m.Files = append(m.Files, backupEntry{Path: e.name, Size: int64(len(e.data)), Revision: api.Revision(e.data)})
	}
	if alter != nil {
		alter(&m)
	}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	all := [][]byte{raw, entries[0].data, entries[1].data}
	names := []string{manifestName, entries[0].name, entries[1].name}
	for i, data := range all {
		if payload != nil {
			data = payload(i, data)
		}
		h := &tar.Header{Name: names[i], Typeflag: tar.TypeReg, Mode: 0600, Size: int64(len(data)), Format: tar.FormatPAX}
		if headers != nil {
			headers(i, h)
		}
		if err := w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := w.Write(data); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestArchiveRejectsUnsafeCorruptIncompleteInputs(t *testing.T) {
	valid := syntheticArchive(t, nil, nil, nil)
	cases := map[string][]byte{"truncated": valid[:len(valid)-1024], "trailing": append(append([]byte(nil), valid...), 1), "garbage": []byte("not a backup")}
	for _, name := range []string{"../escape", "/absolute", "C:/absolute", "note.md/../escape", ".kb/state/pending.json", ".kb/cache/entry", ".git/config", "note.md\\escape"} {
		cases[name] = syntheticArchive(t, func(m *backupManifest) { m.Files[1].Path = name }, nil, nil)
	}
	for _, kind := range []byte{tar.TypeSymlink, tar.TypeLink, tar.TypeDir, tar.TypeFifo} {
		cases[fmt.Sprint(kind)] = syntheticArchive(t, nil, func(i int, h *tar.Header) {
			if i == 2 {
				h.Typeflag = kind
				h.Size = 0
				if kind == tar.TypeLink || kind == tar.TypeSymlink {
					h.Linkname = "outside"
				}
			}
		}, nil)
	}
	cases["duplicate-manifest-path"] = syntheticArchive(t, func(m *backupManifest) { m.Files[1] = m.Files[0] }, nil, nil)
	cases["missing-config"] = syntheticArchive(t, func(m *backupManifest) { m.Files = m.Files[1:] }, nil, nil)
	cases["oversized"] = syntheticArchive(t, func(m *backupManifest) { m.Files[1].Size = maxBackupFileBytes + 1 }, nil, nil)
	cases["wrong-version"] = syntheticArchive(t, func(m *backupManifest) { m.Version = 2 }, nil, nil)
	cases["checksum"] = syntheticArchive(t, nil, nil, func(i int, b []byte) []byte {
		if i == 2 {
			b = bytes.Repeat([]byte("x"), len(b))
		}
		return b
	})
	cases["duplicate-json-key"] = syntheticArchive(t, nil, nil, func(i int, b []byte) []byte {
		if i == 0 {
			return append([]byte(`{"version":1,`), b[1:]...)
		}
		return b
	})
	cases["extra-entry"] = append(append([]byte(nil), valid[:len(valid)-1024]...), valid...)
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			s, p := backupSetup(t)
			put(t, p, ".kb/backups/bad.tar", data)
			_, err := s.Restore(context.Background(), api.RestoreInput{Archive: ".kb/backups/bad.tar", Destination: ".kb/restores/rejected"})
			if err == nil {
				t.Fatal("invalid archive restored")
			}
			if _, err := os.Lstat(filepath.Join(p.root, ".kb/restores/rejected")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failure not cleaned: %v", err)
			}
		})
	}
}

func FuzzOperationalArchive(f *testing.F) {
	f.Add(syntheticArchive(f, nil, nil, nil))
	f.Add([]byte("malformed"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 2<<20 {
			return
		}
		_, _, _ = readBackup(context.Background(), bytes.NewReader(data), verifyBackupEntry)
	})
}
func FuzzBackupPaths(f *testing.F) {
	for _, s := range []string{"note.md", "../escape", ".kb/state/receipts/entry.json", ".kb/backups/archive.tar", "/absolute"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) { _ = backupPath(name); _ = storagePath(name, ".kb/backups") })
}
