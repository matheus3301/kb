package setup

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureEngine(t *testing.T) (*Engine, string, string) {
	t.Helper()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	for _, p := range []string{home, project} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	e, err := New(Options{Home: home, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return e, project, filepath.Join(root, "collection")
}
func installation(agent, project, root string, artifacts ...Artifact) Installation {
	return Installation{Binding: Binding{Agent: agent, Scope: "project", Project: project, Root: root, Mode: "cli", Consult: "always", Learning: "on", Hooks: "off"}, Artifacts: artifacts}
}
func put(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func read(t *testing.T, path string) []byte {
	t.Helper()
	//nolint:gosec // Reads only explicit files below this test's disposable fixture roots.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func applyInstall(t *testing.T, e *Engine, ins ...Installation) *Plan {
	t.Helper()
	p, err := e.PlanInstall(context.Background(), ins)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	return p
}
func uninstall(t *testing.T, e *Engine, project string, agents ...string) {
	t.Helper()
	p, err := e.PlanUninstall(context.Background(), Selection{Scope: "project", Project: project, Agents: agents})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
}

func TestLifecycleSharedBorrowedAndPristine(t *testing.T) {
	e, project, root := fixtureEngine(t)
	ctx := context.Background()
	rule := filepath.Join(project, "AGENTS.md")
	skill := filepath.Join(project, ".agents", "skills", "knowledge-base", "SKILL.md")
	original := []byte("\xef\xbb\xbf# Existing\r\nKeep this exact, without final newline")
	put(t, rule, original)
	borrowed := []byte("complete bundled skill")
	put(t, skill, borrowed)
	artifacts := []Artifact{{Path: rule, Kind: "block", Key: "kb-policy", Content: []byte("Consult and learn."), Shared: true}, {Path: skill, Kind: "file", Content: borrowed, Shared: true}}
	first := installation("codex", project, root, artifacts...)
	second := installation("opencode", project, root, artifacts...)
	applyInstall(t, e, first, second)
	configured := read(t, rule)
	if !bytes.HasPrefix(configured, original) || !bytes.Contains(configured, []byte("\r\n<!-- kb:")) {
		t.Fatal("original bytes/newline changed")
	}
	p, err := e.PlanInstall(ctx, []Installation{first, second})
	if err != nil || len(p.Changes) != 0 {
		t.Fatalf("not idempotent: %v %#v", err, p)
	}
	uninstall(t, e, project, "codex")
	if !bytes.Equal(read(t, rule), configured) {
		t.Fatal("shared rule removed")
	}
	uninstall(t, e, project, "opencode")
	if !bytes.Equal(read(t, rule), original) || !bytes.Equal(read(t, skill), borrowed) {
		t.Fatal("pristine/borrowed bytes lost")
	}
	uninstall(t, e, project, "opencode")
}

func TestPreflightPureAndAllOrNothing(t *testing.T) {
	e, project, root := fixtureEngine(t)
	rule := filepath.Join(project, "AGENTS.md")
	config := filepath.Join(project, "config.json")
	put(t, config, []byte(`{"broken":`))
	_, err := e.PlanInstall(context.Background(), []Installation{installation("codex", project, root, Artifact{Path: rule, Kind: "block", Key: "policy", Content: []byte("Read sources")}, Artifact{Path: config, Kind: "json", Key: "mcpServers.knowledge-base", Content: []byte(`{"command":"kb"}`)})})
	if err == nil {
		t.Fatal("malformed config accepted")
	}
	if _, err := os.Stat(rule); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("preflight wrote rule")
	}
	if _, err := os.Stat(e.options.ConfigHome); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("preflight created state")
	}
	put(t, config, []byte("{}"))
	p, err := e.PlanInstall(context.Background(), []Installation{installation("codex", project, root, Artifact{Path: rule, Kind: "block", Key: "policy", Content: []byte("Read sources")})})
	if err != nil {
		t.Fatal(err)
	}
	_ = p.Result(true)
	if _, err := os.Stat(e.options.ConfigHome); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("dry-run created state")
	}
}

