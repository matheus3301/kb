package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSetupRejectsOversizedGeneratedStateBeforeWrites(t *testing.T) {
	f := newSetupFixture(t)
	original, err := json.Marshal(map[string]string{"foreign": strings.Repeat("x", 3_300_000)})
	if err != nil {
		t.Fatal(err)
	}
	putSetup(t, filepath.Join(f.home, ".claude.json"), string(original))
	beforeHome := treeBytes(t, f.home)
	beforeRoot := treeBytes(t, f.root)
	args := []string{"--root", f.root, "setup", "install", "--global", "--agent", "claude", "--mode", "mcp", "--hooks", "off", "--learning", "off", "--yes"}
	for _, dry := range []bool{true, false} {
		request := append([]string(nil), args...)
		if dry {
			request = append(request, "--dry-run")
		}
		code, result, _, stderr := runSetup(t, f, request, "")
		if code != 2 || result["error"].(map[string]any)["code"] != "INVALID_INPUT" {
			t.Fatalf("oversized generated metadata accepted (dry=%t): code=%d result=%v stderr=%s", dry, code, result, stderr)
		}
		if !reflect.DeepEqual(beforeHome, treeBytes(t, f.home)) || !reflect.DeepEqual(beforeRoot, treeBytes(t, f.root)) {
			t.Fatalf("size rejection changed host or collection (dry=%t)", dry)
		}
	}
	code, result, _, _ := runSetup(t, f, []string{"setup", "recover"}, "")
	if code != 0 || result["data"].(map[string]any)["status"] != "no_pending_transaction" {
		t.Fatal("size rejection left unrecoverable state", code, result)
	}
	code, result, _, _ = runSetup(t, f, f.install("codex"), "")
	if code != 0 {
		t.Fatal("size rejection blocked a subsequent valid installation", code, result)
	}
}

func TestSetupInheritedHookHonorsProjectHooksOff(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "gemini", "pi", "omp"} {
		t.Run(agent, func(t *testing.T) {
			f := newSetupFixture(t)
			code, result, _, _ := runSetup(t, f, []string{"--root", f.root, "setup", "install", "--global", "--agent", agent, "--learning", "off", "--hooks", "auto", "--yes"}, "")
			if code != 0 {
				t.Fatal("global hook setup failed", result)
			}
			code, result, _, _ = runSetup(t, f, f.install(agent), "")
			if code != 0 {
				t.Fatal("project rules-only setup failed", result)
			}
			code, result, _, _ = runSetup(t, f, []string{"setup", "resolve", "--project", f.project, "--agent", agent}, "")
			if code != 0 || result["data"].(map[string]any)["hooks"] != "off" {
				t.Fatal("project hook policy did not override global", result)
			}
			// An opted-out dispatcher must succeed without opening even a missing base.
			if err := os.Rename(f.root, f.root+"-unavailable"); err != nil {
				t.Fatal(err)
			}
			events := []string{"SessionStart", "UserPromptSubmit"}
			if agent == "gemini" {
				events[1] = "BeforeAgent"
			}
			for _, event := range events {
				input, err := json.Marshal(map[string]string{"cwd": f.project, "hook_event_name": event, "prompt": "synthetic knowledge"})
				if err != nil {
					t.Fatal(err)
				}
				code, result, _, stderr := runSetup(t, f, []string{"setup", "hook", "--agent", agent, "--event", event}, string(input))
				if code != 0 {
					t.Fatalf("opted-out inherited hook accessed the base: %d %v %s", code, result, stderr)
				}
				if agent == "pi" || agent == "omp" {
					if result["context"] != "" {
						t.Fatal("opted-out hook emitted context", result)
					}
				} else if result["hookSpecificOutput"].(map[string]any)["additionalContext"] != "" {
					t.Fatal("opted-out hook emitted context", result)
				}
			}
		})
	}
}

func TestSetupPreservesScopedInstructionLineEndings(t *testing.T) {
	for _, format := range []struct {
		name, newline string
	}{
		{"lf", "\n"},
		{"crlf", "\r\n"},
	} {
		t.Run(format.name, func(t *testing.T) {
			f := newSetupFixture(t)
			f.project = f.root
			original := strings.Join([]string{"---", "type: Reference", "title: Instructions", "---", "", "# Instructions", "Keep the existing policy."}, format.newline)
			path := filepath.Join(f.root, "AGENTS.md")
			putSetup(t, path, original)
			before := treeBytes(t, f.root)
			code, result, _, stderr := runSetup(t, f, f.install("codex", "--dry-run"), "")
			if code != 0 || !reflect.DeepEqual(before, treeBytes(t, f.root)) {
				t.Fatal("conformant instruction preview failed or changed collection", code, result, stderr)
			}
			code, result, _, stderr = runSetup(t, f, f.install("codex"), "")
			if code != 0 {
				t.Fatal("conformant instruction installation failed", result, stderr)
			}
			//nolint:gosec // Reads the known instruction fixture within this test's collection.
			installed, err := os.ReadFile(path)
			if err != nil || !bytes.HasPrefix(installed, []byte(original+format.newline+"<!-- kb:")) {
				t.Fatal("setup changed original bytes or marker line endings", err)
			}
			code, result, _, _ = runSetup(t, f, []string{"setup", "uninstall", "--project", f.project, "--agent", "codex", "--yes"}, "")
			if code != 0 {
				t.Fatal("instruction uninstall failed", result)
			}
			//nolint:gosec // Confirms this disposable instruction fixture is restored byte-for-byte.
			restored, err := os.ReadFile(path)
			if err != nil || string(restored) != original {
				t.Fatal("uninstall did not restore instruction bytes", err)
			}
		})
	}
}
