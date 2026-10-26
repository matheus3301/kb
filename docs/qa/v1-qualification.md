# Shipping-candidate v1 qualification evidence

**Reviewable factual candidate snapshot; not a stable-publication decision.** The release-shaped
candidate has one complete passing local D03 matrix. Its fresh 60-second default-log
companion, retrieval/categories, actual-prior-producer archive and 102 observed
interruption cases have completed passing reports. The shipping 1,800-second
D02 attempt **completed failed: seed 31415's final restore exceeded the unchanged
30-second command timeout**. Independent automated V08 publication-authorization
review remains pending. Historical endurance is not reused.
Experimental v0.4.0 remains the observed published/installed release.

## Exact source, executable and build protocol

| Identity | Observed value |
| --- | --- |
| Shipping Go/module source | `b71c0763c87313666b3dd79da08da12b7b3512c1` |
| Build-protocol source | `cdc261fb26e696bf26c7f621b3a2b495b09a2885` |
| Protected PR23 merge | `5c94c993aa8ec97295bdf6bcdb1d5cd2f052a2f7` |
| Build-protocol / protected-main full tree | `f39d12927362971668b2e173ca01db2daec3ca67` |
| Shipping version label | `v1.0.0-candidate.3` — not a published v1 release |
| Shipping executable SHA-256 | `8aecbf6cf22d2c4bc8c97e1d8b63fea725dfa325b2e5f632f2ed97fc815815ac` |
| All 69 Go/module Git-blob-map SHA-256 | `a52cf381561d235c79016226a854e6cb03ac1e4cfc143130c9db54d379babfec` |
| Fixed retrieval corpus SHA-256 | `562af0124822b42e72125ae9307e5bc5a5fe9fb0f1164be13a8811aa6d49cc4c` |

All tracked `*.go` files plus `go.mod` and `go.sum`, including tests, agree across
b71, CDC and protected main: 69 canonical path-to-Git-blob-OID entries. The map is
sorted compact JSON. This is broader than the earlier 35-file production-only
fingerprint, so those two digest populations must not be compared as if identical.
The full b71 source tree differs from CDC/main; only CDC and main share the full
tree above. Build/harness/documentation provenance is recorded separately.

The shipping executable's embedded build information confirms Go 1.27.3,
`CGO_ENABLED=0`, `GOOS=darwin`, `GOARCH=arm64`, `GOARM64=v8.0` and `-trimpath=true`.
The declared build uses `GOWORK=off`, `GOTOOLCHAIN=local`, `-trimpath`, stripping
flags `-s -w` and explicit version/source metadata. The preserved actual v0.3.1
baseline also records CGO=0. The non-race core measurement's build information
confirms CGO=0; its core build record does not report `-trimpath`, so none is
invented for that executable. Race-enabled quality gates remain a separate build
mode, not substituted for shipping artifacts.

A later docs-only release commit can retain these Go/module bytes while embedded
version/commit metadata changes its rebuilt executable hash. Final published
archive, formula, installed executable and consumer hashes must be recorded
separately; source equality never implies executable-byte equality.

## Source gates and current status

