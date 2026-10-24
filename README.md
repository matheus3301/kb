# kb

[![CI](https://github.com/matheus3301/kb/actions/workflows/ci.yml/badge.svg)](https://github.com/matheus3301/kb/actions/workflows/ci.yml)

Local-first knowledge memory for humans and agents. Markdown is the source of
truth, [OKF](https://github.com/GoogleCloudPlatform/open-knowledge-format) is the
knowledge format, and PARA is an optional human navigation profile. A Go binary
will provide CLI commands and an MCP stdio server backed by the same core.

**Pre-release: repository and quality foundation only. There is no application,
working `kb` executable, or release yet.** The documentation describes the
proposed interface. Acceptance criteria remain planned until implementation
and evidence exist; green repository checks do not establish product readiness.

Start with the [public quickstart](docs/quickstart.md) for source installation
prerequisites and the proposed create, search, read, context, revision-checked
patch, dry-run, validation, and MCP workflow. Its product commands are explicitly
marked pre-release until implemented.

The planned tool keeps knowledge in editable OKF Markdown files, preserves YAML
comments and unknown metadata, and uses expected revisions to reject conflicting
writes. Coordinated changes require a durable journal and recovery. The external
agent handles synthesis and semantic review; the core handles deterministic
operations. These are requirements, not claims about existing behavior.

Binary installation, optional agent skill installation, and MCP registration
are separate steps. Workdir instructions activate the memory workflow and
identify its root. An `AGENTS.md` inside the knowledge base is optional.

- [Specification](docs/spec.md)
- [Product and architecture](docs/project-overview.md)
- [Research and public sources](docs/research.md)
- [Quality gates and their limits](docs/quality.md)
- [Acceptance evidence](docs/acceptance.json)
- [Development](CONTRIBUTING.md)
- [Implementation decisions](docs/decisions/0001-foundation.md)

For contributors, use the Go version in `go.mod` and Python 3.12 or newer:

```sh
make tools   # install pinned development tools into .tools/bin
make check   # run the checks used by CI
```

CI validates the repository now. Application tests, coverage, fuzzing, and builds
activate when Go source is introduced. The final distribution and public API
must be settled before a release.

This project is open source under the [MIT License](LICENSE).
