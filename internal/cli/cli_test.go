package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fixture struct {
	root       string
	plan       api.Plan
	search     api.SearchRequest
	read       api.ReadRequest
	clock      time.Time
	dry        bool
	fail       error
	conformant bool
	block      bool
	started    chan struct{}
}

func (f *fixture) Root() string        { return f.root }
func (f *fixture) Config() api.Config  { var c api.Config; c.PARA.Archive = "Archive"; return c }
func (f *fixture) SetAsOf(t time.Time) { f.clock = t }
func (f *fixture) Status(context.Context) (api.Status, error) {
	return api.Status{Root: f.root, Generation: "generation", Notes: 1}, f.fail
}
func (f *fixture) List(_ context.Context, r api.SearchRequest) (api.SearchResult, error) {
	f.search = r
	return api.SearchResult{Generation: "generation", Hits: []api.Hit{{Note: api.Note{ID: "memory", Path: "Notes/a b.md", Revision: "sha256:fixture", Title: "Synthetic note"}}}}, f.fail
}
func (f *fixture) Search(ctx context.Context, r api.SearchRequest) (api.SearchResult, error) {
	if f.block {
		close(f.started)
		<-ctx.Done()
		return api.SearchResult{}, ctx.Err()
	}
	return f.List(ctx, r)
}
func (f *fixture) Read(_ context.Context, r api.ReadRequest) (api.ReadResult, error) {
	f.read = r
	return api.ReadResult{Note: api.Note{Path: r.Path, Revision: "sha256:fixture"}, Markdown: "# Synthetic note\nReusable memory."}, f.fail
}
func (f *fixture) Context(_ context.Context, _ api.ContextRequest) (api.ContextResult, error) {
	return api.ContextResult{Text: "Reusable memory.", Generation: "generation"}, f.fail
}
func (f *fixture) Links(context.Context, string, string) ([]api.Link, error) {
	return []api.Link{}, f.fail
}
func (f *fixture) Validate(_ context.Context, p string) (api.ValidationReport, error) {
	return api.ValidationReport{Profile: p, Generation: "generation", Conformant: f.conformant}, f.fail
}
func (f *fixture) Rebuild(ctx context.Context) (api.Status, error) { return f.Status(ctx) }
func (f *fixture) Export(_ context.Context, d string) (api.ExportResult, error) {
	return api.ExportResult{Destination: d, Generation: "generation"}, f.fail
}
func (f *fixture) Apply(_ context.Context, p api.Plan) (api.Receipt, error) {
	f.plan = p
	return api.Receipt{IdempotencyKey: p.IdempotencyKey, Generation: "generation", DryRun: p.DryRun}, f.fail
}
func (f *fixture) Maintain(_ context.Context, d bool) (api.Receipt, error) {
	f.dry = d
	return api.Receipt{Generation: "generation", DryRun: d}, f.fail
}
func (f *fixture) Recover(ctx context.Context, d bool) (api.Receipt, error) {
	return f.Maintain(ctx, d)
}
func deps(f *fixture) Dependencies {
	return Dependencies{ResolveRoot: func(s string) (string, error) {
		if s != "" {
			f.root = s
		}
		return f.root, nil
	}, Open: func(string) (mcpserver.Reader, mcpserver.Writer, error) { return f, f, nil }, Init: func(_ context.Context, s string, _ bool) (mcpserver.Reader, mcpserver.Writer, error) {
		f.root = s
		return f, f, nil
	}, Version: "v0.1.0", Commit: "test"}
}
func execute(t *testing.T, f *fixture, args []string, input string) (int, map[string]any, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(context.Background(), args, strings.NewReader(input), &out, &errOut, deps(f))
	var e map[string]any
	if err := json.Unmarshal(out.Bytes(), &e); err != nil {
		t.Fatalf("invalid single JSON response %q: %v", out.String(), err)
	}
	return code, e, errOut.String()
}

func TestCLICommands(t *testing.T) {
	for _, args := range [][]string{{"status"}, {"list", "--para", "resources", "--type", "Concept", "--tags", "memory", "--status", "active", "--trust", "unverified", "--stale=false", "--has-sources", "--prefix", "Notes", "--include-archives", "--include-deprecated", "--limit", "2", "--literal", "--cursor", "next"}, {"search", "memory"}, {"read", "Notes/a b.md", "--section", "Topic", "--line-start", "1", "--line-end", "4", "--raw", "--include-links"}, {"context", "memory", "--max-tokens", "100", "--max-bytes", "1000", "--include-links"}, {"links", "Notes/a b.md", "--direction", "outgoing"}, {"validate", "--profile", "health"}, {"index", "rebuild"}, {"export", "bundle"}, {"maintain", "--dry-run"}, {"maintain", "--apply"}, {"maintain", "--apply", "--dry-run"}, {"recover", "--dry-run"}, {"init", "--para"}, {"version"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			f := &fixture{root: "synthetic", conformant: true}
			args = append([]string{"--root", "synthetic", "--json", "--as-of", "2026-01-01T00:00:00Z"}, args...)
			code, e, errOut := execute(t, f, args, "")
			if code != 0 || e["ok"] != true || errOut != "" {
				t.Fatalf("%d %#v %s", code, e, errOut)
			}
			if f.clock.IsZero() && args[len(args)-1] != "version" {
				t.Fatal("as-of was not applied")
			}
			if args[5] == "list" && f.search.Filters.HasSources == nil {
				t.Fatal("filters lost")
			}
		})
	}
}

