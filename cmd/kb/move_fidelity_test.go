package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func TestMovePreservesDependentMetadataThroughCLIAndMCP(t *testing.T) {
	for _, transport := range []string{"cli", "mcp"} {
		for _, style := range []string{"block", "flow", "crlf", "alias"} {
			t.Run(transport+"/"+style, func(t *testing.T) {
				root := realFixture(t)
				source := []byte("---\ntype: Reference\n---\n# Source\n")
				if err := os.WriteFile(filepath.Join(root, "source.md"), source, 0o600); err != nil {
					t.Fatal(err)
				}
				metadata := "sources:\n    # Source annotation\n    - resource: 'source.md' # Source path\n      custom: {decimal: 1234567890.123456789, tiny: 0.1234567890123456789} # Keep flow\n      text: \"café 🧪\" # Keep quotes\n" +
					"basis_revisions:\n    # Basis annotation\n    - path: \"source.md\" # Basis path\n      revision: '" + api.Revision(source) + "'\n      custom: [1.234567890123456789e+23, 9007199254740993123456789] # Keep numbers\n"
				if style == "flow" {
					metadata = "sources: [{resource: 'source.md', custom: 1234567890.123456789}] # Source annotation\n" +
						"basis_revisions: [{path: \"source.md\", revision: '" + api.Revision(source) + "', custom: 0.1234567890123456789}] # Basis annotation\n"
				}
				if style == "alias" {
					metadata = "template: &evidence\n  - resource: 'source.md' # Anchored path\n    custom: 1234567890.123456789 # Anchored precision\nsources: *evidence # Preserve alias annotation\n" +
						"basis_revisions:\n  - path: source.md\n    revision: '" + api.Revision(source) + "'\n"
				}
				original := "---\ntype: Synthesis\n" + metadata + "---\n# Synthesis\n"
				if style == "crlf" {
					original = strings.ReplaceAll(original, "\n", "\r\n")
				}
				name := filepath.Join(root, "synthesis.md")
				if err := os.WriteFile(name, []byte(original), 0o600); err != nil {
					t.Fatal(err)
				}
				request := fmt.Sprintf(`{"path":"source.md","to":"nested/source.md","expected_revision":%q,"actor":"process:fidelity-test","summary":"Relocate synthetic evidence","idempotency_key":"move-fidelity"}`, api.Revision(source))
				if transport == "cli" {
					if code, result := invoke(t, []string{"--root", root, "move", "--input", "-"}, request); code != 0 {
						t.Fatalf("move: %d %+v", code, result)
					}
				} else {
					session, cleanup := realSession(t, root, false, nil)
					defer cleanup()
					if result := protocolCall(t, session, "move", request); result["ok"] != true {
						t.Fatalf("move: %+v", result)
					}
				}
				files := fidelityFiles(t, root)
				after, err := files.ReadFile("synthesis.md")
				if err != nil {
					t.Fatal(err)
				}
				if style != "alias" {
					want := strings.ReplaceAll(metadata, "source.md", "nested/source.md")
					if style == "crlf" {
						want = strings.ReplaceAll(want, "\n", "\r\n")
					}
					if !strings.Contains(string(after), want) {
						t.Fatalf("dependent metadata changed beyond paths\nwant:\n%s\nafter:\n%s", want, after)
					}
				} else {
					for _, want := range []string{"template: &evidence\n  - resource: 'source.md' # Anchored path", "1234567890.123456789", "# Preserve alias annotation", "nested/source.md"} {
						if !strings.Contains(string(after), want) {
							t.Fatalf("missing %q after alias rewrite:\n%s", want, after)
						}
					}
				}
				if code, result := invoke(t, []string{"--root", root, "validate", "--profile", "okf"}, ""); code != 0 {
					t.Fatalf("validation: %+v", result)
				}
			})
		}
	}
}

func fidelityFiles(t *testing.T, root string) *os.Root {
	t.Helper()
	files, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := files.Close(); err != nil {
			t.Error(err)
		}
	})
	return files
}

func TestRelocatedSynthesisCoordinatedMovesAndArchivePreserveMetadata(t *testing.T) {
	for _, scenario := range []string{"outgoing", "coordinated", "archive"} {
		t.Run(scenario, func(t *testing.T) {
			root := realFixture(t)
			files := fidelityFiles(t, root)
			source := []byte("---\ntype: Reference\n---\n# Source\n")
			if err := files.WriteFile("source.md", source, 0o600); err != nil {
				t.Fatal(err)
			}
			metadata := "sources:\n  # Source annotation\n  - resource: 'source.md' # Keep quotes\n    precision: 1234567890.123456789\n" +
				"basis_revisions:\n  # Basis annotation\n  - path: 'source.md'\n    revision: '" + api.Revision(source) + "'\n    precision: 0.1234567890123456789\n"
			synthesis := []byte("---\ntype: Synthesis\n" + metadata + "---\n# Synthesis\n")
			if err := files.WriteFile("synthesis.md", synthesis, 0o600); err != nil {
				t.Fatal(err)
			}
			actor, summary, key := "process:fidelity-test", "Relocate synthetic evidence", "fidelity/"+scenario
			request := map[string]any{"actor": actor, "summary": summary, "idempotency_key": key}
			action, name, wantResource, wantBasis := "move", "nested/synthesis.md", "../source.md", "source.md"
			switch scenario {
			case "outgoing":
				request["path"], request["to"], request["expected_revision"] = "synthesis.md", name, api.Revision(synthesis)
			case "coordinated":
				action, name, wantResource, wantBasis = "apply", "deep/synthesis.md", "../nested/source.md", "nested/source.md"
				request["operations"] = []api.Operation{
					{OperationID: "source", Kind: "move", Path: "source.md", To: "nested/source.md", ExpectedRevision: api.Revision(source)},
					{OperationID: "synthesis", Kind: "move", Path: "synthesis.md", To: name, ExpectedRevision: api.Revision(synthesis)},
				}
			case "archive":
				action, name, wantResource, wantBasis = "archive", "synthesis.md", "04_Archive/source.md", "04_Archive/source.md"
				request["path"], request["expected_revision"] = "source.md", api.Revision(source)
			}
			payload, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			if code, result := invoke(t, []string{"--root", root, action, "--input", "-", "--dry-run"}, string(payload)); code != 0 {
				t.Fatalf("preview: %d %+v", code, result)
			}
			unchanged, err := files.ReadFile("synthesis.md")
			if err != nil || string(unchanged) != string(synthesis) {
				t.Fatal("preview changed synthesis metadata", err)
			}
			if code, result := invoke(t, []string{"--root", root, action, "--input", "-"}, string(payload)); code != 0 {
				t.Fatalf("apply: %d %+v", code, result)
			}
			after, err := files.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			want := strings.Replace(metadata, "resource: 'source.md'", "resource: '"+wantResource+"'", 1)
			want = strings.Replace(want, "path: 'source.md'", "path: '"+wantBasis+"'", 1)
			if !strings.Contains(string(after), want) {
				t.Fatalf("move changed unrelated metadata\nwant:\n%s\nafter:\n%s", want, after)
			}
		})
	}
}
