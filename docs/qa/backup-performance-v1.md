# Backup/restore receipt-history scaling

## Failure and independent reproduction

The first extended reliability attempt against source
`74f0d196408c1354899802e18ce569b0eab33adf` **failed**, not passed: seed 1729
stopped after 257.015 seconds and 477 rounds, with 2,385 acknowledged mutations,
2,385 replays and 1,431 expected CAS conflicts. Its third backup exceeded the
unchanged 30-second command timeout. Earlier checkpoints covered 729 files in
12.96 seconds and 1,589 files in 32.51 seconds for the combined backup/restore
workflow. No lost confirmed effect was demonstrated. These observations do not
satisfy the required 1,800-second qualification.

A separate fresh reproduction generated 250 and 2,000 durable receipts through
actual create/patch operations on five synthetic notes, with index maintenance
on and event-log maintenance off. Receipts were not fabricated or duplicated to
simulate confirmation. Both snapshots included a synthetic immutable binary
source, exact configuration and the service-produced receipts. Fixture generation,
copying and verification were outside the measured operations.

Environment: Apple M4, macOS 25.6 arm64, Go 1.27.3, `GOMAXPROCS=2`. Three samples
per size used fresh CLI children, child-only `wait4` peak RSS, and no OS cache
eviction. The reproduction executable identified itself as
`v0.4.0-candidate` at the full source commit above; it is not relabeled as an
installed release. The preserved actual Homebrew v0.4.0 executable is the baseline
for subsequent paired qualification.

| Confirmed receipts | Receipt bytes | Backup median / p95 | Restore median / p95 | Combined median / p95 |
| ---: | ---: | ---: | ---: | ---: |
| 250 | 217,187 | 558.3 / 558.9 ms | 1,762.8 / 1,807.9 ms | 2,321.7 / 2,358.4 ms |
| 2,000 | 1,726,341 | 24,113.6 / 24,810.8 ms | 23,926.3 / 24,065.9 ms | 48,179.5 / 48,661.8 ms |

For 250/2,000 receipts, backup child peak RSS was 19.7/22.4 MiB and restore peak
was 19.6/21.8 MiB. Every sample preserved exact source/restored file hashes,
configuration and receipt bytes, replayed a confirmed request without another
effect, and accepted a subsequent write in the restored root. Eight times as
many receipts caused about 20.8 times the combined median latency; this is a
bounded reproduction, not a general complexity proof or portable timing SLA.

## Paired released-baseline and committed-candidate measurements

The final comparison used the **actual preserved Homebrew v0.4.0 executable** at
`74f0d196408c1354899802e18ce569b0eab33adf`, not the development reproduction
binary. The candidate identified itself as **dev** at
`cf3860a82ac45fe2d1d1dacc29adc6d6e1dcb0da`. Version and full source commit were
asserted before either matrix; no development binary was relabeled as a release.

| Executable | SHA-256 |
| --- | --- |
| Actual Homebrew v0.4.0 | `bc61edae093dea6896b3f842ca2ab7441341e25f078495f1a6230cd25aa00443` |
| Committed dev candidate | `fe6df0097d87668e7a936b3c89089a939adb789528fee3eecf0b2dc2cbec708c` |

Both binaries used the exact same already-generated synthetic snapshots:
250 receipts/259 included files and 2,000 receipts/2,009 files. The hashes of the
sorted path-to-content-revision inventories were respectively
`5cf20abc89c3ed1dc79c4696b08ca0073f81fe0246cb500f4f4c8b0cdaefaa0a` and
`fcd003ecece6daa54e666ef80dc9848043edecef25edd930223dd9b1dc80ac04`.
Each sample copied its snapshot into a fresh disposable root before timing.

The complete baseline-then-candidate matrix held one shared measurement slot
without overlapping builds, tests or profiles. Hardware/toolchain, `GOMAXPROCS=2`,
three samples per operation/size, fresh CLI processes and child-only RSS method
were unchanged. No profiler was active and OS caches were not evicted. The
measurement harness's process watchdog is not a changed product or qualification
timeout. A three-sample nearest-rank p95 is the observed maximum, not a stable
tail-latency estimate.

| Receipts | Operation | v0.4.0 median / p95 (ms) | Candidate median / p95 (ms) | v0.4.0 / candidate peak RSS (MiB) |
| ---: | --- | ---: | ---: | ---: |
| 250 | Backup | 566.0 / 572.7 | 80.0 / 81.0 | 19.64 / 19.03 |
| 250 | Restore | 1,735.7 / 1,795.0 | 1,282.9 / 1,395.8 | 19.80 / 19.03 |
| 250 | Combined | 2,298.6 / 2,367.7 | 1,360.3 / 1,475.8 | Not a single process |
| 2,000 | Backup | 24,452.3 / 24,523.5 | 465.7 / 467.0 | 22.80 / 22.34 |
| 2,000 | Restore | 23,880.5 / 24,028.5 | 9,934.8 / 9,968.6 | 21.61 / 21.78 |
| 2,000 | Combined | 48,332.8 / 48,552.0 | 10,401.7 / 10,426.6 | Not a single process |

At 2,000 receipts the observed backup median improved about **52.5×** and combined
median about **4.6×**. RSS did not fall uniformly: restore peak increased by
0.17 MiB there. Every sample preserved exact source/restored content, config and
receipt hashes, replayed the saved confirmed request, and continued normal writes
without changing the immutable source.

