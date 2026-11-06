package setup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOwnedHookUpgradePreservesForeignHandler(t *testing.T) {
	e, project, root := fixtureEngine(t)
	path := filepath.Join(project, "settings.json")
	original := []byte("{\n  \"foreign\": 1.000,\n  \"hooks\": {\"UserPromptSubmit\": [{\"hooks\":[{\"command\":\"foreign-command\"}]}]}\n}\n")
	put(t, path, original)
	a := Artifact{Path: path, Kind: "json-array", Key: "hooks.UserPromptSubmit", Content: []byte(`{"hooks":[{"command":"kb-v1 setup hook","timeout":5}]}`)}
	in := installation("claude", project, root, a)
	applyInstall(t, e, in)
	in.Artifacts[0].Content = []byte(`{"hooks":[{"command":"kb-v2 setup hook","timeout":4}]}`)
	applyInstall(t, e, in)
	configured := read(t, path)
	if !bytes.Contains(configured, []byte("foreign-command")) || !bytes.Contains(configured, []byte("kb-v2")) || bytes.Contains(configured, []byte("kb-v1")) {
		t.Fatal("upgrade lost foreign or retained old hook")
	}
	p, err := e.PlanInstall(context.Background(), []Installation{in})
	if err != nil || len(p.Changes) != 0 {
		t.Fatal("upgraded hook repeat was not a no-op", err)
	}
	uninstall(t, e, project, "claude")
	if !bytes.Equal(read(t, path), original) {
		t.Fatal("hook uninstall lost original bytes")
	}
}

func TestTOMLEscapedMultilineHeadersAndUpgrade(t *testing.T) {
	e, project, root := fixtureEngine(t)
	path := filepath.Join(project, "config.toml")
	original := []byte("[foreign]\ntext = \"\"\"\nAn escaped quote sequence: \\\"\"\"\n[mcp_servers.knowledge-base]\nStill inside the string.\n\"\"\"\nlast = 1.000 # preserve\n")
	if _, _, err := tomlTables(original); err != nil {
		t.Fatal("invalid synthetic TOML fixture", err)
	}
	put(t, path, original)
	a := Artifact{Path: path, Kind: "toml", Key: "mcp_servers.knowledge-base", Content: []byte("command = \"kb-v1\"\n")}
	in := installation("codex", project, root, a)
	applyInstall(t, e, in)
	in.Artifacts[0].Content = []byte("command = \"kb-v2\"\n")
	applyInstall(t, e, in)
	if !bytes.HasPrefix(read(t, path), original) {
		t.Fatal("multiline false table boundary damaged foreign data")
	}
	uninstall(t, e, project, "codex")
	if !bytes.Equal(read(t, path), original) {
		t.Fatal("TOML upgrade/uninstall changed original")
	}
}

func TestPrivacyNoGitAndFatalRepositoryCommands(t *testing.T) {
	for _, code := range []string{"1", "0", "128", "revparse"} {
		t.Run(code, func(t *testing.T) {
			e, project, root := fixtureEngine(t)
			a := Artifact{Path: filepath.Join(project, "personal.json"), Kind: "json", Key: "kb", Content: []byte(`{"root":"` + root + `"}`)}
			b := installation("claude", project, root, a).Binding
			for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE"} {
				t.Setenv(key, "")
			}
			bin := filepath.Join(e.options.Home, "bin")
			if err := os.MkdirAll(bin, 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin)
			plain, _, err := PrivacyArtifacts(context.Background(), b, []Artifact{a}, e.options.Home)
			if err != nil || len(plain) != 1 || plain[0].Path != filepath.Join(project, ".gitignore") {
				t.Fatal("plain folder unnecessarily required Git", plain, err)
			}
			if err := os.Mkdir(filepath.Join(project, ".git"), 0700); err != nil {
				t.Fatal(err)
			}
			script := "#!/bin/sh\nif [ \"$3\" = ls-files ]; then exit " + code + "; fi\nif [ \"$4\" = --show-toplevel ]; then printf '%s\\n' \"$2\"; else printf '%s/.git/info/exclude\\n' \"$2\"; fi\n"
			if code == "revparse" {
				script = "#!/bin/sh\nexit 128\n"
			}
			stub := filepath.Join(bin, "git")
			put(t, stub, []byte(script))
			//nolint:gosec // Synthetic Git stub needs owner execute permission to exercise real process exit handling.
			if err := os.Chmod(stub, 0700); err != nil {
				t.Fatal(err)
			}
			artifacts, _, err := PrivacyArtifacts(context.Background(), b, []Artifact{a}, e.options.Home)
			if code == "1" {
				if err != nil || len(artifacts) != 1 || artifacts[0].Path != filepath.Join(project, ".git", "info", "exclude") {
					t.Fatal("untracked exit was not accepted", err)
				}
			} else if err == nil {
				t.Fatalf("Git %s failure/tracked destination accepted", code)
			}
			if _, err := os.Stat(e.statePath()); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("privacy preflight mutated setup state")
			}
		})
	}
}

