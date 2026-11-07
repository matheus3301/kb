package setup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestRoutingBoundaryWorktreeAndPortable(t *testing.T) {
	e, project, root := fixtureEngine(t)
	ctx := context.Background()
	put(t, filepath.Join(project, ".git"), []byte("gitdir: ignored-synthetic\n"))
	global := installation("codex", project, root)
	global.Binding.Scope = "global"
	global.Binding.Project = ""
	global.Binding.Root = root + "-global"
	applyInstall(t, e, global)
	local := installation("codex", project, root)
	applyInstall(t, e, local)
	nested := filepath.Join(project, "sub")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	resolved, err := e.Resolve(ctx, "codex", nested, "unrelated-env")
	if err != nil || resolved.Root != root || len(resolved.Diagnostics) != 1 {
		t.Fatalf("wrong project routing: %#v %v", resolved, err)
	}
	put(t, filepath.Join(nested, ".git"), []byte("gitdir: other-worktree"))
	resolved, err = e.Resolve(ctx, "codex", nested, "")
	if err != nil || resolved.Root != global.Binding.Root {
		t.Fatalf("nested repo leaked parent: %#v %v", resolved, err)
	}
	other := filepath.Join(filepath.Dir(project), "copied-worktree")
	if err := os.MkdirAll(other, 0700); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(other, ".git"), []byte("gitdir: independent"))
	resolved, err = e.Resolve(ctx, "codex", other, "")
	if err != nil || resolved.Root != global.Binding.Root {
		t.Fatal("private binding copied to other worktree")
	}
	uninstall(t, e, project, "codex")
	resolved, err = e.Resolve(ctx, "codex", project, "")
	if err != nil || resolved.Root != global.Binding.Root {
		t.Fatal("global inheritance not restored")
	}
	portable := installation("codex", project, root)
	portable.Binding.Portable = true
	applyInstall(t, e, portable)
	raw := read(t, filepath.Join(project, ".kb-agent", "setup.json"))
	if bytes.Contains(raw, []byte(e.options.Home)) || bytes.Contains(raw, []byte(root)) || bytes.Contains(raw, []byte(project)) {
		t.Fatalf("portable path leaked: %s", raw)
	}
	secondHome := filepath.Join(filepath.Dir(project), "new-home")
	if err := os.MkdirAll(secondHome, 0700); err != nil {
		t.Fatal(err)
	}
	second, err := New(Options{Home: secondHome})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err = second.Resolve(ctx, "codex", project, "")
	if err != nil || resolved.Root != root {
		t.Fatalf("portable resolution failed: %#v %v", resolved, err)
	}
	uninstall(t, e, project, "codex")
}

func TestDisabledPortableBindingStopsGlobal(t *testing.T) {
	e, project, root := fixtureEngine(t)
	global := installation("codex", project, root)
	global.Binding.Scope = "global"
	global.Binding.Project = ""
	applyInstall(t, e, global)
	b := Binding{Agent: "codex", Scope: "project", Root: "../collection", Mode: "cli", Consult: "always", Learning: "off", Hooks: "off", Portable: true, Disabled: true}
	raw, err := encode(scopeManifest{Schema: 1, Bindings: map[string]Binding{"codex": b}})
	if err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(project, ".kb-agent", "setup.json"), raw)
	r, err := e.Resolve(context.Background(), "codex", project, "")
	if err != nil || r.Status != "disabled" || !r.Disabled {
		t.Fatalf("disabled did not stop global: %#v %v", r, err)
	}
}

func TestProtectCollectionDoesNotChangeConfigOrNotes(t *testing.T) {
	_, project, _ := fixtureEngine(t)
	path := filepath.Join(project, "AGENTS.md")
	a := Artifact{Path: path, Kind: "block", Key: "policy", Content: []byte("read")}
	if err := ProtectCollection(project, []Artifact{a}, nil); err == nil {
		t.Fatal("new raw rule inside collection accepted")
	}
	original := []byte("---\ntype: Concept\ncustom: preserve\n---\n# Existing instruction note\n")
	put(t, path, original)
	if err := ProtectCollection(project, []Artifact{a}, nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(read(t, path), original) {
		t.Fatal("guard changed existing note")
	}
	for _, dir := range []string{".pi", ".omp"} {
		skill := Artifact{Path: filepath.Join(project, dir, "skills", "knowledge-base", "SKILL.md"), Kind: "file", Content: []byte("raw skill")}
		if err := ProtectCollection(project, []Artifact{skill}, nil); err == nil {
			t.Fatalf("%s raw skill accepted", dir)
		}
		if err := ProtectCollection(project, []Artifact{skill}, func(string) bool { return false }); err != nil {
			t.Fatal("explicit excluded scope rejected", err)
		}
	}
}

func TestConcurrentInstallAndLockFailure(t *testing.T) {
	e, project, root := fixtureEngine(t)
	in := installation("codex", project, root, Artifact{Path: filepath.Join(project, "AGENTS.md"), Kind: "block", Key: "policy", Content: []byte("read")})
	p1, err := e.PlanInstall(context.Background(), []Installation{in})
	if err != nil {
		t.Fatal(err)
	}
	p2, err := e.PlanInstall(context.Background(), []Installation{in})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, p := range []*Plan{p1, p2} {
		wg.Go(func() { _, err := e.Apply(context.Background(), p); results <- err })
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("concurrent plans committed %d times", success)
	}
	lock, err := e.lock(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer lock()
	p, err := e.PlanUninstall(context.Background(), Selection{Scope: "project", Project: project, Agents: []string{"codex"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Apply(context.Background(), p); err == nil {
		t.Fatal("lock failure ignored")
	}
}

func TestRecoveryUnrecordedRenameAndInterruptedRollback(t *testing.T) {
	e, project, _ := fixtureEngine(t)
	path := filepath.Join(project, "AGENTS.md")
	before := Snapshot{Exists: true, Content: []byte("before"), Mode: 0600}
	after := Snapshot{Exists: true, Content: []byte("after"), Mode: 0600}
	put(t, path, after.Content)
	if err := mkdirSafe(e.stateDir); err != nil {
		t.Fatal(err)
	}
	j := journal{Schema: 1, Changes: []Change{{Path: path, Before: before, After: after}}, Stages: map[string]string{}, Applied: 0}
	if err := e.saveJournal(j); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Recover(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if string(read(t, path)) != "before" {
		t.Fatal("unrecorded publication not rolled back")
	}
	// Recovery may itself have restored a file before its final journal removal.
	if err := e.saveJournal(j); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Recover(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Recover(context.Background(), false); err != nil {
		t.Fatal(err)
	}
}

func TestUnsupportedManifestAndCaseCollision(t *testing.T) {
	e, project, root := fixtureEngine(t)
	put(t, e.statePath(), []byte(`{"schema_version":999,"installations":{},"assets":{},"files":{}}`))
	if _, err := e.PlanInstall(context.Background(), []Installation{installation("codex", project, root)}); err == nil {
		t.Fatal("future manifest accepted")
	}
	if err := os.Remove(e.statePath()); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(project, "agents.md"), []byte("custom"))
	_, err := e.PlanInstall(context.Background(), []Installation{installation("codex", project, root, Artifact{Path: filepath.Join(project, "AGENTS.md"), Kind: "block", Key: "policy", Content: []byte("read")})})
	if err == nil {
		t.Fatal("case collision accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := e.PlanInstall(ctx, []Installation{installation("codex", project, root)}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation ignored", err)
	}
}