func TestJSONAndTOMLPreserveForeignBytes(t *testing.T) {
	for _, kind := range []string{"json", "jsonc", "toml"} {
		t.Run(kind, func(t *testing.T) {
			e, project, root := fixtureEngine(t)
			path := filepath.Join(project, "settings."+kind)
			var original, content []byte
			key := "mcpServers.knowledge-base"
			switch kind {
			case "json":
				original = []byte("{\r\n  \"foreign\": {\"decimal\": 1.000, \"token\": \"FAKE-SECRET\"}\r\n}\r\n")
				content = []byte(`{"command":"kb","args":["mcp"]}`)
			case "jsonc":
				original = []byte("{\n// keep comment\n\"foreign\": \"https://example.com/*literal*/\", /* comma, comment */\n}\n")
				content = []byte(`{"command":"kb"}`)
			case "toml":
				key = "mcp_servers.knowledge-base"
				original = []byte("# user comment\r\n[foreign]\r\nnumber = 1.000 # exact\r\ntext = '''\r\n[not_a_table]\r\n'''\r\n")
				content = []byte("command = \"kb\"\nargs = [\"mcp\"]\n")
			}
			put(t, path, original)
			in := installation("codex", project, root, Artifact{Path: path, Kind: kind, Key: key, Content: content})
			p := applyInstall(t, e, in)
			if !bytes.Contains(read(t, path), bytes.TrimSuffix(original, []byte("}\r\n"))) && kind == "json" {
				t.Fatal("foreign JSON changed")
			}
			result, err := json.Marshal(p.Result(true))
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Contains(result, []byte("FAKE-SECRET")) || bytes.Contains(result, []byte(base64.StdEncoding.EncodeToString(original))) {
				t.Fatal("result leaked unrelated config")
			}
			applyInstall(t, e, in)
			uninstall(t, e, project, "codex")
			if !bytes.Equal(read(t, path), original) {
				t.Fatalf("roundtrip changed %q", read(t, path))
			}
		})
	}
}

func TestBorrowedDriftAndUpgrade(t *testing.T) {
	for _, kind := range []string{"file", "block", "json", "toml"} {
		t.Run(kind, func(t *testing.T) {
			e, project, root := fixtureEngine(t)
			path := filepath.Join(project, "asset")
			a := Artifact{Path: path, Kind: kind, Key: "kb", Content: []byte("policy")}
			var existing []byte
			switch kind {
			case "file":
				existing = a.Content
			case "block":
				existing, _ = blockText(a, nil)
			case "json":
				a.Content = []byte(`{"policy":"read"}`)
				existing = []byte(`{"kb":{"policy":"read"}}`)
			case "toml":
				a.Content = []byte("value = \"policy\"\n")
				existing = []byte("[kb]\nvalue = \"policy\"\n")
			}
			put(t, path, existing)
			in := installation("codex", project, root, a)
			applyInstall(t, e, in)
			put(t, path, []byte(strings.ReplaceAll(string(existing), "policy", "custom")))
			if _, err := e.PlanInstall(context.Background(), []Installation{in}); err == nil {
				t.Fatal("borrowed drift accepted")
			}
			check, err := e.Check(context.Background(), "codex", project, "")
			if err != nil || check.Configured {
				t.Fatalf("borrowed drift not diagnosed: %#v %v", check, err)
			}
			uninstall(t, e, project, "codex")
			if !bytes.Contains(read(t, path), []byte("custom")) {
				t.Fatal("borrowed customization removed")
			}
			put(t, path, existing)
			applyInstall(t, e, in)
			in.Artifacts[0].Content = []byte("upgraded")
			if _, err := e.PlanInstall(context.Background(), []Installation{in}); err == nil {
				t.Fatal("borrowed old asset accepted as upgraded")
			}
		})
	}
}

func TestOwnedConflictAndOutsideEdit(t *testing.T) {
	e, project, root := fixtureEngine(t)
	path := filepath.Join(project, "AGENTS.md")
	put(t, path, []byte("# user\n"))
	in := installation("codex", project, root, Artifact{Path: path, Kind: "block", Key: "policy", Content: []byte("owned policy")})
	applyInstall(t, e, in)
	put(t, path, append([]byte("User added before\n"), read(t, path)...))
	uninstall(t, e, project, "codex")
	after := read(t, path)
	if !bytes.Contains(after, []byte("User added before")) || bytes.Contains(after, []byte("owned policy")) {
		t.Fatal("outside edit lost or block retained")
	}
	applyInstall(t, e, in)
	put(t, path, bytes.ReplaceAll(read(t, path), []byte("owned policy"), []byte("customized policy")))
	if _, err := e.PlanUninstall(context.Background(), Selection{Scope: "project", Project: project, Agents: []string{"codex"}}); err == nil {
		t.Fatal("custom owned block overwritten")
	}
}

