# v1.0 exit criteria

**v1.0.0 is published, Homebrew-installed, consumer-qualified and independently
reviewed.** [PR25](https://github.com/matheus3301/kb/pull/25) records the required
exact-head and merged-main checks for the final documentation integration.
The [actual delivery report](qa/v1-delivery.md) binds released source
`0a6d04d3c25840b2c71c0b98d50478dc6749879a`, published native/archive-source guards,
tap publication and the actual macOS arm64 upgrade/formula test.
The qualified local executable remains `v1.0.0-candidate.5`, Go/module source
`cb66fd77a120d8c8531f9077a32d2bf51ef7921c`, SHA-256
`59d1b477031792cdf539aa3777fafc3942403b80aa8691a3403a8732bf928d46`,
CGO=0. All **72 Go/module files** match the actual published source archive.
Candidate, published-native and Homebrew-built executable hashes remain distinct.

The [current qualification report](qa/v1-qualification.md) and
[source/build manifest](evidence/v1-qualification.json) record actual local full
gates/snapshot, fresh 47-pair retrieval, pinned prior-version upgrades, 105
observed test-child kills, a passing 1,200-ACK default-log companion and one
complete passing local 16-row/three-core-target resource matrix.
**The completed extended D02 observation passed: 34,840 ACK, 27 backup/restores,
zero errors and three seeds each over 600 active seconds.** Exact final PR/main
CI and all five guarded release jobs passed. Independent automated OMP
prepublication review accepted bounded source/evidence; F1 is closed.
Final independent installed consumers passed **CLI+skill 17/17 and MCP-only
20/20 checks**, with bounded independent result acceptance. All six earlier
failed consumer attempts retain their outcomes and specific attribution limits.

[Failure history](evidence/v1-qualification-history.json) retains all three failed
7ceed matrices, the failed source-b71 CGO=1 matrix, the original backup-timeout
endurance and candidate.3's failed 33,680-ACK restore-timeout attempt. The older
7ceed long pass remains historical only. No counts, clocks, old passing cells or
test-only source equivalence erase a failure or qualify another executable.
The committed restore correction and actual candidate.4 6k-receipt scale evidence
have [separate source/artifact attribution](evidence/v1-restore-production.json);
the old prototype's error-path overclaim is not retroactively fixed.

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

## Candidate acceptance — observed evidence and remaining gates

- [x] **V01 — Exact integrated candidate.**
  [The source/build manifest](evidence/v1-qualification.json) binds 72 matching
  Go/module files, harness hashes and the actual candidate.5 executable.
  Local full gates passed **93 Python tests, 88.8% Go coverage, race/scans,
  12 fuzz targets, four cross-builds and 43 documentation requests**; actual
  native/archive-source snapshots passed. Exact [final PR24 CI](https://github.com/matheus3301/kb/actions/runs/40287161946)
  and [released-main CI](https://github.com/matheus3301/kb/actions/runs/40287653868)
  passed; actual published/installed identities are separately bound in the
  [delivery evidence](evidence/v1-delivery.json), not relabeled candidate metrics.
- [x] **V02 — Retrieval usefulness and parity.**
  [Fresh candidate.5 evidence](evidence/v1-retrieval.json) passed 47 CLI/MCP pairs
  and 17 category validations on unchanged corpus
  `562af0124822b42e72125ae9307e5bc5a5fe9fb0f1164be13a8811aa6d49cc4c`.
  All 31 searches, eight context cases, eight read pairs and authored category
  membership remain inspectable. The 24 positive/seven negative judgments retain
  Recall@5 **0.9930555555555555** and MRR@5 **1.0**; negative-only means are null.
  D01 floors, labels, membership/order/no-match and parity were not changed.
- [x] **V03 — Operational reliability.**
  [Current completed observations](evidence/v1-reliability.json) include the
  default-log/index-enabled companion: **600 mutations + 600 replays = 1,200 ACK**,
  360 safe conflicts, six backup/restores, six restored writes and twelve restored
  replays, zero unexpected errors. Three repetitions passed **105 observed
  test-child SIGKILL cases** in 22.336 package seconds, with zero failures/skips.
  The completed candidate.5 extended profile passed **17,420 mutations + 17,420
  replays = 34,840 ACK**, 10,452 expected conflicts and 27 backup/restores.
  Seeds 1729/2718/31415 performed **608.568/616.662/615.944 active seconds**.
  No historical pass, setup/restored work, conflict or failed attempt counts
  toward the ACK gate. These test children do not prove arbitrary public-CLI
  interruption or hardware power-loss safety.
- [x] **V04 — Operational backup/restore.**
  Fresh candidate.5 companion and [prior-produced archive](evidence/v1-compatibility.json)
  observations passed exact authority/config/receipt/source fidelity, replay and
  continued writes. The [committed production review and native 6k scale](evidence/v1-restore-production.json)
  separately identify source equivalence and the actual candidate.4 artifact.
  The original worker-cleanup defect is closed for reviewed production scope,
  not retroactively for its prototype. Candidate.5 completed **27** extended
  fidelity checkpoints, 27 restored writes and 54 restored replays; maximum
  observed restore was **24.227 seconds**, with at most **5,889 checkpoint files**.
  Pending journals are refused, not archived; finite local observations and
  same-disk staging are not universal disaster-recovery guarantees.
- [x] **V05 — Local resource envelope.**
  [One complete fresh candidate.5 matrix](evidence/v1-resources.json) passed all
  16 unchanged latency/RSS rows and three core targets: 20 read, three repair,
  five rebuild and 20 core samples, eight fresh four-worker quiet proofs.
  Core warm p95 **130.2 ms / 130.4 MiB** and fresh 10k CLI rebuild p95
  **177.464 ms** met the original targets. Independent actual-result audit
  accepted bounded D03, not V08. All ordered samples, first/max and drift remain,
  including **10k search p95 +12.34%, 50k CLI-read p95 +19.33% and 50k MCP-read
  p95 +11.07%**. No uniform improvement, whole-machine idleness, cache eviction
  or portable SLA follows.
- [x] **V06 — Compatibility and upgrades.**
  [Fresh pinned executable evidence](evidence/v1-compatibility.json) passed eight
  actual v0.3.1-to-candidate.5 checks and nine v0.4.0-to-candidate.5 checks,
  including an archive actually produced by v0.4.0. Exact authority/receipts,
  replay, continued writes and prior-binary pending recovery passed.
  Prior-produced archives are not applicable to v0.3.1, which lacks those tools.
  Test-only source equivalence never relabels the candidate.4 scale executable.
  No automatic downgrade or blanket older-version guarantee is promised.
- [x] **V07 — Published packages and installed consumers.**
  Actual v1.0.0 assets, Linux amd64/macOS arm64 native and archived-source
  verification, tap publication, macOS arm64 Homebrew upgrade/formula test,
  installed acceptance/43 documentation requests and both prior-version upgrades
  passed. The six installed skill files match public/published source bytes.
  Final fresh-context consumers passed **CLI+skill 17/17 and MCP-only 20/20**;
  independent automated review accepted their actual facts/citations, allowed
  tools, byte purity and teardowns. Earlier failures remain failed history.
- [x] **V08 — Independent source, delivery and documentation review.**
  Independent automated focused OMP review accepted prepublication source/evidence,
  closed F1, accepted both final installed consumers and approved documentation
  head `b33739b` plus the precise receipt/status projection. P01 is closed by
  binding the actual consumer-review receipt. [The delivery evidence](evidence/v1-delivery.json)
  records this automated decision and its unchanged scope. Exact successor-head
  checks, normal protected merge and merged-main CI are required and recorded
  in [PR25](https://github.com/matheus3301/kb/pull/25); no future CI result or human
  approval is asserted by the review record.

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
profiles. [Both actual candidate.5 profiles](evidence/v1-reliability.json) passed
without changing the 30-second per-command timeout. Candidate.3's completed
failed restore-timeout attempt remains in history, not a passing observation.

Use setup creates to establish a fixed corpus, then sustain **patches, conflicting
CAS, identical-key replay, reads/search and periodic backup/restore**. Reads/search
do not count toward either mutation/replay minimum. Retain exact outcomes and
immutable-source hashes; verify receipts/replays, final inventory and continued
writes after restoration. Timed move/preview loops are not claimed. Deterministic
acceptance/crash regressions separately cover create, move, archive, apply,
preview and coordinated multi-file publication.

Repeat all **35 current deterministic cases at least three times each**
(105 observed kills), retaining the original 11 scenarios, multi-file publication/
recovery positions, committed roll-forward, restore positions and receipt-batch
interruption. Retain injected faults, cancellation and concurrency regressions.
Report timeouts, skips or unobserved boundaries as failures, not successful kills.

Rationale: receipt count has no automatic expiry, retention or rotation.
Receipts store before/after full-file diff text while it fits a **256 KiB aggregate
diff-text budget per receipt** (`api.MaxReadBytes`); excess diff text is truncated.
Repeatedly extending the retained log can therefore accumulate roughly quadratic
diff storage **while the growing images fit that cap**, not indefinitely.
Serialized receipts also include revisions and other fields outside the diff
budget. Existing log/note/archive bounds still apply and rejected operations must
report explicit errors; the diff cap is not permission to bypass those bounds.

The extended profile disables logging, not receipt durability or index
maintenance. The shorter default-log profile and deterministic default-log
publication/recovery tests complement it; independent readiness review must assess
the actual observations and this bounded scope. No receipt-format/expiry change
or automatic compaction is approved. Monitor/protect event-log and receipt growth;
deleting receipts breaks replay. Neither profile proves universal crash/power-loss,
network-filesystem, hostile same-user or arbitrary-editor isolation.

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
absolute/percentage drift, even inside budget. Preserve the historical report's
10k search/read non-improvements, 50k repair median +4.9% and preview RSS increase;
also disclose all fresh candidate drift rather than transfer those old numbers.
An exceedance blocks V05 until a quiet repeat and explicit evidence-backed
review; do not silently change ceilings. Results on different hardware are
informational, not pass/fail against these machine-specific numbers.
The original **core-only 10k 150 ms warm p95 / 150 MiB / 10 s rebuild targets
remain unchanged** and need separately identified measurements. The original
paired report did not measure rebuild improvement; current evidence separately
records both binaries' 10k/50k rebuilds without an invented 50k ceiling.
No Linux timing, broader SLA or unmeasured retained-memory improvement is claimed.

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

The [public snapshot](qa/v1-qualification.md) is available for independent review.
Do not wait for V08 to make factual documentation reviewable; do not infer stable
publication from that documentation.

- [x] Identify frozen candidate.5 source/build/executable and current local source/
  snapshot receipts, keeping actual v0.4.0 publication and old CI historical.
- [x] Attach exact released-source hosted CI and final release/source binding.
  Final documentation successors use the same protected CI; actual results live
  in [PR25](https://github.com/matheus3301/kb/pull/25), without self-referential claims.
- [x] Independently validate the completed candidate.5 extended D02 report;
  retain all failed attempts and distinct source/artifact identities.
- [x] Attach V07 installed Homebrew/native Linux and independent consumer evidence;
  state channel/client limits and distinguish source/snapshot/installed checks.
- [x] Record automated V08 reviewer role, scope, findings, disposition and bounded
  readiness decision. P01 is corrected; no independent human approval is invented.
- [x] Reconcile README, llms.txt, spec, interfaces, installation, quickstart,
  roadmap, quality, compatibility, decision summaries and release guidance;
  preserve historical reports and record only observed outcomes.
- [x] Publish unused v1.0.0 through the normal guarded workflow and verify actual
  assets, source, Homebrew and independent consumers. Keep immutable tags/assets
  and the documented support limits.

