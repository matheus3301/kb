# Quality gates

The required GitHub status is **CI gate**. It depends on repository checks,
static analysis, tests on Linux and macOS, cross-build checks, secret and
vulnerability scans, and dependency review for PRs. It must fail when a required
dependency fails, is cancelled, or is unexpectedly skipped. No path filter can
silently bypass this workflow.

## Enforced by CI

| Gate | Policy |
| --- | --- |
| Repository | Local Markdown/llms.txt links, self-contained CLI skill references, runtime MCP catalog parity, complete acceptance inventory, evidence paths, and regression tests for the quality tooling. |
| Workflows | actionlint; actions pinned to full SHAs; read-only token; no pull_request_target or continue-on-error. |
| Go hygiene | gofmt, go vet, go mod tidy -diff, go mod verify, golangci-lint. |
| Analysis | staticcheck, gosec, errcheck, resource and error checks; specific explained lint suppressions. |
| Tests | Real named tests; race detector; shuffled order; fresh execution; five-minute timeout. |
| Coverage | At least 80% of statements over all application packages. No exclusions. |
| Portability | Linux amd64/arm64 and macOS amd64/arm64 cross-builds, with CGO disabled. |
| Fuzz | Every discovered Go fuzz target runs for a bounded smoke period; seed corpus also runs in ordinary tests. |
| Secrets | Gitleaks scans Git history with redacted findings. |
| Release snapshot | Pinned GoReleaser config, four binary packages and source archive, checksums, privacy, source formula, and real native/source-build workflows. |
| Vulnerabilities | govulncheck; dependency review blocks newly introduced high/critical known vulnerabilities. |

Tool versions are pinned. Dependabot proposes Action and Go module updates;
updates to development tools are separate explicit changes. No automatic merge
is configured. Release publication requires an explicit version tag and follows
the [tag-only workflow](releases.md).

## Implemented v0.1 validation and its limits

The application is implemented. `make check` runs the required application
checks plus executable documentation examples and the real CLI acceptance workflow (`make acceptance`). The latter
builds `cmd/kb`, operates on disposable synthetic files, and checks revisions,
coordinated ingestion, query purity, repair, references, export, and source
immutability. [Recorded validation](evidence/validation.json) reports a successful
full run with 88.4% statement coverage, 102 top-level test functions, 314 named
pass events including subtests and fuzz seeds, seven fuzz smoke targets, and
Linux/macOS amd64/arm64 cross-builds. These counts describe that recorded run.

The [CLI workflow evidence](evidence/cli-workflow.json) and
[controlled agent-consumer observation](evidence/consumer/consumer-evidence.json)
provide additional behavior evidence. The consumer explicitly loaded the locally
installed skill in one session; automatic activation across models is not proven.
The [independent interface evaluation](qa/independent-interfaces.md) separately
exercises CLI + skill and MCP initialization/tool guidance without the skill.
Evidence supports tested behavior, not the absence of bugs, universal crash
safety, or semantic correctness.

The quality tooling also tests itself with temporary synthetic Go packages,
including rejection of missing tests and low coverage. These tooling fixtures
are separate from application validation. Without Go, `make repo` skips its
Go-dependent regression cases; that partial run is not a full `make check`.

The [acceptance inventory](acceptance.json) is the source for criterion labels.
A criterion may be marked `verified` only with existing test and evidence paths.
The repository check verifies traceability, not whether the evidence proves the
claim. Do not equate aggregate coverage or one green workflow with all criteria
being satisfied.

[Performance evidence](performance.md) records unmet initial targets: warm core
search p95 162.9 ms versus 150 ms, and peak process RSS 180.3 MiB versus 150 MiB.
Cold core search was 594.4 ms. These measurements used 10,000 synthetic 2 KiB
notes on Apple M4/macOS arm64 with Go 1.27.3 and 20 warm samples. Reindexing was
not independently benchmarked. No quality threshold is reduced by these results.

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

## Evidence required for changes

Use synthetic fixtures for CLI/MCP parity, malformed YAML, unknown metadata,
comment preservation, path traversal, symlinks, CAS conflicts, idempotent retries,
recovery at publication boundaries, external edits and read-only enforcement.
Add fuzz targets for parsing and path resolution and concurrency tests for the
transaction layer. Measure performance against the declared corpus and hardware;
do not convert the spec's performance targets into reported results.

Branch protection was established during repository bootstrap; its API record,
workflow URL, and observed limits are recorded in the bootstrap PR. Recheck
settings when repository policy changes. PR CI remains read-only. The sole
write-token exception is the tag-only publisher after its verification job: it
requires protected-main ancestry, exact-commit CI, and another full check.
Regression tests reject write tokens in other jobs/files or removal of those
prerequisites. The tap token is separate and scoped to the tap repository.
[Release operations](releases.md) documents credentials and publication.
There is no automatic merge and merging code does not create a release.

## Performance workflow

`make benchmark-smoke` runs the actual binary on a bounded synthetic collection
and records CLI/MCP latency, process RSS and preview/committed repair. It is part
of `make check` and the Linux/macOS test jobs. The harness has its own behavior
tests. CI artifacts retain the measurements; shared-runner timing is not treated
as a portable SLA. Large paired runs use `scripts/benchmark.py` and the declared
method in [performance measurements](performance.md). Correctness regressions
cover fresh bytes, directory replacement, immutable images and request-local
validation; optimizing must not disable existing quality or durability gates.
