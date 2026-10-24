package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
)

func testRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func newStore(t *testing.T, para bool) *Store {
	t.Helper()
	s, err := Init(context.Background(), testRoot(t), para)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func put(t *testing.T, s *Store, p, raw string) {
	t.Helper()
	name := filepath.Join(s.Root(), filepath.FromSlash(p))
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		t.Fatal(err)
	}
	//nolint:gosec // Synthetic fixture paths are explicitly controlled by each test.
	if err := os.WriteFile(name, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
}
func noteText(title, body string) string {
	return "---\ntype: Reference\ntitle: " + title + "\n---\n" + body
}
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var domain *api.DomainError
	if !errors.As(err, &domain) || domain.Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}
func TestInitScopeIdempotenceAndConfig(t *testing.T) {
	s := newStore(t, true)
	for _, p := range []string{"index.md", "log.md", "00_Inbox/index.md", "01_Projects/index.md", "02_Areas/index.md", "03_Resources/index.md", "04_Archive/index.md"} {
		//nolint:gosec // Reads an explicit list of synthetic initialized fixture files.
		raw, err := os.ReadFile(filepath.Join(s.Root(), p))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(p, "index.md") && !bytes.Contains(raw, []byte(indexStart)) {
			t.Fatal("missing managed marker")
		}
	}
	put(t, s, "index.md", "# Manual index\n")
	_, err := Init(context.Background(), s.Root(), true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(s.Root(), "index.md"))
	if err != nil || string(raw) != "# Manual index\n" {
		t.Fatal(err, string(raw))
	}
	for _, p := range []string{"custom/topic.md", ".git/private.md", ".kb/cache/private.md", "node_modules/module.md", "nested/.claude/state.md", "sources/original.txt"} {
		put(t, s, p, noteText("Synthetic", "content"))
	}
	snap, err := s.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Notes) != 1 || snap.Notes[0].Path != "custom/topic.md" || snap.Generation != api.Generation(snap.Files) {
		t.Fatalf("%+v", snap)
	}
	conf := s.Config()
	conf.Content.Include[0] = "wrong"
	if s.Config().Content.Include[0] != "**" {
		t.Fatal("mutable config escaped")
	}
	if s.Engine() == nil {
		t.Fatal("missing codec")
	}
	config := "version=1\n[content]\ninclude=['custom/**']\nexclude=['custom/hidden/**']\n[write]\nenabled=false\n"
	put(t, s, ".kb/config.toml", config)
	s, err = Open(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	put(t, s, "other.md", noteText("Other", "content"))
	put(t, s, "custom/hidden/no.md", noteText("Hidden", "content"))
	snap, err = s.Snapshot(context.Background())
	if err != nil || len(snap.Notes) != 1 || s.Config().Write.Enabled {
		t.Fatal(err, len(snap.Notes))
	}
	for _, text := range []string{"version=2", "invalid = [", "version=1\n[content]\ninclude=['../escape/**']", "version=1\n[para]\ninbox='same'\nprojects='SAME'"} {
		put(t, s, ".kb/config.toml", text)
		if _, err := Open(s.Root()); err == nil {
			t.Fatal("invalid config accepted", text)
		}
	}
}
func TestDiscoveryExplicitEnvAncestorAndInitDefault(t *testing.T) {
	s := newStore(t, false)
	t.Setenv("KB_ROOT", s.Root())
	root, err := ResolveRoot("")
	if err != nil || root != s.Root() {
		t.Fatal(root, err)
	}
	explicit := filepath.Join(s.Root(), "new")
	root, err = ResolveRoot(explicit)
	if err != nil || root != explicit {
		t.Fatal(root, err)
	}
	t.Setenv("KB_ROOT", "")
	nested := filepath.Join(s.Root(), "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	root, err = ResolveRoot("")
	if err != nil || root != s.Root() {
		t.Fatal(root, err)
	}
	t.Chdir(testRoot(t))
	if _, err := ResolveRoot(""); err == nil {
		t.Fatal("discovered unrelated base")
	}
	fresh, err := Init(context.Background(), "", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(fresh.Root(), "index.md")); err != nil {
		t.Fatal(err)
	}
}
func TestSnapshotInvalidFreshnessAndIsolation(t *testing.T) {
	s := newStore(t, false)
	put(t, s, "valid.md", noteText("Original", "body"))
	put(t, s, "invalid.md", "not YAML\n")
	snap, err := s.Snapshot(context.Background())
	if err != nil || len(snap.Notes) != 1 || !snap.Degraded {
		t.Fatal(err, snap)
	}
	before := snap.Generation
	snap.Notes[0].Metadata["type"] = "Mutated"
	snap.Files["valid.md"][0] = 'x'
	next, err := s.Snapshot(context.Background())
	if err != nil || next.Notes[0].Type != "Reference" || next.Files["valid.md"][0] != '-' {
		t.Fatal("snapshot mutation escaped", err)
	}
	put(t, s, "valid.md", noteText("Updated!", "body"))
	next, err = s.Snapshot(context.Background())
	if err != nil || before == next.Generation || next.Notes[0].Title != "Updated!" {
		t.Fatal(err, next)
	}
	status, err := s.Status(context.Background())
	if err != nil || status.Invalid != 1 || !status.Degraded || !status.IndexStale {
		t.Fatal(err, status)
	}
	report, err := s.Validate(context.Background(), "okf")
	if err != nil || report.Conformant {
		t.Fatal(err, report)
	}
	read, err := s.Read(context.Background(), api.ReadRequest{Path: "invalid.md", Raw: true})
	if err != nil || read.Markdown != "not YAML\n" || len(read.Diagnostics) == 0 {
		t.Fatal(err, read)
	}
}
func TestSymlinkPathsCollisionsAndPendingRecovery(t *testing.T) {
	s := newStore(t, false)
	outside := testRoot(t)
	if err := os.WriteFile(filepath.Join(outside, "secret.md"), []byte(noteText("Secret", "outside")), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(s.Root(), "escape")); err != nil {
		t.Fatal(err)
	}
	snap, err := s.Snapshot(context.Background())
	if err != nil || !snap.Degraded || len(snap.Notes) != 0 {
		t.Fatal(err, snap)
	}
	if _, err := s.Read(context.Background(), api.ReadRequest{Path: "escape/secret.md"}); err == nil {
		t.Fatal("read symlink escaped")
	}
	for _, p := range []string{"../outside.md", "/absolute.md", "a/../b.md", "a\\b.md"} {
		_, err := s.Read(context.Background(), api.ReadRequest{Path: p})
		requireCode(t, err, "INVALID_PATH")
	}
	put(t, s, "Name.md", noteText("First", "x"))
	put(t, s, "name.md", noteText("Second", "x"))
	snap, err = s.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range snap.Diagnostics {
		if d.Code == "PATH_COLLISION" {
			found = true
		}
	}
	if !found {
		entries, err := os.ReadDir(s.Root())
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, entry := range entries {
			if caseKey(entry.Name()) == caseKey("Name.md") {
				count++
			}
		}
		if count > 1 {
			t.Fatal("collision missing")
		}
	}
	r, err := os.OpenRoot(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if err := safePath(r, "NAME.md", true); err == nil {
		t.Fatal("destination collision accepted")
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	put(t, s, ".kb/state/pending.json", "{}")
	for _, read := range []func() error{func() error { _, e := s.Snapshot(context.Background()); return e }, func() error { _, e := s.Status(context.Background()); return e }, func() error { _, e := s.Search(context.Background(), api.SearchRequest{}); return e }, func() error { _, e := s.Rebuild(context.Background()); return e }, func() error { _, e := s.Export(context.Background(), filepath.Join(testRoot(t), "export")); return e }} {
		requireCode(t, read(), "RECOVERY_REQUIRED")
	}
	if _, err := s.SnapshotUnlocked(context.Background()); err != nil {
		t.Fatal("writer snapshot blocked by journal", err)
	}
}
func TestReadSectionsRangesAndBounds(t *testing.T) {
	s := newStore(t, false)
	put(t, s, "note.md", noteText("Repeated", "# Repeat\nfirst\n## Child\nchild\n# Repeat\nsecond\n"))
	_, err := s.Read(context.Background(), api.ReadRequest{Path: "note.md", Section: "Repeat"})
	requireCode(t, err, "AMBIGUOUS_SECTION")
	got, err := s.Read(context.Background(), api.ReadRequest{Path: "note.md", Section: "#repeat-1"})
	if err != nil || got.Markdown != "# Repeat\nsecond\n" || got.LineStart != 9 {
		t.Fatal(err, got)
	}
	got, err = s.Read(context.Background(), api.ReadRequest{Path: "note.md", LineStart: 6, LineEnd: 8, IncludeLinks: true})
	if err != nil || got.Markdown != "first\n## Child\nchild\n" {
		t.Fatal(err, got)
	}
	for _, request := range []api.ReadRequest{{Path: "note.md", Section: "missing"}, {Path: "note.md", LineStart: 100}, {Path: "note.md", LineEnd: -1}, {Path: "note.md", Section: "Repeat", LineStart: 1}, {Path: "missing.md"}} {
		if _, err := s.Read(context.Background(), request); err == nil {
			t.Fatal("invalid read accepted", request)
		}
	}
	put(t, s, "long.md", noteText("Long", strings.Repeat("ação synthetic long line\n", 20000)))
	got, err = s.Read(context.Background(), api.ReadRequest{Path: "long.md"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(got)
	if err != nil || len(encoded) > api.MaxReadBytes || len(got.Diagnostics) == 0 {
		t.Fatal(err, len(encoded))
	}
}
func TestContextBudgetsCitationsAndOneHop(t *testing.T) {
	s := newStore(t, false)
	put(t, s, "a.md", noteText("Alpha", "# Topic\nSynthetic alpha evidence. [Related](b.md)\n"))
	put(t, s, "b.md", noteText("Related", "# Detail\nOne hop relationship only. [Next](c.md)\n"))
	put(t, s, "c.md", noteText("Next", "Do not recurse.\n"))
	got, err := s.Context(context.Background(), api.ContextRequest{Query: "alpha", IncludeLinks: true, MaxTokens: 2000})
	if err != nil || len(got.Sources) != 2 || got.Sources[0].Revision == "" || !strings.Contains(got.Text, "a.md:") {
		t.Fatal(err, got)
	}
	for _, source := range got.Sources {
		if source.Path == "c.md" {
			t.Fatal("traversed more than one hop")
		}
	}
	got, err = s.Context(context.Background(), api.ContextRequest{Query: "alpha", MaxTokens: 15, MaxBytes: 60})
	if err != nil || !got.Truncated || len(got.Text) > 60 || got.EstimatedTokens > 15 {
		t.Fatal(err, got)
	}
	for _, req := range []api.ContextRequest{{MaxTokens: -1}, {MaxTokens: api.MaxContextTokens + 1}, {MaxBytes: api.MaxReadBytes + 1}, {Query: strings.Repeat("a", api.MaxQueryRunes+1)}} {
		if _, err := s.Context(context.Background(), req); err == nil {
			t.Fatal("context limit accepted")
		}
	}
}
func TestHealthGraphTemporalBasisAndExternalInert(t *testing.T) {
	s := newStore(t, false)
	put(t, s, "basis.md", noteText("Basis", "# Target\nOriginal evidence.\n"))
	basis, err := os.ReadFile(filepath.Join(s.Root(), "basis.md"))
	if err != nil {
		t.Fatal(err)
	}
	put(t, s, "synthesis.md", "---\ntype: Synthesis\nstale_after: 2026-10-24T11:00:00+02:00\nbasis_revisions:\n  basis.md: "+api.Revision(basis)+"\nsources:\n  - resource: basis.md\n  - resource: https://example.invalid/source\n---\n[Target](/basis.md#target)\n[Missing](missing.md)\n[External](https://example.invalid/never-fetch)\n[Ref][basis]\n[basis]: basis.md\n````\n[Code](ignored.md)\n````\n")
	put(t, s, "orphan.md", noteText("Orphan", "Unlinked synthetic note.\n"))
	s.SetAsOf(time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC))
	links, err := s.Links(context.Background(), "synthesis.md", "out")
	if err != nil {
		t.Fatal(err)
	}
	var provenance, external, broken, anchor bool
	for _, link := range links {
		provenance = provenance || link.Kind == "provenance"
		external = external || link.External
		broken = broken || link.To == "missing.md" && !link.Resolved
		anchor = anchor || link.To == "basis.md" && link.Anchor == "target" && link.Resolved
		if link.To == "ignored.md" {
			t.Fatal("link inside code")
		}
	}
	if !provenance || !external || !broken || !anchor {
		t.Fatal(links)
	}
	report, err := s.Validate(context.Background(), "health")
	if err != nil || !report.Conformant {
		t.Fatal(err, report)
	}
	codes := map[string]bool{}
	for _, d := range report.Diagnostics {
		codes[d.Code] = true
	}
	for _, code := range []string{"BROKEN_LINK", "ORPHAN", "UNINDEXED", "STALE"} {
		if !codes[code] {
			t.Fatal("missing", code)
		}
	}
	if codes["BASIS_CHANGED"] {
		t.Fatal("unchanged dependency marked changed")
	}
	put(t, s, "basis.md", noteText("Basis", "Changed evidence.\n"))
	report, err = s.Validate(context.Background(), "health")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range report.Diagnostics {
		found = found || d.Code == "BASIS_CHANGED"
	}
	if !found {
		t.Fatal("missing changed basis")
	}
	if _, err := s.Links(context.Background(), "basis.md", "incoming"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Links(context.Background(), "basis.md", "bad"); err == nil {
		t.Fatal("bad direction accepted")
	}
	if _, err := s.Validate(context.Background(), "unsupported"); err == nil {
		t.Fatal("bad profile accepted")
	}
}
func TestRebuildExportNoOverwriteAndSourcePreservation(t *testing.T) {
	s := newStore(t, false)
	put(t, s, "notes/valid.md", noteText("Valid", "Original source.\n"))
	put(t, s, "sources/raw.bin", "Synthetic original bytes")
	put(t, s, "broken.md", "invalid note")
	before, err := os.ReadFile(filepath.Join(s.Root(), "notes/valid.md"))
	if err != nil {
		t.Fatal(err)
	}
	status, err := s.Rebuild(context.Background())
	if err != nil || status.IndexStale {
		t.Fatal(err, status)
	}
	status, err = s.Status(context.Background())
	if err != nil || status.IndexStale {
		t.Fatal(err, status)
	}
	destination := filepath.Join(testRoot(t), "bundle")
	result, err := s.Export(context.Background(), destination)
	if err != nil || result.Conformant || contains(result.Files, "broken.md") || !contains(result.Files, "notes/valid.md") || !contains(result.Excluded, "sources/raw.bin") {
		t.Fatal(err, result)
	}
	for _, p := range []string{".kb", "sources/raw.bin", "broken.md"} {
		if _, err := os.Lstat(filepath.Join(destination, p)); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("excluded path exported", p, err)
		}
	}
	//nolint:gosec // Reads the test-owned export manifest beneath a temporary destination.
	raw, err := os.ReadFile(filepath.Join(destination, "manifest.json"))
	if err != nil || !bytes.Contains(raw, []byte("INVALID_NOTE")) {
		t.Fatal(err, string(raw))
	}
	_, err = s.Export(context.Background(), destination)
	requireCode(t, err, "ALREADY_EXISTS")
	_, err = s.Export(context.Background(), filepath.Join(s.Root(), "export"))
	requireCode(t, err, "INVALID_PATH")
	after, err := os.ReadFile(filepath.Join(s.Root(), "notes/valid.md"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("read modified note", err)
	}
}
func TestConcurrentReadersAndCancellation(t *testing.T) {
	s := newStore(t, false)
	for i := range 20 {
		put(t, s, "note-"+itoa(i)+".md", noteText("Synthetic", "s3 k8s ação\n"))
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 3 {
				got, err := s.Search(context.Background(), api.SearchRequest{Query: "s3"})
				if err != nil || got.Total != 20 {
					t.Error(err, got.Total)
				}
			}
		})
	}
	wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Snapshot(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestGlobSemantics(t *testing.T) {
	for _, test := range []struct {
		pattern, p string
		want       bool
	}{{"**", "x/y.md", true}, {"notes/**", "notes/a/b.md", true}, {"**/*.md", "root.md", true}, {"notes/*.md", "notes/a/b.md", false}, {"a/**/b.md", "a/b.md", true}, {"a/**/b.md", "a/x/b.md", true}, {"a/**/b.md", "b.md", false}} {
		if got := globMatch(test.pattern, test.p); got != test.want {
			t.Errorf("%+v => %v", test, got)
		}
	}
}
func TestSearchMultilingualTokensRankingFiltersAndCursors(t *testing.T) {
	s := newStore(t, true)
	// All multilingual examples are synthetic and intentionally exercise Unicode.
	fixtures := map[string]string{"03_Resources/ação.md": "---\ntype: Reference\ntitle: Ação para recuperação\ntags: [backup, português]\ndescription: Restauração sintética\n---\nAção recuperação s3 k8s café 東京 中文 Привет.\n", "03_Resources/body.md": noteText("Other", "ação in body and s3backup k8sops\n"), "01_Projects/project.md": "---\ntype: Project\ntitle: Project\ntags: [urgent]\nstatus: draft\n---\ns3 k8s\n", "04_Archive/old.md": noteText("Archived", "s3\n"), "deprecated.md": "---\ntype: Reference\nstatus: deprecated\n---\ns3\n"}
	for p, raw := range fixtures {
		put(t, s, p, raw)
	}
	for _, query := range []string{"acao", "ação", "AÇÃO", "recuperacao", "cafe", "東京", "中文", "привет"} {
		got, err := s.Search(context.Background(), api.SearchRequest{Query: query})
		if err != nil || got.Total == 0 || got.Hits[0].Path != "03_Resources/ação.md" {
			t.Fatal(query, err, got)
		}
	}
	got, err := s.Search(context.Background(), api.SearchRequest{Query: "s3 k8s"})
	if err != nil || got.Total != 2 {
		t.Fatal(err, got)
	}
	for _, hit := range got.Hits {
		if hit.Body != "" || hit.Raw != nil {
			t.Fatal("full body exposed")
		}
	}
	got, err = s.Search(context.Background(), api.SearchRequest{Filters: api.Filters{Tags: []string{"backup", "urgent"}, Types: []string{"Reference"}, PARA: []string{"resources"}}})
	if err != nil || got.Total != 1 || got.Hits[0].Path != "03_Resources/ação.md" {
		t.Fatal(err, got)
	}
	got, err = s.Search(context.Background(), api.SearchRequest{Query: "s3", Filters: api.Filters{IncludeArchives: true, IncludeDeprecated: true}})
	if err != nil || got.Total != 4 {
		t.Fatal(err, got)
	}
	got, err = s.List(context.Background(), api.SearchRequest{Limit: 1})
	if err != nil || got.Cursor == "" {
		t.Fatal(err, got)
	}
	cursor := got.Cursor
	next, err := s.List(context.Background(), api.SearchRequest{Limit: 1, Cursor: cursor})
	if err != nil || next.Hits[0].Path == got.Hits[0].Path {
		t.Fatal(err, next)
	}
	_, err = s.Search(context.Background(), api.SearchRequest{Query: "different", Limit: 1, Cursor: cursor})
	requireCode(t, err, "INVALID_CURSOR")
	put(t, s, "new.md", noteText("New", "new content"))
	_, err = s.List(context.Background(), api.SearchRequest{Limit: 1, Cursor: cursor})
	requireCode(t, err, "STALE_CURSOR")
	_, err = s.Search(context.Background(), api.SearchRequest{Query: strings.Repeat("a", api.MaxQueryRunes+1)})
	requireCode(t, err, "LIMIT_EXCEEDED")
	_, err = s.Search(context.Background(), api.SearchRequest{Limit: 101})
	requireCode(t, err, "LIMIT_EXCEEDED")
	_, err = s.Search(context.Background(), api.SearchRequest{Cursor: "bad!"})
	requireCode(t, err, "INVALID_CURSOR")
	got, err = s.Search(context.Background(), api.SearchRequest{Query: "s3 k8s", Literal: true})
	if err != nil || got.Total != 2 {
		t.Fatal(err, got)
	}
	empty, err := s.List(context.Background(), api.SearchRequest{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{}
	for _, hit := range empty.Hits {
		paths = append(paths, hit.Path)
	}
	if !sort.StringsAreSorted(paths) {
		t.Fatal("unstable inventory order")
	}
}
