# v1 reliability qualification

Status: **proposed D02 observation scope, not v1 readiness**. The harness operates
only on newly created disposable synthetic roots. It cannot select an existing
knowledge base. Evidence from the integrated candidate, independent review and an
explicit readiness decision remain required by the [exit checklist](../stability.md).

## Proposed D02

Run **30 minutes total across seeds 1729, 2718 and 31415**, divided equally,
with **four concurrent writer workers**. Require **at least 10,000 acknowledged
main-root mutations plus identical-request replays**, in addition to CAS
conflicts. Setup and restored-root operations do not count toward this minimum.
Do not lower the duration, worker count or useful-work minimum to obtain a pass.
If the minimum is missed, retain the failed report and investigate throughput or
propose an explicit scope decision. The full observation is parent-run after
integration; a short functional smoke is not that observation.

This is finite local-process correctness on the tested local filesystem. Record
the volume/filesystem type, OS, architecture, candidate commit and toolchain
alongside the JSON report. The harness records the binary SHA-256, platform,
Python version, seeds, timings, command outcomes and exact round counts. It does
not detect the filesystem type portably. No universal power-loss, controller,
filesystem, hostile-same-user or external-editor isolation guarantee follows.

## Workload and invariants

The fixed corpus contains one shared reference, one reference per worker,
reserved navigation/log files and one 2 KiB immutable binary source. Bodies stay
bounded; each round replaces the previous body with a seeded unique token.
Each seed starts from a fresh root. The explicit sustained profile enables
navigation maintenance and disables event-log maintenance (`maintain_log=false`).
Receipts intentionally grow linearly; they are never deleted as disposable cache.
The current receipt format embeds before/after full-file diffs, with an aggregate
**256 KiB diff cap per receipt**. Retained bytes accumulate approximately
quadratically while growing log images fit that cap, not indefinitely; once
capped, receipt count still grows without expiry. Log/note and operational-archive
limits remain finite. This bounded profile isolates sustained CAS, replay and
backup correctness; it does **not** qualify sustained event-log maintenance.
`--maintain-log` selects a complementary short profile using the product's normal
log-enabled behavior. The parent should also run 60 seconds total across the same
three seeds/four writers, with at least 250 acknowledged mutations/replays and
periodic backup/restore. Neither that profile nor deterministic log-enabled
SIGKILL tests substitute for the 1,800-second/10,000-work observation.
Full-log diff storage growth is a disclosed operating limit, not removed by the
profile choice. Each report declares the selected setting and records initial,
periodic and final receipt/log byte counts plus acknowledged work per active second.

Each round launches four independent CLI writer streams using a thread pool:

1. Read and patch each worker's private note; check the confirmed body and exact
   receipt revision through a fresh CLI read. Replay the identical request and
   compare the entire receipt, allowing only `replayed: true` to differ.
   Require the note's immutable-source provenance to survive every confirmed edit.
2. Synchronize the workers at a barrier, then issue four shared-note patches
   against the same revision. Require exactly one success and three
   `REVISION_CONFLICT` results with exit code 4. Replay the winning request, read
   the shared note and require the winning body/revision. No lock, timeout,
   idempotency conflict or other error is considered a safe conflict.
3. Search and require every current worker/shared note. Four workers produce
   exactly five mutations, five replays, three conflicts, ten reads and one
   search per completed round. Command-level success/conflict/error counts and
   per-command call count/total/max elapsed time are recorded separately.

There is no sleep-only observation window: the loop continues doing and checking
real operations until each seed's duration expires, finishing its current round.
Setup and final validation add elapsed time beyond the requested active duration.
Concurrency is real process overlap, not simultaneous filesystem publication:
KB's advisory lock serializes compliant writers. The winner is scheduler-dependent;
seeds reproduce input bodies and requests, not OS scheduling.

At the configured backup interval (default 60 seconds), and at every seed's end,
quiesce writer streams at a complete round and:

- Require exactly the confirmed receipt set and full original receipt values.
  Reject a pending journal or changed binary source. With logs disabled, require
  byte-identical initial log content. With logs enabled, require exactly one
  logical log event per confirmed receipt, with the correct summary/key and actor
  provenance; no missing, duplicated or unexpected events.
