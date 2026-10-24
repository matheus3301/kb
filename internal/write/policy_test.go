package write

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func TestPathConfinementAndCaseCollisions(t *testing.T) {
	for _, name := range []string{"../escape.md", "/absolute.md", "a/../escape.md", "./a.md", "a//b.md", "a\\b.md", "C:drive.md", ".kb/secret.md", "raw/source.md", "node_modules/n.md", ".git/n.md", "non-markdown.txt"} {
		t.Run(name, func(t *testing.T) {
			s, p := setup(t)
			p.config.Content.Exclude = []string{"raw/**"}
			_, err := s.Apply(context.Background(), create(name))
			if err == nil {
				t.Fatal("unsafe path accepted")
			}
		})
	}
	for _, name := range []string{"Note.md", "Folder/n.md"} {
		t.Run(name, func(t *testing.T) {
			s, p := setup(t)
			put(t, p, strings.ToLower(name), document("Concept", "old"))
			_, err := s.Apply(context.Background(), create(name))
			errorCode(t, err, "PATH_COLLISION")
		})
	}
	s, p := setup(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(p.root, "linked")); err != nil {
		t.Fatal(err)
	}
	_, err := s.Apply(context.Background(), create("linked/new.md"))
	errorCode(t, err, "PATH_OUTSIDE_ROOT")
	if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
		t.Fatal("symlink target changed")
	}
}

func TestPlanRejectsInvalidAndOverlappingOperations(t *testing.T) {
	cases := []struct {
		name string
		edit func(*api.Plan)
	}{
		{"empty actor", func(p *api.Plan) { p.Actor = "" }},
		{"empty summary", func(p *api.Plan) { p.Summary = " " }},
		{"empty key", func(p *api.Plan) { p.IdempotencyKey = "" }},
		{"no operations", func(p *api.Plan) { p.Operations = nil }},
		{"too many", func(p *api.Plan) { p.Operations = make([]api.Operation, 51) }},
		{"unknown kind", func(p *api.Plan) { p.Operations[0].Kind = "delete" }},
		{"missing type", func(p *api.Plan) { p.Operations[0].Type = "" }},
		{"conflicting body", func(p *api.Plan) { p.Operations[0].BodyEdits = []api.TextEdit{{OldText: "old", NewText: "new"}} }},
		{"create expected revision", func(p *api.Plan) { p.Operations[0].ExpectedRevision = api.Revision(nil) }},
		{"unencodable metadata", func(p *api.Plan) { p.Operations[0].Metadata = map[string]any{"bad": make(chan int)} }},
		{"verified create", func(p *api.Plan) {
			p.Operations[0].Metadata = map[string]any{"verified": map[string]any{"by": "human"}}
		}},
		{"verified patch", func(p *api.Plan) { p.Operations[0].MetadataPatch = map[string]any{"verified": true} }},
		{"unexpected to", func(p *api.Plan) { p.Operations[0].To = "b.md" }},
		{"document without repair", func(p *api.Plan) { p.Operations[0].Document = body("replacement") }},
		{"mixed document", func(p *api.Plan) { p.Repair = true; p.Operations[0].Document = body("replacement") }},
		{"duplicate IDs", func(p *api.Plan) {
			p.Operations[0].OperationID = "a"
			p.Operations = append(p.Operations, api.Operation{OperationID: "a", Kind: "create", Path: "b.md", Type: "Concept"})
		}},
		{"missing IDs", func(p *api.Plan) {
			p.Operations = append(p.Operations, api.Operation{Kind: "create", Path: "b.md", Type: "Concept"})
		}},
		{"overlapping paths", func(p *api.Plan) {
			p.Operations[0].OperationID = "a"
			p.Operations = append(p.Operations, api.Operation{OperationID: "b", Kind: "create", Path: "a.md", Type: "Concept"})
		}},
		{"oversized plan", func(p *api.Plan) { p.Operations[0].Body = body(strings.Repeat("x", api.MaxPlanBytes)) }},
		{"oversized note", func(p *api.Plan) { p.Operations[0].Body = body(strings.Repeat("x", api.MaxNoteBytes)) }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			s, p := setup(t)
			request := create("a.md")
			test.edit(&request)
			_, err := s.Apply(context.Background(), request)
			if err == nil {
				t.Fatal("invalid plan accepted")
			}
			if len(inventory(t, p)) != 0 {
				t.Fatal("rejected plan changed notes")
			}
		})
	}
}

