# kb

[![CI](https://github.com/matheus3301/kb/actions/workflows/ci.yml/badge.svg)](https://github.com/matheus3301/kb/actions/workflows/ci.yml)

Local-first knowledge memory for humans and agents. Markdown is the source of
truth, [OKF](https://github.com/GoogleCloudPlatform/open-knowledge-format) is the
knowledge format, and PARA is an optional human navigation profile. One Go binary
provides CLI commands and an MCP stdio server backed by the same core.

**v0.1.0 is an experimental first release.** The implementation supports search,
reading, bounded context, revision-checked writing, coordinated plans, recovery,
validation, navigation maintenance, and export. Use the
[quickstart](docs/quickstart.md) to install from source and try an example base.
Download platform archives and checksums from the
[v0.1.0 release](https://github.com/matheus3301/kb/releases/tag/v0.1.0), or build
from the public source checkout below.

```sh
git clone https://github.com/matheus3301/kb.git
cd kb
go install ./cmd/kb
kb --root ./example-knowledge init --para
```

Use the Go version in [go.mod](https://github.com/matheus3301/kb/blob/main/go.mod) and put its binary installation directory
on your `PATH`. Linux and macOS amd64/arm64 are the release build targets.

`kb` preserves untouched YAML metadata and comments, rejects stale expected
revisions, and journals coordinated file changes for recovery. External readers
can observe intermediate multi-file states; external editors do not honor its
advisory lock. The [validation evidence](docs/evidence/validation.json) records
observed behavior, not a guarantee that all bugs are absent.

The external agent handles synthesis and semantic review. The core performs
local deterministic operations without an embedded LLM, telemetry, automatic
Git synchronization, or a required cloud service. Original sources remain
outside the knowledge bundle. Native OKF notes are required; legacy collection
migration is not included.

Binary installation, optional [agent skill](skills/knowledge-memory/SKILL.md)
installation, and MCP registration are separate steps. Workdir instructions
activate the memory workflow and identify the root. An `AGENTS.md` inside the
knowledge base is optional; installation does not change global agent settings.

The initial performance targets are **not met**. On 10,000 synthetic 2 KiB notes,
the recorded warm core search p95 is 162.9 ms and peak process RSS is 180.3 MiB,
against targets of 150 ms and 150 MiB. See [measurement conditions and limits](docs/performance.md).

- [Quickstart](docs/quickstart.md)
- [Product and architecture](docs/project-overview.md)
- [Specification](docs/spec.md) and [v0.1 interface contract](docs/interfaces.md)
- [Quality gates and their limits](docs/quality.md)
- [Acceptance inventory](docs/acceptance.json) and [recorded validation](docs/evidence/validation.json)
- [Research and public sources](docs/research.md)
- [Development](CONTRIBUTING.md) and [security](SECURITY.md)
- [v0.1 implementation decisions](docs/decisions/0002-v0.1.md)

For contributors, use Python 3.12 or newer alongside the pinned Go toolchain:

```sh
make tools   # install pinned development tools into .tools/bin
make check   # repository checks, analysis, tests, builds, fuzzing, and CLI workflow
```

This project is open source under the [MIT License](LICENSE).
