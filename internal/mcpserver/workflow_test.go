package mcpserver

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/core"
	"github.com/matheus3301/kb/internal/write"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Exercise the advertised MCP surface against real services. No skill files,
// product documentation, shell commands, or AGENTS.md are part of the workflow.
func TestStandaloneMCPWorkflow(t *testing.T) {
	root := t.TempDir()
	store, err := core.Init(context.Background(), root, true)
	if err != nil {
		t.Fatal(err)
	}
	files, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := files.Close(); err != nil {
			t.Error(err)
		}
	})
	writer := write.New(store)
	cs, cleanup := session(t, &Adapter{Reader: store, Writer: writer})
	defer cleanup()
	call := func(name string, args map[string]any, wantError bool) map[string]any {
		t.Helper()
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil || result.IsError != wantError {
			t.Fatalf("%s: err=%v result=%#v", name, err, result)
		}
		return result.StructuredContent.(map[string]any)
	}
	call("kb_status", map[string]any{}, false)
	create := map[string]any{
		"path": "03_Resources/policy.md", "type": "Reference", "title": "Retention policy",
		"body": "# Retention policy\n\nRetain records for 14 days.\n", "actor": "process:mcp-test",
		"summary": "Record the synthetic policy", "idempotency_key": "workflow/create", "dry_run": true,
	}
	call("kb_create", create, false)
	if _, err := files.Stat("03_Resources/policy.md"); !os.IsNotExist(err) {
		t.Fatal("MCP preview published its note")
	}
	create["dry_run"] = false
	call("kb_create", create, false)
	replay := call("kb_create", create, false)["data"].(map[string]any)
	if replay["replayed"] != true {
		t.Fatal("MCP retry duplicated the write")
	}
	search := call("kb_search", map[string]any{"query": "retention", "limit": 5}, false)["data"].(map[string]any)
	if len(search["hits"].([]any)) != 1 {
		t.Fatalf("unexpected search result: %#v", search)
	}
	read := call("kb_read", map[string]any{"path": create["path"]}, false)["data"].(map[string]any)
	revision := read["note"].(map[string]any)["revision"]
	patch := map[string]any{
		"path": create["path"], "expected_revision": revision,
		"body_edits": []any{map[string]any{"old_text": "14 days", "new_text": "30 days"}},
		"actor":      "process:mcp-test", "summary": "Update the interval", "idempotency_key": "workflow/patch", "dry_run": true,
	}
	call("kb_patch", patch, false)
	patch["dry_run"] = false
	call("kb_patch", patch, false)
	patch["idempotency_key"] = "workflow/stale"
	conflict := call("kb_patch", patch, true)["error"].(map[string]any)
	if conflict["code"] != "REVISION_CONFLICT" {
		t.Fatalf("missing revision conflict: %#v", conflict)
	}
	beforeNote, err := files.ReadFile("03_Resources/policy.md")
	if err != nil {
		t.Fatal(err)
	}
	beforeLog, err := files.ReadFile("log.md")
	if err != nil {
		t.Fatal(err)
	}
	context := call("kb_context", map[string]any{"query": "retention", "max_tokens": 1000}, false)["data"].(map[string]any)
	if len(context["sources"].([]any)) == 0 || !strings.Contains(context["text"].(string), "30 days") {
		t.Fatal("MCP context lost source citations")
	}
	validation := call("kb_validate", map[string]any{"profile": "health"}, false)["data"].(map[string]any)
	if validation["conformant"] != true {
		t.Fatalf("unexpected health findings: %#v", validation)
	}
	for path, before := range map[string][]byte{"03_Resources/policy.md": beforeNote, "log.md": beforeLog} {
		after, err := files.ReadFile(path)
		if err != nil || string(before) != string(after) {
			t.Fatalf("MCP retrieval changed %s: %v", path, err)
		}
	}
}