func TestPreconditionRaceAndSymlinkHardlink(t *testing.T) {
	e, project, root := fixtureEngine(t)
	path := filepath.Join(project, "AGENTS.md")
	in := installation("codex", project, root, Artifact{Path: path, Kind: "block", Key: "policy", Content: []byte("read")})
	p, err := e.PlanInstall(context.Background(), []Installation{in})
	if err != nil {
		t.Fatal(err)
	}
	put(t, path, []byte("external edit"))
	if _, err := e.Apply(context.Background(), p); err == nil {
		t.Fatal("lost update")
	}
	if string(read(t, path)) != "external edit" {
		t.Fatal("external edit overwritten")
	}
	target := filepath.Join(project, "target")
	put(t, target, []byte("user"))
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := e.PlanInstall(context.Background(), []Installation{in}); err == nil {
		t.Fatal("symlink followed")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := e.PlanInstall(context.Background(), []Installation{in}); err == nil {
		t.Fatal("hardlink accepted")
	}
}

func TestRollbackAtEveryPublicationBoundary(t *testing.T) {
	e, project, root := fixtureEngine(t)
	a := Artifact{Path: filepath.Join(project, "AGENTS.md"), Kind: "block", Key: "policy", Content: []byte("read")}
	in := installation("codex", project, root, a)
	p, err := e.PlanInstall(context.Background(), []Installation{in})
	if err != nil {
		t.Fatal(err)
	}
	boundaries := []string{"prepared"}
	for i := range p.Changes {
		boundaries = append(boundaries, fmt.Sprintf("published:%d", i))
	}
	for _, boundary := range boundaries {
		t.Run(boundary, func(t *testing.T) {
			e.options.Failpoint = func(got string) error {
				if got == boundary {
					return errors.New("injected publication failure")
				}
				return nil
			}
			if _, err := e.Apply(context.Background(), p); err == nil {
				t.Fatal("injected error lost")
			}
			if _, err := os.Stat(a.Path); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("rollback retained new rule")
			}
			if _, err := os.Stat(e.pendingPath()); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("rollback did not clear journal")
			}
		})
	}
	e.options.Failpoint = nil
	applyInstall(t, e, in)
}

func TestRecoveryCommittedAndExternalEdit(t *testing.T) {
	e, project, root := fixtureEngine(t)
	path := filepath.Join(project, "AGENTS.md")
	in := installation("codex", project, root, Artifact{Path: path, Kind: "block", Key: "policy", Content: []byte("read")})
	e.options.Failpoint = func(boundary string) error {
		if boundary == "committed" {
			return errors.New("reply lost")
		}
		return nil
	}
	p, err := e.PlanInstall(context.Background(), []Installation{in})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Apply(context.Background(), p); err == nil {
		t.Fatal("lost commit reply not reported")
	}
	if _, err := e.Resolve(context.Background(), "codex", project, ""); err == nil {
		t.Fatal("pending state ignored")
	}
	if _, err := e.Recover(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(e.pendingPath()); err != nil {
		t.Fatal("dry recovery mutated")
	}
	if _, err := e.Recover(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Resolve(context.Background(), "codex", project, ""); err != nil {
		t.Fatal(err)
	}
	before := Snapshot{Exists: true, Content: []byte("before"), Mode: 0600}
	after := Snapshot{Exists: true, Content: []byte("after"), Mode: 0600}
	put(t, path, []byte("external"))
	j := journal{Schema: 1, Changes: []Change{{Path: path, Before: before, After: after}}, Stages: map[string]string{}}
	if err := e.saveJournal(j); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Recover(context.Background(), false); err == nil {
		t.Fatal("external edit recovery clobber")
	}
	if string(read(t, path)) != "external" {
		t.Fatal("external edit lost")
	}
}

func TestJSONCAndMarkerRejection(t *testing.T) {
	for _, raw := range []string{`{"a":1,"a":2}`, `{"a":{"b":1,"b":2}}`, `{/* open`, `{"a":1,,}`} {
		if _, err := parseJSONFormat([]byte(raw), "jsonc"); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	a := Artifact{Kind: "jsonc", Key: "mcp.kb", Content: []byte(`{"command":"kb"}`)}
	raw := []byte("{ // comment with comma,\n\"s\": \"//not a comment /* either */\", /* trailing */\n}\n")
	out, _, _, err := editArtifact(raw, a, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseJSONFormat(out, "jsonc"); err != nil {
		t.Fatalf("invalid edited JSONC %s: %v", out, err)
	}
	if !bytes.Contains(out, []byte("// comment with comma,")) || !bytes.Contains(out, []byte(`"//not a comment /* either */"`)) {
		t.Fatal("comments/string changed")
	}
	for _, raw := range []string{"<!-- kb:x:v1 begin -->\norph", "<!-- kb:x:v1 end -->\n<!-- kb:x:v1 begin -->", "<!-- kb:x:v1 begin -->\n<!-- kb:x:v1 begin -->\n<!-- kb:x:v1 end -->"} {
		if _, _, err := blockSpan([]byte(raw), "x"); err == nil {
			t.Fatal("malformed markers accepted")
		}
	}
}
