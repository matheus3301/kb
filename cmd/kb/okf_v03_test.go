package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestV03RootMetadataFidelityThroughCLIAndMCP(t *testing.T) {
	for _, transport := range []string{"cli", "mcp"} {
		for _, newline := range []string{"\n", "\r\n"} {
			t.Run(transport+"/"+map[string]string{"\n": "LF", "\r\n": "CRLF"}[newline], func(t *testing.T) {
				root := realFixture(t)
				files := fidelityFiles(t, root)
				header := strings.ReplaceAll("---\n# Preserve bundle metadata exactly\nokf_version: '0.3'\ntitle: '[untrusted]<script>'\ndescription: |\n  <!-- kb:index:start -->\n  <!-- kb:index:end -->\nlicense: ../../outside.txt\nx-fixture: &fixture {precision: 1234567890.123456789, text: 'café'}\nowner: *fixture # Unprefixed extensions are permitted\nverified: {by: 'human:synthetic', at: '2026-01-01T00:00:00Z'}\n---\n", "\n", newline)
				original := header + "# Synthetic bundle\n\nManual prose.\n"
				if err := files.WriteFile("index.md", []byte(original), 0600); err != nil {
					t.Fatal(err)
				}
				untouched, err := files.ReadFile("03_Resources/alpha note.md")
				if err != nil {
					t.Fatal(err)
				}
				if code, result := invoke(t, []string{"--root", root, "maintain", "--dry-run"}, ""); code != 0 {
					t.Fatalf("preview: %d %#v", code, result)
				}
				if raw, err := files.ReadFile("index.md"); err != nil || string(raw) != original {
					t.Fatal("preview changed root index", err)
				}
				if code, result := invoke(t, []string{"--root", root, "maintain", "--apply"}, ""); code != 0 {
					t.Fatalf("maintain: %d %#v", code, result)
				}
				call := func(action, raw string) map[string]any {
					_, result := invoke(t, []string{"--root", root, action, "--input", "-"}, raw)
					return result
				}
				if transport == "mcp" {
					session, cleanup := realSession(t, root, false, nil)
					defer cleanup()
					call = func(action, raw string) map[string]any { return protocolCall(t, session, action, raw) }
				}
				payload, err := json.Marshal(map[string]any{
					"path": "new note.md", "type": "Reference", "body": "Synthetic evidence.\n",
					"metadata": map[string]any{"title": "[unsafe]<script>", "description": "[outside](../../outside.md)"},
					"actor":    "process:v03-fixture", "summary": "Create synthetic evidence", "idempotency_key": "v03-create",
				})
				if err != nil {
					t.Fatal(err)
				}
				if result := call("create", string(payload)); result["ok"] != true {
					t.Fatalf("create: %#v", result)
				}
				after, err := files.ReadFile("index.md")
				if err != nil || !strings.HasPrefix(string(after), header) || !strings.Contains(string(after), "Manual prose.") {
					t.Fatal("write changed bundle header or manual prose", err)
				}
				if !strings.Contains(string(after), "- [\\[unsafe\\]&lt;script&gt;](new%20note.md) — \\[outside\\](../../outside.md)") {
					t.Fatalf("unsafe generated navigation: %s", after)
				}
				if raw, err := files.ReadFile("03_Resources/alpha note.md"); err != nil || string(raw) != string(untouched) {
					t.Fatal("write changed an unrelated note", err)
				}
				read := call("read", `{"path":"new note.md"}`)
				if read["ok"] != true || read["data"].(map[string]any)["note"].(map[string]any)["trust"] != "unverified" {
					t.Fatal("bundle verification leaked into concept trust", read)
				}
				if result := call("validate", `{"profile":"okf"}`); result["data"].(map[string]any)["conformant"] != true {
					t.Fatal(result)
				}
				destination := filepath.Join(t.TempDir(), "bundle")
				if code, result := invoke(t, []string{"--root", root, "export", destination}, ""); code != 0 || result["data"].(map[string]any)["conformant"] != true {
					t.Fatalf("export: %d %#v", code, result)
				}
				exported := fidelityFiles(t, destination)
				if raw, err := exported.ReadFile("index.md"); err != nil || string(raw) != string(after) {
					t.Fatal("export changed root index bytes", err)
				}
			})
		}
	}
}

func TestV03RootMetadataDiagnosticsParity(t *testing.T) {
	root := realFixture(t)
	files := fidelityFiles(t, root)
	session, cleanup := realSession(t, root, false, nil)
	defer cleanup()
	for _, test := range []struct {
		header string
		valid  bool
	}{
		{"okf_version: '0.2'", true},
		{"okf_version: '99.0'", true},
		{"okf_version: '0.2'\ntitle: Synthetic", false},
		{"okf_version: '99.0'\ntitle: Synthetic", false},
		{"title: Synthetic", false},
		{"okf_version: 0.3", false},
		{"okf_version: '0.3'\ntitle: 42\nlicense: {}", false},
		{"okf_version: '0.3'\nx-fixture: {duplicate: one, duplicate: two}", false},
		{"okf_version: '0.3'\ntitle: Synthetic\nlicense: /LICENSE.txt\nx-fixture: {nested: [one, two]}", true},
	} {
		if err := files.WriteFile("index.md", []byte("---\n"+test.header+"\n---\n# Bundle\n"), 0600); err != nil {
			t.Fatal(err)
		}
		_, cli := invoke(t, []string{"--root", root, "validate", "--profile", "okf"}, "")
		remote := protocolCall(t, session, "validate", `{"profile":"okf"}`)
		if !reflect.DeepEqual(cli, remote) || cli["data"].(map[string]any)["conformant"] != test.valid {
			t.Fatalf("%s: CLI %#v MCP %#v", test.header, cli, remote)
		}
	}
	// A metadata license path grants no filesystem access, through either interface.
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("Synthetic outside sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.md")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../outside.txt", "escape.md"} {
		raw, err := json.Marshal(map[string]any{"path": path})
		if err != nil {
			t.Fatal(err)
		}
		code, cli := invoke(t, []string{"--root", root, "read", "--input", "-"}, string(raw))
		if remote := protocolCall(t, session, "read", string(raw)); code == 0 || cli["ok"] != false || !reflect.DeepEqual(cli, remote) {
			t.Fatalf("escape read parity: %d %#v %#v", code, cli, remote)
		}
	}
}