- Hash all fixture content, configuration and durable receipt files. Back up and
  restore through real CLI processes; require identical manifest revisions and
  exact authoritative path/hash inventories.
- Replay a confirmed request in the restored root, rebuild its cache, read it,
  commit a new revision-checked mutation and replay that continuation. Confirm
  the continuation body and immutable source.
- Require the source root's authoritative inventory to remain unchanged.
  Remove only this disposable verified archive/restore pair to bound staging
  storage; never delete receipts from the active synthetic root.

Periodic backups are settled snapshots, not backups during active publication.
The separate deterministic interruption tests cover pending/recovery behavior.
The harness exercises CLI only; existing CLI/MCP parity tests remain necessary.
It does not cover arbitrary inputs, all possible schedules, long-term retention
or off-device disaster recovery. See the [backup contract](../backups.md).

## Reproduction

Build the candidate using the repository's serialized build/check runner where
multiple worktrees share a machine. Use `GOMAXPROCS=2` for routine work. Reserve a
quiet slot with the coordinator for the qualification; do not overlap performance
measurements. The runner path is installation-specific and is not required by
the public harness.

```sh
GOMAXPROCS=2 go build -trimpath -o .build/kb ./cmd/kb
python3 -m unittest discover -s scripts -p test_soak.py -v

# Short real-process smoke suitable for parent CI wiring, not D02 qualification.
python3 scripts/soak.py --binary .build/kb --duration 6 \
  --seeds 1729 2718 31415 --writers 4 --minimum-work 30 \
  --backup-interval 1 --output .build/soak-smoke.json
python3 scripts/soak.py --binary .build/kb --duration 6 \
  --seeds 1729 2718 31415 --writers 4 --minimum-work 30 --maintain-log \
  --backup-interval 1 --output .build/soak-log-smoke.json

# Parent-run integrated-candidate observation; do not substitute the smoke.
python3 scripts/soak.py --binary .build/kb --duration 1800 \
  --seeds 1729 2718 31415 --writers 4 --minimum-work 10000 \
  --backup-interval 60 --output .build/soak-qualification.json

# Complementary normal-log profile; the parent runs this after integration too.
python3 scripts/soak.py --binary .build/kb --duration 60 \
  --seeds 1729 2718 31415 --writers 4 --minimum-work 250 --maintain-log \
  --backup-interval 10 --output .build/soak-log-qualification.json

# Repeated observation-boundary SIGKILL/recovery, separately from the soak.
GOMAXPROCS=2 go test ./internal/write -run '^TestRealProcessInterruption' \
  -count=3 -timeout=5m -v
```

Every CLI invocation has a configurable timeout (default 30 seconds). A timeout,
malformed/inconsistent envelope, unexpected exit/code, missing effect, altered
receipt/source, count mismatch, failed seed or missed minimum fails the report
and process exit. All requested seeds must complete in order. Report files hold
compact aggregate evidence, not note bodies or large raw logs. Failures use
structured category/error-type/check/operation metadata, never raw exception
commands, environment, response bodies or paths. OS errors expose only their
numeric errno. Keep detailed local diagnostics outside public evidence; publish
only reviewed aggregate results.

## Deterministic interruption coverage

The existing `TestRealProcessInterruptionRecoveryAndOperationalArchives` covers
`prepared`, `published`, `committed`, `receipt`, `removed`, `recovered`,
`backup-synced`, `backup-published`, `restore-file`, `restore-validated` and
`restore-published`. Previously the repeated file boundaries always selected the
first file, and recovery interruption covered rollback only.

