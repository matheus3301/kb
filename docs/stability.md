# v1.0 exit criteria

**Candidate qualification; stable-v1 readiness remains pending.** All four
initial [roadmap](roadmap.md) increments are integrated at `a320dea` as of
**2026-10-25**. Published/installed production remains experimental **v0.3.1**.
The next intended qualification release is **v0.4.0**, not yet a published or
installed result here. This checklist extends A01–A26 in the
[acceptance inventory](acceptance.json); no item alone authorizes publication.

## Evidence and status rules

- **Satisfied baseline** is historical evidence for a bounded claim, not sign-off
  on the eventual candidate.
- **Implemented; qualification pending** means source and focused evidence exist,
  but the exact integrated/installed candidate still needs the listed checks.
- **Policy selected** means D01–D04 are delegated coordinator acceptance choices,
  with rationale below. They are not measured results, independent review or
  human approval. Changing a threshold requires a recorded reason and review,
  never silently repinning it to an observed failure.

For each V item, record exact commit/version and binary identity, actual commands
and outcomes, platform/filesystem/toolchain, synthetic fixture hash, samples,
duration/seeds/counts, known limits and an inspectable evidence link. Record the
reviewer's role and scope separately from implementation/self-checks. Do not copy
private collections or transcripts. Coverage, model obedience and green CI are
not proofs of relevance, isolation, durability or readiness.

## Satisfied baseline, not final-candidate sign-off

- [x] **B01 — Local core acceptance.** A01–A26 link tests and historical evidence
  for CLI/MCP parity, metadata preservation, CAS, replay/recovery, confinement,
  purity, bounded context and explicit integration.
  [Original validation](evidence/validation.json) is dated 2026-10-24;
  [independent interface evaluation](qa/independent-interfaces.md) and later
  regression reports extend it without making it final-v1 evidence.
