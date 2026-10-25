package write

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"syscall"
	"unicode"
	"unicode/utf8"

	"github.com/matheus3301/kb/internal/api"
)

const pendingPath = ".kb/state/pending.json"
const maxJournalBytes = 64 << 20

func problem(code, message string) error { return &api.DomainError{Code: code, Message: message} }

func cleanPath(name string) error {
	if name == "" || !utf8.ValidString(name) || strings.ContainsAny(name, "\\\x00\r\n") || strings.HasPrefix(name, "/") || path.Clean(name) != name {
		return problem("PATH_OUTSIDE_ROOT", "a canonical relative path is required")
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." || part == "." || strings.Contains(part, ":") {
			return problem("PATH_OUTSIDE_ROOT", "path traversal and absolute paths are forbidden")
		}
	}
	return nil
}

// inspect rejects symlinks and case collisions, including in existing parents.
func inspect(root *os.Root, name string) error {
	if err := cleanPath(name); err != nil {
		return err
	}
	dir := "."
	for _, part := range strings.Split(name, "/") {
		f, err := openRead(root, dir)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		entries, readErr := f.ReadDir(-1)
		closeErr := f.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), part) && entry.Name() != part {
				return problem("PATH_COLLISION", "path differs only in letter case from an existing entry")
			}
		}
		dir = path.Join(dir, part)
		info, err := root.Lstat(dir)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return problem("PATH_OUTSIDE_ROOT", "symbolic links are forbidden")
		}
	}
	return nil
}

func inScope(name string, config api.Config) bool {
	if cleanPath(name) != nil || !strings.HasSuffix(name, ".md") {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		switch strings.ToLower(part) {
		case ".kb", ".git", "node_modules", "vendor", ".claude", ".codex", ".agents", ".obsidian":
			return false
		}
	}
	if name == "index.md" || name == "log.md" {
		return true
	}
	for _, pattern := range config.Content.Exclude {
		if glob(pattern, name) {
			return false
		}
	}
	if len(config.Content.Include) == 0 {
		return true
	}
	for _, pattern := range config.Content.Include {
		if glob(pattern, name) {
			return true
		}
	}
	return false
}

func glob(pattern, name string) bool {
	patterns, parts := strings.Split(pattern, "/"), strings.Split(name, "/")
	type position struct{ pattern, name int }
	known, results := map[position]bool{}, map[position]bool{}
	var match func(int, int) bool
	match = func(i, j int) bool {
		key := position{i, j}
		if known[key] {
			return results[key]
		}
		known[key] = true
		var result bool
		if i == len(patterns) {
			result = j == len(parts)
		} else if patterns[i] == "**" {
			result = match(i+1, j) || (j < len(parts) && match(i, j+1))
		} else if j < len(parts) {
			matched, err := path.Match(patterns[i], parts[j])
			result = err == nil && matched && match(i+1, j+1)
		}
		results[key] = result
		return result
	}
	return match(0, 0)
}

func caseKey(name string) string {
	ascii := true
	for i := 0; i < len(name); i++ {
		if name[i] >= 128 {
			ascii = false
			break
		}
	}
	if ascii {
		return strings.ToUpper(name)
	}
	return strings.Map(func(r rune) rune {
		minimum := r
		for folded := unicode.SimpleFold(r); folded != r; folded = unicode.SimpleFold(folded) {
			if folded < minimum {
				minimum = folded
			}
		}
		return minimum
	}, name)
}

type image struct {
	Exists bool   `json:"exists"`
	Data   []byte `json:"data,omitempty"`
	Mode   uint32 `json:"mode,omitempty"`
}

