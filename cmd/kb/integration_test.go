package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/internal/filelock"
	"github.com/matheus3301/kb/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func realFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "knowledge")
	if code, e := invoke(t, []string{"--root", root, "init", "--para"}, ""); code != 0 {
		t.Fatalf("init %d %#v", code, e)
	}
	for _, n := range []struct{ path, body string }{{"03_Resources/alpha note.md", "# Synthetic memory\nMemory remains useful.\n\n## Evidence\nA synthetic source."}, {"03_Resources/nested/beta.md", "# Synthetic comparison\nA memory comparison with [Alpha](../alpha%20note.md)."}} {
		raw, err := json.Marshal(map[string]any{"path": n.path, "type": "Concept", "body": n.body, "metadata": map[string]any{"tags": []string{"memory"}, "synthetic_extra": map[string]any{"preserved": true}}, "actor": "agent:integration-test", "summary": "Record synthetic evidence", "idempotency_key": "fixture-" + n.path})
		if err != nil {
			t.Fatal(err)
		}
		if code, e := invoke(t, []string{"--root", root, "create", "--input", "-"}, string(raw)); code != 0 {
			t.Fatalf("create %d %#v", code, e)
		}
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func realSession(t *testing.T, root string, ro bool, configure func(*mcp.Server)) (*mcp.ClientSession, func()) {
	t.Helper()
	reader, writer, err := dependencies().Open(root)
	if err != nil {
		t.Fatal(err)
	}
	server := mcpserver.New(&mcpserver.Adapter{Reader: reader, Writer: writer, ReadOnly: ro}, "test")
	if configure != nil {
		configure(server)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "real-provider-test", Version: "1"}, nil).Connect(ctx, ct, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
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

func wireResult(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var e map[string]any
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatal(err)
	}
	if e["schema_version"] != "kb.v1" || result.IsError == (e["ok"] == true) || len(result.Content) == 0 {
		t.Fatalf("invalid wire result %#v %#v", e, result)
	}
	return e
}
func protocolCall(t *testing.T, cs *mcp.ClientSession, action, raw string) map[string]any {
	t.Helper()
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "kb_" + action, Arguments: json.RawMessage(raw)})
	if err != nil {
		t.Fatal(err)
	}
	return wireResult(t, result)
}

func TestRealProviderCLIAndMCPParity(t *testing.T) {
	root := realFixture(t)
	cs, cleanup := realSession(t, root, false, nil)
	defer cleanup()
	for _, tc := range []struct{ action, raw string }{
		{"status", `{}`}, {"list", `{"limit":1,"filters":{"prefix":"03_Resources/","tags":["memory"]}}`}, {"search", `{"query":"memory","filters":{"type":["Concept"],"tags":["memory"]},"limit":10}`}, {"read", `{"path":"03_Resources/alpha note.md","section":"Evidence","include_links":true}`}, {"context", `{"query":"memory","max_tokens":100,"max_bytes":2000,"include_links":true}`}, {"validate", `{"profile":"okf"}`}, {"validate", `{"profile":"health"}`},
		{"read", `{"path":"missing.md"}`}, {"read", `{"path":"../outside.md"}`}, {"read", `{"path":"03_Resources/alpha note.md","section":"missing"}`}, {"search", `{"query":"memory","cursor":"invalid"}`},
		{"search", `{"root":"other"}`}, {"search", `{"filters":{"unknown":true}}`}, {"read", `null`}, {"search", `{"filters":null}`}, {"search", `{"filters":{"stale":null}}`}, {"search", `{"query":null}`}, {"read", `{"path":null}`}, {"list", `{"limit":null}`},
	} {
		t.Run(tc.action+tc.raw, func(t *testing.T) {
			_, cliEnvelope := invoke(t, []string{"--root", root, tc.action, "--input", "-"}, tc.raw)
			mcpEnvelope := protocolCall(t, cs, tc.action, tc.raw)
			if !reflect.DeepEqual(cliEnvelope, mcpEnvelope) {
				t.Fatalf("CLI %#v != MCP %#v", cliEnvelope, mcpEnvelope)
			}
		})
	}
	// A cursor from the CLI remains valid at the MCP boundary for the same query.
	code, page := invoke(t, []string{"--root", root, "list", "--input", "-"}, `{"limit":1}`)
	if code != 0 {
		t.Fatalf("%#v", page)
	}
	cursor := page["data"].(map[string]any)["cursor"].(string)
	raw, err := json.Marshal(map[string]any{"limit": 1, "cursor": cursor})
	if err != nil {
		t.Fatal(err)
	}
	_, cliEnvelope := invoke(t, []string{"--root", root, "list", "--input", "-"}, string(raw))
	mcpEnvelope := protocolCall(t, cs, "list", string(raw))
	if !reflect.DeepEqual(cliEnvelope, mcpEnvelope) {
		t.Fatal("cross-transport cursor parity differs")
	}
}

