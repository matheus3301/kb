package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHealthChecksLocalAssetPresenceWithoutIndexingOrFollowingSymlinks(t *testing.T) {
	root := realFixture(t)
	dir, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := dir.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := dir.MkdirAll("evidence", 0700); err != nil {
		t.Fatal(err)
	}
	if err := dir.WriteFile("evidence/data.json", []byte(`{"fixture":"private synthetic bytes not indexed"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "evidence/data.json"), filepath.Join(root, "evidence/linked.json")); err != nil {
		t.Fatal(err)
	}
	// Always-excluded operational files remain outside reference validation.
	if err := dir.WriteFile(".kb/hidden.json", []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	content := "---\ntype: Reference\n---\n[valid](evidence/data.json)\n[missing](evidence/missing.json)\n[symlink](evidence/linked.json)\n[excluded](.kb/hidden.json)\n[outside](../../outside.json)\n"
	if err := dir.WriteFile("assets.md", []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	_, result := invoke(t, []string{"--root", root, "validate", "--profile", "health"}, "")
	broken := 0
	for _, value := range result["data"].(map[string]any)["diagnostics"].([]any) {
		d := value.(map[string]any)
		if d["path"] == "assets.md" && d["rule"] == "BROKEN_LINK" {
			broken++
			if d["line"] == float64(4) {
				t.Fatal("existing asset was called broken")
			}
		}
	}
	if broken != 4 {
		t.Fatalf("expected missing/symlink/excluded/outside findings: %#v", result)
	}
	if err := dir.Remove("evidence/data.json"); err != nil {
		t.Fatal(err)
	}
	_, result = invoke(t, []string{"--root", root, "validate", "--profile", "health"}, "")
	broken = 0
	for _, value := range result["data"].(map[string]any)["diagnostics"].([]any) {
		d := value.(map[string]any)
		if d["path"] == "assets.md" && d["rule"] == "BROKEN_LINK" {
			broken++
		}
	}
	if broken != 5 {
		t.Fatal("asset removal not detected without a Markdown change", result)
	}
	code, read := invoke(t, []string{"--root", root, "read", "evidence/data.json"}, "")
	if code == 0 || read["ok"] != false {
		t.Fatal("asset check expanded read permissions")
	}
}
