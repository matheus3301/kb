# Agent setup compatibility

This matrix describes **rendered configuration contracts**, checked against primary
sources on 2026-11-06. A generated file, successful parser fixture, or descriptor
capability is not evidence that an installed agent loaded it or that a model
followed it. Native activation remains pending until a fresh target session is
observed. No minimum native version is inferred from a directory's existence.

The adapter package is `internal/agentsetup`. It returns file, managed-block,
JSON/JSONC-entry, TOML-table and hook-array intents. The setup engine owns
preservation, publication, recovery, shared ownership and uninstall. CLI mode
installs the complete version-matched bundled skill, including its references;
MCP mode does not depend on that skill.

## Supported configuration targets

`~` means the selected user's home on the **execution host**, not necessarily the
machine running an editor. Paths below are defaults. Project paths are relative
to the selected project root. `knowledge-base` is the skill and server ID.

| Agent ID | Project rules / CLI skill directory | Global rules / CLI skill directory | Local stdio MCP configuration | Prompt context adapter |
| --- | --- | --- | --- | --- |
| `claude` | `.claude/rules/knowledge-base.md`; `.claude/skills/knowledge-base` | `~/.claude/rules/knowledge-base.md`; `~/.claude/skills/knowledge-base` | Project `.mcp.json`; global `~/.claude.json`; `mcpServers.knowledge-base` | `SessionStart` and `UserPromptSubmit`, settings command hooks |
| `codex` | `AGENTS.override.md` if present, otherwise `AGENTS.md`; `.agents/skills/knowledge-base` | `<CODEX_HOME>/AGENTS.override.md` if present, otherwise `AGENTS.md`; `~/.agents/skills/knowledge-base` | `.codex/config.toml` or `<CODEX_HOME>/config.toml`; `[mcp_servers.knowledge-base]` | `SessionStart` and `UserPromptSubmit`, `hooks.json` |
| `pi` | `AGENTS.override.md` if present, otherwise `AGENTS.md`; `.pi/skills/knowledge-base` | `<agent-dir>/AGENTS.override.md` if present, otherwise `AGENTS.md`; `<agent-dir>/skills/knowledge-base` | `.pi/mcp.json` or `<agent-dir>/mcp.json`; `mcpServers.knowledge-base` | Native `.pi/extensions/knowledge-base.ts` or `<agent-dir>/extensions/knowledge-base.ts` |
| `omp` | `.omp/RULES.md`; `.omp/skills/knowledge-base` | `<agent-dir>/RULES.md`; `<agent-dir>/skills/knowledge-base` | `.omp/mcp.json` or `<agent-dir>/mcp.json`; `mcpServers.knowledge-base` | Native `.omp/extensions/knowledge-base.ts` or `<agent-dir>/extensions/knowledge-base.ts` |
| `opencode` | `AGENTS.md`; `.opencode/skills/knowledge-base` | `<config-home>/opencode/AGENTS.md`; `<config-home>/opencode/skills/knowledge-base` | `opencode.json` / `opencode.jsonc`, project or global; documented `mcp.knowledge-base` local-server shape | Pending: no speculative plugin installed |
| `cursor` | `.cursor/rules/knowledge-base.mdc` with `alwaysApply: true`; `.cursor/skills/knowledge-base` | Automatic global rules installation unsupported; use Cursor's User Rules UI | Project `.cursor/mcp.json`; `mcpServers.knowledge-base` | Pending: rule + skill/MCP baseline |
| `gemini` | `GEMINI.md`; `.agents/skills/knowledge-base` | `~/.gemini/GEMINI.md`; `~/.agents/skills/knowledge-base` | `.gemini/settings.json` or `~/.gemini/settings.json`; `mcpServers.knowledge-base` | `SessionStart` and **`BeforeAgent`**, settings command hooks |
| `copilot` | `.github/copilot-instructions.md`; `.github/skills/knowledge-base` | `<COPILOT_HOME>/copilot-instructions.md`; `<COPILOT_HOME>/skills/knowledge-base` | Project `.mcp.json`; global `<COPILOT_HOME>/mcp-config.json`; `mcpServers.knowledge-base` | Per-prompt command output is ignored by the current Copilot harness; not claimed |
| `antigravity` | `.agents/rules/knowledge-base.md` with `trigger: always_on`; `.agents/skills/knowledge-base` | `~/.gemini/config/rules/knowledge-base.md`; `~/.gemini/config/skills/knowledge-base` | Project `.agents/mcp_config.json`; global `~/.gemini/config/mcp_config.json`; `mcpServers.knowledge-base` | Pending: rule + skill/MCP baseline |
| `devin` | `AGENTS.md`; `.agents/skills/knowledge-base` | `<config-home>/devin/AGENTS.md`; `<config-home>/devin/skills/knowledge-base` | `.devin/mcp_config.json` or `<config-home>/devin/mcp_config.json`; `mcpServers.knowledge-base` | Native hook feature documented, but cwd/input qualification pending; not installed |

