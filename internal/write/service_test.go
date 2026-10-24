package write

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
)

type fakeCodec struct{}

func (fakeCodec) Parse(name string, raw []byte) (api.Note, []api.Diagnostic) {
	note := api.Note{Path: name, Raw: raw, Revision: api.Revision(raw), Metadata: map[string]any{}}
	parts := bytes.SplitN(raw, []byte("\n---\n"), 2)
	if len(parts) != 2 || !bytes.HasPrefix(parts[0], []byte("---\n")) {
		return note, []api.Diagnostic{{Code: "INVALID_FRONTMATTER", Layer: "conformance_errors"}}
	}
	decoder := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(parts[0], []byte("---\n"))))
	decoder.UseNumber()
	if err := decoder.Decode(&note.Metadata); err != nil {
		return note, []api.Diagnostic{{Code: "INVALID_FRONTMATTER", Layer: "conformance_errors"}}
	}
	note.Type, _ = note.Metadata["type"].(string)
	note.Title, _ = note.Metadata["title"].(string)
	note.Description, _ = note.Metadata["description"].(string)
	note.Body = string(parts[1])
	if layer, ok := note.Metadata["test_diagnostic_layer"].(string); ok {
		return note, []api.Diagnostic{{Code: "SYNTHETIC_FINDING", Layer: layer}}
	}
	return note, nil
}

func (fakeCodec) ValidateReserved(_ string, raw []byte) []api.Diagnostic {
	if bytes.Contains(raw, []byte("INVALID")) {
		return []api.Diagnostic{{Code: "INVALID_RESERVED", Layer: "conformance_errors"}}
	}
	return nil
}

func (codec fakeCodec) Patch(name string, raw []byte, op api.Operation, _ string, _ time.Time) ([]byte, error) {
	note, _ := codec.Parse(name, raw)
	if op.Kind == "create" {
		note.Metadata = map[string]any{}
		for key, value := range op.Metadata {
			note.Metadata[key] = value
		}
		if op.Type != "" {
			note.Metadata["type"] = op.Type
		}
		if op.Title != "" {
			note.Metadata["title"] = op.Title
		}
	}
	if op.Body != nil {
		note.Body = *op.Body
	}
	for _, edit := range op.BodyEdits {
		if edit.OldText == "" || strings.Count(note.Body, edit.OldText) != 1 {
			return nil, problem("INVALID_EDIT", "text must occur exactly once")
		}
		note.Body = strings.Replace(note.Body, edit.OldText, edit.NewText, 1)
	}
	for key, value := range op.MetadataPatch {
		note.Metadata[key] = value
	}
	for _, key := range op.RemoveMetadataKeys {
		delete(note.Metadata, key)
	}
	metadata, err := json.Marshal(note.Metadata)
	if err != nil {
		return nil, err
	}
	return []byte("---\n" + string(metadata) + "\n---\n" + note.Body), nil
}

type fakeProvider struct {
	root        string
	config      api.Config
	snapshotErr error
	extraLinks  []api.Link
}

