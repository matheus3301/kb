package write

import (
	"archive/tar"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"reflect"
	"slices"
	"sort"
	"syscall"

	"github.com/matheus3301/kb/internal/api"
)

type backupOutput struct {
	writer    io.Writer
	remaining int64
}

func (w *backupOutput) Write(data []byte) (int, error) {
	if int64(len(data)) > w.remaining {
		return 0, problem("INPUT_TOO_LARGE", "archive exceeds byte limit")
	}
	n, err := w.writer.Write(data)
	w.remaining -= int64(n)
	return n, err
}

func randomBackupName() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return ".stage-" + hex.EncodeToString(b[:]), nil
}

func pendingBackup(root *os.Root) error {
	if err := inspect(root, pendingPath); err != nil {
		return err
	}
	if _, err := root.Lstat(pendingPath); err == nil {
		return problem("RECOVERY_REQUIRED", "recover the pending transaction before operational backup or restore; a partial snapshot cannot preserve recovery semantics")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func backupInventory(ctx context.Context, root *os.Root) (backupManifest, error) {
	m := backupManifest{Version: backupVersion, Kind: "kb-operational-backup", Files: []backupEntry{}, Excluded: []string{}}
	var total int64
	count := 0
	paths := map[string][]byte{}
	err := walkBackupInventory(ctx, root, func(dir *os.Root, name string, entry fs.DirEntry) error {
		count++
		if count > maxBackupFiles*2 {
			return problem("INPUT_TOO_LARGE", "backup inventory exceeds entry limit")
		}
		if excludedBackupPath(name) {
			if entry.IsDir() {
				m.Excluded = append(m.Excluded, name+"/")
				return fs.SkipDir
			}
			m.Excluded = append(m.Excluded, name)
			return nil
		}
		paths[name] = nil
		if entry.IsDir() {
			return nil
		}
		if err := backupPath(name); err != nil {
			return err
		}
		data, err := backupFile(dir, entry.Name())
		if err != nil {
			return err
		}
		m.Files = append(m.Files, backupEntry{Path: name, Size: int64(len(data)), Revision: api.Revision(data)})
		total += int64(len(data))
		if len(m.Files) > maxBackupFiles || total > maxBackupBytes {
			return problem("INPUT_TOO_LARGE", "backup exceeds file or byte limit")
		}
		return nil
	})
	if err == nil {
		err = inspectInventory(root, paths)
	}
	sortEntries(m.Files)
	sort.Strings(m.Excluded)
	return m, err
}

func backupFile(root *os.Root, name string) (data []byte, err error) {
	parent, err := backupParent(root, name, nil)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, parent.Close()) }()
	base := path.Base(name)
	before, err := parent.Lstat(base)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return nil, problem("PATH_OUTSIDE_ROOT", "symbolic links are forbidden")
	}
	f, err := openRead(parent, base)
	if err != nil {
		return nil, err
	}
	info, statErr := f.Stat()
	if statErr != nil {
		return nil, errors.Join(statErr, f.Close())
	}
	if !os.SameFile(before, info) {
		return nil, errors.Join(problem("REVISION_CONFLICT", "a backup file changed during access"), f.Close())
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || !ok || stat.Nlink != 1 {
		return nil, errors.Join(problem("INVALID_BACKUP", "backup requires regular files without hard links"), f.Close())
	}
	if info.Size() > maxBackupFileBytes {
		return nil, errors.Join(problem("INPUT_TOO_LARGE", "backup file exceeds 64 MiB"), f.Close())
	}
	data, readErr := io.ReadAll(io.LimitReader(f, maxBackupFileBytes+1))
	err = errors.Join(readErr, f.Close())
	if err != nil {
		return nil, err
	}
	if err := backupParentIdentity(root, name, parent); err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBackupFileBytes {
		return nil, problem("INPUT_TOO_LARGE", "backup file exceeds 64 MiB")
	}
	return data, nil
}

func ensureBackupDirectory(root *os.Root, name string) error {
	if err := inspect(root, name); err != nil {
		return err
	}
	if err := root.Mkdir(name, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if err := privateBackupDirectory(info); err != nil {
		return err
	}
	return syncDirectory(root, path.Dir(name))
}

func privateBackupDirectory(info os.FileInfo) error {
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return problem("ACCESS_DENIED", "operational storage must be a private directory")
	}
	return nil
}

func validateBackupDirectory(root *os.Root, name string) error {
	if err := inspect(root, name); err != nil {
		return err
	}
	info, err := root.Lstat(name)
	if err != nil {
		return err
	}
	return privateBackupDirectory(info)
}