func TestRealProviderWriteReplayParity(t *testing.T) {
	root := realFixture(t)
	cs, cleanup := realSession(t, root, false, nil)
	defer cleanup()
	requests := []struct {
		action string
		input  map[string]any
	}{
		{"create", map[string]any{"path": "03_Resources/new.md", "type": "Concept", "body": "# New evidence\nA synthetic finding."}},
		{"patch", map[string]any{"path": "03_Resources/new.md", "body": "# Corrected evidence\nA corrected synthetic finding."}},
		{"move", map[string]any{"path": "03_Resources/new.md", "to": "03_Resources/moved.md"}},
		{"apply", map[string]any{"operations": []map[string]any{{"operation_id": "new-evidence", "kind": "create", "path": "03_Resources/applied.md", "type": "Concept", "body": "# Coordinated evidence\nAnother finding."}}}},
	}
	for _, tc := range requests {
		t.Run(tc.action, func(t *testing.T) {
			tc.input["actor"] = "agent:integration-test"
			tc.input["summary"] = "Persist synthetic evidence"
			tc.input["idempotency_key"] = "replay-" + tc.action
			if tc.action == "patch" || tc.action == "move" {
				_, read := invoke(t, []string{"--root", root, "read", tc.input["path"].(string)}, "")
				tc.input["expected_revision"] = read["data"].(map[string]any)["note"].(map[string]any)["revision"]
			}
			raw, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			first := protocolCall(t, cs, tc.action, string(raw))
			if first["ok"] != true {
				t.Fatalf("first effect %#v", first)
			}
			code, cliEnvelope := invoke(t, []string{"--root", root, tc.action, "--input", "-"}, string(raw))
			mcpEnvelope := protocolCall(t, cs, tc.action, string(raw))
			if code != 0 || !reflect.DeepEqual(cliEnvelope, mcpEnvelope) || mcpEnvelope["data"].(map[string]any)["replayed"] != true {
				t.Fatalf("CLI %#v != MCP %#v", cliEnvelope, mcpEnvelope)
			}
		})
	}
	// Both boundaries retain CAS errors for a fresh request with the wrong revision.
	raw := `{"path":"03_Resources/moved.md","expected_revision":"sha256:obsolete","body":"Bad edit","actor":"agent:integration-test","summary":"Conflict","idempotency_key":"conflict"}`
	code, cliEnvelope := invoke(t, []string{"--root", root, "patch", "--input", "-"}, raw)
	mcpEnvelope := protocolCall(t, cs, "patch", raw)
	if code != 4 || !reflect.DeepEqual(cliEnvelope, mcpEnvelope) {
		t.Fatalf("conflict: %d %#v %#v", code, cliEnvelope, mcpEnvelope)
	}
}

