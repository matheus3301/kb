package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Keep the public catalog synchronized with what an actual client discovers,
// rather than maintaining a second, disconnected set of instructions/schemas.
func TestToolCatalogMatchesRuntime(t *testing.T) {
	b := &backend{}
	cs, closeSession := session(t, &Adapter{Reader: b, Writer: b})
	defer closeSession()
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// The ten advertised tools fit in one default discovery page.
	sort.Slice(tools.Tools, func(i, j int) bool { return tools.Tools[i].Name < tools.Tools[j].Name })
	readonly, closeReadonly := session(t, &Adapter{Reader: b, Writer: b, ReadOnly: true})
	defer closeReadonly()
	catalog := map[string]any{
		"protocol_baseline": "2025-11-25", "schema_version": "kb.v1",
		"instructions":           cs.InitializeResult().Instructions,
		"read_only_instructions": readonly.InitializeResult().Instructions,
		"tools":                  tools.Tools,
	}
	path := "../../schemas/kb.v1.tools.json"
	if os.Getenv("KB_UPDATE_TOOL_CATALOG") == "1" {
		data, err := json.MarshalIndent(catalog, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var expected, actual any
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatal("public MCP catalog differs from initialize/tools/list; regenerate with KB_UPDATE_TOOL_CATALOG=1 go test ./internal/mcpserver -run '^TestToolCatalogMatchesRuntime$'")
	}
}

func TestMCPDiscoveryProvidesIndependentGuidance(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		t.Run(map[bool]string{false: "writable", true: "readonly"}[readOnly], func(t *testing.T) {
			b := &backend{}
			cs, cleanup := session(t, &Adapter{Reader: b, Writer: b, ReadOnly: readOnly})
			defer cleanup()
			if cs.InitializeResult().Instructions == "" {
				t.Fatal("MCP client received no operating instructions")
			}
			result, err := cs.ListTools(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, tool := range result.Tools {
				assertFieldGuidance(t, tool.Name, tool.InputSchema.(map[string]any))
				output, ok := tool.OutputSchema.(map[string]any)
				if !ok || output["description"] == "" {
					t.Fatalf("%s lacks structured result guidance", tool.Name)
				}
				if readOnly && !tool.Annotations.ReadOnlyHint {
					t.Fatalf("read-only discovery advertises %s", tool.Name)
				}
				if tool.Name == "kb_create" {
					required := tool.InputSchema.(map[string]any)["required"].([]any)
					for _, key := range required {
						if key == "expected_revision" {
							t.Fatal("creation must not require a prior note revision")
						}
					}
				}
			}
		})
	}
}

func assertFieldGuidance(t *testing.T, path string, schema map[string]any) {
	t.Helper()
	if properties, ok := schema["properties"].(map[string]any); ok {
		for name, value := range properties {
			property := value.(map[string]any)
			description, _ := property["description"].(string)
			if strings.TrimSpace(description) == "" {
				t.Errorf("%s.%s has no field guidance", path, name)
			}
			assertFieldGuidance(t, path+"."+name, property)
		}
	}
	if items, ok := schema["items"].(map[string]any); ok {
		assertFieldGuidance(t, path+"[]", items)
	}
}

func TestMCPResultEnvelopesSurviveTransport(t *testing.T) {
	b := &backend{}
	cs, cleanup := session(t, &Adapter{Reader: b, Writer: b})
	defer cleanup()
	for _, args := range []map[string]any{{}, {"unexpected": true}} {
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "kb_status", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		envelope := result.StructuredContent.(map[string]any)
		for _, name := range []string{"schema_version", "ok", "warnings", "meta"} {
			if _, exists := envelope[name]; !exists {
				t.Fatalf("missing required result field %s", name)
			}
		}
		_, hasData := envelope["data"]
		_, hasError := envelope["error"]
		if hasData == hasError || result.IsError != hasError {
			t.Fatalf("success/error envelope inconsistent: %#v", envelope)
		}
	}
}