// Backup refuses pending recovery instead of implicitly changing source state.
// Publication is a no-overwrite hard link of a fully synced temporary archive.
func (s *Service) Backup(ctx context.Context, request api.BackupInput) (result api.BackupResult, err error) {
	if err := storagePath(request.Archive, ".kb/backups"); err != nil {
		return result, err
	}
	_, err = s.locked(ctx, func(root *os.Root) (api.Receipt, error) {
		if err := pendingBackup(root); err != nil {
			return api.Receipt{}, err
		}
		if err := ensureBackupDirectory(root, ".kb/backups"); err != nil {
			return api.Receipt{}, err
		}
		m, err := backupInventory(ctx, root)
		if err != nil {
			return api.Receipt{}, err
		}
		raw, err := json.Marshal(m)
		if err != nil {
			return api.Receipt{}, err
		}
		if int64(len(raw)) > maxManifestBytes {
			return api.Receipt{}, problem("INPUT_TOO_LARGE", "backup manifest exceeds byte limit")
		}
		if _, err := decodeManifest(raw); err != nil {
			return api.Receipt{}, err
		}
		stage, err := randomBackupName()
		if err != nil {
			return api.Receipt{}, err
		}
		stage = ".kb/backups/" + stage
		f, err := root.OpenFile(stage, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return api.Receipt{}, err
		}
		defer func() { _ = root.Remove(stage) }()
		writer := tar.NewWriter(&backupOutput{writer: f, remaining: maxArchiveBytes})
		writeEntry := func(name string, data []byte) error {
			if err := writer.WriteHeader(&tar.Header{Name: name, Size: int64(len(data)), Typeflag: tar.TypeReg, Mode: 0600, Format: tar.FormatPAX}); err != nil {
				return err
			}
			_, err := writer.Write(data)
			return err
		}
		writeErr := writeEntry(manifestName, raw)
		for _, entry := range m.Files {
			if writeErr != nil {
				break
			}
			if writeErr = ctx.Err(); writeErr != nil {
				break
			}
			data, err := backupFile(root, entry.Path)
			if err != nil {
				writeErr = err
				break
			}
			if int64(len(data)) != entry.Size || api.Revision(data) != entry.Revision {
				writeErr = problem("REVISION_CONFLICT", "source changed during backup")
				break
			}
			writeErr = writeEntry(entry.Path, data)
		}
		if err := errors.Join(writeErr, writer.Close(), f.Sync(), f.Close()); err != nil {
			return api.Receipt{}, err
		}
		if err := s.boundary("backup-synced", stage); err != nil {
			return api.Receipt{}, err
		}
		current, err := backupInventory(ctx, root)
		if err != nil {
			return api.Receipt{}, err
		}
		if !reflect.DeepEqual(m, current) {
			return api.Receipt{}, problem("REVISION_CONFLICT", "source inventory changed during backup")
		}
		if err := inspect(root, request.Archive); err != nil {
			return api.Receipt{}, err
		}
		if err := root.Link(stage, request.Archive); errors.Is(err, os.ErrExist) {
			return api.Receipt{}, problem("DESTINATION_EXISTS", "backup archive already exists")
		} else if err != nil {
			return api.Receipt{}, err
		}
		if err := syncDirectory(root, ".kb/backups"); err != nil {
			return api.Receipt{}, err
		}
		if err := s.boundary("backup-published", request.Archive); err != nil {
			return api.Receipt{}, err
		}
		result = backupResult(request.Archive, "", m, api.Revision(raw))
		return api.Receipt{}, nil
	})
	return result, err
}

func backupResult(archive, destination string, m backupManifest, revision string) api.BackupResult {
	result := api.BackupResult{Archive: archive, Destination: destination, ManifestRevision: revision, Files: len(m.Files)}
	for _, e := range m.Files {
		result.Bytes += e.Size
	}
	return result
}