func readImage(root *os.Root, name string, limit int64) (image, error) {
	if err := inspect(root, name); err != nil {
		return image{}, err
	}
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return image{}, nil
	}
	if err != nil {
		return image{}, err
	}
	if !info.Mode().IsRegular() {
		return image{}, problem("INVALID_PATH", "a regular file is required")
	}
	f, err := openRead(root, name)
	if errors.Is(err, os.ErrNotExist) {
		return image{}, nil
	}
	if err != nil {
		return image{}, err
	}
	info, statErr := f.Stat()
	if statErr != nil {
		return image{}, errors.Join(statErr, f.Close())
	}
	if !info.Mode().IsRegular() {
		return image{}, errors.Join(problem("INVALID_PATH", "a regular file is required"), f.Close())
	}
	data, readErr := io.ReadAll(io.LimitReader(f, limit+1))
	if err := errors.Join(readErr, f.Close()); err != nil {
		return image{}, err
	}
	if int64(len(data)) > limit {
		return image{}, problem("INPUT_TOO_LARGE", "file exceeds the byte limit")
	}
	return image{Exists: true, Data: data, Mode: uint32(info.Mode().Perm())}, nil
}

// Nonblocking, no-follow opens keep special-file swaps from trapping a lock owner.
func openRead(root *os.Root, name string) (*os.File, error) {
	return root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
}

func sameImage(a, b image) bool { return a.Exists == b.Exists && bytes.Equal(a.Data, b.Data) }

func validatePathSet(files map[string][]byte) error {
	spellings := map[string]string{}
	for name := range files {
		for prefix := name; prefix != "."; prefix = path.Dir(prefix) {
			folded := caseKey(prefix)
			if previous, exists := spellings[folded]; exists && previous != prefix {
				return problem("PATH_COLLISION", "planned directory or file names differ only in letter case")
			}
			spellings[folded] = prefix
			if prefix != name {
				if _, exists := files[prefix]; exists {
					return problem("PATH_COLLISION", "a planned file is also used as a directory")
				}
			}
		}
	}
	return nil
}

func syncDirectory(root *os.Root, name string) error {
	f, err := openRead(root, name)
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func makeParents(root *os.Root, name string) error {
	dir := "."
	for _, part := range strings.Split(path.Dir(name), "/") {
		if part == "." {
			continue
		}
		next := path.Join(dir, part)
		if err := inspect(root, next); err != nil {
			return err
		}
		if err := root.Mkdir(next, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		if err := syncDirectory(root, dir); err != nil {
			return err
		}
		dir = next
	}
	return nil
}

func atomicImage(root *os.Root, name string, value image) error {
	return replaceImage(root, name, value, nil)
}

func replaceImage(root *os.Root, name string, value image, expected *image) error {
	if err := inspect(root, name); err != nil {
		return err
	}
	if !value.Exists {
		if err := compareImage(root, name, expected); err != nil {
			return err
		}
		if err := root.Remove(name); errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		return syncDirectory(root, path.Dir(name))
	}
	if err := makeParents(root, name); err != nil {
		return err
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return err
	}
	temp := path.Join(path.Dir(name), ".kb-tmp-"+hex.EncodeToString(random[:]))
	mode := os.FileMode(value.Mode) & 0777
	if mode == 0 {
		mode = 0600
	}
	f, err := root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(value.Data)
	err = errors.Join(writeErr, f.Sync(), f.Close())
	if err != nil {
		return errors.Join(err, root.Remove(temp))
	}
	if err := inspect(root, name); err != nil {
		return errors.Join(err, root.Remove(temp))
	}
	if err := compareImage(root, name, expected); err != nil {
		return errors.Join(err, root.Remove(temp))
	}
	if err := root.Rename(temp, name); err != nil {
		return errors.Join(err, root.Remove(temp))
	}
	return syncDirectory(root, path.Dir(name))
}

func compareImage(root *os.Root, name string, expected *image) error {
	if expected == nil {
		return nil
	}
	current, err := readImage(root, name, api.MaxNoteBytes)
	if err != nil {
		return err
	}
	if !sameImage(current, *expected) {
		return problem("REVISION_CONFLICT", "file changed immediately before replacement")
	}
	return nil
}
