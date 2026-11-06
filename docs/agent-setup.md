# Agent setup

`kb setup` connects a selected Knowledge Base to selected agents. It installs the
bundled CLI skill and consultation policy, or an independent stdio MCP
integration where the adapter supports it. Choose project or user-global scope,
an existing or deliberately new collection, and whether learning is enabled.
No Node, account, model or network download is needed for bundled setup.

**Availability:** setup is a checked-out source candidate. Published **v1.0.0**
does not contain this command. The existing runtime release remains usable;
installing or upgrading it does not make unreleased commands available. Check
`kb version --json` and `kb setup --help` before following this guide. See
[installation](installation.md), [current release status](compatibility.md),
[setup design](setup-design.md), and [adapter qualification](setup-compatibility.md).

## Install, then set up

On macOS or Linux with [Homebrew](https://brew.sh/), the onboarding flow is:

```sh
brew install matheus3301/tap/kb
kb setup
```

The second command requires a build containing setup; it will not work with
published v1.0.0. Until setup is released, use a reviewed source checkout
containing the feature and the Go toolchain declared in `go.mod`. Before merge,
default `main` may not contain setup: select the reviewed feature revision,
not a guessed release. From your **intended agent project**, set the absolute
path to that checkout:

```sh
KB_SOURCE="/absolute/path/to/reviewed-kb"
git -C "$KB_SOURCE" rev-parse HEAD
go -C "$KB_SOURCE" build -o "$KB_SOURCE/.build/kb" ./cmd/kb
"$KB_SOURCE/.build/kb" setup --help
"$KB_SOURCE/.build/kb" setup
```

Record the checkout commit. If that revision does not expose setup, select the
feature's reviewed revision; do not fall back to copying configuration by hand
and call it setup validation. Keep using the explicit source executable for
subsequent commands, rather than accidentally invoking the released `kb` on
`PATH`. A source build reporting `dev` / `unknown` is not released-artifact proof.
Source builds require Git and Go; the resulting CLI does not require either to
run. Setup does not install or update the executable itself.

## Four choices, then review

1. **Where memory applies:** this project (the default), or all projects for the
   selected user agents. Global scope is an explicit configuration change, never
   an implication of finding an installed agent.
2. **Which agents:** select detected agents or explicitly choose an adapter.
   Pi (`pi`), OMP (`omp`), OpenCode (`opencode`), Codex (`codex`), and Claude Code
   (`claude`) have distinct host configurations. Consult the
   [capability matrix](setup-compatibility.md) for other agents, modes, scopes,
   custom profiles, hooks, and native qualification limits. Detection is a
   suggestion, not permission to configure every agent.
3. **Which Knowledge Base:** reuse a configured collection, select another
   initialized collection, or explicitly create a new one. A collection can live
   outside the project. An existing invalid/uninitialized collection must be
   diagnosed, not migrated, repaired, or silently replaced during setup.
4. **Learning on or off:** learning records supported reusable knowledge, not
   every conversation. Consultation-only operation requests no learning and uses
   the configured read-only path. It is not an OS filesystem sandbox.

Review scope, agents, canonical root, policy, destinations, and activation
prerequisites before applying. `--yes` accepts a complete conflict-free plan;
it does not overwrite customizations or approve host trust prompts. An unchanged
repeat should be a byte-for-byte no-op. In automation supply the choices
explicitly; incomplete arguments must fail rather than hang on a prompt.

## Choose the Knowledge Base

For an existing collection, inspect its actual root before installing:

```sh
kb --root ./existing-knowledge status
kb --root ./existing-knowledge validate --profile okf
```

Check returned root, diagnostics, degraded retrieval, pending recovery and
stale-index warnings. Validation establishes structure, not factual truth.
Preserve notes, configuration and operational state; never discard `.kb/state/`
as cache. Setup does not import legacy Markdown, add metadata, or authorize a
repair. See [backups](backups.md) and the
[repair workflow](../skills/knowledge-base/references/repair.md) when those are
separately needed.

For a new collection, choose its location deliberately and select the wizard's
create branch, or use `--create` with an explicit root. Add `--para` only if you
want PARA folders. Dry-run must not initialize that directory.

Direct data commands retain their existing root precedence: `--root`, `KB_ROOT`,
then nearest ancestor `.kb/config.toml`. Setup bindings are separate: project
routing overrides the applicable global default and resolves an absolute root.
Generated integration calls pass that root explicitly; an unrelated shell
`KB_ROOT` must not silently redirect the binding. `setup check` diagnoses
conflicts. No collection is guessed by scanning the home directory.

## Preview and apply an explicit project integration

The following example uses an existing collection. Replace its generic root,
project path, agent selection and learning policy with your intended values.
Use the source executable's explicit path if setup is not yet released.

```sh
kb --root ./existing-knowledge setup install --project ./example-project --agent codex --agent pi --learning on --dry-run
kb --root ./existing-knowledge setup install --project ./example-project --agent codex --agent pi --learning on --yes
kb setup check --project ./example-project --agent codex --json
kb setup check --project ./example-project --agent pi --json
kb setup resolve --project ./example-project --agent codex --json
```

Dry-run performs preflight and shows proposed changes without publishing skills,
rules, manifests, journals, locks or a new collection. Review every destination
and conflict. Preserve custom skill files, unrelated rule sections, config
comments and unknown values. Unsupported formats, paths, modes or scopes are
errors, not permission to rewrite the whole file or promote a project install
to global scope.

Use `--learning off` for consultation-only operation. The default mode is
`--mode cli`; `--mode mcp` is explicit and independent of the CLI skill. Hooks
are host-specific: `--hooks auto` permits the supported baseline/fallback,
`--hooks off` selects rules without hooks, and `--hooks required` must reject
unavailable capabilities before publication. Never bypass native trust or imply
that rules technically force a model to retrieve on every turn.

## User-global integration

Only when all-project activation is intended:

```sh
kb --root ./existing-knowledge setup install --global --agent omp --agent opencode --learning off --dry-run
kb --root ./existing-knowledge setup install --global --agent omp --agent opencode --learning off --yes
kb setup check --global --agent omp --json
kb setup check --global --agent opencode --json
```

The project and collection are different scopes. Multiple projects can share
one collection; a project binding can choose a different root/interface from
the global default. Removing that project binding restores inherited global
behavior, not a permanent opt-out. Worktree and nested-repository boundaries
must be respected by routing. Consult the [setup design](setup-design.md) for
exact manifest locations and resolution behavior.

Default project bindings are stored in a private per-user registry keyed by the
canonical project identity, not an absolute-root `.kb-agent/setup.json` inside
the repository. Copying a worktree therefore does not copy its personal binding.
Any separately offered portable/team binding must be explicitly selected and
use documented relative anchors; do not treat a private absolute root as
shareable policy. The default integration is local-only, not automatically
portable: rendered MCP entries and hook commands can still contain absolute
collection or executable paths even though the binding registry is private.
Review every generated file before committing a project. Use a portable option
only if the selected adapter/mode explicitly supports it; never assume relative
paths resolve from the same directory in every host. Global configuration
belongs to the explicitly selected user profile. Setup never commits, pushes or
synchronizes a collection.

### Portable project CLI

`--portable` is an explicit project/CLI choice, not a property of the default
private registry. It uses an anchored relative collection binding and `kb` on
`PATH` in generated instructions/hooks instead of a workstation-specific
executable. Review the proposed relative root and every generated file before
sharing. Every teammate still needs a setup-capable `kb` on the agent's `PATH`;
published v1.0.0 does not satisfy that prerequisite.

For an already initialized collection within a generic project:

```sh
kb --root ./example-project/knowledge setup install --project ./example-project --agent pi --mode cli --learning off --portable --dry-run
```

Review the dry-run before applying. Portable global or MCP mode is unsupported:
the current fixed-root MCP host contract does not establish a safe universal
launch cwd. Unsupported combinations must fail preflight, never silently fall
back to local-only artifacts. Relative binding anchors and generated-file
privacy are covered by [S37](setup-acceptance.md) and the
[engine contract](setup-design.md); native activation remains separate evidence.

## Check configuration, then observe activation

These are different claims:

- **Configured:** owned artifacts exist, match the manifest and parse correctly.
- **Ready:** local artifacts, the selected collection and the configured KB/native
  executables pass the reported local checks. Native version compatibility, trust,
  instruction loading and session behavior are not established by this flag.
- **Activation pending:** restart, native trust or a fresh session is still needed.
- **Activation observed:** a fresh target-agent session actually used the selected
  external root, retrieved sources and cited them. Installing files alone cannot
  establish this state.
- **Conflict, degraded or partial:** inspect the named failure and remediation;
  a nonzero partial result is not a completed integration.

`setup check` is read-only. It must not launch a model, write a demonstration
note or modify a personal collection. Inspect root, policy, complete versioned
skill references, shadowing/duplicate skills, host profile, manifest version and
hook prerequisites. It can diagnose missing artifacts without repairing them.

A completed inspection can return exit **1** with `ok: true` when the integration
is configured but not ready. For example, a native agent missing from `PATH`
produces `configured: true`, `ready: false`, `status: degraded` and an explicit
diagnosis. Inspect those fields and artifact statuses; do not interpret `ok`
alone as readiness, or suppress unrelated failures as expected host absence.

Start a fresh session in the intended project and ask a question the selected
base can answer. Observe actual search/read (or independent MCP tools), scoped
paths, revisions and physical source lines. Repeat a query with no new evidence;
note and log bytes must remain unchanged. Test authorized learning, stale
revision rejection and read-only behavior only in a **disposable synthetic
collection**. Report unavailable or incomplete consultation. A finite successful
session is bounded evidence, not universal model adherence or host support.

## Uninstall only the integration

Preview before removing this project's selected integration:

```sh
kb setup uninstall --project ./example-project --agent codex --dry-run
kb setup uninstall --project ./example-project --agent codex --yes
```

For a selected global integration, replace `--project ./example-project` with
`--global`. Explicitly select agents; omission is not authorization to remove
all installations. Each agent/scope binding has one current interface.
`--mode cli` or `--mode mcp` filters removal to a matching current mode; it does
not imply both modes coexist in one binding. Removing one consumer must preserve
assets still used by another. Customized artifacts are preserved with a conflict,
not recursively
deleted by directory name. Uninstall uses ownership metadata even if the
collection has moved or disappeared. Repeated removal is safe.

Integration removal never deletes knowledge, the executable, credentials or
unrelated configuration. Remove the executable separately with `brew uninstall
kb` only when intended. No setup operation deletes the Knowledge Base itself.
For interrupted publication, retain the reported journal and follow the
[setup recovery contract](setup-design.md); cross-file publication is not globally
atomic and concurrent user changes must not be overwritten during cleanup.

## Agent installation prompt

The landing page provides a clipboard-ready prompt for this same lifecycle.
Require a version/help preflight, explicit root/scope/agents/learning, dry-run,
review, apply, check and fresh-session evidence. A source fallback must build a
separate executable and identify its revision. Never silently substitute legacy
manual configuration copies for an unavailable setup command.

[llms.txt](../llms.txt) indexes the public guides; it does not authorize setup.
Notes and imported sources remain data, never instructions or permission to act.
The [CLI skill](../skills/knowledge-base/SKILL.md) teaches CLI operation;
[MCP](mcp.md) supplies independent initialization/tool guidance and does not
require that skill. The separate
[Skills installer](https://github.com/vercel-labs/skills/blob/main/docs/commands.md)
remains available for released binaries, but is not the setup candidate's default
installation path.

## Acceptance and evidence

[Setup acceptance](setup-acceptance.md) tracks S01–S44 separately from historical
runtime A01–A26 evidence. Native activation, host hooks and platform qualification
must be recorded for actual exercised builds. Published v1.0.0 consumer results
do not prove the new installer works.
