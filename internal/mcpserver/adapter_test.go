package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

type backend struct {
	Reader
	Writer
	fail error
	body string
	plan api.Plan
}

func (b *backend) Root() string       { return "synthetic" }
func (b *backend) Config() api.Config { var c api.Config; c.PARA.Archive = "Archive"; return c }
func (b *backend) Status(context.Context) (api.Status, error) {
	return api.Status{Generation: "g", IndexStale: true}, b.fail
}
func (b *backend) List(context.Context, api.SearchRequest) (api.SearchResult, error) {
	return api.SearchResult{Generation: "g"}, b.fail
}
func (b *backend) Search(ctx context.Context, r api.SearchRequest) (api.SearchResult, error) {
	return b.List(ctx, r)
}
func (b *backend) Read(context.Context, api.ReadRequest) (api.ReadResult, error) {
	return api.ReadResult{Markdown: b.body}, b.fail
}
func (b *backend) Context(context.Context, api.ContextRequest) (api.ContextResult, error) {
	return api.ContextResult{Generation: "g", Text: b.body}, b.fail
}
func (b *backend) Links(context.Context, string, string) ([]api.Link, error) {
	return []api.Link{}, b.fail
}
func (b *backend) Validate(_ context.Context, p string) (api.ValidationReport, error) {
	return api.ValidationReport{Profile: p, Generation: "g", Conformant: true}, b.fail
}
func (b *backend) Rebuild(ctx context.Context) (api.Status, error) { return b.Status(ctx) }
func (b *backend) Export(context.Context, string) (api.ExportResult, error) {
	return api.ExportResult{Generation: "g"}, b.fail
}
func (b *backend) Apply(_ context.Context, p api.Plan) (api.Receipt, error) {
	b.plan = p
	return api.Receipt{Generation: "g"}, b.fail
}
func (b *backend) Maintain(context.Context, bool) (api.Receipt, error) {
	return api.Receipt{Generation: "g"}, b.fail
}
func (b *backend) Recover(ctx context.Context, d bool) (api.Receipt, error) {
	return b.Maintain(ctx, d)
}

func TestDispatchAndBounds(t *testing.T) {
	b := &backend{}
	a := &Adapter{Reader: b, Writer: b}
	ctx := context.Background()
	for _, action := range []string{"status", "list", "search", "read", "context", "links", "validate", "export", "rebuild", "maintain", "recover", "apply", "create", "patch", "move"} {
		t.Run(action, func(t *testing.T) {
			e := a.Call(ctx, action, []byte(`{}`))
			if !e.OK || e.Meta.Generation != "g" || !e.Meta.IndexStale || e.Warnings == nil {
				t.Fatalf("%#v", e)
			}
			bad := a.Call(ctx, action, []byte(`{"extra":true}`))
			if bad.OK || bad.Error.Code != "INVALID_INPUT" {
				t.Fatalf("unknown field accepted: %#v", bad)
			}
		})
	}
	for _, tc := range []struct{ action, input string }{
		{"search", `{"limit":101}`}, {"search", `{"limit":-1}`}, {"search", `{"query":"` + strings.Repeat("x", 4097) + `"}`},
		{"context", `{"max_tokens":16001}`}, {"context", `{"max_bytes":262145}`}, {"context", `{"max_tokens":-1}`}, {"context", `{"query":"` + strings.Repeat("x", 4097) + `"}`},
		{"create", `{"kind":"move"}`}, {"archive", `{"path":"../a.md"}`}, {"archive", `{"path":"a.md","to":"elsewhere/a.md"}`}, {"unknown", `{}`},
	} {
		if e := a.Call(ctx, tc.action, []byte(tc.input)); e.OK {
			t.Fatalf("accepted %s %s", tc.action, tc.input)
		}
	}
	if e := a.Call(ctx, "archive", []byte(`{"path":"a.md","to":"Archive/a.md"}`)); !e.OK || b.plan.Operations[0].Kind != "move" {
		t.Fatalf("%#v %#v", e, b.plan)
	}
	p := api.Plan{Operations: make([]api.Operation, 51)}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if e := a.Call(ctx, "apply", raw); e.OK {
		t.Fatal("oversized operation set accepted")
	}
	b.body = strings.Repeat("x", api.MaxReadBytes)
	for _, action := range []string{"read", "context"} {
		if e := a.Call(ctx, action, nil); e.OK || e.Error.Code != "RESPONSE_TOO_LARGE" {
			t.Fatalf("%#v", e)
		}
	}
	b.fail = errors.New("private backend detail")
	e := a.Call(ctx, "status", nil)
	if e.OK || strings.Contains(e.Error.Message, "private") {
		t.Fatalf("%#v", e)
	}
	b.fail = nil
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if e := a.Call(canceled, "status", nil); e.Error.Code != "CANCELED" {
		t.Fatalf("%#v", e)
	}
	a.Writer = nil
	for _, action := range []string{"apply", "maintain", "recover"} {
		if a.Call(ctx, action, nil).OK {
			t.Fatal("missing writer accepted")
		}
	}
	a.ReadOnly = true
	for _, action := range []string{"create", "patch", "move", "apply", "maintain", "recover", "rebuild"} {
		if e := a.Call(ctx, action, nil); e.Error.Code != "READ_ONLY" {
			t.Fatalf("%#v", e)
		}
	}
}

