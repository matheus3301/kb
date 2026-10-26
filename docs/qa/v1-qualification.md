# Candidate.5 v1 qualification evidence

**Historical candidate.5 qualification with separately bound actual v1 delivery.**
Frozen candidate.5 D01–D03 and local V01–V06 evidence completed passing reports,
including **34,840 extended ACK and 27 exact backup/restores** plus the separate
1,200-ACK default-log companion. [v1.0.0 delivery](v1-delivery.md) now records
actual publication, native/source guards and Homebrew installation without
relabeling these measurements. Both final installed consumers passed and received
bounded independent acceptance; only combined-docs review/protected integration
remain pending. All earlier failed consumer attempts remain failed history.

## Exact source and executable

| Identity | Observed value |
| --- | --- |
| Final Go/module source | `cb66fd77a120d8c8531f9077a32d2bf51ef7921c` |
| Documentation observation base | `c372b33bf3a41fd90cd1ee939c1e7c96e19ea37a` |
| Shipping-shaped label | `v1.0.0-candidate.5`, not published v1 |
| Actual executable SHA-256 | `59d1b477031792cdf539aa3777fafc3942403b80aa8691a3403a8732bf928d46` |
| All Go/module files | **72**, including tests |
| Canonical Go/module Git-blob-map SHA-256 | `567d8104823d1f31b0ab22ab94cf2c761fdb2db78438d1afa9cb75434864af25` |
| Fixed retrieval corpus SHA-256 | `562af0124822b42e72125ae9307e5bc5a5fe9fb0f1164be13a8811aa6d49cc4c` |

All 72 tracked `*.go` files plus `go.mod`/`go.sum` agree between frozen source
and documentation base. The map is sorted compact path-to-Git-blob-OID JSON;
[the manifest](../evidence/v1-qualification.json) also contains every source-file
SHA-256 and operational-harness hash. The executable records Go 1.27.3,
Darwin/arm64/v8.0, CGO=0 and trimpath; declared build environment is
`CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local`, stripping `-s -w` with explicit
version/source linker identity. Race correctness builds are a separate mode.

The production restore change was authored at `38ab64dfd2b362fb8848c7e331b0efb5d64b8ff6` and integrated as
`5477946be88ef574499a07a7eabe7aa8b6bcc084`. Final source differs from that implementation in **only two test
files**, `backup_restore_batch_test.go` and `backup_restore_contracts_test.go`:
checked channels and anchored fixture reads, not suppressions or production
changes. This establishes production-source equivalence, **not identical
executable bytes**. Final docs/release commits and rebuilt version/commit metadata
must be bound separately; candidate.4 or candidate.3 hashes are never relabeled.

## Gates and provenance

Completed source receipts record **93 Python tests, 88.8% aggregate Go coverage,
12 fuzz targets, four cross-builds, 43 documentation requests**, race/static/security
and application gates, including retrieval and an 80-ACK smoke. The exact full
check log hash is retained in the manifest. The committed-source snapshot passed
local native Darwin arm64 and archived-source CLI/MCP workflows; four archives
were inspected, not executed on four platforms. It is not actual published-v1,
native Linux or Homebrew proof.

