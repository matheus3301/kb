package write

import (
	"errors"
	"os"
	"path"
	"sort"
)

// inspectInventory performs the same symlink/case checks as per-path inspect,
// but lists each shared parent once for this planning phase. Operation targets
// and every published image still receive fresh uncached checks and CAS.
func inspectInventory(root *os.Root, files map[string][]byte) error {
	paths := map[string]bool{}
	for name := range files {
		if err := cleanPath(name); err != nil {
			return err
		}
		for p := name; p != "."; p = path.Dir(p) {
			paths[p] = true
		}
	}
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	sort.Strings(names)
	type spelling struct {
		name      string
		ambiguous bool
	}
	directories := map[string]map[string]spelling{}
	missing := map[string]bool{}
	for _, name := range names {
		parent := path.Dir(name)
		if missing[parent] {
			missing[name] = true
			continue
		}
		entries, ok := directories[parent]
		if !ok {
			f, err := openRead(root, parent)
			if errors.Is(err, os.ErrNotExist) {
				missing[parent] = true
				missing[name] = true
				continue
			}
			if err != nil {
				return err
			}
			listed, readErr := f.ReadDir(-1)
			if err := errors.Join(readErr, f.Close()); err != nil {
				return err
			}
			entries = map[string]spelling{}
			for _, entry := range listed {
				key := caseKey(entry.Name())
				previous, exists := entries[key]
				entries[key] = spelling{name: entry.Name(), ambiguous: exists && (previous.ambiguous || previous.name != entry.Name())}
			}
			directories[parent] = entries
		}
		base := path.Base(name)
		if entry, ok := entries[caseKey(base)]; ok && (entry.name != base || entry.ambiguous) {
			return problem("PATH_COLLISION", "path differs only in letter case from an existing entry")
		}
		info, err := root.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			missing[name] = true
			continue
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
