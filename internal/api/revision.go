package api

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

const (
	MaxNoteBytes         = 1 << 20
	MaxReadBytes         = 256 << 10
	MaxPlanBytes         = 4 << 20
	MaxOperations        = 50
	MaxQueryRunes        = 4096
	MaxHits              = 100
	DefaultHits          = 10
	DefaultContextTokens = 2000
	MaxContextTokens     = 16000
)

func Revision(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Generation hashes sorted scoped Markdown paths and their content revisions.
func Generation(files map[string][]byte) string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var inventory []byte
	for _, path := range paths {
		inventory = append(inventory, path...)
		inventory = append(inventory, 0)
		inventory = append(inventory, Revision(files[path])...)
		inventory = append(inventory, '\n')
	}
	return Revision(inventory)
}
