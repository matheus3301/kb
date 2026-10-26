package write

import (
	"errors"
	"os"
	"path"
	"strings"

	"github.com/matheus3301/kb/internal/api"
)

// backupParent opens each parent as an anchored directory, rejecting symlinks
// and replacements without enumerating siblings for every receipt. Callers batch
// case-fold checks with inspectInventory at each complete inventory boundary.
// A non-nil directories list permits private parent creation during restore.
func backupParent(root *os.Root, name string, directories *[]string) (*os.Root, error) {
	if err := cleanPath(name); err != nil {
		return nil, err
	}
	current, err := root.OpenRoot(".")
	if err != nil {
		return nil, err
	}
	prefix := "."
	for part := range strings.SplitSeq(path.Dir(name), "/") {
		if part == "." {
			continue
		}
		prefix = path.Join(prefix, part)
		next, err := backupDirectory(current, part, prefix, directories)
		if err != nil {
			return nil, errors.Join(err, current.Close())
		}
		if err := current.Close(); err != nil {
			return nil, errors.Join(err, next.Close())
		}
		current = next
	}
	return current, nil
}

func backupDirectory(parent *os.Root, name, prefix string, directories *[]string) (*os.Root, error) {
	before, err := parent.Lstat(name)
	if errors.Is(err, os.ErrNotExist) && directories != nil {
		if err := parent.Mkdir(name, 0700); err != nil {
			return nil, err
		}
		*directories = append(*directories, prefix)
		if err := syncDirectory(parent, "."); err != nil {
			return nil, err
		}
		before, err = parent.Lstat(name)
	}
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return nil, problem("PATH_OUTSIDE_ROOT", "symbolic links are forbidden")
	}
	if !before.IsDir() {
		return nil, problem("INVALID_PATH", "a parent directory is required")
	}
	directory, err := parent.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	opened, statErr := directory.Stat(".")
	current, currentErr := parent.Lstat(name)
	if err := errors.Join(statErr, currentErr); err != nil {
		return nil, errors.Join(err, directory.Close())
	}
	if current.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, current) || !os.SameFile(before, opened) {
		return nil, errors.Join(problem("REVISION_CONFLICT", "a backup or restore parent changed during access"), directory.Close())
	}
	return directory, nil
}

type restoredFile struct {
	entry    backupEntry
	identity os.FileInfo
}

func restoredFilePaths(files []restoredFile, directories []string) map[string][]byte {
	paths := make(map[string][]byte, len(files)+len(directories))
	for _, entry := range files {
		paths[entry.entry.Path] = nil
	}
	for _, name := range directories {
		paths[name] = nil
	}
	return paths
}

func removeRestoredFile(dest *os.Root, file restoredFile) (err error) {
	entry := file.entry
	parent, err := backupParent(dest, entry.Path, nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, parent.Close()) }()
	name := path.Base(entry.Path)
	data, err := backupFile(parent, name)
	if err != nil {
		return err
	}
	if int64(len(data)) != entry.Size || api.Revision(data) != entry.Revision {
		return nil
	}
	if err := backupParentIdentity(dest, entry.Path, parent); err != nil {
		return err
	}
	return removeOwnedRestoreLeaf(parent, name, file.identity)
}

func removeOwnedRestoreLeaf(parent *os.Root, name string, created os.FileInfo) error {
	current, err := parent.Lstat(name)
	if err != nil {
		return err
	}
	if current.Mode()&os.ModeSymlink != 0 || !os.SameFile(created, current) {
		return problem("REVISION_CONFLICT", "a restored file changed before cleanup")
	}
	return parent.Remove(name)
}