// Restore creates a new root only. Configuration is published last, after every
// archive entry and checksum has been verified and all content has been synced.
// Ordinary failures remove only our created files; interruption can leave an
// uninitialized destination, which is never reused automatically.
func (s *Service) Restore(ctx context.Context, request api.RestoreInput) (result api.BackupResult, err error) {
	if err := storagePath(request.Archive, ".kb/backups"); err != nil {
		return result, err
	}
	if err := storagePath(request.Destination, ".kb/restores"); err != nil {
		return result, err
	}
	_, err = s.locked(ctx, func(root *os.Root) (receipt api.Receipt, err error) {
		if err := pendingBackup(root); err != nil {
			return receipt, err
		}
		if err := validateBackupDirectory(root, ".kb/backups"); err != nil {
			return receipt, err
		}
		if err := inspect(root, request.Archive); err != nil {
			return receipt, err
		}
		archive, err := openRead(root, request.Archive)
		if err != nil {
			return receipt, err
		}
		defer func() { err = errors.Join(err, archive.Close()) }()
		info, err := archive.Stat()
		if err != nil {
			return receipt, err
		}
		if !info.Mode().IsRegular() || info.Size() > maxArchiveBytes {
			return receipt, problem("INVALID_BACKUP", "archive must be a bounded regular file")
		}
		if err := ensureBackupDirectory(root, ".kb/restores"); err != nil {
			return receipt, err
		}
		if err := inspect(root, request.Destination); err != nil {
			return receipt, err
		}
		if err := root.Mkdir(request.Destination, 0700); errors.Is(err, os.ErrExist) {
			return receipt, problem("DESTINATION_EXISTS", "restore requires a nonexistent destination; existing directories are never overwritten")
		} else if err != nil {
			return receipt, err
		}
		created, err := root.Lstat(request.Destination)
		if err != nil {
			return receipt, err
		}
		dest, err := root.OpenRoot(request.Destination)
		if err != nil {
			return receipt, err
		}
		defer func() { err = errors.Join(err, dest.Close()) }()
		var files []restoredFile
		var directories []string
		published := false
		defer func() {
			if !published {
				err = errors.Join(err, cleanupRestore(root, dest, request.Destination, created, files, directories))
			}
		}()
		identity := func() (err error) {
			parent, err := backupParent(root, request.Destination, nil)
			if err != nil {
				return err
			}
			defer func() { err = errors.Join(err, parent.Close()) }()
			current, err := parent.Lstat(path.Base(request.Destination))
			if err != nil {
				return err
			}
			opened, err := dest.Stat(".")
			if err != nil {
				return err
			}
			if !os.SameFile(created, current) || !os.SameFile(created, opened) {
				return problem("DESTINATION_EXISTS", "restore destination changed during publication")
			}
			return nil
		}
		if err := identity(); err != nil {
			return receipt, err
		}
		var config []byte
		consume := func(entry backupEntry, reader io.Reader) error {
			if err := identity(); err != nil {
				return err
			}
			data, err := io.ReadAll(io.LimitReader(reader, entry.Size+1))
			if err != nil || int64(len(data)) != entry.Size || api.Revision(data) != entry.Revision {
				return problem("INVALID_BACKUP", "archive entry checksum or length mismatch")
			}
			if entry.Path == ".kb/config.toml" {
				config = data
				return nil
			}
			if err := restoreFile(dest, entry, data, &directories, &files); err != nil {
				return err
			}
			return s.boundary("restore-file", entry.Path)
		}
		m, revision, err := readBackup(ctx, archive, consume)
		if err != nil {
			return receipt, err
		}
		if err := ctx.Err(); err != nil {
			return receipt, err
		}
		if err := s.boundary("restore-validated", request.Destination); err != nil {
			return receipt, err
		}
		if err := identity(); err != nil {
			return receipt, err
		}
		// Detect injected files, directory swaps and edits before exposing config.
		actual, err := backupInventory(ctx, dest)
		if err != nil {
			return receipt, err
		}
		expected := make([]backupEntry, len(files))
		for i, file := range files {
			expected[i] = file.entry
		}
		sortEntries(expected)
		if !slices.Equal(actual.Files, expected) || len(actual.Excluded) != 0 {
			return receipt, problem("REVISION_CONFLICT", "restore destination changed before publication")
		}
		if err := inspect(root, request.Destination); err != nil {
			return receipt, err
		}
		if err := inspect(dest, ".kb/config.toml"); err != nil {
			return receipt, err
		}
		entry := backupEntry{Path: ".kb/config.toml", Size: int64(len(config)), Revision: api.Revision(config)}
		if err := restoreFile(dest, entry, config, &directories, &files); err != nil {
			return receipt, err
		}
		if err := identity(); err != nil {
			return receipt, err
		}
		if err := inspect(root, request.Destination); err != nil {
			return receipt, err
		}
		if err := syncDirectory(root, ".kb/restores"); err != nil {
			return receipt, err
		}
		published = true
		if err := s.boundary("restore-published", request.Destination); err != nil {
			return receipt, err
		}
		result = backupResult(request.Archive, request.Destination, m, revision)
		return receipt, nil
	})
	return result, err
}

func restoreFile(dest *os.Root, entry backupEntry, data []byte, directories *[]string, files *[]restoredFile) (err error) {
	parent, err := backupParent(dest, entry.Path, directories)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, parent.Close()) }()
	name := path.Base(entry.Path)
	f, err := parent.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return err
	}
	created, err := f.Stat()
	if err != nil {
		return errors.Join(err, f.Close(), parent.Remove(name))
	}
	_, writeErr := f.Write(data)
	if err := errors.Join(writeErr, f.Sync(), f.Close()); err != nil {
		return errors.Join(err, removeOwnedRestoreLeaf(parent, name, created))
	}
	*files = append(*files, restoredFile{entry: entry, identity: created})
	return errors.Join(syncDirectory(parent, "."), backupParentIdentity(dest, entry.Path, parent))
}

func cleanupRestore(parent, dest *os.Root, name string, created os.FileInfo, files []restoredFile, directories []string) error {
	var errs []error
	// A fresh batch avoids quadratic sibling scans. If names became ambiguous,
	// retain the incomplete destination rather than deleting external state.
	if err := inspectInventory(dest, restoredFilePaths(files, directories)); err != nil {
		return err
	}
	for i := len(files) - 1; i >= 0; i-- {
		entry := files[i]
		if err := removeRestoredFile(dest, entry); err != nil {
			errs = append(errs, err)
		}
	}
	for i := len(directories) - 1; i >= 0; i-- {
		if err := dest.Remove(directories[i]); err != nil {
			errs = append(errs, err)
		}
	}
	current, err := parent.Lstat(name)
	if err == nil && os.SameFile(created, current) {
		err = parent.Remove(name)
	}
	if err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
