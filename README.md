# kb

[![CI](https://github.com/matheus3301/kb/actions/workflows/ci.yml/badge.svg)](https://github.com/matheus3301/kb/actions/workflows/ci.yml)

Local-first knowledge memory for humans and agents. Markdown is the source of
truth, [OKF](https://github.com/GoogleCloudPlatform/open-knowledge-format) is the
knowledge format, and PARA is an optional human navigation profile. CLI and MCP
will share one core.

**Status: repository and quality foundation. There is no application yet.**
Go is the implementation baseline. An installed skill will teach memory
workflows; the workdir's instructions will activate them. An `AGENTS.md` inside
the knowledge base is optional.

- [Specification](docs/spec.md)
- [Product and architecture](docs/project-overview.md)
- [Research](docs/research.md)
- [Quality gates and their limits](docs/quality.md)
- [Acceptance evidence](docs/acceptance.json)
- [Development](CONTRIBUTING.md)
- [Implementation decisions](docs/decisions/0001-foundation.md)

```sh
make tools   # install pinned development tools into .tools/bin
make check   # same checks used by CI; requires Go and Python 3.12+
```

CI validates the repository now. Application tests, coverage, fuzzing, and builds
activate when Go source is introduced. A green bootstrap build does not mean the
product or the acceptance criteria have been implemented.

No release or working `kb` executable is available. The final distribution and
public API will be settled before a release.

This is an open-source project licensed under the [MIT License](LICENSE).
