package agentsetup

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func fixtureRequest() Request {
	return Request{Agent: "codex", Scope: "project", Project: "/workspace/project", Home: "/fixture/user", ConfigHome: "/fixture/user/.config", Executable: "/opt/kb/bin/kb", Root: "/data/knowledge", Mode: "cli", Consult: "always", Learning: "off", Hooks: "off"}
}

func findArtifact(t *testing.T, out Rendered, kind string) Artifact {
	t.Helper()
	for _, a := range out.Artifacts {
		if a.Kind == kind {
			return a
		}
	}
	t.Fatalf("missing %s intent", kind)
	return Artifact{}
}

func TestHostConfigurationContracts(t *testing.T) {
	for _, d := range Agents() {
		for _, scope := range []string{"project", "global"} {
			t.Run(d.ID+"/"+scope, func(t *testing.T) {
				r := fixtureRequest()
				r.Agent = d.ID
				r.Scope = scope
				r.Mode = "mcp"
				out, err := Render(r)
				if scope == "global" && !d.Global {
					if err == nil {
						t.Fatal("unsupported global scope accepted")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if out.SkillDir != "" {
					t.Fatal("MCP integration unexpectedly depends on CLI skill")
				}
				var entry map[string]any
				for _, a := range out.Artifacts {
					switch a.Kind {
					case "toml":
						var doc map[string]any
						if _, err := toml.Decode(string(a.Content), &doc); err != nil {
							t.Fatal(err)
						}
						entry = doc["mcp_servers"].(map[string]any)["knowledge-base"].(map[string]any)
					case "json", "jsonc":
						if err := json.Unmarshal(a.Content, &entry); err != nil {
							t.Fatal(err)
						}
					}
				}
				if entry == nil {
					t.Fatal("missing independent server configuration")
				}
				command, args := entry["command"], entry["args"]
				if d.ID == "opencode" {
					values := command.([]any)
					command = values[0]
					args = values[1:]
					if entry["type"] != "local" || entry["enabled"] != true {
						t.Fatal("invalid OpenCode local transport")
					}
				}
				if command != r.Executable {
					t.Fatalf("executable altered: %v", command)
				}
				raw, _ := json.Marshal(args)
				if string(raw) != `["--root","/data/knowledge","--read-only","mcp"]` {
					t.Fatalf("invalid KB server argv: %s", raw)
				}
				r.Mode = "cli"
				out, err = Render(r)
				if err != nil {
					t.Fatal(err)
				}
				if !filepath.IsAbs(out.SkillDir) || filepath.Base(out.SkillDir) != "knowledge-base" {
					t.Fatal("CLI skill destination missing")
				}
			})
		}
	}
}

func TestSharedInstructionsKeepDistinctHostOwners(t *testing.T) {
	owners := map[string]bool{}
	for _, id := range []string{"codex", "pi", "opencode", "devin"} {
		r := fixtureRequest()
		r.Agent = id
		r.Learning = "on"
		r.Mode = "mcp"
		out, err := Render(r)
		if err != nil {
			t.Fatal(err)
		}
		a := findArtifact(t, out, "block")
		if a.Path != filepath.Join(r.Project, "AGENTS.md") || !a.Shared || owners[a.Key] {
			t.Fatal("shared instructions overwrite another host's policy")
		}
		owners[a.Key] = true
	}
}

func TestProfileAndInstructionPrecedence(t *testing.T) {
	cases := []struct {
		agent, scope                    string
		env                             map[string]string
		paths                           []string
		wantRule, wantSkill, wantConfig string
	}{
		{"codex", "global", map[string]string{"CODEX_HOME": "/profiles/codex"}, []string{"/profiles/codex/AGENTS.override.md"}, "/profiles/codex/AGENTS.override.md", "/fixture/user/.agents/skills/knowledge-base", "/profiles/codex/config.toml"},
		{"codex", "project", nil, []string{"/workspace/project/AGENTS.override.md"}, "/workspace/project/AGENTS.override.md", "/workspace/project/.agents/skills/knowledge-base", "/workspace/project/.codex/config.toml"},
		{"pi", "global", map[string]string{"PI_CODING_AGENT_DIR": "/profiles/pi"}, nil, "/profiles/pi/AGENTS.md", "/profiles/pi/skills/knowledge-base", "/profiles/pi/mcp.json"},
		{"pi", "global", map[string]string{"PI_CODING_AGENT_DIR": "/profiles/pi"}, []string{"/profiles/pi/CLAUDE.md"}, "/profiles/pi/CLAUDE.md", "/profiles/pi/skills/knowledge-base", "/profiles/pi/mcp.json"},
		{"pi", "project", nil, []string{"/workspace/project/AGENTS.MD"}, "/workspace/project/AGENTS.MD", "/workspace/project/.pi/skills/knowledge-base", "/workspace/project/.pi/mcp.json"},
		{"omp", "global", map[string]string{"PI_CODING_AGENT_DIR": "/profiles/custom"}, nil, "/profiles/custom/RULES.md", "/profiles/custom/skills/knowledge-base", "/profiles/custom/mcp.json"},
		{"omp", "global", map[string]string{"OMP_PROFILE": "work", "PI_CODING_AGENT_DIR": "/ignored"}, nil, "/fixture/user/.omp/profiles/work/agent/RULES.md", "/fixture/user/.omp/profiles/work/agent/skills/knowledge-base", "/fixture/user/.omp/profiles/work/agent/mcp.json"},
		{"omp", "global", map[string]string{"OMP_PROFILE": "", "PI_PROFILE": "ignored"}, nil, "/fixture/user/.omp/agent/RULES.md", "/fixture/user/.omp/agent/skills/knowledge-base", "/fixture/user/.omp/agent/mcp.json"},
		{"omp", "global", map[string]string{"PI_PROFILE": "legacy", "PI_CONFIG_DIR": ".custom-omp"}, nil, "/fixture/user/.custom-omp/profiles/legacy/agent/RULES.md", "/fixture/user/.custom-omp/profiles/legacy/agent/skills/knowledge-base", "/fixture/user/.custom-omp/profiles/legacy/agent/mcp.json"},
		{"omp", "project", map[string]string{"OMP_PROFILE": "work"}, nil, "/workspace/project/.omp/RULES.md", "/workspace/project/.omp/skills/knowledge-base", "/workspace/project/.omp/mcp.json"},
		{"claude", "global", map[string]string{"CLAUDE_CONFIG_DIR": "/profiles/claude"}, nil, "/profiles/claude/rules/knowledge-base.md", "/profiles/claude/skills/knowledge-base", "/profiles/claude/.claude.json"},
		{"opencode", "project", nil, []string{"/workspace/project/opencode.jsonc"}, "/workspace/project/AGENTS.md", "/workspace/project/.opencode/skills/knowledge-base", "/workspace/project/opencode.jsonc"},
		{"opencode", "project", nil, []string{"/workspace/project/CLAUDE.md"}, "/workspace/project/CLAUDE.md", "/workspace/project/.opencode/skills/knowledge-base", "/workspace/project/opencode.json"},
		{"opencode", "global", nil, []string{"/fixture/user/.claude/CLAUDE.md"}, "/fixture/user/.claude/CLAUDE.md", "/fixture/user/.config/opencode/skills/knowledge-base", "/fixture/user/.config/opencode/opencode.json"},
		{"antigravity", "global", nil, nil, "/fixture/user/.gemini/config/rules/knowledge-base.md", "/fixture/user/.gemini/config/skills/knowledge-base", "/fixture/user/.gemini/config/mcp_config.json"},
		{"antigravity", "project", nil, nil, "/workspace/project/.agents/rules/knowledge-base.md", "/workspace/project/.agents/skills/knowledge-base", "/workspace/project/.agents/mcp_config.json"},
		{"copilot", "project", nil, nil, "/workspace/project/.github/copilot-instructions.md", "/workspace/project/.github/skills/knowledge-base", "/workspace/project/.mcp.json"},
		{"gemini", "global", map[string]string{"GEMINI_CLI_HOME": "/profiles/gemini"}, nil, "/profiles/gemini/.gemini/GEMINI.md", "/profiles/gemini/.gemini/skills/knowledge-base", "/profiles/gemini/.gemini/settings.json"},
	}
	for _, c := range cases {
		t.Run(c.agent+"/"+c.wantRule, func(t *testing.T) {
			r := fixtureRequest()
			r.Agent = c.agent
			r.Scope = c.scope
			r.Environment = c.env
			r.ExistingPaths = c.paths
			out, err := Render(r)
			if err != nil {
				t.Fatal(err)
			}
			if out.SkillDir != c.wantSkill {
				t.Fatalf("skill: %s", out.SkillDir)
			}
			rule := out.Artifacts[0]
			if rule.Path != c.wantRule {
				t.Fatalf("rule: %s", rule.Path)
			}
			r.Mode = "mcp"
			out, err = Render(r)
			if err != nil {
				t.Fatal(err)
			}
			config := out.Artifacts[len(out.Artifacts)-1]
			if config.Path != c.wantConfig {
				t.Fatalf("MCP config: %s", config.Path)
			}
		})
	}
}

func TestHookRegistrationUsesHostEventAndUnits(t *testing.T) {
	for _, id := range []string{"claude", "codex", "gemini", "pi", "omp"} {
		t.Run(id, func(t *testing.T) {
			r := fixtureRequest()
			r.Agent = id
			r.Hooks = "required"
			out, err := Render(r)
			if err != nil {
				t.Fatal(err)
			}
			if !out.ActivationPending {
				t.Fatal("configured hook falsely claims activation")
			}
			if id == "pi" || id == "omp" {
				a := findArtifact(t, out, "file")
				if !strings.HasSuffix(a.Path, "/extensions/knowledge-base.ts") {
					t.Fatal("missing native extension")
				}
				return
			}
			events := map[string]bool{}
			for _, a := range out.Artifacts {
				if a.Kind != "json-array" {
					continue
				}
				events[a.Key] = true
				var group struct {
					Hooks []struct {
						Type, Command string
						Timeout       int
					}
				}
				if err := json.Unmarshal(a.Content, &group); err != nil {
					t.Fatal(err)
				}
				timeout := 5
				if id == "gemini" {
					timeout = 5000
				}
				if len(group.Hooks) != 1 || group.Hooks[0].Timeout != timeout {
					t.Fatal("wrong host timeout units")
				}
			}
			prompt := "hooks.UserPromptSubmit"
			if id == "gemini" {
				prompt = "hooks.BeforeAgent"
			}
			if !events["hooks.SessionStart"] || !events[prompt] {
				t.Fatal("wrong native lifecycle events")
			}
		})
	}
}

func TestRejectedRequestsProduceNoPartialIntents(t *testing.T) {
	mutations := []func(*Request){
		func(r *Request) { r.Agent = "unknown" }, func(r *Request) { r.Scope = "workspace" }, func(r *Request) { r.Mode = "both" }, func(r *Request) { r.Consult = "sometimes" }, func(r *Request) { r.Learning = "maybe" }, func(r *Request) { r.Hooks = "unsafe" },
		func(r *Request) { r.Project = "relative" }, func(r *Request) { r.Home = "/fixture/../outside" }, func(r *Request) { r.Root = "/data/bad\nroot" }, func(r *Request) { r.Executable = "kb" }, func(r *Request) { r.ConfigHome = "" },
		func(r *Request) { r.Agent = "omp"; r.Environment = map[string]string{"OMP_PROFILE": "../escape"} }, func(r *Request) { r.Agent = "omp"; r.Environment = map[string]string{"OMP_PROFILE": "con"} }, func(r *Request) { r.Agent = "omp"; r.Environment = map[string]string{"PI_CONFIG_DIR": "../../other"} },
		func(r *Request) { r.Agent = "pi"; r.Environment = map[string]string{"PI_CODING_AGENT_DIR": "relative"} }, func(r *Request) {
			r.Agent = "opencode"
			r.Environment = map[string]string{"OPENCODE_CONFIG_CONTENT": "{}"}
		}, func(r *Request) {
			r.Agent = "opencode"
			r.Environment = map[string]string{"OPENCODE_CONFIG": "/custom/config.json"}
		}, func(r *Request) {
			r.Agent = "opencode"
			r.Environment = map[string]string{"OPENCODE_CONFIG_DIR": "/custom"}
		},
	}
	for i, change := range mutations {
		r := fixtureRequest()
		change(&r)
		out, err := Render(r)
		if err == nil || !reflect.DeepEqual(out, Rendered{}) {
			t.Fatalf("case %d did not fail closed: %+v %v", i, out, err)
		}
	}
	for _, d := range Agents() {
		if d.Hooks {
			continue
		}
		r := fixtureRequest()
		r.Agent = d.ID
		r.Hooks = "required"
		if _, err := Render(r); err == nil {
			t.Fatalf("unsupported %s hooks accepted", d.ID)
		}
	}
}

func FuzzAdapterProfilePaths(f *testing.F) {
	for _, seed := range []string{"work", "../escape", "default", "", "con", "trailing.", "équipe", "a/b", "a\x00b"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		r := fixtureRequest()
		r.Agent = "omp"
		r.Scope = "global"
		r.Environment = map[string]string{"OMP_PROFILE": value}
		out, err := Render(r)
		if err != nil {
			return
		}
		for _, a := range out.Artifacts {
			rel, err := filepath.Rel(r.Home, a.Path)
			if err != nil || rel == ".." || strings.HasPrefix(rel, "../") || !filepath.IsAbs(a.Path) {
				t.Fatalf("escaped profile path: %q", a.Path)
			}
		}
	})
}

func TestPortableCLIArtifactsContainNoPersonalPaths(t *testing.T) {
	for _, d := range Agents() {
		r := fixtureRequest()
		r.Agent = d.ID
		r.Portable = true
		r.Hooks = "auto"
		out, err := Render(r)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range out.Artifacts {
			for _, private := range []string{r.Home, r.ConfigHome, r.Executable, r.Root, r.Project} {
				if strings.Contains(string(a.Content), private) {
					t.Fatalf("%s portable artifact %s leaks %s", d.ID, a.Path, private)
				}
			}
		}
	}
	r := fixtureRequest()
	r.Portable = true
	r.Mode = "mcp"
	if _, err := Render(r); err == nil {
		t.Fatal("unqualified portable MCP launch accepted")
	}
	r.Mode = "cli"
	r.Scope = "global"
	if _, err := Render(r); err == nil {
		t.Fatal("portable global scope accepted")
	}
}