func (p *fakeProvider) Root() string       { return p.root }
func (p *fakeProvider) Config() api.Config { return p.config }
func (*fakeProvider) Engine() api.Codec    { return fakeCodec{} }
func (p *fakeProvider) SnapshotUnlocked(ctx context.Context) (api.Snapshot, error) {
	if p.snapshotErr != nil {
		return api.Snapshot{}, p.snapshotErr
	}
	snapshot := api.Snapshot{Files: map[string][]byte{}}
	root, err := os.OpenRoot(p.root)
	if err != nil {
		return snapshot, err
	}
	defer func() { _ = root.Close() }()
	err = filepath.WalkDir(p.root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(p.root, name)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") && rel != "." {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !inScope(filepath.ToSlash(rel), p.config) {
			return nil
		}
		data, err := root.ReadFile(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		snapshot.Files[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		return snapshot, err
	}
	for name, data := range snapshot.Files {
		note, diagnostics := p.Engine().Parse(name, data)
		if reserved(name) {
			note.Body = string(data)
			diagnostics = nil
		}
		snapshot.Notes = append(snapshot.Notes, note)
		snapshot.Diagnostics = append(snapshot.Diagnostics, diagnostics...)
		for _, span := range markdownDestinations(note.Body) {
			target, anchor, ok := resolveKnown(name, span.value, snapshot.Files)
			snapshot.Links = append(snapshot.Links, api.Link{From: name, To: target, Anchor: anchor, Resolved: ok})
		}
	}
	snapshot.Generation = api.Generation(snapshot.Files)
	snapshot.Links = append(snapshot.Links, p.extraLinks...)
	return snapshot, nil
}

func setup(t *testing.T) (*Service, *fakeProvider) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := &fakeProvider{root: root}
	p.config.Write.Enabled = true
	p.config.Write.MaintainIndexes = true
	p.config.Write.MaintainLog = true
	s := New(p)
	s.now = func() time.Time { return time.Date(2026, 10, 24, 23, 30, 0, 0, time.FixedZone("offset", -3*60*60)) }
	return s, p
}
func put(t *testing.T, p *fakeProvider, name string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(p.root, filepath.FromSlash(path.Dir(name))), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.root, filepath.FromSlash(name)), data, 0600); err != nil {
		t.Fatal(err)
	}
}
func get(t *testing.T, p *fakeProvider, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(p.root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func document(kind, body string) []byte {
	return []byte("---\n{\"type\":\"" + kind + "\"}\n---\n" + body)
}
func body(value string) *string { return &value }
func plan(op api.Operation) api.Plan {
	return api.Plan{Operations: []api.Operation{op}, Actor: "synthetic-agent", Summary: "Record synthetic learning", IdempotencyKey: "request-1"}
}
func create(name string) api.Plan {
	return plan(api.Operation{Kind: "create", Path: name, Type: "Concept", Body: body("Synthetic body\n")})
}
func errorCode(t *testing.T, err error, code string) {
	t.Helper()
	var domain *api.DomainError
	if !errors.As(err, &domain) || domain.Code != code {
		t.Fatalf("expected %s, got %v", code, err)
	}
}
func inventory(t *testing.T, p *fakeProvider) map[string][]byte {
	t.Helper()
	snapshot, err := p.SnapshotUnlocked(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return snapshot.Files
}
func equalInventory(t *testing.T, want, got map[string][]byte) {
	t.Helper()
	if changed(want, got) {
		t.Fatalf("inventory changed: want %v, got %v", mapRevisions(want), mapRevisions(got))
	}
}
func mapRevisions(files map[string][]byte) map[string]string {
	out := map[string]string{}
	for name, data := range files {
		out[name] = api.Revision(data)
	}
	return out
}

func TestCreateDryRunReceiptReplayAndSourceImmutability(t *testing.T) {
	s, p := setup(t)
	p.config.Content.Exclude = []string{"raw/**"}
	put(t, p, "raw/source.md", []byte("immutable synthetic source"))
	request := create("topic/new.md")
	request.DryRun = true
	preview, err := s.Apply(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.DryRun || len(preview.Changes) != 4 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	for _, name := range []string{"topic", "index.md", "log.md", pendingPath, ".kb/state/receipts"} {
		if _, err := os.Stat(filepath.Join(p.root, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("dry run created %s: %v", name, err)
		}
	}
	request.DryRun = false
	receipt, err := s.Apply(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Generation != preview.Generation {
		t.Fatal("dry run differed from apply")
	}
	if !bytes.Contains(get(t, p, "log.md"), []byte("## 2026-10-25")) {
		t.Fatal("log did not use UTC")
	}
	if string(get(t, p, "raw/source.md")) != "immutable synthetic source" {
		t.Fatal("raw source changed")
	}
	before := inventory(t, p)
	for _, dry := range []bool{false, true} {
		request.DryRun = dry
		retry, err := s.Apply(context.Background(), request)
		if err != nil || !retry.Replayed || retry.Generation != receipt.Generation {
			t.Fatalf("retry: %+v %v", retry, err)
		}
	}
	equalInventory(t, before, inventory(t, p))
	request.Summary = "Different request"
	_, err = s.Apply(context.Background(), request)
	errorCode(t, err, "IDEMPOTENCY_CONFLICT")
}

func TestCASMultiNoteConflictAndConcurrentWriters(t *testing.T) {
	s, p := setup(t)
	put(t, p, "a.md", document("Concept", "old"))
	put(t, p, "b.md", document("Concept", "old"))
	before := inventory(t, p)
	request := plan(api.Operation{OperationID: "a", Kind: "patch", Path: "a.md", ExpectedRevision: api.Revision(before["a.md"]), Body: body("new")})
	request.Operations = append(request.Operations, api.Operation{OperationID: "b", Kind: "patch", Path: "b.md", ExpectedRevision: api.Revision([]byte("obsolete")), Body: body("new")})
	_, err := s.Apply(context.Background(), request)
	errorCode(t, err, "REVISION_CONFLICT")
	equalInventory(t, before, inventory(t, p))
	request.Operations = request.Operations[:1]
	var wait sync.WaitGroup
	errorsOut := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			r := request
			r.IdempotencyKey = fmt.Sprintf("concurrent-%d", i)
			_, err := New(p).Apply(context.Background(), r)
			errorsOut <- err
		}(i)
	}
	wait.Wait()
	close(errorsOut)
	success, conflicts := 0, 0
	for err := range errorsOut {
		if err == nil {
			success++
		} else {
			errorCode(t, err, "REVISION_CONFLICT")
			conflicts++
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflicts=%d", success, conflicts)
	}
}

func TestJournalRecoveryAtEveryDurableBoundary(t *testing.T) {
	probe, _ := setup(t)
	var boundaries []string
	probe.fault = func(stage, name string) error { boundaries = append(boundaries, stage+":"+name); return nil }
	if _, err := probe.Apply(context.Background(), create("nested/new.md")); err != nil {
		t.Fatal(err)
	}
	for _, boundary := range boundaries {
		t.Run(boundary, func(t *testing.T) {
			s, p := setup(t)
			put(t, p, "prior.md", document("Concept", "prior"))
			before := inventory(t, p)
			s.fault = func(stage, name string) error {
				if stage+":"+name == boundary {
					return errors.New("injected interruption")
				}
				return nil
			}
			request := create("nested/new.md")
			if _, err := s.Apply(context.Background(), request); err == nil {
				t.Fatal("interruption reported success")
			}
			s.fault = nil
			preview, err := s.Recover(context.Background(), true)
			if err != nil {
				t.Fatal(err)
			}
			if !preview.DryRun {
				t.Fatal("recovery preview missing dry-run")
			}
			if _, err := s.Recover(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			committed := strings.HasPrefix(boundary, "committed:") || strings.HasPrefix(boundary, "receipt:") || strings.HasPrefix(boundary, "removed:")
			if !committed {
				equalInventory(t, before, inventory(t, p))
			}
			receipt, err := s.Apply(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if receipt.Replayed != committed {
				t.Fatalf("replayed=%v committed=%v", receipt.Replayed, committed)
			}
			if strings.Count(string(get(t, p, "log.md")), "Record synthetic learning") != 1 {
				t.Fatal("duplicate log event")
			}
			if _, err := os.Stat(filepath.Join(p.root, pendingPath)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("pending journal remained: %v", err)
			}
		})
	}
}

func TestInterruptedRecoveryIsIdempotentAndMutationRecovers(t *testing.T) {
	for _, commit := range []bool{false, true} {
		t.Run(fmt.Sprint(commit), func(t *testing.T) {
			s, p := setup(t)
			s.fault = func(stage, _ string) error {
				if (!commit && stage == "published") || (commit && stage == "committed") {
					return errors.New("stop")
				}
				return nil
			}
			request := create("a.md")
			if _, err := s.Apply(context.Background(), request); err == nil {
				t.Fatal("missing fault")
			}
			s.fault = func(stage, _ string) error {
				if stage == "recovered" {
					return errors.New("recovery interrupted")
				}
				return nil
			}
			if _, err := s.Recover(context.Background(), false); err == nil {
				t.Fatal("missing recovery fault")
			}
			s.fault = nil
			receipt, err := s.Apply(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if receipt.Replayed != commit {
				t.Fatal("incorrect retry status")
			}
			if len(get(t, p, "a.md")) == 0 {
				t.Fatal("note missing")
			}
		})
	}
}

func TestExternalEditAtPublicationAndRecoveryNeverOverwrites(t *testing.T) {
	s, p := setup(t)
	put(t, p, "a.md", document("Concept", "before"))
	external := document("Concept", "external editor")
	s.fault = func(stage, _ string) error {
		if stage == "prepared" {
			put(t, p, "a.md", external)
		}
		return nil
	}
	request := plan(api.Operation{Kind: "patch", Path: "a.md", ExpectedRevision: api.Revision(get(t, p, "a.md")), Body: body("tool edit")})
	_, err := s.Apply(context.Background(), request)
	errorCode(t, err, "REVISION_CONFLICT")
	s.fault = nil
	_, err = s.Recover(context.Background(), false)
	errorCode(t, err, "REVISION_CONFLICT")
	if !bytes.Equal(get(t, p, "a.md"), external) {
		t.Fatal("external edit overwritten")
	}
}

func TestNoopDoesNotChangeNotesIndexesOrLog(t *testing.T) {
	s, p := setup(t)
	put(t, p, "a.md", document("Concept", "same"))
	before := inventory(t, p)
	request := plan(api.Operation{Kind: "patch", Path: "a.md", ExpectedRevision: api.Revision(before["a.md"]), Body: body("same")})
	receipt, err := s.Apply(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Changes) != 0 {
		t.Fatalf("no-op changes: %+v", receipt.Changes)
	}
	equalInventory(t, before, inventory(t, p))
}

func TestSyntheticIngestionIsAtomicAndOriginalSnapshotRemainsImmutable(t *testing.T) {
	s, p := setup(t)
	p.config.Content.Exclude = []string{"raw/**"}
	original := []byte("Synthetic original source, version one.\n")
	put(t, p, "raw/evidence.md", original)
	put(t, p, "a.md", document("Concept", "Prior A"))
	put(t, p, "b.md", document("Concept", "Prior B"))
	before := inventory(t, p)
	request := api.Plan{Actor: "synthetic-agent", Summary: "Incorporate synthetic evidence", IdempotencyKey: "ingestion", Operations: []api.Operation{
		{OperationID: "source", Kind: "create", Path: "source.md", Type: "Source", Body: body("Synthetic source description"), Metadata: map[string]any{"resource": "https://example.org/synthetic-evidence", "snapshot_digest": api.Revision(original)}},
		{OperationID: "a", Kind: "patch", Path: "a.md", ExpectedRevision: api.Revision(before["a.md"]), Body: body("Revised A with [evidence](source.md)")},
		{OperationID: "b", Kind: "patch", Path: "b.md", ExpectedRevision: api.Revision(before["b.md"]), Body: body("Revised B with [evidence](source.md)")},
	}}
	for _, index := range []int{1, 2} {
		bad := request
		bad.Operations = append([]api.Operation(nil), request.Operations...)
		bad.Operations[index].ExpectedRevision = api.Revision([]byte("obsolete"))
		_, err := s.Apply(context.Background(), bad)
		errorCode(t, err, "REVISION_CONFLICT")
		equalInventory(t, before, inventory(t, p))
	}
	if _, err := s.Apply(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	patch := plan(api.Operation{Kind: "patch", Path: "a.md", ExpectedRevision: api.Revision(get(t, p, "a.md")), BodyEdits: []api.TextEdit{{OldText: "Revised A", NewText: "Clarified A"}}})
	patch.IdempotencyKey = "clarification"
	if _, err := s.Apply(context.Background(), patch); err != nil {
		t.Fatal(err)
	}
	move := plan(api.Operation{Kind: "move", Path: "b.md", To: "archive/b.md", ExpectedRevision: api.Revision(get(t, p, "b.md"))})
	move.IdempotencyKey = "archive"
	if _, err := s.Apply(context.Background(), move); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(get(t, p, "raw/evidence.md"), original) {
		t.Fatal("original source snapshot changed")
	}
	if strings.Count(string(get(t, p, "log.md")), "Incorporate synthetic evidence") != 1 {
		t.Fatal("ingestion did not produce exactly one logical event")
	}
}