func TestReadOnlyDestinationPreflightAndPrivateState(t *testing.T) {
	e, project, root := fixtureEngine(t)
	path := filepath.Join(project, "AGENTS.md")
	put(t, path, []byte("user instructions"))
	if err := os.Chmod(path, 0400); err != nil {
		t.Fatal(err)
	}
	in := installation("codex", project, root, Artifact{Path: path, Kind: "block", Key: "policy", Content: []byte("read")})
	if _, err := e.PlanInstall(context.Background(), []Installation{in}); err == nil {
		t.Fatal("read-only destination passed preflight")
	}
	if _, err := os.Stat(e.stateDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed preflight created private state")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	e.options.Failpoint = func(boundary string) error {
		if boundary == "prepared" {
			info, err := os.Stat(e.pendingPath())
			if err != nil {
				return err
			}
			if info.Mode().Perm() != 0600 {
				t.Error("journal exposed private before-image")
			}
		}
		return nil
	}
	applyInstall(t, e, in)
	for _, file := range []string{e.statePath(), filepath.Join(e.stateDir, "lock")} {
		info, err := os.Stat(file)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("private state mode", file, err)
		}
	}
	//nolint:gosec // Negative fixture intentionally makes only its disposable state directory insecure; Apply must reject it unchanged.
	if err := os.Chmod(e.stateDir, 0755); err != nil {
		t.Fatal(err)
	}
	p, err := e.PlanUninstall(context.Background(), Selection{Scope: "project", Project: project, Agents: []string{"codex"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Apply(context.Background(), p); err == nil {
		t.Fatal("insecure state directory accepted")
	}
	info, err := os.Stat(e.stateDir)
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatal("setup silently repaired broad permissions")
	}
}

func TestOperationReceiptReplayConflictsAndPending(t *testing.T) {
	e, project, root := fixtureEngine(t)
	in := installation("codex", project, root)
	p, err := e.PlanInstall(context.Background(), []Installation{in})
	if err != nil {
		t.Fatal(err)
	}
	request := map[string]string{"root": root, "agent": "codex"}
	if cached, err := e.Operation(p, "operation", request); err != nil || cached != nil {
		t.Fatal(cached, err)
	}
	if _, err := e.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if cached, err := e.Operation(nil, "operation", request); err != nil || cached == nil {
		t.Fatal("same operation did not replay", err)
	}
	if _, err := e.Operation(nil, "operation", map[string]string{"root": root + "-different"}); err == nil {
		t.Fatal("changed operation request replayed")
	}
	if _, err := e.Operation(nil, " padded ", request); err == nil {
		t.Fatal("padded operation id accepted")
	}
	if err := e.saveJournal(journal{Schema: 1, Stages: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Operation(nil, "operation", request); err == nil {
		t.Fatal("receipt bypassed pending recovery")
	}
}

func TestEmptyJSONAndSharedPolicyConflicts(t *testing.T) {
	e, project, root := fixtureEngine(t)
	path := filepath.Join(project, "settings.json")
	put(t, path, []byte{})
	if _, err := e.PlanInstall(context.Background(), []Installation{installation("claude", project, root, Artifact{Path: path, Kind: "json", Key: "kb", Content: []byte(`{}`)})}); err == nil {
		t.Fatal("invalid existing empty JSON accepted")
	}
	a := Artifact{Path: filepath.Join(project, "AGENTS.md"), Kind: "block", Key: "same-policy", Content: []byte("learning on"), Shared: true}
	one := installation("codex", project, root, a)
	a.Content = []byte("learning off")
	two := installation("pi", project, root, a)
	if _, err := e.PlanInstall(context.Background(), []Installation{one, two}); err == nil {
		t.Fatal("incompatible shared policies combined")
	}
}
