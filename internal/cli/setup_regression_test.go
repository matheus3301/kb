package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func nativeFixture(t *testing.T, f *setupFixture, agent string) {
	t.Helper()
	bin := filepath.Join(f.home, "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"kb", agent} {
		path := filepath.Join(bin, name)
		putSetup(t, path, "#!/bin/sh\nexit 0\n")
		//nolint:gosec // Disposable executable fixture requires owner execute permission for PATH discovery.
		if err := os.Chmod(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
}
func TestSetupSharedMCPEntryRemovalRetainsOtherConsumer(t *testing.T) {
	f := newSetupFixture(t)
	args := f.install("claude", "--agent", "copilot", "--mode", "mcp")
	code, e, _, stderr := runSetup(t, f, args, "")
	if code != 0 {
		t.Fatal("identical shared MCP entry rejected", e, stderr)
	}
	path := filepath.Join(f.project, ".mcp.json")
	//nolint:gosec // Explicit synthetic MCP configuration below this test's disposable project.
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	code, e, _, _ = runSetup(t, f, []string{"setup", "uninstall", "--project", f.project, "--agent", "claude", "--yes"}, "")
	if code != 0 {
		t.Fatal(e)
	}
	//nolint:gosec // Reads the same synthetic shared entry after one consumer uninstalls.
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("removing Claude broke Copilot MCP", err)
	}
	code, e, _, _ = runSetup(t, f, []string{"setup", "uninstall", "--project", f.project, "--agent", "copilot", "--yes"}, "")
	if code != 0 {
		t.Fatal(e)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("last shared MCP reference retained config")
	}
}

func TestSetupBareRerunChecksArtifactRootAndCommand(t *testing.T) {
	for _, broken := range []string{"skill", "root", "command", "shadow"} {
		t.Run(broken, func(t *testing.T) {
			f := newSetupFixture(t)
			nativeFixture(t, &f, "codex")
			command := filepath.Join(f.home, "kb-runtime")
			putSetup(t, command, "#!/bin/sh\nexit 0\n")
			//nolint:gosec // Disposable recorded-command fixture must initially be executable to test later removal.
			if err := os.Chmod(command, 0700); err != nil {
				t.Fatal(err)
			}
			f.deps.SetupExecutable = command
			code, e, _, stderr := runSetup(t, f, f.install("codex"), "")
			if code != 0 {
				t.Fatal(e, stderr)
			}
			t.Chdir(f.project)
			code, e, _, _ = runSetup(t, f, []string{"setup"}, "")
			if code != 0 || e["data"].(map[string]any)["status"] != "already_configured" {
				t.Fatal("valid bare rerun failed", e)
			}
			switch broken {
			case "skill":
				if err := os.Remove(filepath.Join(f.project, ".agents", "skills", "knowledge-base", "SKILL.md")); err != nil {
					t.Fatal(err)
				}
			case "root":
				if err := os.Rename(f.root, f.root+"-moved"); err != nil {
					t.Fatal(err)
				}
			case "command":
				if err := os.Remove(command); err != nil {
					t.Fatal(err)
				}
			case "shadow":
				putSetup(t, filepath.Join(f.project, "AGENTS.override.md"), "New higher-precedence instructions\n")
			}
			code, e, _, _ = runSetup(t, f, []string{"setup"}, "")
			if code != 1 || e["data"].(map[string]any)["ready"] != false {
				t.Fatal("bare rerun hid missing/conflicting prerequisite", broken, code, e)
			}
		})
	}
}

func TestSetupProfileChangesAndExistingFallbackInstructions(t *testing.T) {
	f := newSetupFixture(t)
	nativeFixture(t, &f, "codex")
	args := []string{"--root", f.root, "setup", "install", "--global", "--agent", "codex", "--learning", "off", "--hooks", "off", "--yes"}
	code, e, _, _ := runSetup(t, f, args, "")
	if code != 0 {
		t.Fatal(e)
	}
	t.Setenv("CODEX_HOME", filepath.Join(f.home, "different-profile"))
	code, e, _, _ = runSetup(t, f, []string{"setup", "check", "--global", "--agent", "codex"}, "")
	if code != 1 || e["data"].(map[string]any)["ready"] != false {
		t.Fatal("changed native profile reported ready", e)
	}
	t.Setenv("CODEX_HOME", "")
	fallback := filepath.Join(f.project, "CLAUDE.md")
	original := "# Shared project instructions\nKeep this original fallback.\n"
	putSetup(t, fallback, original)
	for _, agent := range []string{"pi", "opencode"} {
		code, e, _, _ = runSetup(t, f, f.install(agent), "")
		if code != 0 {
			t.Fatal(e)
		}
		if _, err := os.Stat(filepath.Join(f.project, "AGENTS.md")); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("new preferred file shadowed existing fallback")
		}
		//nolint:gosec // Reads only the known fallback-instruction fixture in this test's project.
		raw, err := os.ReadFile(fallback)
		if err != nil || !strings.HasPrefix(string(raw), original) {
			t.Fatal("fallback instructions overwritten", err)
		}
	}
}

