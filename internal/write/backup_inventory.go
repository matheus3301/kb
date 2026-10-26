package write

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"sort"
)

// Each phase enumerates actual entries afresh. Parent handles are scoped to one
// traversal and checked after use, including empty included directories.
func walkBackupInventory(ctx context.Context, root *os.Root, visit func(*os.Root, string, fs.DirEntry) error) error {
	var walk func(*os.Root, string) error
	walk = func(dir *os.Root, prefix string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		file, err := openRead(dir, ".")
		if err != nil {
			return err
		}
		entries, readErr := file.ReadDir(-1)
		if err := errors.Join(readErr, file.Close()); err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			name := path.Join(prefix, entry.Name())
			if err := visit(dir, name, entry); errors.Is(err, fs.SkipDir) {
				continue
			} else if err != nil {
				return err
			}
			if !entry.IsDir() {
				continue
			}
			child, err := backupDirectory(dir, entry.Name(), name, nil)
			if err != nil {
				return err
			}
			opened, statErr := child.Stat(".")
			if statErr != nil {
				return errors.Join(statErr, child.Close())
			}
			walkErr := walk(child, name)
			current, statErr := dir.Lstat(entry.Name())
			if err := errors.Join(walkErr, statErr, child.Close()); err != nil {
				return err
			}
			if current.Mode()&os.ModeSymlink != 0 || !os.SameFile(opened, current) {
				return problem("REVISION_CONFLICT", "a directory changed during backup inventory")
			}
		}
		return nil
	}
	return walk(root, "")
}

func backupParentIdentity(root *os.Root, name string, opened *os.Root) (err error) {
	current, err := backupParent(root, name, nil)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, current.Close()) }()
	before, err := opened.Stat(".")
	if err != nil {
		return err
	}
	after, err := current.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(before, after) {
		return problem("REVISION_CONFLICT", "a backup or restore parent changed while in use")
	}
	return nil
}