func TestCLIWriteFlatteningAndPreconditions(t *testing.T) {
	for _, tc := range []struct {
		action string
		args   []string
		input  string
		kind   string
	}{
		{"create", nil, `{"path":"Notes/new.md","type":"Concept","body":"Learned","actor":"agent:test","summary":"Learn","idempotency_key":"one"}`, "create"},
		{"patch", []string{"Notes/old.md"}, `{"expected_revision":"sha256:old","body_edits":[{"old_text":"Old","new_text":"New"}],"actor":"agent:test","summary":"Correct","idempotency_key":"two"}`, "patch"},
		{"move", []string{"Notes/old.md", "Notes/new.md"}, `{"expected_revision":"sha256:old","actor":"agent:test","summary":"Organize","idempotency_key":"three"}`, "move"},
		{"archive", []string{"Notes/old.md"}, `{"to":"Archive/old.md","expected_revision":"sha256:old","actor":"agent:test","summary":"Archive","idempotency_key":"four"}`, "move"},
		{"apply", nil, `{"operations":[{"kind":"patch","path":"Notes/old.md","expected_revision":"sha256:old","document":"---\ntype: Concept\n---\nRepaired"}],"actor":"agent:test","summary":"Repair","idempotency_key":"five"}`, "patch"},
	} {
		t.Run(tc.action, func(t *testing.T) {
			f := &fixture{root: "synthetic"}
			args := append([]string{tc.action}, tc.args...)
			args = append(args, "--input", "-", "--dry-run")
			if tc.action == "apply" {
				args = append(args, "--repair")
			}
			code, e, _ := execute(t, f, args, tc.input)
			if code != 0 || e["ok"] != true {
				t.Fatalf("%d %#v", code, e)
			}
			if len(f.plan.Operations) != 1 || f.plan.Operations[0].Kind != tc.kind || !f.plan.DryRun || f.plan.Actor != "agent:test" {
				t.Fatalf("bad plan %#v", f.plan)
			}
			if tc.action == "apply" && !f.plan.Repair {
				t.Fatal("repair lost")
			}
		})
	}
}

func TestCLIInputAndExitFailures(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		input string
		want  int
	}{
		{[]string{"search", "--input", "-"}, `{"unknown":true}`, 2},
		{[]string{"search", "--input", "-"}, `{"filters":{"unknown":true}}`, 2},
		{[]string{"search", "--input", "-"}, `{} {}`, 2},
		{[]string{"search", "--input", "-"}, `null`, 2},
		{[]string{"search", "--input", "-"}, `{"filters":null}`, 2},
		{[]string{"patch", "a.md", "--input", "-"}, `{"path":"b.md"}`, 2},
		{[]string{"move", "--input", "-"}, `{"kind":"patch"}`, 2},
		{[]string{"archive", "--input", "-"}, `{"path":"../a.md"}`, 2},
		{[]string{"archive", "--input", "-"}, `{"path":"a.md","to":"Other/a.md"}`, 2},
		{[]string{"create"}, ``, 2},
		{[]string{"--as-of", "yesterday", "status"}, ``, 2},
		{[]string{"read", "a.md", "extra"}, ``, 2},
		{[]string{"wat"}, ``, 2},
		{[]string{"status", "--wat"}, ``, 2},
		{[]string{"--read-only", "init"}, ``, 5},
		{[]string{"--read-only", "create", "--input", "-"}, `{}`, 5},
		{[]string{"validate"}, ``, 1},
		{[]string{"read", "--input", "missing-file.json"}, ``, 3},
	} {
		t.Run(strings.Join(tc.args, " ")+tc.input, func(t *testing.T) {
			f := &fixture{root: "synthetic"}
			code, e, errOut := execute(t, f, tc.args, tc.input)
			if code != tc.want {
				t.Fatalf("got %d want %d: %#v %s", code, tc.want, e, errOut)
			}
			if code != 1 && errOut == "" {
				t.Fatal("missing stderr error")
			}
		})
	}
}

