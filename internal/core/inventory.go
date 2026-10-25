package core

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path"
	"sort"
	"syscall"
)

// walkInventory uses a short-lived confined handle for each directory. File
// opens use basenames, avoiding repeated ancestor resolution for every note.
// Each directory is checked before and after traversal; symlink/replacement
// races abort rather than returning a graph from a different directory.
func walkInventory(ctx context.Context, root *os.Root, visit func(*os.Root, string, fs.DirEntry) error, drained func()) error {
	var walk func(*os.Root, string) error
	walk = func(dir *os.Root, prefix string) error {
		// Queued reads must finish before a child directory handle is closed,
		// including cancellation and traversal errors.
		defer drained()
		file, err := dir.OpenFile(".", os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
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
			before, err := dir.Lstat(entry.Name())
			if err != nil {
				return err
			}
			if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
				return fault("CONFLICT", "directory changed during inventory")
			}
			child, err := dir.OpenRoot(entry.Name())
			if err != nil {
				return err
			}
			opened, statErr := child.Stat(".")
			if statErr != nil || !os.SameFile(before, opened) {
				_ = child.Close()
				if statErr != nil {
					return statErr
				}
				return fault("CONFLICT", "directory replaced during inventory")
			}
			walkErr := walk(child, name)
			after, statErr := dir.Lstat(entry.Name())
			closeErr := child.Close()
			if err := errors.Join(walkErr, statErr, closeErr); err != nil {
				return err
			}
			if after.Mode()&os.ModeSymlink != 0 || !os.SameFile(opened, after) {
				return fault("CONFLICT", "directory replaced during inventory")
			}
		}
		return nil
	}
	return walk(root, "")
}
