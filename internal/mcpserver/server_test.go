package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func session(t *testing.T, a *Adapter) (*mcp.ClientSession, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	st, ct := mcp.NewInMemoryTransports()
	ss, err := New(a, "test").Connect(ctx, st, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	return cs, func() {
		if err := cs.Close(); err != nil {
			t.Error(err)
		}
		if err := ss.Close(); err != nil {
			t.Error(err)
		}
		cancel()
	}
}

func TestOfficialSDKToolsAndResourceProtocol(t *testing.T) {
	b := &backend{}
	cs, cleanup := session(t, &Adapter{Reader: b, Writer: b})
	defer cleanup()
	ctx := context.Background()
	tools, err := cs.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 10 {
		t.Fatalf("tools: %v %#v", err, tools)
	}
	for _, tool := range tools.Tools {
		if tool.Description == "" {
			t.Fatal("missing description")
		}
		schema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(schema), `"additionalProperties":false`) {
			t.Fatal("open schema")
		}
	}
	for _, action := range []string{"status", "list", "search", "read", "context", "create", "patch", "move", "apply", "validate"} {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "kb_" + action, Arguments: map[string]any{}})
		if err != nil || res.IsError || res.StructuredContent == nil || len(res.Content) != 1 {
			t.Fatalf("%s: %v %#v", action, err, res)
		}
	}
	if _, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: ResourceURI("Notes/a b.md")}); err != nil {
		t.Fatal(err)
	}
	if _, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "kb://local/%2e%2e/a.md"}); err == nil {
		t.Fatal("traversal accepted")
	}
	b.fail = &api.DomainError{Code: "NOT_FOUND", Message: "missing"}
	if _, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: ResourceURI("missing.md")}); err == nil {
		t.Fatal("resource failure lost")
	}
	if _, err := cs.ListResources(ctx, nil); err == nil {
		t.Fatal("resource inventory failure lost")
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "kb_read", Arguments: map[string]any{"path": "missing.md"}})
	if err != nil || !res.IsError {
		t.Fatalf("domain error not a tool error: %v %#v", err, res)
	}
}

func TestOfficialSDKReadonly(t *testing.T) {
	b := &backend{}
	cs, cleanup := session(t, &Adapter{Reader: b, Writer: b, ReadOnly: true})
	defer cleanup()
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 6 {
		t.Fatalf("%v %#v", err, tools)
	}
	for _, action := range []string{"create", "patch", "move", "apply"} {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "kb_" + action, Arguments: map[string]any{}})
		if err != nil || !res.IsError {
			t.Fatalf("write not rejected: %v %#v", err, res)
		}
	}
}