func TestCLIFileInputAndDefaultInit(t *testing.T) {
	t.Setenv("KB_ROOT", "synthetic-env")
	f := &fixture{}
	code, _, _ := execute(t, f, []string{"init"}, "")
	if code != 0 || f.root != "synthetic-env" {
		t.Fatalf("%d %s", code, f.root)
	}
	t.Setenv("KB_ROOT", "")
	code, _, _ = execute(t, f, []string{"init"}, "")
	if code != 0 || f.root != "." {
		t.Fatalf("%d %s", code, f.root)
	}
	p := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(p, []byte(`{"query":"memory","limit":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	code, _, _ = execute(t, f, []string{"search", "--input", p}, "")
	if code != 0 || f.search.Query != "memory" || f.search.Limit != 2 {
		t.Fatalf("bad request %#v", f.search)
	}
	code, _, _ = execute(t, f, []string{"maintain"}, "")
	if code != 0 || !f.dry {
		t.Fatal("maintain must default to preview")
	}
}

func connect(t *testing.T, a *mcpserver.Adapter) (*mcp.ClientSession, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	s := mcpserver.New(a, "test")
	st, ct := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
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

func TestCLIMCPParity(t *testing.T) {
	f := &fixture{root: "synthetic", conformant: true}
	cs, cleanup := connect(t, &mcpserver.Adapter{Reader: f, Writer: f})
	defer cleanup()
	for _, tc := range []struct {
		action string
		args   []string
		input  string
	}{
		{"status", nil, `{}`}, {"search", []string{"--input", "-"}, `{"query":"memory","limit":2,"filters":{"type":["Concept"]}}`}, {"read", []string{"--input", "-"}, `{"path":"Notes/a b.md"}`}, {"context", []string{"--input", "-"}, `{"query":"memory","max_tokens":200}`}, {"validate", []string{"--input", "-"}, `{"profile":"health"}`}, {"create", []string{"--input", "-"}, `{"path":"Notes/new.md","type":"Concept","actor":"agent:test","summary":"Learn","idempotency_key":"key","dry_run":true}`},
	} {
		t.Run(tc.action, func(t *testing.T) {
			args := append([]string{tc.action}, tc.args...)
			_, cliOut, _ := execute(t, f, args, tc.input)
			var input map[string]any
			if err := json.Unmarshal([]byte(tc.input), &input); err != nil {
				t.Fatal(err)
			}
			res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "kb_" + tc.action, Arguments: input})
			if err != nil {
				t.Fatal(err)
			}
			b, err := json.Marshal(res.StructuredContent)
			if err != nil {
				t.Fatal(err)
			}
			var mcpOut map[string]any
			if err = json.Unmarshal(b, &mcpOut); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(cliOut, mcpOut) {
				t.Fatalf("CLI %#v != MCP %#v", cliOut, mcpOut)
			}
		})
	}
	f.fail = &api.DomainError{Code: "REVISION_CONFLICT", Message: "revision changed"}
	code, cliOut, _ := execute(t, f, []string{"read", "Notes/a b.md"}, "")
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "kb_read", Arguments: map[string]any{"path": "Notes/a b.md"}})
	if err != nil {
		t.Fatal(err)
	}
	if code != 4 || !res.IsError {
		t.Fatalf("%d %#v", code, res)
	}
	b, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpOut map[string]any
	if err = json.Unmarshal(b, &mcpOut); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cliOut, mcpOut) {
		t.Fatal("failure parity differs")
	}
}

func TestMCPToolsResourcesAndReadOnly(t *testing.T) {
	for _, ro := range []bool{false, true} {
		t.Run(fmtBool(ro), func(t *testing.T) {
			f := &fixture{root: "synthetic"}
			cs, cleanup := connect(t, &mcpserver.Adapter{Reader: f, Writer: f, ReadOnly: ro})
			defer cleanup()
			ctx := context.Background()
			tools, err := cs.ListTools(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := 10
			if ro {
				want = 6
			}
			if len(tools.Tools) != want {
				t.Fatalf("tools %d", len(tools.Tools))
			}
			for _, tool := range tools.Tools {
				b, err := json.Marshal(tool.InputSchema)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(b, []byte(`"additionalProperties":false`)) {
					t.Fatal("schema not closed")
				}
				if tool.Annotations.ReadOnlyHint == mcpserver.IsWrite(strings.TrimPrefix(tool.Name, "kb_")) {
					t.Fatal("incorrect annotation")
				}
			}
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "kb_validate", Arguments: map[string]any{}})
			if err != nil || res.IsError {
				t.Fatalf("validation findings must be successful: %v %#v", err, res)
			}
			res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "kb_search", Arguments: map[string]any{"root": "elsewhere"}})
			if err != nil || !res.IsError {
				t.Fatalf("unknown field accepted: %v %#v", err, res)
			}
			if ro {
				res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "kb_apply", Arguments: map[string]any{}})
				if err != nil || !res.IsError {
					t.Fatalf("read-only call not rejected: %v %#v", err, res)
				}
			}
			resources, err := cs.ListResources(ctx, nil)
			if err != nil || len(resources.Resources) != 1 || resources.Resources[0].URI != "kb://local/Notes/a%20b.md" {
				t.Fatalf("resources %v %#v", err, resources)
			}
			rr, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: resources.Resources[0].URI})
			if err != nil || rr.Contents[0].Meta["revision"] != "sha256:fixture" {
				t.Fatalf("resource read %v %#v", err, rr)
			}
			_, err = cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "kb://local/../secret.md"})
			if err == nil {
				t.Fatal("traversal resource read accepted")
			}
		})
	}
}
func fmtBool(v bool) string {
	if v {
		return "read-only"
	}
	return "writable"
}

func TestMCPCancellation(t *testing.T) {
	f := &fixture{root: "synthetic", block: true, started: make(chan struct{})}
	cs, cleanup := connect(t, &mcpserver.Adapter{Reader: f, Writer: f})
	defer cleanup()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "kb_search", Arguments: map[string]any{"query": "memory"}})
		done <- err
	}()
	select {
	case <-f.started:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("missing cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not finish")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }
func TestOutputFailureAndHelp(t *testing.T) {
	f := &fixture{}
	if code := Run(context.Background(), []string{"version"}, strings.NewReader(""), brokenWriter{}, io.Discard, deps(f)); code != 6 {
		t.Fatal(code)
	}
	var out bytes.Buffer
	if code := Run(context.Background(), []string{"--help"}, strings.NewReader(""), &out, io.Discard, deps(f)); code != 0 || !strings.Contains(out.String(), "mcp") {
		t.Fatalf("%d %s", code, out.String())
	}
}

func TestCLIStdioMCPAndStartupFailure(t *testing.T) {
	f := &fixture{root: "synthetic"}
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	var errOut bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan int, 1)
	go func() {
		done <- Run(ctx, []string{"mcp", "--read-only"}, serverIn, serverOut, &errOut, deps(f))
		_ = serverOut.Close()
	}()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "1"}, nil).Connect(ctx, &mcp.IOTransport{Reader: clientIn, Writer: clientOut}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	if cs.InitializeResult().ProtocolVersion != "2025-11-25" {
		t.Fatal("baseline was not negotiated")
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "kb_status", Arguments: map[string]any{}})
	if err != nil || res.IsError {
		t.Fatalf("%v %#v", err, res)
	}
	if err := cs.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("shutdown %d: %s", code, errOut.String())
		}
	case <-ctx.Done():
		t.Fatal("server did not shut down")
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected diagnostics %s", errOut.String())
	}
	for _, args := range [][]string{{"mcp", "--unknown"}, {"mcp", "--as-of", "invalid"}} {
		var out, diagnostics bytes.Buffer
		if code := Run(context.Background(), args, strings.NewReader(""), &out, &diagnostics, deps(f)); code != 2 || out.Len() != 0 || diagnostics.Len() == 0 {
			t.Fatalf("startup: %d stdout %s stderr %s", code, out.String(), diagnostics.String())
		}
	}
}

func TestCLIArchiveDefaultDestination(t *testing.T) {
	f := &fixture{root: "synthetic"}
	input := `{"expected_revision":"sha256:old","actor":"agent:test","summary":"Archive a finding","idempotency_key":"archive-default"}`
	code, e, _ := execute(t, f, []string{"archive", "Notes/old.md", "--input", "-", "--dry-run"}, input)
	if code != 0 || e["ok"] != true {
		t.Fatalf("%d %#v", code, e)
	}
	op := f.plan.Operations[0]
	if op.Path != "Notes/old.md" || op.To != "Archive/old.md" || op.ExpectedRevision != "sha256:old" || op.Kind != "move" {
		t.Fatalf("%#v", op)
	}
}

func TestCLIExplicitBooleanFlagsOverridePlan(t *testing.T) {
	f := &fixture{root: "synthetic"}
	input := `{"operations":[{"kind":"create","path":"new.md","type":"Concept"}],"actor":"agent:test","summary":"Learn","idempotency_key":"boolean-flags","dry_run":true,"repair":true}`
	code, e, _ := execute(t, f, []string{"apply", "--input", "-", "--dry-run=false", "--repair=false"}, input)
	if code != 0 || e["ok"] != true || f.plan.DryRun || f.plan.Repair {
		t.Fatalf("%d %#v %#v", code, e, f.plan)
	}
}