func TestSetupPortableAndTildeCreation(t *testing.T) {
	f := newSetupFixture(t)
	nativeFixture(t, &f, "omp")
	args := f.install("omp", "--portable", "--hooks", "auto")
	code, e, _, _ := runSetup(t, f, args, "")
	if code != 0 {
		t.Fatal(e)
	}
	for path, raw := range treeBytes(t, f.project) {
		if strings.HasSuffix(path, "/") {
			continue
		}
		for _, private := range []string{f.home, f.root, f.project, f.deps.SetupExecutable} {
			if strings.Contains(raw, private) {
				t.Fatalf("portable generated file %s leaked personal path", path)
			}
		}
	}
	other := f
	other.home = filepath.Join(filepath.Dir(f.home), "other-home")
	if err := os.MkdirAll(other.home, 0700); err != nil {
		t.Fatal(err)
	}
	other.deps.SetupHome = other.home
	other.deps.SetupConfigHome = filepath.Join(other.home, "config")
	code, e, _, _ = runSetup(t, other, []string{"setup", "resolve", "--project", f.project, "--agent", "omp"}, "")
	if code != 0 || e["data"].(map[string]any)["root"] != f.root {
		t.Fatal("second-user portable binding failed", e)
	}
	code, e, _, _ = runSetup(t, f, []string{"--root", "~/new-collection", "setup", "install", "--project", f.project, "--agent", "claude", "--learning", "off", "--hooks", "off", "--create", "--yes"}, "")
	if code != 0 {
		t.Fatal(e)
	}
	newRoot := filepath.Join(f.home, "new-collection")
	if _, err := os.Stat(filepath.Join(newRoot, ".kb", "config.toml")); err != nil {
		t.Fatal("tilde creation targeted literal cwd directory", err)
	}
	// A successful explicit create can repeat without an operation id too.
	before := treeBytes(t, newRoot)
	code, e, _, _ = runSetup(t, f, []string{"--root", "~/new-collection", "setup", "install", "--project", f.project, "--agent", "claude", "--learning", "off", "--hooks", "off", "--create", "--yes"}, "")
	if code != 0 || !reflect.DeepEqual(before, treeBytes(t, newRoot)) {
		t.Fatal("unchanged create rerun failed or mutated collection", e)
	}
	if _, err := expandSetupPath("~another/private", f.home); err == nil {
		t.Fatal("unsupported home alias accepted")
	}
}

func TestSetupHookMCPUsesNativeContextWithoutSkill(t *testing.T) {
	f := newSetupFixture(t)
	putSetup(t, filepath.Join(f.root, "alpha.md"), "---\ntype: Concept\n---\nAlpha is a synthetic MCP hook marker.\n")
	code, e, _, _ := runSetup(t, f, f.install("claude", "--mode", "mcp", "--hooks", "auto"), "")
	if code != 0 {
		t.Fatal(e)
	}
	var out, stderr bytes.Buffer
	input := `{"cwd":"` + f.project + `","hook_event_name":"UserPromptSubmit","prompt":"alpha"}`
	code = Run(context.Background(), []string{"setup", "hook", "--agent", "claude", "--event", "UserPromptSubmit"}, strings.NewReader(input), &out, &stderr, f.deps)
	if code != 0 || !strings.Contains(out.String(), "alpha.md") || strings.Contains(out.String(), "--read-only") || strings.Contains(out.String(), "SKILL.md") {
		t.Fatal("MCP hook was reminder-only or required CLI/skill", code, out.String(), stderr.String())
	}
}