func TestDecodeAndReadInput(t *testing.T) {
	for _, raw := range []string{`[]`, `null`, `{`, `{} {}`, `{"unknown":true}`, strings.Repeat(" ", api.MaxPlanBytes+1)} {
		if err := Decode([]byte(raw), &struct{}{}); err == nil {
			t.Fatalf("accepted %.60q", raw)
		}
	}
	for _, raw := range []string{"", `{}`, " \n {} \t"} {
		if err := Decode([]byte(raw), &struct{}{}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ReadInput(strings.NewReader(strings.Repeat("x", api.MaxPlanBytes+1))); err == nil {
		t.Fatal("input not bounded")
	}
	want := []byte(`{"query":"memory"}`)
	got, err := ReadInput(bytes.NewReader(want))
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := ReadInput(errorReader{}); err == nil {
		t.Fatal("read error lost")
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestExitCodeAndSafeErrors(t *testing.T) {
	for _, tc := range []struct {
		code string
		want int
	}{{"NOT_FOUND", 3}, {"SECTION_NOT_FOUND", 3}, {"REVISION_CONFLICT", 4}, {"IDEMPOTENCY_CONFLICT", 4}, {"PATH_OUTSIDE_ROOT", 5}, {"READ_ONLY", 5}, {"IO_ERROR", 6}, {"RECOVERY_REQUIRED", 6}, {"INVALID_FRONTMATTER", 1}, {"INVALID_INPUT", 2}} {
		if got := ExitCode(Failure("synthetic", &api.DomainError{Code: tc.code})); got != tc.want {
			t.Fatalf("%s: %d", tc.code, got)
		}
	}
	if ExitCode(Envelope{}) != 6 || ExitCode(Envelope{OK: true}) != 0 || ExitCode(Envelope{OK: true, Data: api.ValidationReport{}}) != 1 {
		t.Fatal("incorrect success/report exit")
	}
	for _, tc := range []struct {
		err  error
		code string
	}{{os.ErrNotExist, "NOT_FOUND"}, {os.ErrPermission, "ACCESS_DENIED"}, {context.DeadlineExceeded, "CANCELED"}, {errors.New("private"), "IO_ERROR"}} {
		if e := Failure("synthetic", tc.err); e.Error.Code != tc.code {
			t.Fatalf("%#v", e)
		}
	}
}

func TestResourceConfinement(t *testing.T) {
	for _, p := range []string{"note.md", "Folder/a b.md", "Folder/memória.md", "Folder/a%20b.md", "Folder/a#b?.md"} {
		u := ResourceURI(p)
		got, err := ResourcePath(u)
		if err != nil || got != p {
			t.Fatalf("%s %q %v", u, got, err)
		}
	}
	for _, u := range []string{"https://example.invalid/a.md", "kb://elsewhere/a.md", "kb://user@local/a.md", "kb://local/a.md?q=x", "kb://local/a.md#fragment", "kb://local/", "kb://local/../a.md", "kb://local/%2e%2e/a.md", "kb://local/%2fetc/a.md", "kb://local/a/./b.md", "kb://local/a//b.md", "kb://local/a%5cb.md", "kb://local/a%00.md", "kb://local/a.txt", "kb://local/%zz"} {
		if _, err := ResourcePath(u); err == nil {
			t.Fatalf("accepted %s", u)
		}
	}
}

func FuzzDecode(f *testing.F) {
	for _, s := range []string{`{}`, `{"path":"note.md"}`, `{"body_edits":[{"old_text":"a","new_text":"b"}]}`, `{} {}`, `null`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > api.MaxPlanBytes {
			return
		}
		var input SingleInput
		if Decode([]byte(s), &input) == nil {
			raw, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			if err := Decode(raw, &SingleInput{}); err != nil {
				t.Fatal(err)
			}
		}
	})
}
func FuzzResourcePath(f *testing.F) {
	for _, s := range []string{"kb://local/Notes/a%20b.md", "kb://local/../a.md", "https://example.invalid/a.md"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p, err := ResourcePath(s)
		if err != nil {
			return
		}
		round, err := ResourcePath(ResourceURI(p))
		if err != nil || round != p {
			t.Fatalf("unstable scoped path %q", p)
		}
	})
}

func TestArchiveDefaultDestinationRetainsCAS(t *testing.T) {
	b := &backend{}
	a := &Adapter{Reader: b, Writer: b}
	e := a.Call(context.Background(), "archive", []byte(`{"path":"Notes/nested/old.md","expected_revision":"sha256:old","actor":"agent:test","summary":"Archive","idempotency_key":"archive-default"}`))
	if !e.OK {
		t.Fatalf("%#v", e)
	}
	op := b.plan.Operations[0]
	if op.Kind != "move" || op.Path != "Notes/nested/old.md" || op.To != "Archive/old.md" || op.ExpectedRevision != "sha256:old" {
		t.Fatalf("%#v", op)
	}
}