The expanded tests select each file position in a three-note transaction's
publication and both rollback and committed roll-forward recovery. A child
announces the exact stage/path only after reaching that boundary, then blocks;
the parent observes the marker and sends SIGKILL. Marker polling is not a guess
about when publication occurs. Each case verifies settled final revisions,
all coordinated bodies, one logical event, identical replay receipts, preservation
of an already acknowledged receipt and unchanged source bytes. Restore tests
select each included non-configuration file position, including receipts and
binary source, then verify fresh-destination restore/replay fidelity. Configuration
publication remains covered by `restore-published`.
These children re-execute the Go test binary with the synthetic filesystem
provider and real write service; they are not SIGKILLs of the public CLI binary.
The concurrent soak separately exercises actual CLI invocations.

Uncovered: interruption inside an individual write/fsync/rename/link system call,
all instruction-level gaps between hooks, storage/host failure, and adversarial
same-user interference. Named hooks bracket observable boundaries; they are not
proof that every possible interruption has been enumerated.

## Worker validation

The following short functional observations used macOS/Darwin 25.6.0 arm64,
APFS temporary roots, Go 1.27.3, Python 3.14.4 and `GOMAXPROCS=2`.
Each profile requested six active seconds total, two per seed, with four writers
and one-second backup intervals. These are small correctness smokes, not paired
performance benchmarks, long-run throughput predictions or the D02 observation.

| Log maintenance | Seed | Mutations / replays / conflicts | Work/s | Receipt bytes, setup → final | Log bytes, setup → final |
| --- | --- | --- | --- | --- | --- |
| Disabled | 1729 | 30 / 30 / 18 | 28.27 | 6,431 → 36,953 | 23 → 23 |
| Disabled | 2718 | 30 / 30 / 18 | 28.18 | 6,431 → 36,955 | 23 → 23 |
| Disabled | 31415 | 30 / 30 / 18 | 28.99 | 6,436 → 37,036 | 23 → 23 |
| Enabled | 1729 | 25 / 25 / 15 | 24.53 | 10,773 → 138,329 | 567 → 3,182 |
| Enabled | 2718 | 25 / 25 / 15 | 24.72 | 10,775 → 138,334 | 567 → 3,182 |
| Enabled | 31415 | 25 / 25 / 15 | 23.44 | 10,805 → 139,311 | 572 → 3,212 |

Work/s counts only verified main-root mutations plus replays, divided by actual
active duration including periodic backup checks. Setup/final checks are outside
that denominator. The disabled profile completed 180 acknowledged work items,
54 safe conflicts and six backup/restore checks in 8.452 seconds overall.
The enabled profile completed 150 work items, 45 safe conflicts and six checks
in 8.371 seconds overall. Both reported zero command errors. The report counts
setup and restored-root mutations/replays separately.

Eleven harness regressions passed, covering CAS winner cardinality, exact replay
receipts, missing/inflated counts, minimum-work/seed failures, config/receipt/source
fidelity, log mutation, missing/duplicate/wrong-actor events, provenance loss and
rejection of non-CAS errors as safe conflicts. Real timeout and permission-denied
subprocess failures verify classified nonzero reports with no executable, root,
temporary-directory or home prefix. Untrusted exception/response messages are
also excluded from public failure metadata.

An actual CLI negative smoke confirmed exit 1 and `ok:false` at a one-microsecond
command timeout, with `TimeoutExpired`/`cli-command-timeout`/`init` classification.
A second confirmed exit 1 for ten acknowledged work items against a required
999,999. Successful operations alone cannot manufacture a qualifying report.

The final focused interruption run passed three repetitions in 20.253 seconds:
34 observed-boundary child SIGKILL cases per iteration, 102 total. This includes
six publication positions, six rollback positions, six committed roll-forward
positions, five restore-file positions and the 11 existing named-boundary cases.

Full qualification is intentionally not run in this worker lane. The parent must
rerun the documented profiles against the integrated candidate; short-run rates
and storage growth are not a promise that either longer observation will pass.

Post-privacy-fix `make check` passed: 64 Python tests, static analysis with zero
issues, Go tests with 88.6% aggregate statement coverage, four build targets,
12 fuzz smoke targets, secret/vulnerability checks, real CLI acceptance,
43 executable documentation requests, benchmark smoke and retrieval evaluation.
The shared runner serialized heavy commands; no gate or threshold was weakened.
Coverage is not a durability or semantic-correctness proof.