- [x] **B02 — Original local core targets.** The
  [v0.3.1 report](performance.md#original-targets) records 117.7 ms warm p95,
  134.0 MiB RSS and 179.0 ms rebuild p95 against 150 ms/150 MiB/10 s targets:
  Apple M4/macOS arm64, Go 1.27.3, 10,000 synthetic 2 KiB notes. Core RSS and
  end-to-end child-process RSS are distinct; no portable SLA follows.
- [x] **B03 — Released package proof, macOS arm64 only.**
  [v0.3.1 release verification](https://github.com/matheus3301/kb/releases/tag/v0.3.1)
  records extracted binary/Homebrew source installation, formula test, acceptance,
  43 documentation requests and installed MCP on synthetic data. Other archive
  targets are not installed-tested by this record.
- [x] **B04 — Required gates without a readiness bypass.**
  [Quality policy](quality.md) and [CI](../.github/workflows/ci.yml) require
  repository/static checks, Linux/macOS tests, race/fuzz, at least 80% coverage,
  four cross-builds, scans and release snapshot. Original counts of 88.4%,
  102 tests, 314 pass events and seven fuzz targets remain dated observations.

## Candidate acceptance — all pending

- [ ] **V01 — Exact integrated candidate.** Run `make check` and
  `make release-snapshot`; link exact-commit CI gate, application acceptance and
  A01–A26 regressions. Record every failure, skip and limitation; no weakened gates
  or historical counts as current results. All later evidence must identify the
  same candidate or explain which affected checks were rerun after a change.
- [ ] **V02 — Retrieval usefulness and parity.** Implemented:
  [corpus, metric definitions and evaluator](qa/retrieval-quality.md) plus
  [focused evidence](evidence/retrieval-quality.json). Rerun `make retrieval-eval`
  on the candidate; retain per-case/category results and exact corpus hash.
  Apply D01 to all deterministic membership/order/no-match, citation/revision,
  budget/truncation, purity and CLI/MCP assertions. Existing macro metrics are
  baseline evidence, not candidate sign-off or semantic-understanding proof.
- [ ] **V03 — Operational reliability.** Existing
  [mixed-workload and observed-boundary SIGKILL tests](../internal/write/operational_resilience_test.go)
  exercise 24 iterations and 11 interruption scenarios; they are not an extended
  soak. Supply D02's multi-seed concurrent workload and repeated interruption
  results, including recovery interruption, CAS conflicts and identical-key
  retries. No partial acknowledged transaction, duplicate logical effect, lost
  confirmed receipt or changed immutable source is allowed.
- [ ] **V04 — Operational backup/restore.** Implemented:
  [version-1 contract](backups.md), [CLI/MCP parity tests](../cmd/kb/backup_test.go),
  [archive/failure/replay tests](../internal/write/backup_test.go) and parser/path
  fuzz targets. Record candidate synthetic restoration with exact revisions,
  configuration, assets and receipts, idempotent replay and continued writes.
  Exercise pending-state refusal, new-only destinations, unsafe/corrupt archives,
  read-only enforcement, publication faults and interrupted restore retries.
  Pending journals are refused, not preserved in archives. Same-disk staging is
  not disaster recovery; export and disposable cache have different contracts.
- [ ] **V05 — Local resource envelope.** Implemented:
  [quiet paired 10k/50k report](performance-large-corpus.md) and
  [raw measurements](evidence/performance-large-corpus.json). Rerun installed
  v0.3.1 versus the exact candidate in a reserved quiet window under D03, with
  **both 10k and 50k fresh-process index rebuild measured separately for each
  binary**. Report p50/p95, first request/apply and peak RSS
  for fresh CLI, reused MCP, preview/apply and rebuild, with all drift disclosed.
  Link freshness/ranking/confinement/durability regressions. Do not claim uniform
  improvements: existing 10k non-improvements, preview RSS and 50k repair-median
  regression remain visible.
- [ ] **V06 — Compatibility and upgrades.** Apply D04 and the
  [v1.x policy](compatibility.md#conditional-v1x-commitments). Exercise supported
  prior CLI/JSON requests and MCP discovery/workflows against the candidate:
  flags, exits, errors, schemas, read-only and fixed-root behavior. On synthetic
  v0.3.1 fixtures, verify exact content/config/receipt preservation, settled-state
  upgrade, pending recovery with the prior binary, restored replay and subsequent
  writes. Qualify the v0.4.0-to-v1.0.0 path before stable release. Disclose
  downgrade limits and any migration; builds alone do not prove compatibility.
- [ ] **V07 — Packages and consumers.** Under D04, record actual macOS arm64
  Homebrew install/upgrade and formula test, extracted native archive execution,
  and Linux amd64 native archive plus archived-source workflows. Record exact
  version/commit/checksums and CLI acceptance. Exercise CLI + separately installed
  skill, and independently initialized/discovered stdio MCP without skill files.
  Identify client versions and activation limits. Cross-builds and snapshot
  execution are not a substitute for installed Homebrew qualification.
- [ ] **V08 — Independent readiness decision.** Review V01–V07 and all public
  docs/catalogs/skill against one candidate, list findings/remaining risks and
  explicitly decide whether experimental status can be removed. Record a review
  distinct from implementation authorship; do not label automated review as human
  approval. Human approval, when obtained, must be explicit. Green CI or an
  author's self-check cannot satisfy this item. Publication and installed-upgrade
  verification remain separate under [release operations](releases.md).

## Qualification policy — D01–D04

These choices bound what must be observed before stable v1, rather than promise
absence of bugs. They are selected for candidate qualification by delegated
coordination, subject to evidence and independent readiness review.

### D01 — Ceiling-aware lexical retrieval floor

Keep the [existing authored corpus](../testdata/retrieval/corpus.json) and metric
definitions; do not relabel expected documents to fit the output. All exact
membership/order/no-match, source/revision/citation, budget/truncation, purity
and CLI/MCP parity assertions must pass. No category may regress without an
explained, reviewed contract change.

For every positive query with relevant set R, require Recall@5 at least
`min(5, |R|) / |R|`, and RR@5 at least its recorded baseline. Thus the current
24 positive judgments require macro Recall@5 **at least 0.9930555555555555**
(within numeric roundoff) and MRR@5 **at least 1.0**. The six-relevant case has
a **5/6** floor, not an impossible 1.0; the other positive cases fit in five.
All seven negative judgments require zero hits/total and are excluded from
macro means. Report category and per-case values, not aggregates alone.
Rationale: retain the demonstrated lexical contract without inventing semantic
recall or a cutoff-incompatible target. A changed corpus needs a new baseline
and explicit review of floors, not silent comparison across different labels.

### D02 — Bounded extended reliability observation

Require **at least 30 minutes total across three distinct recorded seeds**, at
least ten minutes per seed, with **four concurrent writers per seed** on local
supported filesystems. This extended profile explicitly sets
**`write.maintain_log = false` and `write.maintain_indexes = true`**.
It is not a 30-minute default-log-enabled observation. Across these runs require
**at least 10,000 acknowledged successful mutations and identical-request replays
combined**; report unique committed effects and replay acknowledgments separately.
Previews, expected conflicts and failed attempts do not count toward that minimum.
The qualification harness must expose an explicit `--maintain-log` control and
record effective configuration; this is not a new `kb` command flag.

Complement it with a **default-log-enabled profile of at least 60 seconds total
across three seeds, four writers per seed and at least 250 acknowledged successful
mutations and identical-request replays combined**, retaining index maintenance.
Exercise backup/restore and restored replay/continued writes in that profile too.
Report unique committed effects, replays and elapsed time separately for both
profiles; neither has passing candidate evidence here.

Use setup creates to establish a fixed corpus, then sustain **patches, conflicting
CAS, identical-key replay, reads/search and periodic backup/restore**. Reads/search
do not count toward either mutation/replay minimum. Retain exact outcomes and
immutable-source hashes; verify receipts/replays, final inventory and continued
writes after restoration. Timed move/preview loops are not claimed. Deterministic
acceptance/crash regressions separately cover create, move, archive, apply,
preview and coordinated multi-file publication.

Repeat the existing 11 real observed-boundary SIGKILL scenarios **at least three
times each**, in addition to injected publication/recovery faults and concurrency
tests. Report timeouts/unobserved boundaries as failures, not successful kills.

Rationale: durable receipts retain full-file diffs. Repeatedly extending the
retained event log can cause quadratic aggregate receipt storage, so the extended
profile disables logging explicitly, not receipt durability or index maintenance.
The shorter default-log profile and existing deterministic default-log
publication/recovery tests complement it; independent review must assess this
scope. There is no approved receipt-format/expiry change or automatic compaction.
Operators must account for event-log/receipt growth; deleting receipts would
break the replay contract. Both profiles are finite observations, not universal
crash/power-loss, network-filesystem, hostile same-user or arbitrary-editor
isolation proof.

### D03 — Measured local resource budgets, not a portable SLA

Use the [paired report](evidence/performance-large-corpus.json)'s Apple M4,
macOS arm64, Go 1.27.3, `GOMAXPROCS=8`, synthetic 2 KiB-note method: serialized
10k/50k baseline/candidate runs, at least five read and three repair samples,
fresh CLI children, one reused MCP process, child-only RSS, no overlapping
build/test/soak/profile work and no claimed OS cache eviction.

The local candidate ceilings below are **110% of the worse recorded
baseline/candidate p95 or RSS, rounded up to the next 10 ms or 10 MiB**.
Ten percent is explicit qualification headroom, not an empirically established
noise interval. It accommodates the disclosed trade-offs without claiming every
operation improved. MCP RSS is the whole-session peak, not per-request memory.

| Operation | 10k p95 ceiling (ms) | 10k RSS ceiling (MiB) | 50k p95 ceiling (ms) | 50k RSS ceiling (MiB) |
| --- | ---: | ---: | ---: | ---: |
| CLI search | 380 | 130 | 2870 | 550 |
| CLI read | 200 | 110 | 2070 | 470 |
| CLI validate | 190 | 120 | 1910 | 470 |
| CLI repair preview | 730 | 200 | 3770 | 830 |
| CLI committed repair | 1460 | 230 | 8940 | 1050 |
| Warm MCP search | 300 | 150 | 1530 | 640 |
| Warm MCP read | 110 | 150 | 1540 | 640 |
| Warm MCP validate | 110 | 150 | 1690 | 640 |

Report every operation's baseline/candidate p50/p95, RSS, first request/apply and
absolute/percentage drift, even inside budget. Disclose the existing 10k
search/read non-improvements, 50k repair median +4.9% and preview RSS increase.
An exceedance blocks V05 until a quiet repeat and explicit evidence-backed
review; do not silently change ceilings. Results on different hardware are
informational, not pass/fail against these machine-specific numbers.
The original **core-only 10k 150 ms warm p95 / 150 MiB / 10 s rebuild targets
remain unchanged** and need separately identified measurements. The paired
report did not measure a new rebuild improvement. No Linux timing, broader SLA
or unmeasured retained-memory improvement is claimed.

### D04 — Conditional v1.x compatibility and support

Adopt the [documented CLI/JSON/MCP commitments](compatibility.md) for v1.x only
after V06–V08 qualify them. Scope includes native pinned OKF v0.2, configuration
version 1, settled version-1 operational state/receipts and version-1 backup
archives. Qualify v0.3.1 → v0.4.0 → v1.0.0 on synthetic fixtures, preserving
unknown metadata and exact revisions; pending journals need the prior binary's
recovery first. No blanket older-version upgrade or downgrade promise.

Required native channels: **macOS arm64 Homebrew** (including formula and
installed CLI/MCP) and **Linux amd64 native archive/archived source**.
Linux arm64 and macOS amd64 remain **cross-build/package-only**, not
installed-qualified. No Windows, remote transport, network-filesystem support,
LTS lifetime, response SLA or automatic backport pledge. Rationale: commit to
interfaces and channels that can be exercised, not infer support from packaging.

## Finalization checklist

Update this section in a later docs-only change when exact evidence exists:

- [ ] Identify the integrated candidate commit, actual v0.4.0 release and binary
  hashes; link exact-commit CI/full-check/snapshot results without private paths.
- [ ] Attach V02 retrieval, V03 reliability, V04 restoration, V05 quiet paired
  matrix/rebuild and V06 upgrade results with counts, seeds and disclosed drift.
- [ ] Attach V07 installed Homebrew/native Linux and independent consumer evidence;
  state channel/client limits and distinguish source/snapshot/installed checks.
- [ ] Record V08 reviewer identity/role, scope, findings, disposition and explicit
  readiness decision. Do not invent independent human approval.
- [ ] Reconcile README, llms.txt, spec, interfaces, installation, quickstart,
  roadmap, quality, compatibility, decision summaries and release guidance;
  preserve historical reports and update status only to observed facts.
- [ ] If stable publication is authorized, select unused v1.0.0, qualify any
  intervening changes, publish under the normal protected workflow and verify
  actual installed artifacts. Add only real release/evidence URLs, then update
  current status with observed version/commit and remaining support limits.

