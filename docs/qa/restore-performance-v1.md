# Restore receipt-history diagnostics

**Historical diagnostic snapshot plus separately attributed production follow-up;
not a readiness decision.** The [whitelisted evidence](../evidence/restore-performance-v1.json)
preserves the older synthetic observations/source overlays and their limitations.
The appended production/6k section records later facts. Current candidate.5 local
qualification passed; hosted CI, V08 and actual installed-v1 proof remain pending.

## Shipping failure remains failed

The immutable `v1.0.0-candidate.3` executable, source
`b71c0763c87313666b3dd79da08da12b7b3512c1`, CGO=0, SHA-256
`8aecbf6cf22d2c4bc8c97e1d8b63fea725dfa325b2e5f632f2ed97fc815815ac`,
completed its shipping 1,800-second observation with **`ok=false`**. Seed 31415's
final restore reached **30.004420 seconds** at **5,450 confirmed receipts** and
exceeded the unchanged **30-second command timeout**. Seeds 1729 and 2718 passed.

The aggregate **16,840 mutations + 16,840 replays = 33,680 main-root
acknowledgments**, 10,104 expected CAS conflicts and sufficient work clocks do
not offset that failure. There were **25 backup attempts / 25 restore attempts,
but only 24 completed fidelity/replay/continued-write checkpoints**. The failed
final checkpoint's fidelity was not verified; no final storage inventory was
recorded for that seed. No lost confirmed state was demonstrated.

Original failed-report SHA-256:
`943e3a519b5a83c0b60849071025378abd4bcda42f729c5fcfae9594f97f2b45`.
The [qualification report](v1-qualification.md) retains that failure; successful
diagnostics below do not replace it or qualify a changed runtime.

## Immutable executable observations

Fresh synthetic receipts were produced through actual MCP create/patch, then
copied into disposable fixtures. Setup, backup and verification were outside
restore timing. The immutable shipping CLI ran one normal restore per size and
one additional externally sampled 5,450-receipt restore. No OS cache eviction
was claimed. Environment: macOS 26.6.2 arm64; receipt generation used
`GOMAXPROCS=2`, while these diagnostic restores used **8**, unlike the shipping
soak's **2**. They are not interchangeable workloads.

| Receipts | Included files | Normal child elapsed (s) | Normal peak RSS (MiB) | Extra sampled elapsed (s) |
| ---: | ---: | ---: | ---: | ---: |
| 250 | 259 | 1.434491 | 19.875000 | Not sampled |
| 2,000 | 2,009 | 10.701633 | 22.812500 | Not sampled |
| 5,450 | 5,459 | 27.634440 | 32.234375 | 29.811201 |

Each observation reported matching manifest identity, exact restored inventory,
configuration and receipts, and published configuration. These single successful
samples near 30 seconds are not a latency margin, tail estimate or endurance pass.

## Timers and the rejected syscall attribution

A separate Go overlay retained the original operations while recording timers,
CPU and allocation profiles. Observed build-info: Go 1.27.3, CGO=0, Darwin/arm64,
`-trimpath=true`; declared environment `GOWORK=off GOTOOLCHAIN=local`.
The **120-second diagnostic process guard does not change the 30-second shipping
qualification timeout**. Fixture generation was not profiled.

At 5,450 receipts, the timed reference observed:

| Timer | Aggregate seconds | Calls | Interpretation |
| --- | ---: | ---: | --- |
| `Restore.total` | 27.240591 | 1 | Service total; child elapsed was 27.437050 s |
| `restoreFile.file_sync` | 19.952687 | 5,459 | File synchronization dominates this observation |
| `restoreFile.parent_sync` | 1.775536 | 5,459 | Parent synchronization retained |
| `backupParent` | 2.672263 | 27,296 | Inclusive/nested parent-opening helper |
| `backupInventory` | 0.541610 | 1 | Final destination inventory |

**Do not add nested/inclusive timers or subtract them to invent a wall-time
breakdown.** In parallel variants, accumulated per-call durations also overlap
and can exceed the total wall time.

The first Python `os.fsync` control was a **mismatched primitive**, not a valid
attribution for Darwin Go `File.Sync`, which uses `F_FULLFSYNC`. Its 200-file
per-file-and-directory run took 16.871958 ms, with a 0.016209 ms file median.
It remains retained as rejected attribution, not evidence that product file
synchronization is cheap or that parent sync can be removed.

