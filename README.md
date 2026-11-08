<div align="center">
  <img src="assets/logo.png" alt="Knowledge Base blue-jay mascot" width="104" height="104" />
  <h1>KB — Knowledge Base</h1>
  <p><strong>Your Knowledge Base. In Markdown. In your terminal.</strong></p>
  <p>Local files for you. A CLI and a skill for your agent.</p>
</div>

<p align="center">
  <a href="https://github.com/matheus3301/kb/actions/workflows/ci.yml"><img src="https://github.com/matheus3301/kb/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://github.com/matheus3301/kb/blob/main/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/matheus3301/kb" alt="Go version" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-3662EA" alt="MIT license" /></a>
  <a href="#project-status"><img src="https://img.shields.io/badge/status-stable-516276" alt="Stable" /></a>
</p>

`kb` lets you search, read, and update a Knowledge Base stored as
[OKF](https://github.com/GoogleCloudPlatform/open-knowledge-format) Markdown.
Use any editor to read the files, and the CLI to make changes with revision
checks, previews, and recoverable transactions. Add the `knowledge-base` skill
to give your agent the same workflow. Optional PARA folders keep projects,
areas, resources, and archives easy to navigate.

## Getting started

### Set up your agents

Install KB and connect your agents in two commands:

```sh
brew install matheus3301/tap/kb
kb setup
```

**Setup requires v1.1.0 or later.** If KB is already installed, run `brew update`
and `brew upgrade matheus3301/tap/kb` first. Check `kb version --json` and
`kb setup --help`; [source installation](docs/agent-setup.md#install-then-set-up)
is also available.

Setup asks where memory applies (project or user-global), which agents to
configure, which collection to reuse or deliberately create, and whether to
record reusable learning. Select Pi, OMP, OpenCode, Codex, Claude Code and other
available adapters; see [capabilities and native qualification](docs/setup-compatibility.md)
before relying on a host mode or hook. Review a dry-run, apply, run
`kb setup check`, then observe actual retrieval in a fresh agent session.
Use `kb setup uninstall` to remove owned integration without deleting knowledge.

### Try the released CLI

With [Homebrew](https://brew.sh/) installed on macOS or Linux, choose a new
directory for this example. Do not replace an existing `KB_ROOT` selection
without checking which collection it identifies:

```sh
brew install matheus3301/tap/kb
export KB_ROOT="$PWD/example-kb"
kb init --para
```

Create a note and find it:

```sh
kb create --input - <<'JSON'
{
  "path": "03_Resources/decisions.md",
  "type": "Reference",
  "title": "Project decisions",
  "body": "# Project decisions\n\nSearch existing notes before adding new ones.\n",
  "summary": "Record the Knowledge Base workflow",
  "actor": "process:quickstart",
  "idempotency_key": "quickstart/create-decisions"
}
JSON

kb search "project decisions" --limit 5
kb read 03_Resources/decisions.md
kb context "search existing notes" --max-tokens 1000
```

Data commands return JSON by default. `KB_ROOT` selects the Knowledge Base;
all note paths are relative to it. Follow the [quickstart](docs/quickstart.md)
for editing and coordinated changes, or use [other installation methods](docs/installation.md).

For an existing collection, preserve its notes, configuration, and operational
state first. Follow [agent setup](docs/agent-setup.md#choose-the-knowledge-base)
to select or initialize that same directory deliberately; initialization does
not convert notes. Validate with `kb validate --profile okf` before writing.
Agents can [repair diagnosed issues in batches](skills/knowledge-base/references/repair.md)
with an explicitly reviewed `kb apply --repair --incremental` plan.

## At a glance

- **Search and read:** lexical ranking, filters, sections, and bounded excerpts
  with source paths, revisions, and line references.
- **Write:** create, patch, move, and archive notes; preview changes and reconcile
  revision conflicts before replacing content.
- **Coordinate:** apply multi-note plans with idempotent retries and an explicit
  recovery journal.
- **Maintain:** inspect links, validate format/health, update managed navigation,
  export validated knowledge snapshots, and stage operational backup/restore.
- **Work with your agent:** use explicit setup to install a bundled CLI skill
  and consultation policy, or an independent MCP integration. The runtime
  needs no model, Node, Git or network.

### Reuse the format in Go

The public [`github.com/matheus3301/kb/pkg/okf`](pkg/okf/README.md) package provides
parsing, structured validation and presentation-preserving metadata/body editing
without the KB runtime. It is **source-only pending a later release**;
published v1.1.0 does not contain it. See its [supported OKF v0.2 contract](pkg/okf/CONFORMANCE.md)
and compiling examples. All packages under `internal/` remain private.

## CLI + skill

`kb setup` is the default integration workflow in v1.1.0 and later. It bundles the
complete version-matched [knowledge-base skill](skills/knowledge-base/SKILL.md),
so normal setup needs no Node or mutable network-fetched skill templates.
Project and global scope are explicit; installing assets is not proof that a
fresh target-agent session has activated them.

The separate optional [Skills installer](https://skills.sh) remains available:

```sh
npx skills add matheus3301/kb --skill knowledge-base
```

That channel requires Node/npm and does not activate a consultation policy.
Preserve existing custom skills and instructions. Follow the
[setup lifecycle guide](docs/agent-setup.md) for version availability,
root/scope/agent/learning selection, preflight, checks and safe uninstall.

### Let your agent set it up

Copy this prompt into your agent:

```text
Set up KB through its setup lifecycle. Read
https://raw.githubusercontent.com/matheus3301/kb/main/docs/agent-setup.md
and https://raw.githubusercontent.com/matheus3301/kb/main/llms.txt as references.
Inspect command -v kb, kb version --json and kb setup --help first.
Setup requires v1.1.0 or later. If installation or upgrade is authorized,
use brew install matheus3301/tap/kb, or brew update followed by
brew upgrade matheus3301/tap/kb for an existing installation. Verify version/help
again. Use the installed executable from my intended agent project.
Preserve existing bases, skills and instructions. Establish an absolute root,
project path or authorized global scope, selected agent IDs, and learning on
or off; ask only for choices not already supplied. Preflight with
kb --root ROOT setup install --project PROJECT --agent AGENT --learning on|off --dry-run
(replace --project PROJECT with --global only when authorized).
Add --create only for an explicitly selected new base; --para is optional.
Review all destinations/conflicts, then apply the same plan using --yes
instead of --dry-run. Run kb setup check with the same scope/agents and --json.
Verify the resolved root and actual search/read with citations in a fresh
agent session. Distinguish configured from ready and activation pending.
Do not bypass host trust, repair a collection or commit/push/sync knowledge.
```

## Command map

```text
kb
├── init --para                    Create a Knowledge Base and optional PARA folders
├── setup install / check          Configure and inspect explicit agent integration
├── setup resolve / uninstall      Resolve bindings or remove owned integration
├── status / validate              Inspect scope, format, and structural health
├── list / search                  Discover notes with filters and pagination
├── read / context / links         Read sources, collect excerpts, and inspect references
├── create / patch                 Add or revise a note
├── move / archive                 Relocate notes and update resolved references
├── apply                          Preview or execute a coordinated plan
├── maintain / index rebuild       Maintain navigation or rebuild derived search state
├── recover / export               Recover a pending transaction or export knowledge
├── backup / restore               Stage and verify exact operational archives
└── mcp / version                  Serve MCP or inspect the installed build
```

The data commands, including backup/restore, have been available since v0.4.0;
setup is available since v1.1.0. Use `kb COMMAND --help` for installed flags.
The [CLI guide](docs/cli.md) contains syntax, JSON requests, errors and limits.

## Project status

**Stable [v1.1.0](https://github.com/matheus3301/kb/releases/tag/v1.1.0) is published
and installed through macOS arm64 Homebrew.** It adds the explicit `kb setup`
lifecycle with bundled CLI skills, scoped bindings, optional native hooks and
safe integration removal.

Actual published native archives and archived source passed verification on
Linux amd64 and macOS arm64 before the tap updated. A real Homebrew upgrade
from v1.0.0, formula test, installed CLI acceptance, **43 data documentation
requests**, **12 setup requests** and CLI/MCP operational consumers passed.
See the [v1.1.0 release record](docs/qa/setup-release-v1.1.0.md) for exact
identities, job outcomes and activation limits.

[Setup acceptance](docs/setup-acceptance.md) records bounded host qualification.
Installing files does not approve native trust or prove fresh-session activation.
The [compatibility policy](docs/compatibility.md#current-qualification-status)
retains the earlier v1.0.0 consumer and resource observations with their original
identities; they are not new v1.1.0 measurements.

Linux arm64 and macOS amd64 remain package/cross-build-only. No arbitrary-editor
isolation, universal power-loss guarantee, LTS or SLA is claimed; see
[compatibility/support](docs/compatibility.md).

## Documentation

- [Documentation index](docs/README.md): task-oriented routes through the existing guides.
- [Quickstart](docs/quickstart.md), [CLI guide](docs/cli.md), and [installation](docs/installation.md).
- [Agent setup](docs/agent-setup.md), [CLI skill](skills/knowledge-base/SKILL.md), and [llms.txt](llms.txt).
- [Architecture](docs/project-overview.md), [specification](docs/spec.md), and [interface contract](docs/interfaces.md).
- [Quality gates](docs/quality.md), [backups](docs/backups.md), and [compatibility/support](docs/compatibility.md).

### Optional MCP interface

MCP is the alternative agent interface and needs neither the CLI skill nor
shell commands for agent operations. The server supplies initialization guidance
and describes inputs, results, previews, revision conflicts, retries and access
boundaries through discovery.

Register a stdio server in your MCP client. Start with read-only access:

```json
{
  "mcpServers": {
    "kb": {
      "command": "kb",
      "args": ["--root", "./example-kb", "mcp", "--read-only"]
    }
  }
}
```

Use an absolute root if the client cannot set the intended working directory.
[MCP guide](docs/mcp.md) explains write-enabled access and exposed operations.
Binary installation, skill installation, and MCP registration are separate actions.

## Contributing

Use the Go version in [go.mod](go.mod), Python 3.12+, and Conventional Commits.
Run `make tools` and `make check`. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE). Third-party dependency notices accompany release archives.
