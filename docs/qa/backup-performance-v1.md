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
cleanup from timing. One baseline iteration per size measured:

| Receipts | Timed backup + restore | Allocated bytes/op | Allocations/op |
| ---: | ---: | ---: | ---: |
| 250 | 2.346 s | 247,641,128 | 2,171,596 |
| 2,000 | 49.613 s | 14,690,601,872 | 122,407,649 |

The separate Go CPU/allocation profiles include fixture generation, including
its create/patch transactions. Their totals are **not** operation-only CPU,
peak RSS or timed benchmark allocations. Across that whole profile, 94.5% of
sampled CPU was in syscall dispatch; directory enumeration/stat/dirent construction
dominated allocation attribution. The benchmark's bytes/op above exclude fixture
generation and are distinct from child-process RSS.

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

## Qualification boundary

The complete restarted 1,800-second reliability observation remains a separate
coordinator gate with the default 30-second command timeout unchanged. Small and
large receipt samples, profiles, green checks and an experimental release do not
establish that observation or stable-v1 readiness. Full candidate checks, actual
released-baseline paired measurements and committed-source release snapshot must
be linked with their exact candidate identity before final integration claims.
