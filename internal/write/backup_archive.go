package write

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/matheus3301/kb/internal/api"
)

const (
	backupVersion            = 1
	maxBackupFiles           = 100000
	maxBackupBytes     int64 = 1 << 30
	maxBackupFileBytes int64 = 64 << 20
	maxManifestBytes   int64 = 16 << 20
	maxArchiveBytes    int64 = maxBackupBytes + 128<<20
	manifestName             = "kb-backup-manifest.json"
)

type backupEntry struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Revision string `json:"revision"`
}

type backupManifest struct {
	Version  int           `json:"version"`
	Kind     string        `json:"kind"`
	Files    []backupEntry `json:"files"`
	Excluded []string      `json:"excluded"`
}

// Storage paths are deliberately separate from knowledge scope, and cannot
// select another server root or overwrite a live base.
func storagePath(name, directory string) error {
	if err := cleanPath(name); err != nil {
		return err
	}
	if path.Dir(name) != directory || len(path.Base(name)) > 128 || strings.HasPrefix(path.Base(name), ".") {
		return problem("INVALID_PATH", "select a direct child of "+directory)
	}
	return nil
}

func backupPath(name string) error {
	if err := cleanPath(name); err != nil {
		return err
	}
	if len(name) > 1024 || strings.Count(name, "/") > 32 || !utf8.ValidString(name) || name == manifestName {
		return problem("INVALID_BACKUP", "invalid archive file path")
	}
	if excludedBackupPath(name) {
		return problem("INVALID_BACKUP", "archive contains excluded state")
	}
	if strings.HasPrefix(name, ".kb/") && name != ".kb/config.toml" && !strings.HasPrefix(name, ".kb/state/receipts/") {
		return problem("INVALID_BACKUP", "unknown operational state cannot be restored")
	}
	if strings.HasPrefix(name, ".kb/state/receipts/") {
		base := strings.TrimSuffix(strings.TrimPrefix(name, ".kb/state/receipts/"), ".json")
		decoded, err := hex.DecodeString(base)
		if err != nil || len(decoded) != 32 || name != ".kb/state/receipts/"+base+".json" {
			return problem("INVALID_BACKUP", "invalid durable receipt path")
		}
	}
	return nil
}

func excludedBackupPath(name string) bool {
	for _, prefix := range []string{".kb/cache", ".kb/backups", ".kb/restores"} {
		if name == prefix || strings.HasPrefix(name, prefix+"/") {
			return true
		}
	}
	if name == ".kb/state/lock" {
		return true
	}
	for _, part := range strings.Split(name, "/") {
		switch strings.ToLower(part) {
		case ".git", "node_modules", "vendor", ".claude", ".codex", ".agents", ".obsidian", ".venv", "venv", "__pycache__", ".idea":
			return true
		}
	}
	return false
}

func decodeManifest(raw []byte) (backupManifest, error) {
	var m backupManifest
	if err := decode(raw, &m); err != nil {
		return m, problem("INVALID_BACKUP", "malformed manifest")
	}
	// A canonical encoding rejects duplicate keys and alternative ambiguous JSON.
	canonical, err := json.Marshal(m)
	if err != nil || !bytes.Equal(raw, canonical) || m.Version != backupVersion || m.Kind != "kb-operational-backup" || len(m.Files) == 0 || len(m.Files) > maxBackupFiles || m.Excluded == nil {
		return m, problem("INVALID_BACKUP", "unsupported or noncanonical manifest")
	}
	names := map[string][]byte{}
	var total int64
	config := false
	for i, e := range m.Files {
		if err := backupPath(e.Path); err != nil {
			return m, err
		}
		if i > 0 && m.Files[i-1].Path >= e.Path || e.Size < 0 || e.Size > maxBackupFileBytes || !validRevision(e.Revision) {
			return m, problem("INVALID_BACKUP", "invalid or duplicate manifest entry")
		}
		if e.Path == ".kb/config.toml" && e.Size > api.MaxNoteBytes {
			return m, problem("INPUT_TOO_LARGE", "configuration exceeds the base-open byte limit")
		}
		total += e.Size
		if total > maxBackupBytes {
			return m, problem("INPUT_TOO_LARGE", "backup exceeds total byte limit")
		}
		names[e.Path] = nil
		config = config || e.Path == ".kb/config.toml"
	}
	if !config {
		return m, problem("INVALID_BACKUP", "backup is missing configuration")
	}
	if err := validatePathSet(names); err != nil {
		return m, err
	}
	for i, name := range m.Excluded {
		if err := cleanPath(strings.TrimSuffix(name, "/")); err != nil {
			return m, err
		}
		if i > 0 && m.Excluded[i-1] >= name {
			return m, problem("INVALID_BACKUP", "duplicate or unsorted exclusions")
		}
	}
	return m, nil
}

