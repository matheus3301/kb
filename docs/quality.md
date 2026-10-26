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

## Implemented validation and its limits

Published production remains experimental v0.3.1; integrated source is preparing
v0.4.0 qualification with all four roadmap increments implemented. `make check`
runs required application gates, executable documentation examples, the real CLI
acceptance workflow (`make acceptance`), bounded CLI/MCP benchmark smoke and
deterministic retrieval evaluation (`make retrieval-eval`). All product workflows
use disposable synthetic roots, never an operator's live collection.
[Original validation](evidence/validation.json), dated **2026-10-24**, reports
a successful v0.1-era run with 88.4% statement coverage, 102 top-level test
functions, 314 named pass events including subtests and fuzz seeds, seven fuzz
smoke targets, and Linux/macOS amd64/arm64 cross-builds. These are historical
counts, not current source suite totals or final v1.0 evidence.

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

[Performance evidence](performance.md) retains the initial v0.1 core misses:
warm search p95 162.9 ms and peak process RSS 180.3 MiB against 150 ms/150 MiB,
with 594.4 ms cold search. The 2026-10-25 v0.3.1 report records 117.7 ms warm
core p95, 134.0 MiB RSS and 179.0 ms rebuild p95, meeting the original local
targets. Both core runs used Apple M4/macOS arm64, Go 1.27.3, 10,000 synthetic
2 KiB notes and 20 warm queries; rebuild was timed separately in the later run.
These are not fresh CLI/MCP measurements or portable SLAs. That historical
50k-note CLI search measured 2.19 s median/483.1 MiB RSS; the later
[paired matrix](performance-large-corpus.md) reports its own baseline and
candidate rather than substituting this earlier result. No gate was reduced.

Source tests run on **ubuntu-24.04** and **macos-15**; four-target cross-builds
are not four-target runtime tests. The Ubuntu release snapshot executes its
native archive and archived-source build. The
[v0.3.1 release record](https://github.com/matheus3301/kb/releases/tag/v0.3.1)
separately documents actual Homebrew installation/formula and installed CLI/MCP
verification on **macOS arm64 only**. See [compatibility/support](compatibility.md)
for platform boundaries and the [exit checklist](stability.md) for pending
exact-candidate qualification. Implemented retrieval, backup and scale evidence
does not itself satisfy V01–V08. Stable status requires the recorded readiness
decision, not merely a green workflow.

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

## Candidate qualification beyond routine CI

The [retrieval evaluation](qa/retrieval-quality.md) runs 31 search judgments,
eight context cases and eight exact source reads through actual CLI and a
separately initialized stdio MCP consumer. The
[baseline report](evidence/retrieval-quality.json) records macro Recall@5
0.9930556 and MRR@5 1.0, not arbitrary semantic understanding. D01 requires all
deterministic assertions, ceiling-aware per-query recall, no unexplained category
regression and exact citation/budget/parity/purity behavior.

The [backup contract](backups.md) and synthetic CLI/MCP/archive/fault/concurrency
regressions are implemented. Short mixed workloads and 11 observed-boundary
SIGKILL scenarios are not D02's extended observation: at least 30 minutes across
three seeds, four writers and 10,000 acknowledged mutation/replay outcomes, with
**log maintenance disabled and index maintenance retained**. The harness must
record an explicit `--maintain-log` choice and effective config; this is not a
product CLI flag. A separate **default-log-enabled profile requires at least
60 seconds total, three seeds/four writers and 250 acknowledged mutation/replay
outcomes combined**, with backup/restore and restored continued operation.
Report unique effects and replays separately. Both profiles use setup creates,
then sustained patch/CAS/replay/read/search and periodic backup/restore; reads and
search do not count toward mutation/replay minima. Deterministic acceptance/crash
regressions separately cover create/move/archive/apply/preview and coordinated
multi-file publication; timed move/preview loops are not claimed. Repeat each
interruption scenario three times. Neither profile nor its independent review
is claimed passed here; default-log boundaries retain deterministic tests.

Receipt count has no automatic expiry, retention or rotation. Aggregate raw diff
text is capped at **256 KiB per receipt**; growing full-file log diffs can
accumulate roughly quadratic storage only while their images fit that cap.
Excess diff text is truncated, not stored without bound; other serialized receipt
fields remain outside the diff budget. Log/note/archive limits still constrain
operations and rejections must report explicit errors. Monitor/protect that state;
no automatic compaction, format change or altered replay guarantee is implied.

The [large-corpus report](performance-large-corpus.md) retains the paired matrix,
including 10k non-improvements, preview RSS increase and 50k committed-repair
median regression. D03 selects explicit local p95/RSS ceilings with 10% headroom
over the worse paired observation, plus per-operation drift reporting and a
reserved quiet candidate rerun. These are not shared-runner timing gates or a
portable SLA; original core targets and every existing correctness gate remain.

D04 selects conditional v1.x compatibility, synthetic upgrade checks, macOS
arm64 Homebrew and Linux amd64 native archive/source qualification. Cross-built
Linux arm64/macOS amd64 are not installed-qualified. V01–V08 require the exact
candidate and independently reviewed evidence; D01–D04 are delegated policy
choices, not fabricated human approval. Final evidence and status changes belong
in the [finalization checklist](stability.md#finalization-checklist).

Candidate integration additionally requires a **required native v0.3.1-to-candidate
compatibility CI job** using the actual v0.3.1 executable pinned to source commit
`2ca104f48cc4a6551f4bbc4fae12d51432e42dff`, and asserting that candidate build
metadata identifies the checked-out candidate HEAD. Use disposable synthetic
fixtures, not duplicated current implementations or cross-build proof. Exercise
the documented upgrade/client contract and retain evidence without leaking local
workstation paths, including failure reports.
This is an integration requirement, not a claim that the inherited workflow in
the original `a320dea` baseline already ran that job; V01/V06 need the resulting
workflow and exact-candidate passing evidence before finalization.

