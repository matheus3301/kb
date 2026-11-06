package setup

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzJSONCEntryPreservation(f *testing.F) {
	for _, seed := range []string{`{}`, `{"foreign":1.000}`, "{ //keep\n\"url\":\"https://example.test/*not*/\",\n}", `{"a":{"a":2}}`, `{"duplicate":1,"duplicate":2}`, `{"x":[1,2,]}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 8192 {
			return
		}
		raw := []byte(text)
		if _, err := parseJSONFormat(raw, "jsonc"); err != nil {
			return
		}
		a := Artifact{Kind: "jsonc", Key: "kb-test", Content: []byte(`{"command":"kb"}`)}
		out, managed, borrowed, err := editArtifact(raw, a, nil, false)
		if err != nil {
			return
		}
		if _, err := parseJSONFormat(out, "jsonc"); err != nil {
			t.Fatalf("edit produced invalid configuration: %v", err)
		}
		if borrowed {
			return
		}
		restored, _, _, err := editArtifact(out, a, &asset{Artifact: a, Managed: managed}, true)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parseJSONFormat(restored, "jsonc"); err != nil {
			t.Fatalf("removal produced invalid configuration: %v", err)
		}
	})
}
func FuzzManagedBlockParsing(f *testing.F) {
	for _, seed := range []string{"# User instructions", "\ufeff# Header\r\nNo final newline", "<!-- kb:test:v1 begin -->\nmissing", "<!-- kb:test:v1 end -->\n<!-- kb:test:v1 begin -->"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 8192 {
			return
		}
		raw := []byte(text)
		a := Artifact{Kind: "block", Key: "test", Content: []byte("Bounded synthetic policy")}
		out, managed, borrowed, err := editArtifact(raw, a, nil, false)
		if err != nil || borrowed {
			return
		}
		if !bytes.HasPrefix(out, raw) {
			t.Fatal("insertion changed foreign bytes")
		}
		restored, _, _, err := editArtifact(out, a, &asset{Artifact: a, Managed: managed}, true)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(restored, raw) {
			t.Fatal("removal changed foreign prefix")
		}
	})
}
func FuzzConfinedSetupPath(f *testing.F) {
	for _, seed := range []string{".agents/skills/a", "../escape", "/absolute", "a/../../escape", "a\x00b", "CaseAlias"} {
		f.Add(seed)
	}
	root := f.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, name string) {
		if len(name) > 512 || strings.ContainsRune(name, 0) {
			return
		}
		if !filepath.IsLocal(name) {
			return
		}
		path := filepath.Join(root, name)
		if !within(root, path) {
			t.Fatal("local path escaped selected root")
		}
		canonical, err := Canonical(path)
		if err != nil {
			return
		}
		if !within(root, canonical) {
			t.Fatal("canonical path escaped selected root")
		}
		if err := safePath(canonical); err != nil {
			return
		}
	})
}
