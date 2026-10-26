<div align="center">
  <img src="assets/logo.svg" alt="Knowledge Base logo" width="104" height="104" />
  <h1>KB — Knowledge Base</h1>
  <p><strong>A local Knowledge Base for humans and AI agents.</strong></p>
  <p>Readable Markdown. A revision-aware CLI. A shared MCP interface.</p>
</div>

<p align="center">
  <a href="https://github.com/matheus3301/kb/actions/workflows/ci.yml"><img src="https://github.com/matheus3301/kb/actions/workflows/ci.yml/badge.svg" alt="CI" /></a>
  <a href="https://github.com/matheus3301/kb/blob/main/go.mod"><img src="https://img.shields.io/github/go-mod/go-version/matheus3301/kb" alt="Go version" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-3662EA" alt="MIT license" /></a>
  <a href="#project-status"><img src="https://img.shields.io/badge/status-stable-516276" alt="Stable" /></a>
</p>

`kb` lets you search, read, and update a Knowledge Base stored as
[OKF](https://github.com/GoogleCloudPlatform/open-knowledge-format) Markdown.
Use any editor to read the files. Use the CLI or MCP server to make changes
with revision checks, previews, and recoverable transactions. Optional PARA
folders keep projects, areas, resources, and archives easy to navigate.

## Getting started

Install on macOS or Linux with Homebrew, then initialize a new directory:

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

For an existing collection, initialize the same directory, then run
`kb validate --profile okf`. Agents can [repair diagnosed issues in batches](skills/knowledge-base/references/repair.md)
with `kb apply --repair --incremental`; initialization does not convert notes.

## At a glance

- **Search and read:** lexical ranking, filters, sections, and bounded excerpts
  with source paths, revisions, and line references.
- **Write:** create, patch, move, and archive notes; preview changes and reconcile
  revision conflicts before replacing content.
- **Coordinate:** apply multi-note plans with idempotent retries and an explicit
  recovery journal.
- **Maintain:** inspect links, validate format/health, update managed navigation,
  export validated knowledge snapshots, and stage operational backup/restore.
- **Connect:** choose CLI + the `knowledge-base` skill, or a self-contained stdio
  MCP server. The binary needs no model, Node, Git, or network at runtime.

## CLI + skill

Install the [knowledge-base skill](skills/knowledge-base/SKILL.md) from the
agent's workdir using [Skills](https://skills.sh):

```sh
npx skills add matheus3301/kb --skill knowledge-base
```

The skill teaches agents how to use the CLI to consult the Knowledge Base, record
reusable learning, and handle revisions, retries, and recovery. Its
references travel with the installed skill. Configure `KB_ROOT` for CLI access
and add a deliberate workdir instruction to use the skill. Start a new agent
turn/session after setup and verify consultation of the intended root.
Installation alone does not guarantee consultation on every question.

See [setup and upgrading the renamed skill](docs/agent-setup.md). Earlier
releases used the installation name `knowledge-memory`.

## MCP

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

## Command map

```text
kb
├── init --para                    Create a Knowledge Base and optional PARA folders
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

This command map, including backup/restore, has been available since v0.4.0.
Use `kb COMMAND --help` for installed flags. The [CLI guide](docs/cli.md)
contains syntax, JSON requests, errors and operating limits.

## Project status

**Stable [v1.0.0](https://github.com/matheus3301/kb/releases/tag/v1.0.0) is published
and installed through macOS arm64 Homebrew.** Actual published native archives
and archived source passed verification on Linux amd64 and macOS arm64 before
the tap update. The Homebrew upgrade, formula test, installed CLI acceptance,
43 documentation requests and both prior-version upgrade workflows passed.

Fresh controlled installed **CLI + knowledge-base skill** and independently
descriptive **stdio MCP** consumers both passed, including separate learning/query
contexts and byte-pure no-write follow-ups. MCP needed neither the skill nor CLI
data commands. See [release/installed identities, consumer results and limits](docs/compatibility.md#current-qualification-status).
Independent automated review approved the bounded delivery and documentation.
The [delivery record](docs/qa/v1-delivery.md) retains evidence, earlier failed
attempts and the normal protected-integration conditions.

The unchanged runtime source has completed bounded local candidate qualification,
including the 30-minute log-disabled/index-enabled profile and a separate shorter
default-log companion. Those observations do not establish portable timing or
capacity for arbitrary receipt histories. The [qualification record](docs/qa/v1-qualification.md)
retains the prepublication evidence and failed history; the [exit checklist](docs/stability.md)
tracks final closure separately from publication.

Linux arm64 and macOS amd64 remain package/cross-build-only. No arbitrary-editor
isolation, universal power-loss guarantee, LTS or SLA is claimed; see
[compatibility/support](docs/compatibility.md).

## Documentation

- [Quickstart](docs/quickstart.md), [CLI guide](docs/cli.md), and [installation](docs/installation.md).
- [Agent setup](docs/agent-setup.md), [MCP guide](docs/mcp.md), [CLI skill](skills/knowledge-base/SKILL.md), and [llms.txt](llms.txt).
- [Architecture](docs/project-overview.md), [specification](docs/spec.md), and [interface contract](docs/interfaces.md).
- [Quality gates](docs/quality.md), [retrieval evaluation](docs/qa/retrieval-quality.md), [backups](docs/backups.md), [performance](docs/performance.md), and [release operations](docs/releases.md).
- [Roadmap](docs/roadmap.md), [v1.0 exit criteria](docs/stability.md), and [compatibility/support](docs/compatibility.md).

## Contributing

Use the Go version in [go.mod](go.mod), Python 3.12+, and Conventional Commits.
Run `make tools` and `make check`. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE). Third-party dependency notices accompany release archives.
