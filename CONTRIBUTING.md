# Development

Use the Go version in [go.mod](https://github.com/matheus3301/kb/blob/main/go.mod), Python 3.12 or newer, Git, and Make.
Development tooling is separate from the standalone runtime binary.

```sh
make tools
make check
```

`make check` runs repository validation, static analysis, race-enabled application
tests with at least 80% statement coverage, four CGO-free cross-builds, fuzz smoke
tests, secret/vulnerability scans, and the real CLI workflow. See
[quality policy](docs/quality.md) for exact gates and limitations.

`make repo` runs repository validation and tooling regression tests; Go-dependent
regression cases are skipped without a Go toolchain. Such a partial run does not
validate the application. `make lint`, `make test`, `make build`, `make fuzz`,
`make security`, and `make acceptance` expose individual checks. Tool versions
are recorded in `scripts/tools.json` and `.golangci-lint-version`; local tool
binaries and generated `.build/` artifacts are ignored by Git.

For a focused build or local experiment:

```sh
go build -o .build/kb ./cmd/kb
make acceptance
```

Use only synthetic fixtures and public sources. Keep Markdown, comments,
examples, and descriptions in English; label deliberate multilingual search
fixtures as synthetic. Do not add private knowledge, credentials, or workstation
paths to tests, logs, or public evidence.

Open a PR describing the problem, resulting behavior, validation, affected
acceptance criteria, and limitations. Add regression evidence for reproducible
bugs. Read the [interface contract](docs/interfaces.md) before changing DTOs or
transaction behavior. Record architecture changes under `docs/decisions`.
Acceptance labels require existing test and evidence paths; coverage alone does
not establish conformance, isolation, durability, or semantic correctness.

Keep code and policy changes reviewable. Do not weaken checks as an incidental
part of implementing a feature. Changes to workflows, policy, and agent
instructions are covered by CODEOWNERS and the same required CI. Independent
approval count remains zero because the sole maintainer also authors PRs;
independent review is recommended, especially for recovery and policy changes.

Release publication starts with an explicitly requested new version tag after
required main CI. Install the GoReleaser version pinned in `.tool-versions`, then
validate packaging locally without publishing:

```sh
make release-snapshot
```

This checks the GoReleaser config, builds four binary archives and a source
archive, verifies checksums/metadata/docs/licenses, generates the Homebrew
source formula, and exercises the extracted binary and archived source build.
PR CI requires this job as part of CI gate. It does not install globally,
create a tag, publish a release, or update the tap. See
[release operations](docs/releases.md) for the dedicated tag workflow and
required tap credential. Do not reuse the withdrawn v0.1.0 version.

There is no automatic merge. Performance changes must retain existing
correctness checks and report measurements using the
[documented benchmark](docs/performance.md).
