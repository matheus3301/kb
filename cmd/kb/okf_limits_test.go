package main

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func TestMetadataLimitFailuresKeepCLIAndMCPParity(t *testing.T) {
	var deep any = "leaf"
	for range 40 {
		deep = map[string]any{"child": deep}
	}
	large := strings.Repeat("x", api.MaxNoteBytes+1)
	for _, test := range []struct {
		name, original string
		value          any
		code, rule     string
		exit           int
	}{
		{"create-deep", "", deep, "INVALID_NOTE", "YAML_LIMIT", 1},
		{"create-large", "", large, "INVALID_NOTE", "LIMIT_EXCEEDED", 1},
		{"patch-new-large", "---\ntype: Reference\n---\nBody\n", large, "INVALID_NOTE", "LIMIT_EXCEEDED", 1},
		{"patch-scalar-large", "---\ntype: Reference\ncustom: old\n---\nBody\n", large, "LIMIT_EXCEEDED", "", 2},
		{"patch-block-large", "---\ntype: Reference\ncustom: |\n  old\n---\nBody\n", large, "INVALID_NOTE", "LIMIT_EXCEEDED", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := realFixture(t)
			files := fidelityFiles(t, root)
			op := api.Operation{OperationID: "change", Kind: "create", Path: "note.md", Type: "Reference", Metadata: map[string]any{"custom": test.value}}
			if test.original != "" {
				if err := files.WriteFile("note.md", []byte(test.original), 0600); err != nil {
					t.Fatal(err)
				}
				op = api.Operation{OperationID: "change", Kind: "patch", Path: "note.md", ExpectedRevision: api.Revision([]byte(test.original)), MetadataPatch: map[string]any{"custom": test.value}}
			}
			request, err := json.Marshal(api.Plan{Operations: []api.Operation{op}, Actor: "process:limits", Summary: "Check synthetic metadata bounds", IdempotencyKey: test.name})
			if err != nil {
				t.Fatal(err)
			}
			exit, cli := invoke(t, []string{"--root", root, "apply", "--input", "-"}, string(request))
			if exit != test.exit {
				t.Fatalf("exit %d, want %d: %+v", exit, test.exit, cli)
			}
			session, cleanup := realSession(t, root, false, nil)
			defer cleanup()
			mcp := protocolCall(t, session, "apply", string(request))
			if !reflect.DeepEqual(cli["error"], mcp["error"]) {
				t.Fatalf("CLI error %+v != MCP %+v", cli["error"], mcp["error"])
			}
			failure := cli["error"].(map[string]any)
			if failure["code"] != test.code {
				t.Fatal(failure)
			}
			if test.rule == "" {
				if failure["details"] != nil {
					t.Fatal(failure)
				}
			} else {
				details, ok := failure["details"].([]any)
				if !ok || len(details) != 1 {
					t.Fatal(failure)
				}
				finding := details[0].(map[string]any)
				if finding["code"] != "INVALID_NOTE" || finding["rule"] != test.rule || finding["layer"] != "conformance_errors" || finding["path"] != "note.md" || finding["severity"] != "error" {
					t.Fatal(finding)
				}
			}
			if test.original == "" {
				if _, err := files.Stat("note.md"); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("rejected create published a note: %v", err)
				}
			} else {
				unchanged, err := files.ReadFile("note.md")
				if err != nil || string(unchanged) != test.original {
					t.Fatal("rejected patch changed note", err)
				}
			}
		})
	}
}
