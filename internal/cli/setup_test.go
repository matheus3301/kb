package cli

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/core"
	"github.com/matheus3301/kb/internal/mcpserver"
	"github.com/matheus3301/kb/internal/write"
)

type setupFixture struct {
	deps                Dependencies
	home, project, root string
}

func newSetupFixture(t *testing.T) setupFixture {
	t.Helper()
	t.Setenv("KB_ROOT", "")
	for _, key := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "PI_CONFIG_DIR", "OMP_PROFILE", "PI_PROFILE", "PI_CODING_AGENT_DIR", "OPENCODE_CONFIG", "OPENCODE_CONFIG_DIR", "OPENCODE_CONFIG_CONTENT", "COPILOT_HOME", "GEMINI_CLI_HOME", "OPENCODE_DISABLE_CLAUDE_CODE", "OPENCODE_DISABLE_CLAUDE_CODE_PROMPT"} {
		t.Setenv(key, "")
	}
	base := t.TempDir()
	base, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	f := setupFixture{home: filepath.Join(base, "home"), project: filepath.Join(base, "project"), root: filepath.Join(base, "collection")}
	for _, path := range []string{f.home, f.project} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	f.deps = Dependencies{Version: "test", SetupHome: f.home, SetupConfigHome: filepath.Join(f.home, "config"), SetupExecutable: executable, Interactive: func() bool { return false }, ResolveRoot: core.ResolveRoot,
		Open: func(root string) (mcpserver.Reader, mcpserver.Writer, error) {
			s, err := core.Open(root)
			if err != nil {
				return nil, nil, err
			}
			return s, write.New(s), nil
		},
		Init: func(ctx context.Context, root string, para bool) (mcpserver.Reader, mcpserver.Writer, error) {
			s, err := core.Init(ctx, root, para)
			if err != nil {
				return nil, nil, err
			}
			return s, write.New(s), nil
		}}
	if _, err := core.Init(context.Background(), f.root, false); err != nil {
		t.Fatal(err)
	}
	return f
}
func runSetup(t *testing.T, f setupFixture, args []string, input string) (int, map[string]any, string, string) {
	t.Helper()
	var out, stderr bytes.Buffer
	args = append([]string{"--json"}, args...)
	code := Run(context.Background(), args, strings.NewReader(input), &out, &stderr, f.deps)
	var envelope map[string]any
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid setup JSON (%d) %s: %v stderr=%s", code, out.String(), err, stderr.String())
	}
	return code, envelope, out.String(), stderr.String()
}
func (f setupFixture) install(agent string, extra ...string) []string {
	args := []string{"--root", f.root, "setup", "install", "--project", f.project, "--agent", agent, "--learning", "off", "--hooks", "off", "--yes"}
	return append(args, extra...)
}
func treeBytes(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			files[path+"/"] = d.Name()
			return nil
		}
		//nolint:gosec // The walk visits only the explicit disposable synthetic fixture root.
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = string(raw)
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return files
}
func putSetup(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSetupCLIInstallResolveCheckAndUninstall(t *testing.T) {
	f := newSetupFixture(t)
	before := treeBytes(t, f.root)
	code, e, stdout, stderr := runSetup(t, f, f.install("codex"), "")
	if code != 0 || e["ok"] != true {
		t.Fatalf("install %d %s %s", code, stdout, stderr)
	}
	if !reflect.DeepEqual(before, treeBytes(t, f.root)) {
		t.Fatal("setup changed existing collection")
	}
	data := e["data"].(map[string]any)
	if data["status"] != "activation_pending" {
		t.Fatal("native activation falsely claimed")
	}
	skill := filepath.Join(f.project, ".agents", "skills", "knowledge-base")
	for _, path := range []string{"SKILL.md", "references/retrieval.md", "references/writing.md", "references/repair.md", "references/maintenance.md", "references/results.md"} {
		if _, err := os.Stat(filepath.Join(skill, filepath.FromSlash(path))); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("KB_ROOT", filepath.Join(f.home, "unrelated"))
	code, e, _, _ = runSetup(t, f, []string{"setup", "resolve", "--agent", "codex", "--project", f.project}, "")
	if code != 0 || e["data"].(map[string]any)["root"] != f.root {
		t.Fatal("ambient root displaced binding", e)
	}
	// Missing native executable must not be reported ready, even with valid assets.
	t.Setenv("PATH", t.TempDir())
	code, e, _, _ = runSetup(t, f, []string{"setup", "check", "--agent", "codex", "--project", f.project}, "")
	if code != 1 || e["data"].(map[string]any)["ready"] != false {
		t.Fatal("missing native executable reported ready", e)
	}
	if err := os.Rename(f.root, f.root+"-moved"); err != nil {
		t.Fatal(err)
	}
	args := []string{"setup", "uninstall", "--project", f.project, "--agent", "codex", "--yes"}
	code, e, _, stderr = runSetup(t, f, args, "")
	if code != 0 || e["ok"] != true {
		t.Fatal("uninstall required missing base", stderr)
	}
	code, e, _, _ = runSetup(t, f, args, "")
	if code != 0 {
		t.Fatal("repeat uninstall failed", e)
	}
	if _, err := os.Stat(filepath.Join(f.project, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("owned rule retained")
	}
}

func TestSetupCLIPureDryRunAndValidation(t *testing.T) {
	f := newSetupFixture(t)
	beforeProject := treeBytes(t, f.project)
	beforeHome := treeBytes(t, f.home)
	beforeRoot := treeBytes(t, f.root)
	args := f.install("claude")
	args = append(args, "--dry-run")
	code, e, _, stderr := runSetup(t, f, args, "")
	if code != 0 || e["ok"] != true {
		t.Fatal(code, e, stderr)
	}
	if !reflect.DeepEqual(beforeProject, treeBytes(t, f.project)) || !reflect.DeepEqual(beforeHome, treeBytes(t, f.home)) || !reflect.DeepEqual(beforeRoot, treeBytes(t, f.root)) {
		t.Fatal("dry-run changed filesystem")
	}
	newRoot := f.root + "-new"
	code, e, _, _ = runSetup(t, f, []string{"--root", newRoot, "setup", "install", "--project", f.project, "--agent", "claude", "--learning", "on", "--create", "--para", "--dry-run"}, "")
	if code != 0 {
		t.Fatal(e)
	}
	if _, err := os.Stat(newRoot); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("dryrun created base")
	}
	for _, args := range [][]string{{"setup"}, {"setup", "install", "--project", f.project, "--agent", "codex", "--yes"}, {"--root", f.root, "setup", "install", "--project", f.project, "--global", "--agent", "codex", "--learning", "off", "--yes"}, {"--root", f.root, "setup", "install", "--project", f.project, "--agent", "unknown", "--learning", "off", "--yes"}, {"--root", f.root, "setup", "install", "--project", f.project, "--agent", "codex", "--learning", "off", "--para", "--yes"}, {"setup", "uninstall", "--project", f.project, "--yes"}, {"setup", "resolve", "--project", f.project}, {"--root", f.root, "setup", "install", "--global", "--agent", "cursor", "--learning", "off", "--yes"}, {"--root", f.root, "setup", "install", "--project", f.project, "--agent", "opencode", "--hooks", "required", "--learning", "off", "--yes"}} {
		code, e, _, _ = runSetup(t, f, args, "")
		if code != 2 || e["ok"] != false {
			t.Fatalf("incomplete/unsupported args accepted: %#v => %d %#v", args, code, e)
		}
	}
	putSetup(t, filepath.Join(f.root, "invalid.md"), "missing metadata")
	code, e, _, _ = runSetup(t, f, f.install("codex"), "")
	if code != 1 || e["error"].(map[string]any)["code"] != "NONCONFORMANT" {
		t.Fatal("invalid collection not diagnosed", e)
	}
}

func TestSetupCLICreateReplayAndModePolicyInheritance(t *testing.T) {
	f := newSetupFixture(t)
	newRoot := f.root + "-created"
	args := []string{"--root", newRoot, "setup", "install", "--project", f.project, "--agent", "claude", "--mode", "mcp", "--consult", "relevant", "--learning", "off", "--hooks", "off", "--create", "--para", "--operation-id", "create-one", "--yes"}
	code, e, _, stderr := runSetup(t, f, args, "")
	if code != 0 {
		t.Fatal(code, e, stderr)
	}
	before := treeBytes(t, newRoot)
	code, e, _, stderr = runSetup(t, f, args, "")
	if code != 0 {
		t.Fatal("create replay failed", code, e, stderr)
	}
	if !reflect.DeepEqual(before, treeBytes(t, newRoot)) {
		t.Fatal("create replay changed collection")
	}
	code, e, _, _ = runSetup(t, f, []string{"setup", "install", "--project", f.project, "--agent", "claude", "--yes"}, "")
	if code != 0 {
		t.Fatal(e)
	}
	binding := e["data"].(map[string]any)["bindings"].([]any)[0].(map[string]any)
	if binding["mode"] != "mcp" || binding["consult"] != "relevant" || binding["hooks"] != "off" || binding["root"] != newRoot || binding["learning"] != "off" {
		t.Fatal("omitted options reset prior choices", binding)
	}
	changed := append(append([]string{}, args...), "--learning", "on")
	code, e, _, _ = runSetup(t, f, changed, "")
	if code != 4 || e["error"].(map[string]any)["code"] != "IDEMPOTENCY_CONFLICT" {
		t.Fatal("changed operation id accepted", e)
	}
	args = []string{"--root", newRoot, "setup", "install", "--project", f.project, "--agent", "claude", "--mode", "cli", "--learning", "off", "--hooks", "off", "--yes"}
	code, e, _, _ = runSetup(t, f, args, "")
	if code != 0 {
		t.Fatal(e)
	}
	if _, err := os.Stat(filepath.Join(f.project, ".claude", "skills", "knowledge-base", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.project, ".mcp.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unused owned MCP config remained")
	}
}

func TestSetupCLIRedactionAndCollectionRootConflict(t *testing.T) {
	f := newSetupFixture(t)
	config := filepath.Join(f.project, ".mcp.json")
	original := `{"mcpServers":{"foreign":{"command":"other","env":{"TOKEN":"SYNTHETIC-NOT-A-SECRET"}}}}`
	putSetup(t, config, original)
	for _, dry := range []bool{true, false} {
		args := f.install("claude", "--mode", "mcp")
		if dry {
			args = append(args, "--dry-run")
		}
		code, e, out, stderr := runSetup(t, f, args, "")
		if code != 0 {
			t.Fatal(e, stderr)
		}
		if strings.Contains(out, "SYNTHETIC-NOT-A-SECRET") || strings.Contains(stderr, "SYNTHETIC-NOT-A-SECRET") || strings.Contains(out, base64.StdEncoding.EncodeToString([]byte(original))) || strings.Contains(out, "c2NoZW1hX3ZlcnNpb24") {
			t.Fatal("setup output leaked before-images/assets")
		}
	}
	f.project = f.root
	code, e, _, _ := runSetup(t, f, f.install("codex"), "")
	if code != 4 {
		t.Fatal("raw AGENTS would invalidate collection", code, e)
	}
	putSetup(t, filepath.Join(f.root, "AGENTS.md"), "---\ntype: Concept\ncustom: preserve\n---\n# Existing note\n")
	code, e, _, _ = runSetup(t, f, f.install("codex"), "")
	if code != 0 {
		t.Fatal("existing OKF AGENTS note should be preserved", e)
	}
	store, err := core.Open(f.root)
	if err != nil {
		t.Fatal(err)
	}
	report, err := store.InspectSetup(context.Background())
	if err != nil || !report.Conformant {
		t.Fatal("integration broke collection", report, err)
	}
	for _, agent := range []string{"pi", "omp", "gemini"} {
		code, e, _, _ = runSetup(t, f, f.install(agent), "")
		if code != 4 {
			t.Fatalf("%s new raw markdown not rejected: %d %#v", agent, code, e)
		}
	}
}

func TestSetupCLIWizardReviewAndHumanOutput(t *testing.T) {
	f := newSetupFixture(t)
	f.deps.Interactive = func() bool { return true }
	var out, stderr bytes.Buffer
	input := "codex\n" + f.root + "\noff\nyes\n"
	code := Run(context.Background(), []string{"setup", "--project", f.project}, strings.NewReader(input), &out, &stderr, f.deps)
	if code != 0 {
		t.Fatal("buffered confirmation lost", code, out.String(), stderr.String())
	}
	if strings.HasPrefix(out.String(), "{") || !strings.Contains(out.String(), "activation_pending") {
		t.Fatal("human setup was not concise", out.String())
	}
	if !strings.Contains(stderr.String(), filepath.Join(f.project, "AGENTS.md")) || !strings.Contains(stderr.String(), filepath.Join(f.project, ".codex", "hooks.json")) {
		t.Fatal("review omitted actual targets or activation diagnostics", stderr.String())
	}
	// A requested policy change must not take the already-configured shortcut.
	code, e, _, _ := runSetup(t, f, []string{"setup", "--project", f.project, "--agent", "codex", "--consult", "relevant", "--yes"}, "")
	if code != 0 {
		t.Fatal(e)
	}
	binding := e["data"].(map[string]any)["bindings"].([]any)[0].(map[string]any)
	if binding["consult"] != "relevant" {
		t.Fatal("explicit policy change ignored")
	}
}

func TestSetupCLIHookNativeOutputAndReadOnlyPolicy(t *testing.T) {
	f := newSetupFixture(t)
	putSetup(t, filepath.Join(f.root, "alpha.md"), "---\ntype: Concept\ntitle: Alpha\n---\nAlpha is a synthetic routing marker.\n")
	code, e, _, _ := runSetup(t, f, f.install("claude", "--hooks", "auto"), "")
	if code != 0 {
		t.Fatal(e)
	}
	before := treeBytes(t, f.root)
	var out, stderr bytes.Buffer
	payload := fmt.Sprintf(`{"cwd":%q,"hook_event_name":"UserPromptSubmit","prompt":"alpha","session_id":"synthetic"}`, f.project)
	code = Run(context.Background(), []string{"setup", "hook", "--agent", "claude", "--event", "UserPromptSubmit"}, strings.NewReader(payload), &out, &stderr, f.deps)
	if code != 0 || strings.Contains(out.String(), "schema_version") || !strings.Contains(out.String(), "alpha.md") {
		t.Fatal("hook did not return native retrieval", code, out.String(), stderr.String())
	}
	if !reflect.DeepEqual(before, treeBytes(t, f.root)) {
		t.Fatal("hook wrote collection")
	}
	out.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"setup", "hook", "--agent", "claude", "--event", "UserPromptSubmit"}, strings.NewReader(`{"cwd":false}`), &out, &stderr, f.deps)
	if code != 2 || !json.Valid(out.Bytes()) || strings.Contains(out.String(), "schema_version") {
		t.Fatal("hook error corrupted native protocol", code, out.String())
	}
}