[PR23](https://github.com/matheus3301/kb/pull/23) merged as the commit above after
[exact PR CI](https://github.com/matheus3301/kb/actions/runs/40263657403) passed.
[exact merged-main CI](https://github.com/matheus3301/kb/actions/runs/40264275136)
also passed. Coordinator-attributed source gates passed 91 tooling tests, 88.7%
aggregate Go coverage, race/static/security gates, 12 fuzz targets, four cross-builds,
43 documentation requests and existing acceptance/evaluation workflows. CGO=0
committed-source snapshots passed native temporary-prefix and archived-source
CLI/MCP workflows. Only the host-native snapshot was executed locally; inspecting
four archives and cross-building four targets is not four-platform runtime proof.

| Criterion | Current observed evidence | Remaining boundary |
| --- | --- | --- |
| V01 exact integrated candidate | Same 69 Go/module files, source gates, shipping snapshots, exact protected PR and merged-main CI | Final docs/release binding |
| V02 retrieval/parity | Fresh shipping 47-pair report and authored category projection passed | Independent final evidence review |
| V03 reliability | Default-log companion and 102 observed interruption cases passed; shipping long attempt failed a real restore timeout | Resolve/dispose failure and obtain fresh passing D02; no old-long reuse |
| V04 backup/restore | Prior-produced archive/companion/positional checks passed; 24 long-run checkpoints completed before a final restore failed | Failed final checkpoint did not complete fidelity/replay/continued-write checks |
| V05 local resources | One complete shipping matrix passed all 16 rows and three core checks; independent automated actual-result audit passed | Final readiness disposition; not a general SLA |
| V06 compatibility | Fresh nine-stage prior-v0.4.0 archive/upgrade and eight-stage preserved-v0.3.1 upgrade passed | Final release binding and post-publication verification |
| V07 packages/consumers | Experimental v0.4.0 installed proof remains historical | Actual published-v1 archives/Homebrew/installed consumers are post-publication proof |
| V08 readiness | Separate independent automated reviewer role exists | Decision pending; no human approval claimed |

## D03 — One complete passing shipping matrix

The reserved experiment used Apple M4, macOS 26.6.2 / Darwin 25.6.0 arm64,
Go 1.27.3, Python 3.14.4 and `GOMAXPROCS=8`. Synthetic corpora contain 10k/50k
2 KiB notes. The unchanged request/fixture evaluator used 20 read, three repair,
five independent rebuild and 20 core samples. CLI children are fresh; MCP reuses
one stdio process per corpus. Baseline precedes candidate once. OS page caches
were not evicted and background applications were not controlled.

Eight native quiet-state proofs record all four coordinated workers settled,
not streaming/compacting, with no pending async work or queued messages at the
recorded boundaries. Public projections retain timestamps, state facts and source
record hashes, not request/session/process identifiers. These proofs establish
coordinated worker state, not whole-machine idleness or product correctness.
The complete matrix preceded the new shipping soak.

An independent automated actual-result audit reproduced 36 metric sets, all 16
candidate rows, three core targets, 284 CLI durations/individual child-RSS values,
240 warm MCP durations and 12 separate first-request durations. It checked all
eight quiet proofs and the unchanged nearest-rank calculation, complete event
coverage and all paired drift. Its bounded D03 decision passed with no concrete
findings; it explicitly does **not** certify V08, stable v1 or post-publication
completion. The sanitized projection retains its source/input hashes and method
scope, not raw private control metadata.

| Operation | 10k p95 / ceiling (ms) | 10k RSS / ceiling (MiB) | 50k p95 / ceiling (ms) | 50k RSS / ceiling (MiB) |
| --- | ---: | ---: | ---: | ---: |
| CLI search | 291.584 / 380 | 115.750 / 130 | 2,227.160 / 2,870 | 491.406 / 550 |
| CLI read | 177.750 / 200 | 98.984 / 110 | 1,444.015 / 2,070 | 400.891 / 470 |
| CLI validate | 177.666 / 190 | 99.719 / 120 | 1,607.952 / 1,910 | 399.406 / 470 |
| CLI repair preview | 524.845 / 730 | 178.594 / 200 | 3,605.977 / 3,770 | 743.125 / 830 |
| CLI committed repair | 1,207.517 / 1,460 | 212.906 / 230 | 7,747.954 / 8,940 | 817.938 / 1,050 |
| Warm MCP search | 101.835 / 300 | 135.531 / 150 | 1,089.205 / 1,530 | 547.109 / 640 |
| Warm MCP read | 101.452 / 110 | 135.531 / 150 | 1,300.823 / 1,540 | 547.109 / 640 |
| Warm MCP validate | 101.176 / 110 | 135.531 / 150 | 1,280.394 / 1,690 | 547.109 / 640 |

All 16 operation/size rows passed their unchanged latency and RSS constraints.
The separate core observation passed 20-sample warm p95 **113.4 ms** and process
RSS **131.4 MiB** against the unchanged 150 ms / 150 MiB targets. The fresh 10k
CLI rebuild p95 was **179.690 ms**, below 10 seconds; it includes startup/core
work, not an invented pure-core timing. Core cold time was separately **287.3 ms**.

Both binaries' rebuilds remain visible, including non-improvements:

| Corpus | Baseline p50 / p95 (ms) | Candidate p50 / p95 (ms) | p95 drift | Baseline / candidate RSS (MiB) |
| --- | ---: | ---: | ---: | ---: |
| 10k | 176.170 / 177.666 | 177.421 / 179.690 | +1.139% | 100.266 / 97.031 |
| 50k | 1,441.045 / 1,446.556 | 1,427.882 / 1,691.151 | +16.909% | 428.188 / 399.859 |

There is no invented 50k rebuild ceiling. The 50k rebuild median improved while
its p95 regressed; neither is hidden. The planned resource JSON retains every
baseline/candidate p50/p95, first request/apply, maximum, RSS and absolute/percentage
drift, including all non-improvements. Twenty-sample nearest-rank p95 is the 19th
ordered sample; three-sample repair p95 is the observed maximum. MCP per-request
RSS remains null: its peak is whole-session, not replicated into fake samples.
The unchanged core benchmark emits no raw warm arrays/max/per-query RSS; those
fields remain null. Measurements are local observations, not portable SLAs or
statistical-significance claims.

### All earlier failures remain failures

| Historical complete matrix | Failing row(s), observed p95 / unchanged ceiling |
| --- | --- |
| Legacy 7ceed attempt 1 | 50k CLI search 4,191.294 / 2,870 ms; preview 3,827.358 / 3,770 ms |
| Legacy 7ceed attempt 2 | 10k warm MCP validate 117.925 / 110 ms |
| Legacy 7ceed attempt 3 | 10k CLI search 384.470 / 380 ms |
| Source b71 candidate.2, CGO=1 | 10k CLI validate 234.312 / 190 ms |

The candidate.2 executable hash was
`66b08d25e0fa939057602d9bd185fb13e01c0be290db55c70a2f1f52f940dd4e`;
its embedded build record confirms CGO=1. Candidate.3 uses the release-shaped
CGO=0 protocol. This is a concrete build-protocol correction, **not a controlled
CGO causal experiment** and not proof that CGO caused any prior spike. No failed
matrix is relabeled, no best cells are aggregated, no sample is dropped and no
ceiling is increased. Original measurements, plans and hashes remain retained.

## D01 — Fresh shipping retrieval, judgments and categories

The fixed corpus SHA-256 above is unchanged. The fresh shipping report passed
31 search judgments, eight context cases and eight exact source-read pairs:
**47 CLI/MCP pairs**, with query and isolated-home purity. It reports 24 positive
and seven negative search judgments, macro Recall@5 **0.9930555555555555** and
MRR@5 **1.0**, plus 17 category/negative-case validator checks.

Original `relevant`, `matches`, filter and ordering membership is retained with
all category-to-case-ID mappings and measured ranked outputs. A six-relevant
query has the attainable 5/6 Recall@5 floor, not an impossible 1.0. Seven negatives
require zero ranked hits/total; their per-case metrics and negative-only category
means remain **null**, excluded from positive-query means. Category grouping
creates no new product executions. Source-read groups identify the expected
source set while runtime evidence reports eight exact read pairs in aggregate;
no per-path runtime records are invented. The evaluator itself omits executable
identity, so the source/hash binding is explicit coordinator attribution to the
preserved shipping candidate, not a self-reported evaluator identity.

## D02 — Companion passed; shipping extended attempt failed

The new shipping log-enabled/index-enabled companion's final report passed:
**610 mutations + 610 exact replays = 1,220 main-root acknowledgments**, 366 expected
CAS conflicts, six backup/restore checkpoints and zero unexpected command errors.
Seeds 1729, 2718 and 31415 used four writers for **20.505157, 20.234761 and
20.465543 active seconds**. Overall elapsed time was **68.247934 seconds**, including
setup/final checks. Restored roots separately accepted six new mutations and
twelve replay acknowledgments. This is not a 30-minute log-enabled observation.

Measured companion throughput was **19.932862 acknowledgments per active
second**, or **17.875999 per overall elapsed second**. Each seed began with five
receipts: seed 1729 grew from 10,775 to 5,016,454 receipt bytes and 567 to 22,165
log bytes; seed 2718 ended at 5,016,430 receipt bytes / 22,165 log bytes; seed
31415 grew from 10,804 to 4,829,429 receipt bytes and 572 to 21,842 log bytes.
Final receipt counts were 210, 210 and 205. The largest companion checkpoint
covered 219 files / 5,042,078 authoritative content bytes; restore maximum was
1.587809 seconds. These short measurements expose retained-diff growth rather
than hiding it behind the extended log-disabled profile.

The shipping **1,800-second** index-enabled/log-disabled report completed with
`ok=false`: **16,840 mutations + 16,840 replays = 33,680 reported main-root
acknowledgments**, 10,104 expected CAS conflicts and 24 completed backup/restore
checks. One unexpected command error was seed 31415's final **restore timeout**.
Its measured restore maximum was **30.004420 seconds**, with the unchanged
**30-second per-command timeout**. Final failed-checkpoint fidelity, restored
replay and continued-write checks did not complete. No lost confirmed state was
demonstrated; absence of demonstrated loss is not a passing qualification.

| Seed | Actual active seconds | Mutations | Replays | Expected conflicts | Completed checkpoints | Result |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| 1729 | 626.915562 | 5,585 | 5,585 | 3,351 | 9 | Passed seed |
| 2718 | 600.119074 | 5,810 | 5,810 | 3,486 | 8 | Passed seed |
| 31415 | 600.165371 | 5,445 | 5,445 | 3,267 | 7 | Failed final restore |

Overall elapsed time was **1,923.206964 seconds**. Counts exceeded 10,000 and
all three work windows exceeded 600 seconds, but these do not offset a real
timeout. Setup/restored-root operations, reads/searches and conflicts are
excluded from the acknowledgment threshold. An independent completed-report
validator rejected the actual failed report; it did not combine good seeds into
a pass. The original report SHA-256 is
`943e3a519b5a83c0b60849071025378abd4bcda42f729c5fcfae9594f97f2b45`.
The frozen original and failure remain distinct from any future rerun.

A combined successful checkpoint may exceed 30 seconds because it comprises
separate commands and exact-byte/replay/continued-write checks. That is different
from this actual restore timeout. No timeout or acceptance threshold was raised.

The failed extended attempt reported **18.432574 main-root acknowledgments per
active second**, not passing reliability throughput. Its largest completed
checkpoint covered **5,824 files / 6,000,506 authoritative content bytes**.
Maximum backup duration was 1.540135 seconds; successful-seed restore maximum
was 27.981605 seconds and maximum completed combined checkpoint was 31.628005
seconds. The separate failed restore reached 30.004420 seconds. The failed seed
has no final storage inventory; no final-fidelity claim is inferred for it.

The bounded corpus uses five mutable notes plus the immutable 2,048-byte
`raw/source.bin` fixture generated as `bytes(range(256)) * 8`. Its expected
SHA-256 is `10fc3c51a152e90e5b90319b601d92ccf37290ef53c35ff92507687d8a911a08`,
computed from the committed fixture constructor, not a separately exported
per-run hash. Receipt verification and restored continuation compare exact
source bytes. Receipts, setup/restored-root acknowledgments and manifest
identities remain separately inspectable in the structured evidence.
The fresh CGO=0 shipping-source interruption run passed **102 observed SIGKILL
cases across three repetitions**, with zero failures/skips and package elapsed
time **24.901 seconds**: 33 archive/recovery cases, 54 publication/recovery-position
cases and 15 restore-file-position cases. Completed JSONL events independently
confirm 34 leaf cases each passed three times. The test children are not the public
CLI executable; observation-boundary handshakes do not model hardware power loss.

The old 7ceed run's **32,850 acknowledgments / 24 checkpoints / three 600-second
seeds** remains a historical pass for executable `83327b…`, never a substitute for
the new shipping run. The earlier pre-fix v0.4.0-source attempt remains failed at
its backup timeout; no lost confirmed state was demonstrated. Neither a historical
pass nor absence of demonstrated loss proves universal durability or qualifies a
changed runtime. No running soak, executable or sandbox was modified by this editor.

The extended profile explicitly disables log maintenance. The companion and
deterministic multi-file log-enabled crash scenarios exercise the default-log
path separately. Retaining full-log diffs in receipts increases storage/work as
the log grows, bounded by the **256 KiB per-receipt diff cap** rather than an
indefinitely applicable quadratic-growth formula. This remains a disclosed
operating limit, not a guarantee for arbitrary log/history sizes.

## D04 — Fresh actual producer archive and release boundaries

The actual preserved v0.4.0 Homebrew executable, source
`74f0d196408c1354899802e18ce569b0eab33adf`, hash
`bc61edae093dea6896b3f842ca2ab7441341e25f078495f1a6230cd25aa00443`,
produced the archive restored by shipping candidate.3. **Nine checks passed**,
including exact authority/config/receipt fidelity, old-request replay, continued
patch/create, source immutability and prior-binary prepared-journal recovery.

The prior-produced archive includes **11 files / 9,003 content bytes**. Archive
SHA-256 is `f19025d3ce702647b93ae7db77699acbd5ffa0a982d937a122319c49205127cf`;
producer/restored manifest identity agrees at
`sha256:d91be78706ae101c7593217c60450bc4948ddb41f520a937967b82373a3daabf`.
Three old requests replayed after restore and subsequent patch/create succeeded.
The actual v0.4.0 producer is not a relabeled development binary. The separate
preserved-v0.3.1 executable also passed eight shipping-upgrade checks, including
CLI/MCP, receipts, old prepared-journal recovery, rollback boundary and candidate
backup/restore. Its source is `2ca104f48cc4a6551f4bbc4fae12d51432e42dff`, SHA-256
`0f6267b05e3eb80ae053cd113aecdb61b3d6e29bd74451b9dc658578b9643238`.
That old release has no operational archive tools: old-producer archive coverage
is explicitly **not applicable** for v0.3.1, not silently counted as a pass.
Hosted-build and final-release reports retain their distinct identities.

Required native scope remains macOS arm64 Homebrew and Linux amd64 native
archive/archived-source workflows. Linux arm64 and macOS amd64 remain
cross-build/package-only. Actual published-v1 assets/formula/Homebrew/independent
installed consumers are a separate **post-publication** proof, not inferred from
source snapshots, historical v0.4.0 installation or an implemented release guard.

## Readiness, public commit and source binding

The separate independent automated reviewer must assess the actual shipping
results, retained failures, build protocol and stated platform/safety limits.
**V08 publication-authorization review is pending; no human approval is claimed.** D01–D04 policy choices are
delegated coordinator decisions, not model-obedience or safety proofs.

The coordinator authorized this whitelisted synthetic public snapshot so V08 can
review completed documentation. V08 is not a prerequisite for making factual
evidence reviewable. The actual shipping long report is now retained as a
**failure**, not a pending or passing run. A fresh passing qualification is still
required. All original failures remain failures. Release-byte claims cannot
follow solely from unchanged
Go/module source. Stable-tag publication, actual Homebrew upgrade and installed-v1
verification remain separately authorized and separately evidenced.


## Inspectable public evidence

| Evidence | Inspectable content |
| --- | --- |
| [Source/qualification manifest](../evidence/v1-qualification.json) | All 69 source-file identities, measured executable, source gates, exact CI URLs, input hashes and pending publication boundaries |
| [Retrieval and judgments](../evidence/v1-retrieval.json) | Authored membership/order/filter labels, measured ranked outputs, category membership and null negative-only means |
| [Reliability](../evidence/v1-reliability.json) | Effective profiles, clocks, counts, checkpoint fidelity, current interruption cases and explicit long-profile status |
| [Complete resource matrix](../evidence/v1-resources.json) | All 16 unchanged-budget rows, baseline/candidate drift, separate rebuilds, native quiet-state facts and bounded independent audit |
| [Prior-version compatibility](../evidence/v1-compatibility.json) | Actual preserved v0.3.1/v0.4.0 identities, upgrade/rollback/archive/recovery observations and post-publication limits |
| [Failed-attempt history](../evidence/v1-qualification-history.json) | All four failed matrices, original pre-fix failed endurance, shipping restore-timeout failure and historical-runtime scope |

The selected projections contain synthetic product data and measured metadata,
not a private collection or raw control/test transcripts. Input hashes anchor
retained source observations without exposing physical executable, sandbox,
home or output paths. Unreported core sample arrays/per-query RSS remain null,
not reconstructed. The original [backup scaling report](backup-performance-v1.md)
retains its actual 2k-receipt paired samples, freshness checks and RSS regression;
this snapshot does not broaden that claim.

## Reproduction and smoke commands

Supply preserved executables with the exact identities above; never point product
mutation tests at a live collection. These harnesses create fresh disposable
synthetic roots. The following relative paths are examples, not original physical
workstation paths. Serialize the resource experiment separately from correctness
soaks/builds/profiling and retain failed reports under distinct names.

```sh
# Fast functional smoke for CI wiring; not a D02 qualification.
GOMAXPROCS=2 python3 scripts/soak.py --binary .build/kb-candidate --duration 3 --seeds 1729 --writers 4 --minimum-work 1 --backup-interval 1 --command-timeout 30 --output .build/v1-smoke.json

# D02 extended profile: the absence of --maintain-log explicitly selects false.
GOMAXPROCS=2 python3 scripts/soak.py --binary .build/kb-candidate --duration 1800 --seeds 1729 2718 31415 --writers 4 --minimum-work 10000 --backup-interval 60 --command-timeout 30 --output .build/v1-endurance.json

# Separate short default-log profile; not 30-minute default-log evidence.
GOMAXPROCS=2 python3 scripts/soak.py --binary .build/kb-candidate --duration 60 --seeds 1729 2718 31415 --writers 4 --minimum-work 250 --backup-interval 10 --command-timeout 30 --maintain-log --output .build/v1-default-log.json

# Deterministic observed-boundary children, repeated three times.
CGO_ENABLED=0 GOMAXPROCS=2 go test ./internal/write -run '^TestRealProcessInterruption(RecoveryAndOperationalArchives|EveryPublicationAndRecoveryPosition|EveryRestoreFilePosition)$' -count=3 -json -timeout=5m

# Fresh retrieval/parity with the unchanged authored corpus.
GOMAXPROCS=2 python3 scripts/retrieval_eval.py --binary .build/kb-candidate --output .build/v1-retrieval.json

# Actual v0.4.0 producer archive plus upgrade/receipt/recovery checks.
GOMAXPROCS=2 python3 scripts/compatibility.py --baseline .build/kb-v040 --baseline-version v0.4.0 --baseline-commit 74f0d196408c1354899802e18ce569b0eab33adf --candidate .build/kb-candidate --candidate-version v1.0.0-candidate.3 --candidate-commit b71c0763c87313666b3dd79da08da12b7b3512c1 --output .build/v1-v040-compatibility.json

# D03 operation matrices, baseline then candidate, during a reserved quiet window.
GOMAXPROCS=8 python3 scripts/benchmark.py --binary .build/kb-v031 --sizes 10000 50000 --samples 20 --repair-samples 3 --output .build/v1-baseline-resources.json
GOMAXPROCS=8 python3 scripts/benchmark.py --binary .build/kb-candidate --sizes 10000 50000 --samples 20 --repair-samples 3 --output .build/v1-candidate-resources.json
GOMAXPROCS=8 python3 scripts/benchmark.py --binary .build/kb-v031 --sizes 10000 50000 --samples 5 --rebuild-only --output .build/v1-baseline-rebuild.json
GOMAXPROCS=8 python3 scripts/benchmark.py --binary .build/kb-candidate --sizes 10000 50000 --samples 5 --rebuild-only --output .build/v1-candidate-rebuild.json
```

The benchmark commands reproduce the public fixture/operation method, not native
quiet-state orchestration. The accepted recording-only hook and evaluator-match
checks have separate input hashes in the resource evidence. It retained all
successful returned duration/RSS observations without response/request payloads;
no failed sample or complete failed matrix was filtered into a pass. Core timing
is separately measured from the identified unchanged Go benchmark and is not
inferred from CLI or MCP timings.