func TestRealProviderReadOnlyResourcesAndExclusions(t *testing.T) {
	root := realFixture(t)
	cs, cleanup := realSession(t, root, true, nil)
	defer cleanup()
	ctx := context.Background()
	tools, err := cs.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 6 {
		t.Fatalf("%v %#v", err, tools)
	}
	for _, tool := range tools.Tools {
		if !tool.Annotations.ReadOnlyHint {
			t.Fatalf("write tool exposed %s", tool.Name)
		}
	}
	_, before := invoke(t, []string{"--root", root, "status"}, "")
	for _, action := range []string{"create", "patch", "move", "apply"} {
		e := protocolCall(t, cs, action, `{"root":"other"}`)
		if e["ok"] != false || e["error"].(map[string]any)["code"] != "READ_ONLY" {
			t.Fatalf("%#v", e)
		}
		for _, raw := range []string{`null`, `{"root":"other"}`} {
			code, cliEnvelope := invoke(t, []string{"--root", root, "--read-only", action, "--input", "-"}, raw)
			if code != 5 || !reflect.DeepEqual(cliEnvelope, protocolCall(t, cs, action, raw)) {
				t.Fatalf("read-only parity: %d %#v", code, cliEnvelope)
			}
		}
	}
	_, after := invoke(t, []string{"--root", root, "status"}, "")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("read-only calls changed inventory")
	}
	// Every listed resource is canonical, scoped, and exposes a current revision.
	files, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := files.Close(); err != nil {
			t.Error(err)
		}
	}()
	count := 0
	for resource, err := range cs.Resources(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		p, err := mcpserver.ResourcePath(resource.URI)
		if err != nil {
			t.Fatal(err)
		}
		rr, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: resource.URI})
		if err != nil || len(rr.Contents) != 1 {
			t.Fatalf("%v %#v", err, rr)
		}
		raw, err := files.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if rr.Contents[0].Meta["revision"] != api.Revision(raw) || rr.Contents[0].Text != string(raw) {
			t.Fatalf("resource differs from scoped file %s", p)
		}
		count++
	}
	if count != 2 {
		t.Fatalf("incomplete inventory %d", count)
	}
	for _, uri := range []string{"kb://local/%2e%2e/outside.md", "kb://local/.kb/config.toml", "kb://local/.git/secret.md", "kb://other/03_Resources/alpha%20note.md"} {
		if _, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri}); err == nil {
			t.Fatalf("out-of-scope resource accepted %s", uri)
		}
	}
}

