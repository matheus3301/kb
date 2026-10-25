package write

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func TestOperationalManifestBoundaries(t *testing.T) {
	valid := backupManifest{Version: 1, Kind: "kb-operational-backup", Files: []backupEntry{{Path: ".kb/config.toml", Size: 0, Revision: api.Revision(nil)}, {Path: "nested/note.md", Size: 1, Revision: api.Revision([]byte("x"))}}, Excluded: []string{}}
	for _, tc := range []struct {
		name  string
		alter func(*backupManifest)
	}{
		{"negative-size", func(m *backupManifest) { m.Files[1].Size = -1 }},
		{"invalid-revision", func(m *backupManifest) { m.Files[1].Revision = "sha256:invalid" }},
		{"file-directory-conflict", func(m *backupManifest) { m.Files[1].Path = ".kb/config.toml/child" }},
		{"case-fold-collision", func(m *backupManifest) {
			m.Files = append(m.Files, backupEntry{Path: "Nested/other.md", Revision: api.Revision(nil)})
			sortEntries(m.Files)
		}},
		{"unsafe-exclusion", func(m *backupManifest) { m.Excluded = []string{"../outside/"} }},
		{"duplicate-exclusion", func(m *backupManifest) { m.Excluded = []string{".git/", ".git/"} }},
		{"invalid-receipt-path", func(m *backupManifest) { m.Files[1].Path = ".kb/state/receipts/fake.json" }},
		{"too-many-files", func(m *backupManifest) { m.Files = make([]backupEntry, maxBackupFiles+1) }},
		{"total-size", func(m *backupManifest) {
			m.Files = m.Files[:1]
			for i := range 17 {
				m.Files = append(m.Files, backupEntry{Path: string(rune('a'+i)) + ".bin", Size: maxBackupFileBytes, Revision: api.Revision(nil)})
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := valid
			m.Files = append([]backupEntry(nil), valid.Files...)
			tc.alter(&m)
			raw, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeManifest(raw); err == nil {
				t.Fatal("unsafe manifest accepted")
			}
		})
	}
}

func TestOperationalUnicodeArchiveAndUnsupportedExtensions(t *testing.T) {
	name := "notes/" + string(bytes.Repeat([]byte("a"), 100)) + "/café.md"
	data := []byte("exact Unicode path content")
	m := backupManifest{Version: 1, Kind: "kb-operational-backup", Files: []backupEntry{{Path: ".kb/config.toml", Size: 0, Revision: api.Revision(nil)}, {Path: name, Size: int64(len(data)), Revision: api.Revision(data)}}, Excluded: []string{}}
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, extension := range []bool{false, true} {
		var buffer bytes.Buffer
		writer := tar.NewWriter(&buffer)
		for _, entry := range []struct {
			name string
			data []byte
		}{{manifestName, raw}, {".kb/config.toml", nil}, {name, data}} {
			header := &tar.Header{Name: entry.name, Size: int64(len(entry.data)), Typeflag: tar.TypeReg, Mode: 0600, Format: tar.FormatPAX}
			if extension {
				header.PAXRecords = map[string]string{"comment": "unsupported"}
			}
			if err := writer.WriteHeader(header); err != nil {
				t.Fatal(err)
			}
			if _, err := writer.Write(entry.data); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		_, _, err := readBackup(context.Background(), bytes.NewReader(buffer.Bytes()), verifyBackupEntry)
		if (err != nil) != extension {
			t.Fatalf("extension=%v: %v", extension, err)
		}
	}
}

// Hashing streams is kept separate for fuzz/consumer checks without retaining
// archive-sized images. Individual source files remain bounded above.
func verifyBackupEntry(entry backupEntry, reader io.Reader) error {
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(reader, entry.Size+1))
	if err != nil {
		return err
	}
	if count != entry.Size || "sha256:"+hex.EncodeToString(hash.Sum(nil)) != entry.Revision {
		return problem("INVALID_BACKUP", "archive entry checksum or length mismatch")
	}
	return nil
}

func TestConfigurationOnlyRootBackupRestore(t *testing.T) {
	s, p := setup(t)
	put(t, p, ".kb/config.toml", []byte("version = 1\n"))
	archive := api.BackupInput{Archive: ".kb/backups/config-only.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/config-only"}); err != nil {
		t.Fatal(err)
	}
}