Restore's per-file write/sync cost remains material: the candidate still took
about **9.93 seconds** for 2,000 receipts. This matrix neither measures arbitrary
larger histories nor establishes that they fit the qualification's 30-second
command limit. It does not replace the full restarted endurance observation.


## Cause and implementation

The old `inspect` listed each ancestor directory in full for every path.
`backupInventory` called it for each entry and `backupFile` repeated it over
multiple passes. Restore and cleanup also repeated parent-directory scans as
the receipt directory grew. This amplified a small number of source bytes into
quadratic directory enumeration/allocation work.

The repair in [backup.go](../../internal/write/backup.go),
[backup_paths.go](../../internal/write/backup_paths.go) and
[backup_inventory.go](../../internal/write/backup_inventory.go) keeps:

- A fresh complete actual-entry traversal for every inventory phase, including
  current file bytes, sizes, counts and exclusions. Empty included directories
  participate in fresh phase-local case/symlink checks. An expected-path map is
  never substituted for discovering the current inventory.
- Basename-by-basename anchored directory opens with non-symlink/type checks and
  `SameFile` identity comparisons. Traversals recheck directory identity after
  use; leaf operations recheck their current parent identity. Read leaves remain
  nonblocking/no-follow regular single-link files with bounded fresh bytes.
- Initial source inventory, per-file size/hash comparison during archive writing,
  then a fresh full inventory/case/symlink/byte comparison after stage sync and
  before no-overwrite publication.
- New-only restore roots and exclusive private leaf creation, checksums, file and
  directory sync, fresh final destination inventory and identity checks, fresh
  configuration/outer-path guards, and configuration-last publication.
- Fresh cleanup validation and exact bytes/size/type plus created-file identity
  before deleting owned output through anchored parents. Ambiguous or externally
  replaced/edited paths are retained, not removed to make cleanup appear complete.

There is no global cache, timestamp-only trust, new dependency, changed receipt
or archive format, receipt expiry/compaction, raised qualification timeout or
removed durability call. Advisory locking and these checks detect observed
replacements; they do not provide absolute isolation against malicious same-user
changes between filesystem calls.

## Profiling and regression scope

`BenchmarkBackupReceiptHistory` produces receipts through the transaction service
before `ResetTimer`, times backup plus restore, and excludes archive/destination
cleanup from timing. One baseline and one candidate iteration per size measured:

| Receipts | Baseline / candidate time | Baseline / candidate allocated bytes/op | Baseline / candidate allocations/op |
| ---: | ---: | ---: | ---: |
| 250 | 2.346 / 1.326 s | 247,641,128 / 11,833,664 | 2,171,596 / 134,069 |
| 2,000 | 49.613 / 10.723 s | 14,690,601,872 / 93,774,592 | 122,407,649 / 1,047,810 |

The separate Go CPU/allocation profiles include fixture generation, including
its create/patch transactions. Their totals are **not** operation-only CPU,
peak RSS or timed benchmark allocations. Across that whole profile, 94.5% of
sampled CPU was in syscall dispatch; directory enumeration/stat/dirent construction
dominated baseline allocation attribution. The candidate's timed allocation bytes
grew about 7.9× for 8× as many receipts, versus about 59.3× in the baseline.
These are bounded observations, not a universal linearity guarantee. Benchmark
bytes/op exclude fixture generation and are distinct from child-process RSS.

Reproduce the bounded service benchmark from the repository root:

```sh
GOMAXPROCS=2 go test ./internal/write -run '^$' -bench '^BenchmarkBackupReceiptHistory/(250|2000)$' -benchtime=1x -count=1
```

For actual CLI comparisons, generate the receipt history once through real
create/patch requests, copy each settled snapshot unchanged for each binary/sample,
then time `backup` followed by `restore` to a new destination. Keep generation,
copying, hash checks, restored replay/continued writes and cleanup outside timing;
retain the exact source identities and do not overlap the matrix with profiles.

[Receipt-history regressions](../../internal/write/backup_scaling_test.go) verify
all saved receipts/replays, continued writes, post-sync added/removed receipts
and unrelated files, same-size/same-mtime edits, spelling changes, symlink/hard-link
changes and ordinary extraction-fault cleanup.
[Parent regressions](../../internal/write/backup_parent_test.go) cover in-root and
outside symlink replacements, same-byte directory identity replacement during
anchored use/traversal, injected configuration case aliases and cleanup preserving
a same-byte foreign inode. Existing hostile archive, private-source, concurrency,
publication-fault, real-process interruption and recovery tests remain required.
Focused race-enabled backup/restore/recovery tests passed on the local platform.
The implementation's full `make check` also passed: 76 tooling tests, 88.7%
aggregate statement coverage, 12 fuzz targets, four cross-builds, scans, real CLI
acceptance, 43 documentation requests, retrieval evaluation and bounded soak smoke.
Those counts identify this worker's run, not a later integrated suite.

## Qualification boundary

The complete restarted 1,800-second reliability observation remains a separate
coordinator gate with the default 30-second command timeout unchanged. Small and
large receipt samples, profiles, green checks and an experimental release do not
establish that observation or stable-v1 readiness. The paired matrix identifies
one committed runtime candidate; later documentation-only changes do not create
new runtime measurements. Integrated candidate/release-snapshot checks and the
restarted full observation remain coordinator gates. Keep real failures and
filesystem/sample limits visible rather than increasing a timeout to hide them.
