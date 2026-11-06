package agentsetup_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/matheus3301/kb/internal/agentsetup"
	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/internal/core"
)

// TestGeneratedMCPLaunchUsesIndependentServer exercises the generated executable
// and argv, not a copied expected argv or an in-memory server. No agent binary,
// shell, CLI skill, setup resolver, account or network is needed by the consumer.
func TestGeneratedMCPLaunchUsesIndependentServer(t *testing.T) {
	t.Setenv("KB_ROOT", "")
	binary := filepath.Join(t.TempDir(), "kb")
	buildCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// #nosec G204 -- Builds this repository into a test-owned disposable path; argv is fixed.
	build := exec.CommandContext(buildCtx, "go", "build", "-trimpath", "-o", binary, "./cmd/kb")
	build.Dir = filepath.Join("..", "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build KB: %v\n%s", err, output)
	}
	root := filepath.Join(t.TempDir(), "external knowledge é")
	store, err := core.Init(context.Background(), root, false)
	if err != nil {
		t.Fatal(err)
	}
	root = store.Root()
	note := []byte("---\ntype: Reference\ntitle: Synthetic Alpha\n---\n# Alpha\nAlpha evidence: synthetic orbit code is DELTA-7.\n")
	if err := os.WriteFile(filepath.Join(root, "alpha.md"), note, 0600); err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- Reads only the log initialized in this test's disposable collection.
	logBefore, err := os.ReadFile(filepath.Join(root, "log.md"))
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	home := t.TempDir()
	for _, d := range agentsetup.Agents() {
		t.Run(d.ID, func(t *testing.T) {
			rendered, err := agentsetup.Render(agentsetup.Request{Agent: d.ID, Scope: "project", Project: project, Home: home, ConfigHome: filepath.Join(home, "config"), Executable: binary, Root: root, Mode: "mcp", Consult: "always", Learning: "off", Hooks: "off"})
			if err != nil {
				t.Fatal(err)
			}
			if rendered.SkillDir != "" {
				t.Fatal("MCP consumer requires a CLI skill")
			}
			command, args := generatedCommand(t, d.ID, rendered)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			// #nosec G204 -- Intentionally executes the renderer's launch contract using a binary built by this test.
			process := exec.CommandContext(ctx, command, args...)
			process.Dir = project
			stdin, err := process.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			stdout, err := process.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			process.Stderr = &stderr
			if err := process.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = stdin.Close()
				if err := process.Wait(); err != nil {
					t.Errorf("generated MCP command: %v; stderr=%s", err, stderr.String())
				}
			}()
			client := wireClient{writer: json.NewEncoder(stdin), reader: json.NewDecoder(stdout)}
			var initialized struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			client.call(t, 1, "initialize", map[string]any{"protocolVersion": "2025-11-25", "capabilities": map[string]any{}, "clientInfo": map[string]string{"name": "synthetic-mcp-only-consumer", "version": "1"}}, &initialized)
			if initialized.ProtocolVersion != "2025-11-25" {
				t.Fatalf("protocol version: %s", initialized.ProtocolVersion)
			}
			if err := client.writer.Encode(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"}); err != nil {
				t.Fatal(err)
			}
			var tools struct{ Tools []struct{ Name string } }
			client.call(t, 2, "tools/list", map[string]any{}, &tools)
			names := map[string]bool{}
			for _, tool := range tools.Tools {
				names[tool.Name] = true
			}
			if !names["kb_search"] || !names["kb_read"] || names["kb_create"] || names["kb_patch"] {
				t.Fatalf("read-only tool surface: %v", names)
			}
			var search struct {
				IsError           bool
				StructuredContent struct {
					OK   bool
					Data api.SearchResult
				}
			}
			client.call(t, 3, "tools/call", map[string]any{"name": "kb_search", "arguments": map[string]string{"query": "alpha"}}, &search)
			if search.IsError || !search.StructuredContent.OK || len(search.StructuredContent.Data.Hits) != 1 || search.StructuredContent.Data.Hits[0].Path != "alpha.md" {
				t.Fatalf("search failed: %+v", search)
			}
			var read struct {
				IsError           bool
				StructuredContent struct {
					OK   bool
					Data api.ReadResult
					Meta struct{ Root string }
				}
			}
			client.call(t, 4, "tools/call", map[string]any{"name": "kb_read", "arguments": map[string]string{"path": "alpha.md"}}, &read)
			result := read.StructuredContent
			if read.IsError || !result.OK || !strings.Contains(result.Data.Markdown, "DELTA-7") || !strings.HasPrefix(result.Data.Note.Revision, "sha256:") || result.Meta.Root != root {
				t.Fatalf("citable fixed-root read failed: %+v", read)
			}
		})
	}
	// #nosec G304 -- Reads only the synthetic note created above in this test's disposable root.
	noteAfter, err := os.ReadFile(filepath.Join(root, "alpha.md"))
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G304 -- Reads only this test's initialized disposable collection log.
	logAfter, err := os.ReadFile(filepath.Join(root, "log.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(note, noteAfter) || !bytes.Equal(logBefore, logAfter) {
		t.Fatal("read-only MCP consumer modified note or log")
	}
}

func generatedCommand(t *testing.T, agent string, rendered agentsetup.Rendered) (string, []string) {
	t.Helper()
	for _, artifact := range rendered.Artifacts {
		var entry map[string]any
		switch artifact.Kind {
		case "json", "jsonc":
			if err := json.Unmarshal(artifact.Content, &entry); err != nil {
				t.Fatal(err)
			}
		case "toml":
			var document map[string]any
			if _, err := toml.Decode(string(artifact.Content), &document); err != nil {
				t.Fatal(err)
			}
			entry = document["mcp_servers"].(map[string]any)["knowledge-base"].(map[string]any)
		default:
			continue
		}
		command, ok := entry["command"].(string)
		var args []string
		if agent == "opencode" {
			values := entry["command"].([]any)
			command = values[0].(string)
			entry["args"] = values[1:]
			ok = true
		}
		if !ok {
			t.Fatal("missing executable in generated MCP entry")
		}
		switch values := entry["args"].(type) {
		case []any:
			for _, value := range values {
				args = append(args, value.(string))
			}
		case []string:
			args = values
		default:
			t.Fatal("missing generated MCP argument array")
		}
		return command, args
	}
	t.Fatal("no generated MCP command")
	return "", nil
}

type wireClient struct {
	writer *json.Encoder
	reader *json.Decoder
}

func (client wireClient) call(t *testing.T, id int, method string, params any, result any) {
	t.Helper()
	if err := client.writer.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		t.Fatal(err)
	}
	for {
		var response struct {
			ID     *int
			Result json.RawMessage
			Error  json.RawMessage
		}
		if err := client.reader.Decode(&response); err != nil {
			if err == io.EOF {
				t.Fatal("generated MCP command exited before responding")
			}
			t.Fatal(err)
		}
		if response.ID == nil {
			continue
		}
		if *response.ID != id || len(response.Error) > 0 {
			t.Fatalf("MCP %s response: %+v", method, response)
		}
		if err := json.Unmarshal(response.Result, result); err != nil {
			t.Fatal(err)
		}
		return
	}
}