func TestRepairRequiresExplicitCorrectionAndConformantFinalState(t *testing.T) {
	s, p := setup(t)
	put(t, p, "bad.md", []byte("unformatted"))
	request := plan(api.Operation{Kind: "patch", Path: "bad.md", ExpectedRevision: api.Revision(get(t, p, "bad.md")), Body: body("new")})
	_, err := s.Apply(context.Background(), request)
	errorCode(t, err, "INVALID_FRONTMATTER")
	request.Repair = true
	_, err = s.Apply(context.Background(), request)
	errorCode(t, err, "INVALID_PLAN")
	request.Operations[0].Body = nil
	request.Operations[0].Document = body("still invalid")
	_, err = s.Apply(context.Background(), request)
	errorCode(t, err, "INVALID_FRONTMATTER")
	request.Operations[0].Document = body(string(document("Concept", "repaired")))
	if _, err := s.Apply(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	note, _ := p.Engine().Parse("bad.md", get(t, p, "bad.md"))
	if note.Type != "Concept" {
		t.Fatal("repair failed")
	}
	s2, p2 := setup(t)
	put(t, p2, "bad.md", []byte("unformatted"))
	request = plan(api.Operation{Kind: "patch", Path: "bad.md", ExpectedRevision: api.Revision(get(t, p2, "bad.md")), MetadataPatch: map[string]any{"type": "ExplicitType"}})
	request.Repair = true
	if _, err := s2.Apply(context.Background(), request); err != nil {
		t.Fatal(err)
	}
}

func TestMoveRewritesResolvedLinksProvenanceAndBasisPreservingAnchors(t *testing.T) {
	s, p := setup(t)
	put(t, p, "a.md", document("Concept", "A\n"))
	put(t, p, "index.md", []byte("# Index\n\n- [Manual description](a.md#manual)\n"))
	put(t, p, "other.md", document("Concept", "Other\n"))
	meta := map[string]any{"type": "Synthesis", "verified": map[string]any{"by": "human"}, "sources": []any{map[string]any{"resource": "a.md#source", "title": "Evidence"}}, "basis_revisions": []any{map[string]any{"path": "a.md", "revision": api.Revision(get(t, p, "a.md"))}}}
	metadata, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	text := "[local](a.md#anchor) [root](/a.md#root) [external](https://example.org/a.md#keep)\n`[code](a.md)`\n```md\n[fenced](a.md)\n```\n"
	put(t, p, "ref.md", []byte("---\n"+string(metadata)+"\n---\n"+text))
	put(t, p, "outgoing.md", document("Concept", "[other](other.md#heading)\n"))
	request := plan(api.Operation{OperationID: "a", Kind: "move", Path: "a.md", To: "nested/a.md", ExpectedRevision: api.Revision(get(t, p, "a.md"))})
	request.Operations = append(request.Operations, api.Operation{OperationID: "b", Kind: "move", Path: "outgoing.md", To: "nested/outgoing.md", ExpectedRevision: api.Revision(get(t, p, "outgoing.md"))})
	if _, err := s.Apply(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	ref := string(get(t, p, "ref.md"))
	for _, expected := range []string{"nested/a.md#anchor", "/nested/a.md#root", "https://example.org/a.md#keep", "`[code](a.md)`", "[fenced](a.md)", "nested/a.md#source", "\"path\":\"nested/a.md\""} {
		if !strings.Contains(ref, expected) {
			t.Fatalf("missing %s in %s", expected, ref)
		}
	}
	if strings.Contains(ref, "verified") {
		t.Fatal("verification remained after link changes")
	}
	if !strings.Contains(string(get(t, p, "nested/outgoing.md")), "../other.md#heading") {
		t.Fatal("moved note outgoing relative link not updated")
	}
	if !strings.Contains(string(get(t, p, "log.md")), "verification cleared because content changed") {
		t.Fatal("log omitted trust reason")
	}
	if !strings.Contains(string(get(t, p, "index.md")), "[Manual description](nested/a.md#manual)") {
		t.Fatal("manual index link was not relocated")
	}
}

func TestPlannedParentCaseCollisionAndFileOverlapFailBeforePublication(t *testing.T) {
	for _, paths := range [][2]string{{"Topic/a.md", "topic/b.md"}, {"a.md", "a.md/b.md"}, {"Σ/a.md", "ς/b.md"}} {
		s, p := setup(t)
		request := create(paths[0])
		request.Operations[0].OperationID = "one"
		request.Operations = append(request.Operations, api.Operation{OperationID: "two", Kind: "create", Path: paths[1], Type: "Concept"})
		if _, err := s.Apply(context.Background(), request); err == nil {
			t.Fatal("overlapping or case-colliding parents accepted")
		}
		if len(inventory(t, p)) != 0 {
			t.Fatal("rejected plan published files")
		}
		if _, err := os.Stat(filepath.Join(p.root, pendingPath)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("rejected plan created journal")
		}
	}
}

func TestScopeGlobsSupportUnicodeAndCharacterClasses(t *testing.T) {
	for _, test := range []struct {
		pattern, name string
		want          bool
	}{
		{"**", "a/b.md", true}, {"**/*.md", "a.md", true}, {"**/*.md", "a/b.md", true},
		{"café/**", "café/note.md", true}, {"[ab]?.md", "az.md", true}, {"*.md", "nested/a.md", false}, {"[invalid", "a.md", false},
	} {
		if got := glob(test.pattern, test.name); got != test.want {
			t.Errorf("glob(%q,%q)=%v", test.pattern, test.name, got)
		}
	}
	if !strings.EqualFold("Σ", "ς") || caseKey("Σ") != caseKey("ς") {
		t.Fatal("Unicode fold classes diverged")
	}
}

func TestExternalDependencyChangeBeforeCommitBlocksCommit(t *testing.T) {
	s, p := setup(t)
	put(t, p, "basis.md", document("Concept", "original"))
	request := create("synthesis.md")
	request.Operations[0].Metadata = map[string]any{"basis_revisions": []any{map[string]any{"path": "basis.md", "revision": api.Revision(get(t, p, "basis.md"))}}}
	s.fault = func(stage, _ string) error {
		if stage == "published" {
			put(t, p, "basis.md", document("Concept", "external edit"))
		}
		return nil
	}
	_, err := s.Apply(context.Background(), request)
	errorCode(t, err, "REVISION_CONFLICT")
	s.fault = nil
	if _, err := s.Recover(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(get(t, p, "basis.md")), "original") {
		t.Fatal("external dependency edit was lost")
	}
	if _, err := os.Stat(filepath.Join(p.root, "synthesis.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("uncommitted synthesis remained")
	}
}

func TestRecoveryRechecksFilesAtEachReplacement(t *testing.T) {
	s, p := setup(t)
	put(t, p, "a.md", document("Concept", "before"))
	s.fault = func(stage, _ string) error {
		if stage == "committed" {
			return errors.New("interrupted")
		}
		return nil
	}
	request := plan(api.Operation{Kind: "patch", Path: "a.md", ExpectedRevision: api.Revision(get(t, p, "a.md")), Body: body("after")})
	if _, err := s.Apply(context.Background(), request); err == nil {
		t.Fatal("missing interruption")
	}
	var pending journal
	if err := decode(get(t, p, pendingPath), &pending); err != nil {
		t.Fatal(err)
	}
	if len(pending.Files) < 2 {
		t.Fatal("expected multiple recovery files")
	}
	target := pending.Files[1].Path
	external := []byte("# External edit\n")
	s.fault = func(stage, name string) error {
		if stage == "recovered" && name == pending.Files[0].Path {
			put(t, p, target, external)
		}
		return nil
	}
	_, err := s.Recover(context.Background(), false)
	errorCode(t, err, "REVISION_CONFLICT")
	if string(get(t, p, target)) != string(external) {
		t.Fatal("recovery overwrote concurrent external edit")
	}
}

func TestReplacementChecksCASAfterPreparingTemporaryFile(t *testing.T) {
	_, p := setup(t)
	put(t, p, "a.md", document("Concept", "current"))
	root, err := os.OpenRoot(p.root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	expected := image{Exists: true, Data: document("Concept", "stale")}
	for _, next := range []image{{Exists: true, Data: document("Concept", "replacement")}, {}} {
		err := replaceImage(root, "a.md", next, &expected)
		errorCode(t, err, "REVISION_CONFLICT")
		if string(get(t, p, "a.md")) != string(document("Concept", "current")) {
			t.Fatal("stale replacement overwrote file")
		}
	}
	entries, err := os.ReadDir(p.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".kb-tmp-") {
			t.Fatal("failed CAS leaked temporary file")
		}
	}
}

func TestAmbiguousExternalAndCodeReferencesArePreserved(t *testing.T) {
	s, p := setup(t)
	put(t, p, "a.md", document("Concept", "A"))
	put(t, p, "a/index.md", []byte("# Index\n"))
	content := "[ambiguous](a) [remote](https://example.org/a.md) [query](a.md?q=x)\n    [indented](a.md)\n~~~\n[fence](a.md)\n~~~\n"
	put(t, p, "ref.md", document("Concept", content))
	request := plan(api.Operation{Kind: "move", Path: "a.md", To: "b.md", ExpectedRevision: api.Revision(get(t, p, "a.md"))})
	if _, err := s.Apply(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	note, _ := p.Engine().Parse("ref.md", get(t, p, "ref.md"))
	if note.Body != content {
		t.Fatalf("unresolved references changed: %s", note.Body)
	}
}

func TestMaintainPreservesManualTextAndProgressiveNavigation(t *testing.T) {
	s, p := setup(t)
	manual := "# Manual index\n\nKeep this introduction.\n\n- [Hand-written title](a.md) — hand-written description\n"
	put(t, p, "index.md", []byte(manual))
	put(t, p, "a.md", document("Concept", "A"))
	put(t, p, "nested/deeper/b.md", document("Concept", "B"))
	put(t, p, "special.md", []byte("---\n{\"type\":\"Concept\",\"title\":\"[unsafe] *title* <tag>\",\"description\":\"Description\"}\n---\nBody"))
	before := inventory(t, p)
	preview, err := s.Maintain(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.DryRun || len(preview.Changes) == 0 {
		t.Fatal("missing maintenance preview")
	}
	equalInventory(t, before, inventory(t, p))
	if _, err := s.Maintain(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	index := string(get(t, p, "index.md"))
	if !strings.HasPrefix(index, manual) || strings.Count(index, "(a.md)") != 1 || !strings.Contains(index, "(nested/index.md)") || strings.Contains(index, "nested/deeper/b.md") {
		t.Fatalf("manual/progressive index failed: %s", index)
	}
	if !strings.Contains(index, "\\[unsafe\\] \\*title\\* &lt;tag&gt;") {
		t.Fatal("title was not escaped")
	}
	if !strings.Contains(string(get(t, p, "nested/index.md")), "(deeper/index.md)") {
		t.Fatal("missing progressive child")
	}
	before = inventory(t, p)
	receipt, err := s.Maintain(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Changes) != 0 {
		t.Fatal("second maintenance was not a no-op")
	}
	equalInventory(t, before, inventory(t, p))
}

func TestMalformedManagedIndexAndInvalidLogBlockPublication(t *testing.T) {
	for _, name := range []string{"index.md", "log.md"} {
		t.Run(name, func(t *testing.T) {
			s, p := setup(t)
			raw := []byte("INVALID\n")
			if name == "index.md" {
				raw = []byte("# Index\n" + indexStart + "\n")
			}
			put(t, p, name, raw)
			before := inventory(t, p)
			_, err := s.Apply(context.Background(), create("a.md"))
			if err == nil {
				t.Fatal("invalid navigation accepted")
			}
			equalInventory(t, before, inventory(t, p))
		})
	}
}

func TestSynthesisDependenciesValidatedBeforePublication(t *testing.T) {
	for _, basis := range []any{true, []any{"invalid"}, []any{map[string]any{"path": "../outside.md", "revision": api.Revision(nil)}}, []any{map[string]any{"path": "a.md", "revision": "bad"}}, []any{map[string]any{"path": "a.md", "revision": api.Revision([]byte("stale"))}}} {
		s, p := setup(t)
		put(t, p, "a.md", document("Concept", "A"))
		before := inventory(t, p)
		request := create("synthesis.md")
		request.Operations[0].Metadata = map[string]any{"basis_revisions": basis}
		_, err := s.Apply(context.Background(), request)
		if err == nil {
			t.Fatalf("invalid dependency accepted: %v", basis)
		}
		equalInventory(t, before, inventory(t, p))
	}
	s, p := setup(t)
	put(t, p, "a.md", document("Concept", "A"))
	request := create("synthesis.md")
	request.Operations[0].Metadata = map[string]any{"basis_revisions": []any{map[string]any{"path": "a.md", "revision": api.Revision(get(t, p, "a.md"))}}}
	if _, err := s.Apply(context.Background(), request); err != nil {
		t.Fatal(err)
	}
}

func TestUntrustedJournalIsBoundedAndConfined(t *testing.T) {
	for _, name := range []string{"../outside.md", "/outside.md", ".kb/config.toml", "raw/source.md", "link/a.md"} {
		t.Run(name, func(t *testing.T) {
			s, p := setup(t)
			p.config.Content.Exclude = []string{"raw/**"}
			if err := os.Symlink(t.TempDir(), filepath.Join(p.root, "link")); err != nil {
				t.Fatal(err)
			}
			j := journal{Version: 1, State: "committed", Key: "key", Digest: api.Revision(nil), Receipt: api.Receipt{IdempotencyKey: "key"}, Files: []mutation{{Path: name, After: image{Exists: true, Data: []byte("malicious")}}}}
			data, err := json.Marshal(j)
			if err != nil {
				t.Fatal(err)
			}
			put(t, p, pendingPath, data)
			_, err = s.Recover(context.Background(), false)
			if err == nil {
				t.Fatal("unsafe journal accepted")
			}
			if len(inventory(t, p)) != 0 {
				t.Fatal("unsafe recovery wrote notes")
			}
		})
	}
	for _, raw := range []string{"{}", "{", `{"version":1,"unexpected":true}`, `{} {}`} {
		s, p := setup(t)
		put(t, p, pendingPath, []byte(raw))
		_, err := s.Recover(context.Background(), false)
		errorCode(t, err, "INVALID_STATE")
	}
	_, p := setup(t)
	root, err := os.OpenRoot(p.root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	put(t, p, "big.md", []byte("12345"))
	_, err = readImage(root, "big.md", 4)
	errorCode(t, err, "INPUT_TOO_LARGE")
}

func TestReadOnlyCancellationAndPendingDryRun(t *testing.T) {
	s, p := setup(t)
	p.config.Write.Enabled = false
	_, err := s.Apply(context.Background(), create("a.md"))
	errorCode(t, err, "READ_ONLY")
	_, err = s.Maintain(context.Background(), true)
	errorCode(t, err, "READ_ONLY")
	_, err = s.Recover(context.Background(), false)
	errorCode(t, err, "READ_ONLY")
	p.config.Write.Enabled = true
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Apply(ctx, create("a.md"))
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	s.fault = func(stage, _ string) error {
		if stage == "prepared" {
			return errors.New("stop")
		}
		return nil
	}
	if _, err := s.Apply(context.Background(), create("a.md")); err == nil {
		t.Fatal("missing interruption")
	}
	s.fault = nil
	request := create("b.md")
	request.DryRun = true
	_, err = s.Apply(context.Background(), request)
	errorCode(t, err, "RECOVERY_REQUIRED")
	_, err = s.Maintain(context.Background(), true)
	errorCode(t, err, "RECOVERY_REQUIRED")
}

func FuzzPlanPaths(f *testing.F) {
	for _, seed := range []string{"a.md", "../escape.md", "/absolute.md", "topic/a.md", "a\\b.md", "a//b.md", "世界.md", ".kb/x.md"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, name string) {
		if cleanPath(name) == nil {
			if filepath.IsAbs(name) || pathEscape(name) {
				t.Fatalf("unsafe canonical path: %q", name)
			}
		}
		_ = inScope(name, api.Config{})
	})
}

func pathEscape(name string) bool {
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func FuzzMarkdownDestinations(f *testing.F) {
	for _, seed := range []string{"[title](a.md#anchor)", "`[code](x)`", "```\n[x](x)\n```", "[x]: <a.md> \"title\"", "[x](with(nested).md)"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		last := 0
		for _, span := range markdownDestinations(text) {
			if span.start < last || span.end < span.start || span.end > len(text) || text[span.start:span.end] != span.value {
				t.Fatalf("invalid destination: %+v", span)
			}
			last = span.end
		}
	})
}