func TestRealProviderCancellationWhileWriterWaitsForLock(t *testing.T) {
	root := realFixture(t)
	path := "03_Resources/alpha note.md"
	_, read := invoke(t, []string{"--root", root, "read", path}, "")
	revision := read["data"].(map[string]any)["note"].(map[string]any)["revision"]
	started, finished := make(chan struct{}), make(chan struct{})
	cs, cleanup := realSession(t, root, false, func(s *mcp.Server) {
		s.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
			return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
				if p, ok := req.GetParams().(*mcp.CallToolParamsRaw); method == "tools/call" && ok && p.Name == "kb_patch" {
					close(started)
					defer close(finished)
				}
				return next(ctx, method, req)
			}
		})
	})
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	lock, err := filelock.Acquire(ctx, root, true)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lock.Close(); err != nil {
			t.Error(err)
		}
	}()
	callCtx, stopCall := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() {
		_, err := cs.CallTool(callCtx, &mcp.CallToolParams{Name: "kb_patch", Arguments: map[string]any{"path": path, "expected_revision": revision, "body": "Canceled mutation", "summary": "This must never publish", "actor": "agent:integration-test", "idempotency_key": "canceled-mutation"}})
		done <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("request did not start")
	}
	stopCall()
	select {
	case <-finished:
	case <-ctx.Done():
		t.Fatal("server did not release the canceled lock wait")
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("%v", err)
		}
	case <-ctx.Done():
		t.Fatal("client did not cancel")
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	code, after := invoke(t, []string{"--root", root, "read", path}, "")
	if code != 0 || after["data"].(map[string]any)["note"].(map[string]any)["revision"] != revision {
		t.Fatalf("canceled mutation changed note %#v", after)
	}
	// The same session remains usable, and another writer can acquire the base.
	if e := protocolCall(t, cs, "status", `{}`); e["ok"] != true {
		t.Fatalf("%#v", e)
	}
	next, err := filelock.Acquire(ctx, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := next.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRealProviderRejectsNullControlFields(t *testing.T) {
	root := realFixture(t)
	cs, cleanup := realSession(t, root, false, nil)
	defer cleanup()
	for _, tc := range []struct{ action, raw string }{{"search", `{"query":null}`}, {"search", `{"filters":null}`}, {"search", `{"filters":{"stale":null}}`}, {"list", `{"limit":null}`}, {"patch", `{"path":"03_Resources/alpha note.md","body":null,"expected_revision":"sha256:obsolete","actor":"agent:test","summary":"Reject null","idempotency_key":"null-control"}`}, {"apply", `{"operations":[null],"actor":"agent:test","summary":"Reject null","idempotency_key":"null-operation"}`}} {
		t.Run(tc.action+tc.raw, func(t *testing.T) {
			code, e := invoke(t, []string{"--root", root, tc.action, "--input", "-"}, tc.raw)
			if code != 2 || e["error"].(map[string]any)["code"] != "INVALID_INPUT" {
				t.Fatalf("null control was accepted or coerced: %d %#v", code, e)
			}
			mcpEnvelope := protocolCall(t, cs, tc.action, tc.raw)
			if !reflect.DeepEqual(e, mcpEnvelope) {
				t.Fatalf("CLI %#v != MCP %#v", e, mcpEnvelope)
			}
		})
	}
}

func TestRealProviderReadHasOneBoundedContentCopy(t *testing.T) {
	root := realFixture(t)
	cs, cleanup := realSession(t, root, false, nil)
	defer cleanup()
	e := protocolCall(t, cs, "read", `{"path":"03_Resources/alpha note.md"}`)
	data := e["data"].(map[string]any)
	note := data["note"].(map[string]any)
	if _, exists := note["body"]; exists {
		t.Fatal("read duplicates complete note body")
	}
	if !strings.Contains(data["markdown"].(string), "Synthetic memory") {
		t.Fatal("source content missing")
	}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > api.MaxReadBytes {
		t.Fatal("response exceeds ceiling")
	}
}

func TestRealProviderEarlyErrorsUseResolvedRoot(t *testing.T) {
	root := realFixture(t)
	cs, cleanup := realSession(t, root, false, nil)
	defer cleanup()
	for _, raw := range []string{`null`, `{"filters":null}`} {
		_, e := invoke(t, []string{"--root", root + "/.", "search", "--input", "-"}, raw)
		peer := protocolCall(t, cs, "search", raw)
		if !reflect.DeepEqual(e, peer) {
			t.Fatalf("root/error parity: %#v %#v", e, peer)
		}
	}
}

func TestRealProviderResourcePaginationAndSymlinkRejection(t *testing.T) {
	root := realFixture(t)
	operations := make([]api.Operation, 12)
	for i := range operations {
		body := "# Synthetic pagination\nA paginated note."
		operations[i] = api.Operation{OperationID: fmt.Sprintf("resource-%02d", i), Kind: "create", Path: fmt.Sprintf("03_Resources/page-%02d.md", i), Type: "Concept", Body: &body}
	}
	plan := api.Plan{Operations: operations, Actor: "agent:integration-test", Summary: "Create a synthetic paginated inventory", IdempotencyKey: "resources-pagination"}
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if code, e := invoke(t, []string{"--root", root, "apply", "--input", "-"}, string(raw)); code != 0 {
		t.Fatalf("%d %#v", code, e)
	}
	cs, cleanup := realSession(t, root, true, nil)
	defer cleanup()
	ctx := context.Background()
	first, err := cs.ListResources(ctx, nil)
	if err != nil || len(first.Resources) != api.DefaultHits || first.NextCursor == "" {
		t.Fatalf("first page %v %#v", err, first)
	}
	second, err := cs.ListResources(ctx, &mcp.ListResourcesParams{Cursor: first.NextCursor})
	if err != nil || len(second.Resources) != 4 || second.NextCursor != "" {
		t.Fatalf("second page %v %#v", err, second)
	}
	seen := map[string]bool{}
	for _, page := range []*mcp.ListResourcesResult{first, second} {
		for _, r := range page.Resources {
			if seen[r.URI] {
				t.Fatalf("duplicate resource %s", r.URI)
			}
			seen[r.URI] = true
		}
	}
	// A symlink in the selected scope cannot expose a different synthetic base.
	external := filepath.Join(t.TempDir(), "external.md")
	secret := []byte("---\ntype: Concept\n---\nExternal synthetic marker\n")
	if err := os.WriteFile(external, secret, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "03_Resources", "shortcut.md")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	if _, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: mcpserver.ResourceURI("03_Resources/shortcut.md")}); err == nil {
		t.Fatal("symlink resource was opened")
	}
}

