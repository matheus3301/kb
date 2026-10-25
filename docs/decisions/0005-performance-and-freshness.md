# 0005 — Faster retrieval with exact freshness

Status: adopted and released in v0.3.1 on 2026-10-25
([release record](https://github.com/matheus3301/kb/releases/tag/v0.3.1)).
The [performance report](../performance.md) retains the pre-release paired
measurements; the release records subsequent macOS arm64 installed verification.
The original local core targets were met. The subsequent integrated
[allocation-reduction increment](../performance-large-corpus.md) preserves these
freshness/publication decisions and discloses both gains and regressions.
Current source is preparing v0.4.0 qualification; production remains experimental
v0.3.1 and [stable-v1 exit evidence](../stability.md) remains pending. No portable
SLA follows from these local observations. The historical decision below is unchanged.

Profiling v0.3.0 found filesystem calls dominating warm inventory checks, a
second full scan for response metadata, eager graph construction for operations
that do not need relationships, and repeated parsing/directory listing in plans.

Keep Markdown as source of truth and read current scoped bytes on every relevant
operation. Do not use mtime/size as evidence of unchanged content, persist a
trusted parsed-note cache, require a daemon/database, or relax write barriers.

- Inventory uses short-lived confined directory handles and at most eight read
  workers. Reads finish before those handles close. Directory identities are
  checked before and after traversal; symlinks/replacements fail or are reported.
  Errors retain traversal order rather than worker completion order.
- Compare fresh bytes against the in-process snapshot. Exact unchanged inventories
  reuse the generation; valid unchanged notes can reuse their parsed metadata.
  Temporal and synthesis-dependency flags still refresh. Added/removed empty
  files, same-size/same-mtime edits and scan diagnostics invalidate appropriately.
- Build relationship graphs only for requested links, health/context neighborhoods
  and full writer snapshots. Search/read/OKF validation avoid unrelated work.
- Response metadata uses the result's generation and a lightweight marker check;
  it does not scan a newer inventory just to decorate an older result. Metadata
  failure cannot hide a confirmed receipt.
- Stream normalized lexical words and intern compact term keys without retaining
  entire folded documents through small substrings. Preserve Unicode folding,
  accents, weights and ordering; malformed UTF-8 uses the existing reference path.
- Plan maps share immutable byte images and replace changed images. Validation
  memoization is request-local, keyed by path and exact content revision, and
  retains only needed findings/title/description. Parent inspection lists each
  shared directory once during planning. Targets/publication still receive fresh
  confinement checks, inventory checks and revision comparisons; journaling,
  fsync, recovery and idempotency remain unchanged.

The [benchmark harness](../../scripts/benchmark.py) measures fresh CLI processes,
reused stdio MCP, process RSS and preview/committed repair. CI runs a small
synthetic smoke corpus and retains metrics. Machine-dependent latency thresholds
are not fabricated from shared-runner timing; source behavior tests guard the
specific duplicated-work regressions and correctness invariants. Larger paired
runs and profiles provide performance evidence separately.