### Important host boundaries

- **Claude Code:** `CLAUDE_CONFIG_DIR` changes the user directory; its global MCP
  file is `<CLAUDE_CONFIG_DIR>/.claude.json`. Project configuration remains
  project-local. Hook entries go into `settings.json`, not skill frontmatter.
  Managed-only/disabled hooks and native settings restrictions remain effective.
- **Codex:** `CODEX_HOME` defaults to `~/.codex`; shared user skills remain under
  `~/.agents/skills`. Existing instruction overrides are selected rather than
  installing a shadowed `AGENTS.md`. Hooks accumulate across layers and require
  native review/trust. Setup never writes trust databases or uses trust-bypass
  flags. Local and cloud Codex are not interchangeable qualification targets.
- **Pi:** the agent directory defaults to `~/.pi/agent` and honors
  `PI_CODING_AGENT_DIR`. Current upstream Pi **does have native MCP**, documented
  independently in its MCP reference: `mcpServers`, one executable in `command`,
  separate `args`, project configuration after native project trust. Older Pi
  builds and replacement MCP extensions can have different capabilities.
  Existing non-empty uppercase or Claude-named context files are reused in
  documented precedence order, rather than creating an AGENTS.md that hides
  existing instructions.
- **OMP:** the inspected source is v18.16.0, commit
  `650f3a108229ee11bf68ba3377f4e8c3391cd424`. Its agent directory defaults to
  `~/.omp/agent`. `OMP_PROFILE` precedes `PI_PROFILE`, including an explicitly
  empty `OMP_PROFILE` selecting default. Named profiles use
  `~/<PI_CONFIG_DIR>/profiles/<name>/agent` and ignore `PI_CODING_AGENT_DIR`;
  default mode honors that directory override. The adapter supports a single
  directory-name `PI_CONFIG_DIR` override. Invalid/reserved profile names and
  unsupported path forms fail before publication. Native project resources stay
  in `.omp`, **not `.pi`**. XDG data/state/cache relocation is not the agent
  configuration directory and must not redirect skills or extensions.
- **OpenCode:** the public `/docs/` pages document the JSON/JSONC local-server
  layout rendered here. Existing `.jsonc` is selected without migrating it to
  JSON. Current V2 source documents a different plugin API; V1 plugins must not
  be assumed compatible with V2. Native V2 plugin/config qualification is not
  established by this matrix. Custom `OPENCODE_CONFIG`, `OPENCODE_CONFIG_DIR`
  and inline `OPENCODE_CONFIG_CONTENT` layers are rejected rather than silently
  writing an ineffective default layer.
  Existing non-empty project CLAUDE.md, or the enabled global Claude fallback,
  is reused when no native AGENTS.md exists, preserving loaded instructions.
- **Cursor:** User Rules are UI-managed. A requested global install fails rather
  than promoting a project operation or inventing a global rule-file location.