func TestRealProviderLargeReadRemainsBoundedAndUseful(t *testing.T) {
	root := realFixture(t)
	body := strings.Repeat("Synthetic bounded content. ", 12000)
	request := map[string]any{"path": "03_Resources/large.md", "type": "Concept", "body": body, "actor": "agent:integration-test", "summary": "Create a large synthetic source", "idempotency_key": "large-read"}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if code, e := invoke(t, []string{"--root", root, "create", "--input", "-"}, string(raw)); code != 0 {
		t.Fatalf("create: %d %#v", code, e)
	}
	cs, cleanup := realSession(t, root, true, nil)
	defer cleanup()
	code, e := invoke(t, []string{"--root", root, "read", "03_Resources/large.md"}, "")
	peer := protocolCall(t, cs, "read", `{"path":"03_Resources/large.md"}`)
	if code != 0 || !reflect.DeepEqual(e, peer) {
		t.Fatalf("large read: %d %#v", code, e)
	}
	encoded, err := json.Marshal(peer)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > api.MaxReadBytes {
		t.Fatalf("read exceeded limit: %d", len(encoded))
	}
	data := peer["data"].(map[string]any)
	if data["markdown"] == "" || len(data["diagnostics"].([]any)) == 0 {
		t.Fatal("bounded source or truncation diagnostic missing")
	}
}

type observedInput struct {
	io.ReadCloser
	started chan struct{}
}

func (r *observedInput) Read(p []byte) (int, error) {
	if r.started != nil {
		close(r.started)
		r.started = nil
	}
	return r.ReadCloser.Read(p)
}

func TestRealProviderCancelsBlockedCLIInput(t *testing.T) {
	root := realFixture(t)
	_, before := invoke(t, []string{"--root", root, "status"}, "")
	reader, writer := io.Pipe()
	defer func() { _ = writer.Close() }()
	input := &observedInput{ReadCloser: reader, started: make(chan struct{})}
	started := input.started
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out, diagnostics bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- run(ctx, []string{"--root", root, "create", "--input", "-"}, input, &out, &diagnostics)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("stdin read never started")
	}
	cancel()
	select {
	case code := <-done:
		if code != 6 {
			t.Fatalf("cancellation exit %d: %s", code, out.String())
		}
	case <-time.After(time.Second):
		t.Fatal("canceled stdin read remained blocked")
	}
	var e map[string]any
	if err := json.Unmarshal(out.Bytes(), &e); err != nil {
		t.Fatal(err)
	}
	if e["error"].(map[string]any)["code"] != "CANCELED" {
		t.Fatalf("%#v", e)
	}
	_, after := invoke(t, []string{"--root", root, "status"}, "")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("canceled input changed base")
	}
}