The matched `F_FULLFSYNC` control used command 51, 200 files of 1,024 bytes and
zero recorded fallbacks: **967.774542 ms elapsed**, **805.616542 ms aggregate file
sync**, file median **3.697729 ms**; aggregate directory sync **95.720257 ms**.
This supports the file-call cost explanation on this host, not a universal
power-loss guarantee or an exact prediction for a product restore.

A further matched scheduling control retained **all 200 file + 200 parent syncs**:

| Synthetic Python scheduling | Elapsed (ms) |
| --- | ---: |
| Immediate file/parent pair | 840.039292 |
| Prepare four, serial flush | 777.412583 |
| Prepare four, parallel flush | 447.511834 |

These are syscall controls, **not product evidence or an approved implementation**.
Order was not randomized and caches were uncontrolled. Overlapping call totals
cannot be summed into elapsed time.

## Profiled Go prototype: successful path only

One reference and one four-file-batch overlay observation used the same copied
fixtures at each size, reference first, with CGO=0/trimpath and timers plus
CPU/allocation profiling. No sample distribution, median or p95 is inferred.
Child elapsed includes startup/profile finalization; `Restore.total` is separate.

| Receipts | Reference child elapsed (s) | Batch-four child elapsed (s) | File / parent sync calls, each variant | Reference → batch peak RSS (MiB) |
| ---: | ---: | ---: | ---: | ---: |
| 2,000 | 12.001334 | 5.502124 | 2,009 / 2,009 | 25.687500 → 26.062500 |
| 5,450 | 28.147379 | 15.023788 | 5,459 / 5,459 | 33.734375 → 35.546875 |

The observations matched manifest/file/byte counts, exact restored
inventory/configuration/receipts and unchanged source. **RSS regressed:**
+0.375 MiB at 2,000 and **+1.8125 MiB (+5.373%)** at 5,450. At the larger size,
`Restore.total` was 28.016700 versus 14.833155 seconds; batch-four aggregate
file-sync duration was 34.160449 seconds because calls overlapped. No total CPU,
allocation, durability or uniform-memory improvement follows. Post-restore
replay/continued-write correctness is not inferred merely from equal bytes.

**The old producer boolean
`all_workers_join_before_ordered_hooks_and_cleanup=true` was an overclaim and is
not accepted.** Independent review found `removeOwnedRestoreLeaf` reachable
inside a worker before all workers had joined. These timings establish
**successful-path evidence only; the error-path invariant is not established**.
That old prototype required coordinator hooks/cleanup after all joins, retaining
ownership and parent handles. The separately committed correction is described
below; its source review does not retroactively validate these prototype bytes.

| Diagnostic executable | Version label | SHA-256 |
| --- | --- | --- |
| Timed reference | `diagnostic-restore-profile` | `5c164e338dff495d223fc5e65b58ee0744acbc843a7ab8024f789eaf12079594` |
| Old batch-four prototype | `diagnostic-restore-batch4` | `204f8a35a0cfec44ea6b1822681fece56b5d187c867a0b8c6f23ae9ae35d8ada` |

