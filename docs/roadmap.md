# Roadmap toward v1.0

Status as of **2026-10-25**: the local core is delivered in experimental
[v0.3.1](https://github.com/matheus3301/kb/releases/tag/v0.3.1), source commit
`2ca104f48cc4a6551f4bbc4fae12d51432e42dff`. The four workstreams below are
approved directions, not shipped additions, deadlines, assigned release numbers,
or authorization to publish v1.0. Read the [exit checklist](stability.md) and
[compatibility/support policy](compatibility.md) before relying on readiness claims.

## Released baseline

The CLI and self-contained stdio MCP share native OKF parsing, lexical retrieval,
bounded cited context, revision-checked writes, durable receipts and recovery.
Incremental repair is explicit; normal writes remain strict. Binary installation,
CLI skill installation, MCP registration and workdir activation remain separate.
The [specification](spec.md), [interface contract](interfaces.md),
[acceptance inventory](acceptance.json) and version-specific
[release notes](https://github.com/matheus3301/kb/releases) define current behavior.

The v0.3.1 performance work met the original local 10,000-note core targets:
117.7 ms warm-search p95, 134.0 MiB peak RSS and 179.0 ms reindex p95.
Those observations are not portable SLAs or fresh CLI latency. At 50,000 synthetic
2 KiB notes, fresh CLI search still measured 2.19 s median and 483.1 MiB peak RSS.
The [performance report](performance.md) preserves method, samples and history.
Installed-package proof is macOS arm64; cross-builds do not establish installed
runtime behavior on the other targets.

## 1. Retrieval quality

**Pending implementation and review.** Establish a public, synthetic, annotated
Portuguese/English corpus for accents, ranking, filters, no-match queries and
context composition. Measure top-k relevance, reciprocal rank and recall with
explicit definitions, expected results and actionable failures. Check exact
source revisions/line citations, token/byte ceilings, truncation and query purity.
Exercise actual CLI and MCP on the same fixtures and run a bounded deterministic
evaluation in CI without replacing existing gates.

Record a baseline before changing ranking. Any runtime change needs failure
evidence and must retain lexical semantics; this roadmap does not authorize
embeddings, an embedded model or tuning solely to a few favorable queries.
Completion requires the linked corpus, evaluator, parity results and reviewed
quality thresholds described in the exit checklist, not merely a harness file.

## 2. Reliability and operational backup/restore

**Pending implementation and review.** Extend concurrency and recovery evidence
with longer workloads and actual process interruption. Add an explicit operational
backup/restore workflow preserving authoritative content, configuration and durable
receipt/journal semantics while distinguishing disposable cache and excluded files.
Specify behavior when recovery is pending. Restore only to an explicitly selected
empty destination; reject unsafe entries, overwrites, corrupt or incomplete
manifests, and integrity/confinement failures.

Prove exact restored revisions, idempotent replay and continued normal operation
on synthetic fixtures, with CLI/MCP parity and read-only enforcement. CLI guidance,
independent MCP discovery and skill guidance must agree. **v0.3.1 has no operational
backup/restore command or MCP tool.** Its `export` is a scoped knowledge bundle,
not a backup of configuration, journal or receipts. Do not use a proposed command
from a branch as an installed recovery procedure.

## 3. Larger collections

**Pending implementation and review.** Profile the v0.3.1 baseline at 10,000 and
50,000 synthetic 2 KiB notes. Reduce remaining fresh-process startup and peak RSS
costs with controlled paired measurements declaring hardware, samples, cache
conditions and remaining limits. Preserve retrieval ordering/context semantics,
exact fresh-byte checks, confinement, CAS, durable publication and recovery.

No daemon/database or timestamp-only freshness shortcut is introduced by this
plan. Performance improvements must retain all existing behavior and quality
gates. Results from concurrent benchmark/build/test workloads are not controlled
comparisons; publish the method and uncertainties rather than a portable SLA.

## 4. Stability, compatibility and current documentation

**Documentation increment; readiness remains pending.** Publish this roadmap,
the [v1.0 exit checklist](stability.md) and [compatibility/support policy](compatibility.md).
Keep historical evidence dated, current release status explicit and packaging
claims separated into cross-build, native execution and actual installation proof.
Link evidence for each readiness decision; do not remove experimental warnings
because a plan exists or CI is green.

## Integration and release decisions

Each workstream is a separately reviewable PR. A branch result is not a released
feature. After integration, reconcile retrieval thresholds, operational format
versions/tool catalog, scale measurements and this checklist against the same
candidate commit. Update current documentation only with merged behavior and
observed evidence; preserve historical reports unchanged.

No dates or future version numbers are committed here. The maintainer must review
remaining risks, approve the compatibility commitments and independently assess
all exit criteria before considering v1.0. Merge, tag, publication and an installed
upgrade are separate decisions under [release operations](releases.md).

Out of scope: UI, remote transport, automatic Git sync, telemetry, embedded LLM,
embeddings, legacy compatibility layer and simultaneous multi-base operation.
Health checks and format conformance do not establish factual truth; advisory
locks do not isolate arbitrary external editors.