- **Gemini CLI:** `GEMINI_CLI_HOME` names an alternative user-home root, with
  configuration under its `.gemini` child (not the variable's directory itself).
  The adapter keeps custom-home rules, settings, hooks and skills together there;
  project scope remains project-local. Native skill consent is not bypassed.
- **GitHub Copilot / VS Code:** this target is the shared **Copilot harness**, not
  the older Local harness, and not whichever model happens to be selected.
  Workspace-root `.mcp.json` requires the documented
  `chat.mcp.workspaceRootConfig.enabled` setting. Setup reports that prerequisite;
  it does not change editor settings. `userPromptSubmitted` command-hook output
  is ignored, so registering that event would not establish context injection.
- **Antigravity:** current App/CLI/IDE docs specify `.agents` workspace paths and
  `.gemini/config` global paths. Legacy `.agent` or older Antigravity-specific
  directories are not silently used. Runtime migration from Gemini CLI is not
  performed by KB setup.
- **Devin:** targets local Devin CLI, not remote Devin Cloud. Dedicated
  `mcp_config.json` files require v3000.3 or newer per the official reference.
  The native hook documentation does not establish this adapter's required cwd
  input contract; hook-required fails instead of guessing. Remote access to a
  workstation-local collection is not provided.

## Routing and interface independence

Managed instruction blocks have distinct `kb-policy-<agent-id>` ownership keys.
They are guarded by the actual harness, not the model name. One host's
installation must not replace another host's policy in shared `AGENTS.md` files.

A project policy replaces less-specific/global KB policy **for that same host**.
The global policy is a fallback only when no applicable project policy is loaded.
This applies equally to global CLI + project MCP and global MCP + project CLI;
never mandate both interfaces. Native instruction loading and same-name server
precedence still require fresh-session qualification.

- **CLI policy:** calls `kb setup resolve --agent ID --project WORKDIR --json`,
  uses the effective root/policies/skill directory, then passes the root
  explicitly to data commands. Ambient `KB_ROOT` does not select the integration.
- **MCP policy:** uses the configured `knowledge-base` server's fixed root and its
  self-contained initialization/tool guidance. The consumer does **not** need a
  shell command, the CLI resolver, or the CLI skill. Failure is reported instead
  of silently switching interfaces or borrowing another host's binding.

Default project bindings live in a private per-user registry, keyed by canonical
project identity. This does **not** make native configuration private:
`.mcp.json`, `.codex/config.toml` and other personal activation files can contain
an absolute collection root or binary path, and can be tracked or copied. Review
the setup engine's disclosed local exclusions and changed paths; exclusions
cannot protect an already tracked file. Do not publish personal activation
artifacts as a portable team integration.

Portable **project CLI** rendering uses `kb` on PATH and obtains the skill
directory from binding resolution, with no personal home, collection, project or binary path in rendered
content. The public binding's relative-root anchoring is owned by the setup
engine. Portable global scope and portable MCP fail explicitly: native server
launch cwd has not been qualified for a binding-aware MCP launcher. No relative
root is guessed from an arbitrary nested session cwd.

## Hook protocol and limitations

Implemented command protocols are host-specific: Claude and Codex return
`hookSpecificOutput` with the event name and `additionalContext`; Gemini returns
its documented `additionalContext` shape. Pi/OMP wrappers consume `{ "context":
"..." }`, then deliver native custom messages. No `kb.v1` CLI envelope or
progress text appears on hook stdout.

The dispatcher must pass the effective binding's interface to `HookContextMode`.
Both CLI and MCP hooks perform the same bounded direct-core retrieval; MCP mode
is not reminder-only. Mode changes only the continuation guidance: MCP directs
the model exclusively to the configured `knowledge-base` server tools and
their self-contained guidance, without shell commands, CLI flags, a resolver
invocation, or a CLI skill. The unchanged `HookContext` convenience API selects
CLI guidance for callers that already operate in CLI mode.

The Pi/OMP wrapper subscribes to `session_start` and `before_agent_start`;
OMP revival and Pi custom-message turns are not submitted user prompts. It
launches the selected KB executable with an argv array and JSON stdin, never a
shell. No npm dependency, prompt log, transcript parsing, note write, network
request or automatic learning action is added. The host can persist delivered
context in its own session transcript; KB does not control host retention.

Bounds and behavior:

- Input: one UTF-8 JSON object, at most 64 KiB; duplicate top-level fields,
  mismatched/unsupported events, invalid cwd and trailing objects fail.
- Session start: root/learning/availability reminder only; no taskless search.
- Prompt: first 32 distinct lexical terms within the core query ceiling; full
  conjunctive retrieval first. When no sources match, at most eight single-term
  probes (terms of at least three runes, prompt order) broaden the query. Errors
  are not retried. Partial relevance is explicitly reported; this is not semantic
  search or a completeness guarantee.
- Evidence: at most five sources and 1,200 estimated tokens / 4,800 excerpt bytes,
  with paths, revisions, line ranges, truncation and omissions. Retrieved material
  is separately JSON-encoded untrusted data, not authority for actions.
- Whole encoded response: at most 16 KiB. Oversize fails rather than silently
  dropping citations. Diagnostics include invalid/degraded/stale-index state.
- Core retrieval: cooperative two-second context deadline; native wrapper process
  the CLI hook dispatcher supplies an earlier 1.5-second deadline, which wins
  in that invocation path. Native wrapper process timeout is 3.5 seconds with
  `SIGKILL` for an uncooperative immediate child;
  command-hook timeout five seconds (5,000 ms for Gemini). These are configured
  bounds, not portable latency SLAs or general descendant-process supervision.
- Empty/no-hit queries are reported honestly. Missing root, pending recovery,
  invalid input, timeout and unsupported capability remain failures. Ordinary
  policy directs manual consultation; no stop-hook loop or permission bypass.
- Global/project dispatchers can both fire. No exactly-once or prompt-text-based
  deduplication is claimed; bounded duplicate read-only delivery is possible.

For pending adapters, use the rule + CLI skill or independent MCP baseline,
restart/reload as the host documents, and inspect its loaded instruction/server
list. Do not manually register KB's Claude protocol handler in another host's
hook format. `--hooks required` rejects unavailable per-prompt adapters;
`--hooks auto` explains the baseline fallback without claiming observed hooks.

## Verification boundary and native checklist

Package tests exercise configuration formats, scope/profile precedence, portable
content, unsupported combinations, malformed host input, budgets, citable core
retrieval, no-write repeat queries and concurrent read-only consultation.
Fuzz targets cover host JSON parsing and profile-path confinement.

A disposable generated-argv smoke launched the **compiled KB CLI** for all ten
project descriptors and completed MCP initialize, read-only tool discovery,
search and read against an external synthetic collection, without a consumer
shell, skill or resolver. Synthetic Pi/OMP host-API callbacks also exercised the
actual generated wrapper through argv/stdin into the direct-core handler,
including no-shell prompt text and oversized input. These are protocol/wrapper
checks, **not native host activation or model-adherence evidence**.

The generated-launch regression is retained in
`internal/agentsetup/mcp_integration_test.go`; it builds the real executable and
consumes only MCP JSON-RPC. Reproduce with:

```sh
env -u KB_ROOT go test ./internal/agentsetup \
  -run '^TestGeneratedMCPLaunchUsesIndependentServer$' -count=1
```

Native qualification must use disposable homes/profiles/projects and an external
synthetic collection, never real personal configuration:

1. Record host executable/version, KB build, OS/architecture, scope/profile and
   before-images or hashes of fixture files.
2. Install/check the integration. Inspect native loaded rules, full skill bundle,
   MCP tools and hook/extension diagnostics. Complete native trust through the
   host's normal interactive review; never add bypass flags.
3. For OMP, set `PI_CODING_AGENT_DIR` to a disposable agent directory, clear
   `OMP_PROFILE`/`PI_PROFILE` for the default-profile case, start in the synthetic
   project and inspect extension discovery. Repeat with a named OMP profile and
   assert it uses the profile's directory rather than the default override.
   For Pi, use its disposable `PI_CODING_AGENT_DIR` and `.pi` project resources,
   grant native project trust, then `/reload` or start a fresh session.
4. Submit a task using a unique synthetic term without repeating the KB rule in
   the prompt. Observe session reminder, prompt context, subsequent source reads
   and cited path/revision/lines. A plausible answer alone is not evidence.
5. Repeat without new learning; assert note/index/log bytes unchanged. Exercise
   disabled hooks, unavailable root, oversized/malformed input and resume.
6. Exercise nested cwd, separate worktree/nested repository, project-over-global
   binding, mixed CLI/MCP scopes, and two agents sharing assets. Remove one
   integration and observe the other still works; then uninstall and inspect
   fixture teardown. Never configure production agents merely to qualify them.

The review qualification records actual OMP 18.16.0 compaction, changed-root
session resume and both CLI/MCP scope directions in
[evidence/setup-native-lifecycle.json](evidence/setup-native-lifecycle.json).
Disabled hooks, no-hit, failed subprocesses and ignored-SIGTERM timeouts were
observed in OMP 18.16.0 and 18.17.0; see
[hook qualification](setup-hook-qualification.md). These identified native
observations do not extend to untested hosts or versions. Local `setup check`
explicitly leaves version/platform, trust and activation unobserved.

## Primary sources

- Claude Code: [memory/rules](https://code.claude.com/docs/en/memory),
  [hooks](https://code.claude.com/docs/en/hooks),
  [MCP](https://code.claude.com/docs/en/mcp),
  [environment variables](https://code.claude.com/docs/en/env-vars).
- Codex: [instruction discovery](https://learn.chatgpt.com/docs/agent-configuration/agents-md),
  [skills](https://learn.chatgpt.com/docs/build-skills),
  [native hooks](https://learn.chatgpt.com/docs/hooks).
- Pi: [configuration](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/configuration.md),
  [extensions](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/extensions.md),
  [MCP](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/mcp.md).
- OMP v18.16.0: [profile/path source](https://github.com/can1357/oh-my-pi/blob/650f3a108229ee11bf68ba3377f4e8c3391cd424/packages/utils/src/dirs.ts),
  [native discovery](https://github.com/can1357/oh-my-pi/blob/650f3a108229ee11bf68ba3377f4e8c3391cd424/packages/coding-agent/src/discovery/builtin.ts),
  [extension event types](https://github.com/can1357/oh-my-pi/blob/650f3a108229ee11bf68ba3377f4e8c3391cd424/packages/coding-agent/src/extensibility/extensions/types.ts).
- OpenCode: [rules](https://opencode.ai/docs/rules/),
  [configuration](https://opencode.ai/docs/config/),
  [MCP](https://opencode.ai/docs/mcp-servers/),
  [plugins](https://opencode.ai/docs/plugins/),
  [current V2 plugin API](https://github.com/anomalyco/opencode/blob/dev/packages/plugin/src/README.md).
- Cursor: [rules](https://cursor.com/docs/rules),
  [skills](https://cursor.com/docs/skills), [MCP](https://cursor.com/docs/mcp).
- Gemini CLI: [configuration](https://geminicli.com/docs/reference/configuration/),
  [hooks](https://geminicli.com/docs/hooks/reference/),
  [skills](https://geminicli.com/docs/cli/skills/).
- GitHub Copilot / VS Code: [instruction locations](https://code.visualstudio.com/docs/agents/reference/custom-instructions-reference),
  [MCP configuration](https://code.visualstudio.com/docs/agent-customization/mcp-servers),
  [skill locations](https://code.visualstudio.com/docs/agent-customization/agent-skills),
  [hook event/output boundaries](https://code.visualstudio.com/docs/agent-customization/hooks).
- Antigravity: [rules](https://antigravity.google/docs/rules),
  [skills](https://antigravity.google/docs/skills),
  [MCP](https://antigravity.google/docs/mcp).
- Devin CLI: [rules](https://docs.devin.ai/cli/extensibility/rules),
  [skills](https://docs.devin.ai/cli/extensibility/skills/overview),
  [MCP configuration](https://docs.devin.ai/cli/extensibility/mcp/configuration),
  [hooks](https://docs.devin.ai/cli/extensibility/hooks/overview),
  [lifecycle payloads](https://docs.devin.ai/cli/extensibility/hooks/lifecycle-hooks).