Both diagnostic artifacts report commit metadata
`5c94c993aa8ec97295bdf6bcdb1d5cd2f052a2f7` and `vcs.modified=false`.
**That does not fingerprint overlay contents or identify a production-fix
revision.** The JSON records separate overlay filenames/source hashes and binary
hashes. The underlying [public source](https://github.com/matheus3301/kb/tree/5c94c993aa8ec97295bdf6bcdb1d5cd2f052a2f7)
and [release-build recipe PR](https://github.com/matheus3301/kb/pull/23) do not turn
private diagnostic overlays into shipped tools.

The preserved first prototype attempt failed during diagnostic version-metadata
setup because profiling output configuration was missing: **zero restore samples
executed**, build succeeded. It is neither a product defect nor a failed
qualification run. Its sanitized disposition and original artifact hash remain
in the evidence; no panic transcript or private location is published.

## Reproduction and pending proof

These existing checked-in commands exercise public contracts and operational
workloads on disposable synthetic roots. **Run them only after profiling ends,
serially with other controlled measurements**; they were not executed to produce
this docs-only snapshot. Supply a separately verified candidate executable.

```sh
# Correctness/race mode is deliberately separate from CGO=0 performance mode.
GOMAXPROCS=2 CGO_ENABLED=1 go test ./internal/write -run '^TestRestoreContract' -race -shuffle=on -count=3

# Existing service benchmark: 250/2,000 receipts, backup PLUS restore, not CLI-only restore.
GOMAXPROCS=2 CGO_ENABLED=0 go test ./internal/write -run '^$' -bench '^BenchmarkBackupReceiptHistory/(250|2000)$' -benchtime=1x -count=3 -benchmem

# Actual extended qualification; absence of --maintain-log selects false.
GOMAXPROCS=2 python3 scripts/soak.py --binary .build/kb --duration 1800 --seeds 1729 2718 31415 --writers 4 --minimum-work 10000 --backup-interval 60 --command-timeout 30 --output .build/restore-endurance.json

# Separate default-log companion, not 30-minute default-log evidence.
GOMAXPROCS=2 python3 scripts/soak.py --binary .build/kb --duration 60 --seeds 1729 2718 31415 --writers 4 --minimum-work 250 --backup-interval 10 --command-timeout 30 --maintain-log --output .build/restore-default-log.json
```

The service benchmark generates real receipts before timing, measures backup
plus restore, and excludes output cleanup. It does **not** reproduce the private
5,450-receipt overlays or syscall-control helpers, which are not shipped entrypoints.
The [earlier receipt-scan repair](backup-performance-v1.md) is a separate bounded
comparison and retains its own RSS regression.

At the earlier diagnostic checkpoint, production correction, 6k scale and fresh
qualification were pending. The later section below updates those facts with
distinct source/artifact identities; the original measurements and rejected
error-path claim above remain unchanged. See [current qualification](v1-qualification.md)
and [stability](../stability.md) for the active candidate and remaining gates.


## Later committed production correction and native 6k scale

The old prototype and its rejected cleanup claim above remain unchanged.
Separately committed production source
`38ab64dfd2b362fb8848c7e331b0efb5d64b8ff6`, root integration
`5477946be88ef574499a07a7eabe7aa8b6bcc084`, now joins all file-sync/close workers
before coordinator-owned hooks, cleanup, retained-parent closure, final inventory
and config publication. Every prepared ownership record survives cancellation
and error draining. Independent focused source/R1 acceptance covers that committed
implementation, not the old overlay artifact, final V08 or universal power loss.
Targeted batch/contract race executions are owner-reported; later actual final
full-source gates are separately receipted.

Final source `cb66fd77a120d8c8531f9077a32d2bf51ef7921c` differs from 38ab in only
`backup_restore_batch_test.go` and `backup_restore_contracts_test.go`, for checked
channels and anchored fixture reads. All production Go/module bytes match.
[Source/review attribution](../evidence/v1-restore-production.json) preserves that
relationship without relabeling an executable.

Three paired native samples at **6,000 transaction-produced receipts** used actual
candidate.4/38ab, SHA-256
`78dc608ad6810188b695c4af796a0811e713f6fce03db5a218739a9ac9d22b93`, versus the
immutable candidate.3 baseline. All six samples finished within the unchanged
30-second guard and matched 6,009 files / 5,235,814 content bytes, exact
inventory/config/receipts, three restored old-request replays, continued writes,
original receipt purity and unchanged source/archive.

| Pair | Candidate.3 (s) | Candidate.4 (s) | Candidate.3 RSS (MiB) | Candidate.4 RSS (MiB) |
| ---: | ---: | ---: | ---: | ---: |
| 1 | 29.400285 | 24.701220 | 32.515625 | 31.781250 |
| 2 | 29.043362 | 24.588339 | 32.500000 | 32.484375 |
| 3 | 28.990590 | 24.834842 | 31.796875 | 32.812500 |

The pair-level peak RSS increased **32.515625 → 32.812500 MiB**. Native samples
used `GOMAXPROCS=2`, while the earlier instrumented prototype used 8; different
implementation/instrumentation contexts prevent a single-variable causal claim
or a promise of uniform timing gains. This is **candidate.4 evidence, not a
candidate.5 binary measurement**, despite the later test-only source delta.

Candidate.5 is separately built from cb66, SHA-256
`59d1b477031792cdf539aa3777fafc3942403b80aa8691a3403a8732bf928d46`.
Its actual 93-Python/88.8%-Go source gates and local native/source snapshot passed;
fresh D03 passed all 16 rows and three core targets; fresh functional reports and
the separate 1,200-ACK default-log companion passed. The completed extended
observation passed **34,840 ACK / 27 exact backup/restores**, with three seeds
over 600 active seconds, zero errors and maximum restore **24.227 seconds**
at no more than **5,889 checkpoint files**. This is distinct candidate.5 evidence,
not a relabeling of the candidate.4 6k pairs or historical failed candidate.3 run.
PR24 initial-head CI passed all 11 jobs; latest integrated-head CI, final V08
and actual publication/Homebrew/installed-consumer evidence remain pending.
[Current qualification](v1-qualification.md) contains all identities, drift,
historical failures and evidence limits.
