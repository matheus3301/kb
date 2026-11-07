package setup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"
)

// Canonical resolves existing ancestor aliases but never follows a final symlink.
// Missing descendants are resolved relative to their nearest existing ancestor.
func Canonical(name string) (string, error) {
	if name == "" || strings.IndexByte(name, 0) >= 0 {
		return "", invalid("path must be nonempty and contain no NUL")
	}
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if info, err := os.Lstat(abs); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", conflict("symlink destination is unsupported: " + abs)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent := abs
	var suffix []string
	for {
		_, err := os.Lstat(parent)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		suffix = append(suffix, filepath.Base(parent))
		next := filepath.Dir(parent)
		if next == parent {
			return "", err
		}
		parent = next
	}
	parent, err = filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	for i := len(suffix) - 1; i >= 0; i-- {
		parent = filepath.Join(parent, suffix[i])
	}
	return parent, nil
}

// safePath rejects symlinks, hardlinks, special files and case-fold aliases at
// every existing component. Publication additionally uses confined os.Roots.
func safePath(name string) error {
	if !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return invalid("target must be an absolute clean path: " + name)
	}
	parts := strings.Split(strings.TrimPrefix(name, string(filepath.Separator)), string(filepath.Separator))
	parent := string(filepath.Separator)
	for i, part := range parts {
		if part == "" {
			continue
		}
		entries, err := os.ReadDir(parent)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Name() != part && strings.EqualFold(entry.Name(), part) {
				return conflict("case alias at " + filepath.Join(parent, part))
			}
		}
		path := filepath.Join(parent, part)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return conflict("symlink target is unsupported: " + path)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return conflict("non-directory ancestor: " + path)
		}
		if i == len(parts)-1 && !info.IsDir() && !info.Mode().IsRegular() {
			return conflict("non-regular destination: " + path)
		}
		if info.Mode().IsRegular() {
			if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Nlink > 1 {
				return conflict("hardlinked destination is unsupported: " + path)
			}
		}
		parent = path
	}
	return nil
}

func readSnapshot(name string) (Snapshot, error) {
	return readSnapshotBounded(name, maxFileBytes)
}

// Recovery can contain several encoded snapshots of otherwise valid native files.
// Accept older oversized catalogs here so their journal can still be rolled back.
func (e *Engine) readSnapshot(name string) (Snapshot, error) {
	if name == e.pendingPath() || name == e.statePath() {
		return readSnapshotBounded(name, maxRecoveryBytes)
	}
	return readSnapshot(name)
}

func readSnapshotBounded(name string, limit int) (Snapshot, error) {
	var snap Snapshot
	if err := safePath(name); err != nil {
		return snap, err
	}
	f, err := os.Open(name) // #nosec G304 -- canonical adapter/state path, checked above and rechecked under publication lock.
	if errors.Is(err, os.ErrNotExist) {
		return snap, nil
	}
	if err != nil {
		return snap, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return snap, err
	}
	if !info.Mode().IsRegular() {
		return snap, conflict("destination is not a regular file: " + name)
	}
	if info.Size() > int64(limit) {
		return snap, invalid(fmt.Sprintf("setup target exceeds %d MiB: %s", limit>>20, name))
	}
	raw, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return snap, err
	}
	if len(raw) > limit {
		return snap, invalid(fmt.Sprintf("setup target exceeds %d MiB: %s", limit>>20, name))
	}
	return Snapshot{Exists: true, Content: raw, Mode: uint32(info.Mode().Perm())}, nil
}

func within(parent, child string) bool {
	r, err := filepath.Rel(parent, child)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}
func validKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("._-", r) {
			return false
		}
	}
	return true
}
func pathError(path string, err error) error { return fmt.Errorf("%s: %w", path, err) }