[PR24](https://github.com/matheus3301/kb/pull/24) merged through protected main
`0a6d04d3c25840b2c71c0b98d50478dc6749879a`. Final head
`dcc7ad90665209886009c8674ad2d10e8c0de4de` passed all 11 jobs in
[CI 40287161946](https://github.com/matheus3301/kb/actions/runs/40287161946);
[exact merged-main CI](https://github.com/matheus3301/kb/actions/runs/40287653868)
passed application jobs and its gate, with an intentional push-only Dependency
Review skip. Earlier head CI remains historical. Editor verification was static
Git/artifact hashing and arithmetic, not a gate or runtime rerun.

| Criterion | Current observed evidence | Remaining boundary |
| --- | --- | --- |
| V01 source/integration | Frozen 72-file map, actual source gates/snapshot, exact final PR/main CI and separate released-source binding | Later docs-only integration retains its own CI requirement |
| V02 retrieval | Fresh 47 parity pairs / 17 category validations, original labels/floors | Accepted bounded source evidence, not arbitrary semantic usefulness |
| V03 reliability | 34,840 extended ACK / 27 checkpoints; 1,200-ACK default-log companion; 105 observed test-child kills | Finite local-process/filesystem observation, not arbitrary history or power-loss proof |
| V04 backup/restore | All 27 extended exact-fidelity checkpoints, replay/continued writes, fresh archive compatibility and distinct production/scale evidence | Actual delivery bound separately; no universal disaster guarantee |
| V05 resources | One whole fresh 16-row + three-core-target matrix passed; independent D03 audit accepted | Bounded host result, not V08 or a portable SLA |
| V06 upgrades | Candidate.5 eight/nine-check reports retained; separate actual installed-v1 eight/nine-check upgrades passed | No automatic downgrade or blanket older-version promise |
| V07 packages/consumers | Actual published/native/source/tap/Homebrew proofs and final CLI17/MCP20 consumers passed with bounded independent result acceptance | Finite tested scope; separate released/candidate/installed identities |
| V08 final closure | Prepublication approval, F1 closure and final installed-consumer result acceptance observed | Combined postpublication documentation-head review and protected integration pending; not human approval |

## D01 — Unchanged judgments, fresh observations

[Retrieval evidence](../evidence/v1-retrieval.json) retains authored `relevant`,
`matches`, filter/order constraints and measured ranked outputs, with every
category-to-case mapping. All **31 searches, eight context cases and eight exact
source-read pairs** passed CLI/MCP parity. The 24 positive cases retain macro
Recall@5 **0.9930555555555555** and MRR@5 **1.0**; all seven negatives require zero
hits/total. Negative per-case and negative-only category means remain **null**,
excluded from positive means. The six-relevant judgment retains its attainable
5/6 cutoff floor; no corpus, membership or floor changed.

The editor recomputed all search membership/order/metrics and category means,
context membership/budgets/truncation, aggregate read count and parity coverage.
The raw evaluator omits executable identity: a completed functional receipt and
coordinator attribution bind this fresh invocation to candidate.5. Source-read
groups describe expected fixture sources; eight runtime read pairs are exported
in aggregate, not invented per-path runtime records. Finite lexical evidence is
not arbitrary semantic understanding.

## D02 — Both actual profiles passed

[Current reliability evidence](../evidence/v1-reliability.json) retains the complete
candidate.5 extended report, SHA-256
`4c79486678cf7fc8a6f4c54a5c9c2659a5c1458993036ed70ecf912cc2b163e1`,
and separately attributed coordinator/editor accounting checks. Its three seeds
and four writers per seed completed:

| Seed | Active seconds | Mutations | Identical-request replays | Expected conflicts | Exact backup/restores |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1729 | 608.568190 | 5,680 | 5,680 | 3,408 | 9 |
| 2718 | 616.662381 | 5,865 | 5,865 | 3,519 | 9 |
| 31415 | 615.944407 | 5,875 | 5,875 | 3,525 | 9 |
| Total | 1,841.174978 | 17,420 | 17,420 | 10,452 | 27 |

All runs passed with zero errors/failures, **34,840 main-root ACK**, 34,840 reads,
3,484 searches, 27 restored writes and 54 restored replays. Overall elapsed was
**1,923.408805 seconds**; active-work throughput was **18.922699 ACK/second**.
Log maintenance was disabled and indexes enabled on the five-note bounded corpus.
Final receipt inventories were 5,685/5,870/5,880 files and
5,862,557/6,053,845/6,081,878 receipt bytes; disabled-log bytes stayed unchanged.
The largest checkpoint had **5,889 files / 6,085,377 authoritative bytes**.
Maximum restore was **24.227168 seconds**, maximum backup **1.371179 seconds**,
and maximum combined backup/restore/fidelity checkpoint **27.953329 seconds**.
Both qualification supervisor children exited zero. The coordinator reran original
qualification/accounting; the editor independently reconciled configuration,
clocks, receipts, commands, checkpoints and report/source hashes without rerunning
the workload or claiming a postmortem read of disposed roots.

The separate default-log/index-enabled profile passed **600 mutations + 600
identical-request replays = 1,200 main-root ACK**, **360 expected conflicts**,
six exact backup/restores, six restored continued writes and twelve restored
replays, with zero errors. Seeds 1729/2718/31415 performed **20.001324,
20.136574 and 20.296424 active seconds**; overall elapsed was **66.521894 seconds**.
Receipt inventories, main/setup/restored acknowledgments and command/checkpoint
counts independently reconcile from its separate completed report.

The default-log profile is **not** a 30-minute log-enabled run. The separate
extended profile exceeded all three **600-second** windows, the four-writer
requirement and **10,000 main-root mutation/replay ACK combined**. The unchanged
**30-second per-command timeout** applies to each CLI operation, not a combined
checkpoint. Setup, restored-root work, reads/searches, conflicts and failed
attempts do not inflate the ACK gate. No old pass substitutes for this new report.

Three current CGO=0 observed-boundary repetitions passed **105 cases in 22.336
package seconds**: 33 archive/recovery, 54 publication/recovery positions,
15 restore positions and three receipt-batch interruptions. Parsed complete
records show 35 distinct cases each passed three times, zero failures/skips.
These are **test children**, not public-CLI interruption at arbitrary syscalls,
hardware power loss or universal filesystem isolation.

## D03 — One complete fresh resource matrix

Window **2026-10-26T06:23:05Z–2026-10-26T06:31:28Z**, before the long observation:
macOS 26.6.2 / Darwin 25.6.0 arm64, 10 logical CPUs, Go 1.27.3, Python 3.14.4,
`GOMAXPROCS=8`, synthetic 10k/50k corpora of 2,048-byte notes. The fixed method
used **20 read / three repair / five separate rebuild / 20 core samples**.
Fresh CLI children and reused stdio MCP have distinct costs; MCP RSS is a whole
session peak, not per-request memory. Baseline preceded candidate, OS caches were
not evicted and background applications were not controlled.

Eight fresh native proofs observed all four coordinated workers settled, without
streaming/compaction/pending async work or queued messages. The separate
independent audit checked their 32 original acknowledgments. These are stage
snapshots, not continuous whole-machine idleness or product-correctness proof.

| Operation | 10k p95 / ceiling (ms) | 10k RSS / ceiling (MiB) | 50k p95 / ceiling (ms) | 50k RSS / ceiling (MiB) |
| --- | ---: | ---: | ---: | ---: |
| CLI search | 362.319 / 380 | 115.250 / 130 | 2314.668 / 2870 | 491.562 / 550 |
| CLI read | 176.244 / 200 | 99.125 / 110 | 1738.669 / 2070 | 398.766 / 470 |
| CLI validate | 177.375 / 190 | 98.812 / 120 | 1670.774 / 1910 | 401.734 / 470 |
| CLI repair preview | 519.554 / 730 | 179.266 / 200 | 3420.311 / 3770 | 755.156 / 830 |
| CLI committed repair | 1196.158 / 1460 | 180.359 / 230 | 7826.353 / 8940 | 834.094 / 1050 |
| Warm MCP search | 101.890 / 300 | 134.641 / 150 | 1209.165 / 1530 | 569.375 / 640 |
| Warm MCP read | 100.350 / 110 | 134.641 / 150 | 1401.045 / 1540 | 569.375 / 640 |
| Warm MCP validate | 100.135 / 110 | 134.641 / 150 | 1320.324 / 1690 | 569.375 / 640 |

All 16 operation/size rows and three original core targets passed unchanged
ceilings. Core warm p95 was **130.2 ms**, process RSS **130.4 MiB**, cold search
**286.2 ms**; fresh 10k CLI rebuild p95 was **177.464 ms** against 10 seconds.
The core build reports CGO=0 but no trimpath setting. Its unchanged benchmark
exports aggregates only: raw warm arrays, warm maximum and per-query RSS remain
null, never reconstructed from CLI/MCP timings.

[Complete resource evidence](../evidence/v1-resources.json) retains every ordered
CLI/MCP duration/RSS array, separate first request, sampled/warm and all-request
maxima, paired drift and both rebuild sizes. The editor independently recomputed
**36 metric sets, 18 paired drift sets / 90 scalar deltas**, covering **548 raw
records**: 524 summarized CLI/warm durations, twelve separate MCP first requests,
eight fixture initializations and four session peaks. Generic raw warm summaries'
first sample is distinct from the separately retained MCP first request. The
independent actual-result audit accepted this whole matrix, not V08.

**Passing ceilings do not imply uniform improvement.** Versus this paired actual
v0.3.1 baseline, 10k CLI-search p95 increased **12.34%**, 50k CLI-read p95 **19.33%**,
and 50k warm-MCP-read p95 **11.07%**. The 10k search sampled maximum **428.706 ms**
is retained; 380 ms is a p95 ceiling, not a maximum bound. All median/first/max/RSS
regressions remain inspectable, with no selected-cell aggregation or rerun waiver.

| Rebuild notes | Baseline p50 / p95 (ms) | Candidate p50 / p95 (ms) | Baseline → candidate RSS (MiB) | p95 drift |
| ---: | ---: | ---: | ---: | ---: |
| 10,000 | 176.320 / 180.027 | 176.137 / 177.464 | 99.047 → 97.500 | -1.424% |
| 50,000 | 1443.507 / 1677.981 | 1428.322 / 1438.505 | 431.922 → 391.297 | -14.272% |

CLI rebuild includes startup plus core work. No separate 50k rebuild ceiling,
unmeasured pure-core improvement, population-tail guarantee or portable SLA is
invented. The four prior failed matrices remain failures, not candidate.5 cells.

## Production restore correction and separately identified 6k scale

[Source/review/scale attribution](../evidence/v1-restore-production.json) records
the committed bounded-four restore implementation: all file synchronization and
close attempts join before coordinator hooks, cleanup, retained-parent closure,
final inventory or config publication. Every prepared ownership record survives
error/cancellation draining. File and parent syncs, fresh confinement/inventory
checks, private new-only outputs and config-last publication remain.

Focused independent source/R1 review closed the old worker-cleanup blocker for
that committed implementation and its regression scope. Six batch tests plus
three contracts cover deterministic blocked-peer joining, archive-ordered hooks,
fault/cancellation/residual handling, foreign-leaf preservation, bounded-descriptor
operation, real interruption, private usable outputs, exact bytes/replay/continued
writes, cancellation/lock release and outside-only hardlinks. Targeted race/shuffle
executions are **owner-reported**, not rerun by the editor/reviewer; later actual
final-source full gates are separately receipted. None is final V08 or universal
crash/power-loss certification.

The three native **6,000-receipt** pairs used actual **candidate.4**, source
`38ab64dfd2b362fb8848c7e331b0efb5d64b8ff6`, SHA-256
`78dc608ad6810188b695c4af796a0811e713f6fce03db5a218739a9ac9d22b93`,
not candidate.5. All samples retained the 30-second guard, exact manifest/inventory/
config/receipt fidelity, three old replays, continued writes and source/archive
purity. Each archive contained 6,009 files / 5,235,814 authoritative content bytes.

| Pair | Candidate.3 baseline (s) | Candidate.4 (s) | Baseline RSS (MiB) | Candidate.4 RSS (MiB) |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 29.400285 | 24.701220 | 32.515625 | 31.781250 |
| 2 | 29.043362 | 24.588339 | 32.500000 | 32.484375 |
| 3 | 28.990590 | 24.834842 | 31.796875 | 32.812500 |

Peak RSS increased **32.515625 → 32.812500 MiB**. Native scale used
`GOMAXPROCS=2`, unlike the older instrumented prototype's 8, and the implementation/
profiling contexts differ. This is not a controlled single-variable comparison
or a promise of the prototype's timing. Test-only source equivalence does not
relabel the candidate.4 executable, samples or hashes as candidate.5.

The [older diagnostic report](restore-performance-v1.md) retains its rejected
`all_workers_join_before_ordered_hooks_and_cleanup=true` overclaim. The old
prototype's error invariant remains unestablished; later production correction
never retroactively repairs or qualifies those old artifact bytes.

## D04, retained failures and remaining publication boundaries

[Fresh compatibility evidence](../evidence/v1-compatibility.json) uses pinned
actual preserved v0.3.1 and v0.4.0 executables: eight and nine checks respectively.
The v0.4.0 binary actually produced the consumed old archive. Exact authority,
configuration, receipts, replay and subsequent writes passed; old pending state
was recovered by the prior executable before upgrade. Old-produced operational
archives are explicitly not applicable to v0.3.1. No automatic downgrade or
blanket earlier-version guarantee follows.

[Historical evidence](../evidence/v1-qualification-history.json) retains all three
failed 7ceed resource matrices, the failed source-b71 CGO=1 matrix, the original
pre-fix backup-timeout endurance, and candidate.3's **failed** 33,680-ACK shipping
attempt. Its final restore exceeded 30 seconds; final failed-checkpoint fidelity
was not verified, and no confirmed-data loss was demonstrated. The older 7ceed
32,850-ACK / 24-checkpoint long **pass** remains bounded to its own runtime,
never reused for candidate.5. Candidate.3's passing resource window is historical
and linked separately. No CGO-causality, discarded-failure or best-cell claim.

No qualification policy was relaxed: corpus labels/floors, the
1,800-second/10,000-ACK and 60-second/250-ACK profile gates, 30-second command
timeout, original core targets and every D03 ceiling remain unchanged.
The actual completed candidate.5 report is retained. Protected integration,
independent automated prepublication approval and real v1.0.0 release/native/tap/
Homebrew proofs are now separately recorded in the [delivery report](v1-delivery.md).
They do not rename candidate measurements as released-binary observations.

Both final fresh-context installed consumers passed and received bounded
independent result acceptance. All six older failures remain failed, with
observer false negatives, observed capacity rejection and unattributed failures
distinguished in the [delivery record](v1-delivery.md). No blanket provider or
product attribution follows. Final combined-documentation review and protected
integration remain pending. The [stability checklist](../stability.md) centralizes
status; automated review is not human approval or overall final-completion sign-off.

## Public reproduction, after the active qualification slot is released

Use verified executables with the identities above. These checked-in harnesses
create disposable synthetic roots; never test mutations on a live collection.
Serialize controlled measurements and retain failed outputs under distinct names.
The following are reproduction instructions, **not additional runs by this editor**.

```sh
# Small functional smoke for CI wiring, not D02 qualification.
GOMAXPROCS=2 python3 scripts/soak.py --binary .build/kb-candidate5 --duration 3 --seeds 1729 --writers 4 --minimum-work 1 --backup-interval 1 --command-timeout 30 --output .build/v1-smoke.json
GOMAXPROCS=2 python3 scripts/soak.py --binary .build/kb-candidate5 --duration 1800 --seeds 1729 2718 31415 --writers 4 --minimum-work 10000 --backup-interval 60 --command-timeout 30 --output .build/v1-endurance.json
GOMAXPROCS=2 python3 scripts/soak.py --binary .build/kb-candidate5 --duration 60 --seeds 1729 2718 31415 --writers 4 --minimum-work 250 --backup-interval 10 --command-timeout 30 --maintain-log --output .build/v1-default-log.json
GOMAXPROCS=2 python3 scripts/retrieval_eval.py --binary .build/kb-candidate5 --output .build/v1-retrieval.json
```

For D03, run `scripts/benchmark.py` serially for baseline and candidate at
`--sizes 10000 50000 --samples 20 --repair-samples 3`, then both binaries with
`--samples 5 --rebuild-only`, `GOMAXPROCS=8`, separate output files and no overlapping
build/test/soak/profile workload. Core measurements and native quiet-state
provenance are separate, not inferred from those CLI commands.
