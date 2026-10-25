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
	// Stream the canonical records instead of retaining a second inventory-sized
	// byte buffer. The separators and hexadecimal revisions are wire-stable.
	digest := sha256.New()
	var record [1 + len("sha256:") + sha256.Size*2 + 1]byte
	copy(record[1:], "sha256:")
	record[len(record)-1] = '\n'
	for _, path := range paths {
		sum := sha256.Sum256(files[path])
		hex.Encode(record[1+len("sha256:"):len(record)-1], sum[:])
		_, _ = digest.Write([]byte(path))
		_, _ = digest.Write(record[:])
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}
