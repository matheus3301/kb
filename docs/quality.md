# Quality gates

The required GitHub status is **CI gate**. It depends on repository checks,
static analysis, tests on Linux and macOS, cross-build checks, secret and
vulnerability scans, and dependency review for PRs. It must fail when a required
dependency fails, is cancelled, or is unexpectedly skipped. No path filter can
silently bypass this workflow.

## Enforced by CI

| Gate | Policy |
| --- | --- |
| Repository | Local Markdown links, complete acceptance inventory, evidence paths, and regression tests for the quality tooling. |
| Workflows | actionlint; actions pinned to full SHAs; read-only token; no pull_request_target or continue-on-error. |
| Go hygiene | gofmt, go vet, go mod tidy -diff, go mod verify, golangci-lint. |
| Analysis | staticcheck, gosec, errcheck, resource and error checks; specific explained lint suppressions. |
| Tests | Real named tests; race detector; shuffled order; fresh execution; five-minute timeout. |
| Coverage | At least 80% of statements over all application packages once Go source exists. No exclusions. |
| Portability | Linux amd64/arm64 and macOS amd64/arm64 cross-builds, with CGO disabled. |
| Fuzz | Every discovered Go fuzz target runs for a bounded smoke period; seed corpus also runs in ordinary tests. |
| Secrets | Gitleaks scans Git history with redacted findings. |
| Vulnerabilities | govulncheck once Go source exists; dependency review blocks newly introduced high/critical known vulnerabilities. |

Tool versions are pinned. Dependabot proposes Action and Go module updates;
updates to development tools are separate explicit changes. No automatic merge
or release is configured.

## Bootstrap honesty

There is no Go application yet. Go-specific scripts report **NOT IMPLEMENTED**
when there is no source. Configuration, workflow lint, secret scan and repository
tests still execute. This is not an application test pass and does not establish
coverage. The first Go file activates the Go checks automatically and requires
actual tests. Fuzz targets are required when parser/path handling lands, rather
than inventing one for an empty program.

The acceptance inventory starts with every criterion `planned`. A criterion can
be marked `verified` only with existing test paths and evidence paths. This is a
traceability check: CI cannot determine whether the evidence proves the claim.

## GitHub settings

Main must require a PR, the current **CI gate** status, an up-to-date branch,
resolved conversations, and linear history. Force pushes and deletion are
disabled, and administrators are subject to the rules. Only squash merge is
enabled. Actions cannot approve PRs or write repository content by default.

CODEOWNERS identifies the maintainer. Required approval count is zero while the
sole maintainer's identity is also used to create agent commits and PRs: GitHub
does not let a PR author approve their own PR. Human review remains recommended,
especially for write/recovery behavior and policy changes. A second reviewer or
separate bot identity is needed to enforce independent approval without locking
every PR. Changing a quality gate is itself possible through a PR and remains
a review concern; these checks are not a proof against a malicious maintainer.

## Evidence required as implementation progresses

Use synthetic fixtures for CLI/MCP parity, malformed YAML, unknown metadata,
comment preservation, path traversal, symlinks, CAS conflicts, idempotent retries,
recovery at publication boundaries, external edits and read-only enforcement.
Add fuzz targets for parsing and path resolution and concurrency tests for the
transaction layer. Measure performance against the declared corpus and hardware;
do not convert the spec's performance targets into reported results.

Branch protection must be verified after repository creation. The API record,
successful workflow URL, and observed limits are recorded in the bootstrap PR.