// readBackup accepts exactly the manifest followed by its ordered regular files.
// No links, device nodes, directories, sparse data or concatenated archives.
func readBackup(ctx context.Context, input io.Reader, consume func(backupEntry, io.Reader) error) (backupManifest, string, error) {
	bounded := &io.LimitedReader{R: input, N: maxArchiveBytes + 1}
	reader := tar.NewReader(bounded)
	header, err := reader.Next()
	if err != nil {
		return backupManifest{}, "", problem("INVALID_BACKUP", "missing archive manifest")
	}
	if err := regularHeader(header, manifestName, maxManifestBytes); err != nil {
		return backupManifest{}, "", err
	}
	raw, err := io.ReadAll(io.LimitReader(reader, maxManifestBytes+1))
	if err != nil || int64(len(raw)) != header.Size {
		return backupManifest{}, "", problem("INVALID_BACKUP", "incomplete manifest")
	}
	m, err := decodeManifest(raw)
	if err != nil {
		return m, "", err
	}
	for _, entry := range m.Files {
		if err := ctx.Err(); err != nil {
			return m, "", err
		}
		header, err := reader.Next()
		if err != nil {
			return m, "", problem("INVALID_BACKUP", "missing archive entry")
		}
		if err := regularHeader(header, entry.Path, entry.Size); err != nil {
			return m, "", err
		}
		if header.Size != entry.Size {
			return m, "", problem("INVALID_BACKUP", "entry size differs from manifest")
		}
		if err := consume(entry, reader); err != nil {
			return m, "", err
		}
	}
	// Require the final entry padding and two end-of-archive blocks. tar.Reader
	// alone accepts EOF without terminators, which would hide truncated backups.
	padding := (512 - m.Files[len(m.Files)-1].Size%512) % 512
	trailer := make([]byte, padding+1024)
	if _, err := io.ReadFull(bounded, trailer); err != nil || len(bytes.Trim(trailer, "\x00")) != 0 {
		return m, "", problem("INVALID_BACKUP", "missing archive terminator or extra entry")
	}
	buffer := make([]byte, 32768)
	for {
		n, err := bounded.Read(buffer)
		if len(bytes.Trim(buffer[:n], "\x00")) != 0 {
			return m, "", problem("INVALID_BACKUP", "trailing archive data")
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return m, "", err
		}
		if bounded.N == 0 {
			return m, "", problem("INPUT_TOO_LARGE", "archive exceeds byte limit")
		}
	}
	if bounded.N == 0 {
		return m, "", problem("INPUT_TOO_LARGE", "archive exceeds byte limit")
	}
	return m, api.Revision(raw), nil
}

func regularHeader(h *tar.Header, name string, limit int64) error {
	if h.Typeflag != tar.TypeReg || h.Name != name || h.Size < 0 || h.Size > limit || h.Linkname != "" {
		return problem("INVALID_BACKUP", "unsafe or oversized archive entry")
	}
	for key := range h.PAXRecords {
		if key != "path" {
			return problem("INVALID_BACKUP", "unsupported archive extension")
		}
	}
	return nil
}

func sortEntries(entries []backupEntry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
}
