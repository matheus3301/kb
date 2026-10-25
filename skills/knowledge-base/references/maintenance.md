# Review, maintenance, and recovery

## Choose the check

```sh
kb status
kb validate --profile okf
kb validate --profile health
```

`okf` checks format/policy conformance. `health` adds structural/reference and
synthesis-dependency findings. A validation request may succeed (`ok: true`) with
`data.conformant: false` and CLI exit 1. Inspect diagnostics, not only `ok`. Reports also expose `clean` and
`summary.errors/warnings`; `--strict` exits 1 for warnings too. For a degraded
collection, follow [Repair](repair.md) to correct notes in bounded batches.

Review outdated synthesis by reading its claims and current source notes. Changed
basis revisions and stale timestamps flag a review need; they do not prove the
claim false. Reading and evidence evaluation are required to resolve a
contradiction. Neither validation nor maintenance certifies semantic truth or
renews verification.

## Navigation versus search state

```sh
kb maintain --dry-run
kb maintain --apply
kb index rebuild
```

`maintain` previews by default. It updates managed index regions and associated
bookkeeping while preserving manual content. Inspect its changes before applying
under the task's write authorization. It does not synthesize new knowledge.

`index rebuild` rebuilds disposable search state. It is a write-class operation,
but does not replace semantic review or repair malformed source files. Fresh
inventory and source revisions remain authoritative. Do not delete operational
journal/receipt files as if they were the search cache.

## Pending transaction

For `RECOVERY_REQUIRED`, explain the interrupted operation and inspect it:

```sh
kb recover --dry-run
```

Read the result and, when recovery is authorized, run:

```sh
kb recover
kb status
kb validate --profile okf
```

Prepared transactions roll back; committed transactions roll forward. Reads
blocked by a pending journal do not silently recover it. A new mutation can
perform automatic recovery before its own operation. If recovery fails, preserve
the journal and affected files and report the failure. Never hand-edit journals
or receipts, force a revision over an external edit, or delete evidence to make
a read succeed. Advisory locks do not isolate arbitrary editors; ask users to
coordinate external editing when an actual conflict requires it.

## Export and integration boundaries

```sh
kb export ./example-export
```

Export validates the scoped content and writes a snapshot with an inclusion/
exclusion manifest to the selected destination. It does not back up operational
journals, durable retry receipts, or certify excluded files. Choose an appropriate
new destination; do not overwrite a user's unrelated directory.

## Operational backup and restore

Export is not a complete operational backup. For an authorized operational
snapshot, first resolve any pending recovery with the workflow above, then:

```sh
kb backup .kb/backups/snapshot.tar
kb restore .kb/backups/snapshot.tar .kb/restores/verification
kb --root ./example-kb/.kb/restores/verification status
kb --root ./example-kb/.kb/restores/verification validate --profile okf
```

The first two commands use the already selected root. Replace `./example-kb` in
the last two with that same explicit root; restore never switches it implicitly.
Both operations require writes enabled and have no dry-run. Archive/destination
must be direct children of `.kb/backups`/`.kb/restores` respectively. Restore
requires a nonexistent destination, refusing even an existing empty directory.
It never overwrites a live base. Rebuild the disposable cache with `index rebuild`
on the explicitly selected restored root when needed.

Backup preserves exact regular-file bytes throughout the root, including
out-of-scope notes/assets, `.kb/config.toml` and durable receipts. It excludes
cache, backup/restore storage, locks, Git, dependency and agent directories; the
versioned manifest lists actual excluded paths and included file checksums.
Unknown operational state, pending journals, symlinks and hard links are refused,
not silently discarded. Review exclusions; source artifacts outside the root
need separate protection. No conformance, authenticity or off-device protection
is implied. Protect archives like the original content, and copy them to separate
protected storage through the user's chosen backup mechanism.

Restore checks paths, sizes, entry kinds, manifest and SHA-256 before publishing
configuration last. Ordinary failures remove only unchanged files it created;
external edits are retained. A process interruption can leave an uninitialized
destination or a complete root after configuration publication. Never initialize
or resume into an incomplete directory: inspect it, retain necessary evidence,
and select a fresh destination. Existing outputs are conflicts, not idempotent
request replays. Do not delete pending journals or receipts to make backup work.
Coordinate external editing; advisory locks do not protect against arbitrary
same-user filesystem mutation or guarantee recovery from hardware power loss.


Respect CLI read-only and write-disabled errors; do not change roots or
permissions to bypass them. Each CLI invocation opens the configured base afresh.
See [Results](results.md) for interpreting errors and output limits.
