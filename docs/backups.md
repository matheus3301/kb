# Operational backup and restore

Implemented in integrated source preparing **v0.4.0 qualification**, not in
published production **v0.3.1**. Candidate restoration/reliability evidence and
stable-v1 review remain pending under [V03/V04](stability.md).

Operational archives are distinct from `kb export`: export produces a validated
knowledge bundle and excludes operational state. Backup preserves exact bytes
and durable request receipts, without claiming that the collection is conformant.

## Workflow and authority

```sh
kb --root ./example-kb backup .kb/backups/snapshot.tar
kb --root ./example-kb restore .kb/backups/snapshot.tar .kb/restores/verification
kb --root ./example-kb/.kb/restores/verification status
kb --root ./example-kb/.kb/restores/verification validate --profile okf
kb --root ./example-kb/.kb/restores/verification index rebuild
```

Both operations require writes enabled and take the exclusive per-base advisory
lock. They have no dry-run, actor, summary or idempotency key. JSON inputs are
`{"archive":".kb/backups/snapshot.tar"}` and
`{"archive":".kb/backups/snapshot.tar","destination":".kb/restores/verification"}`.
The same inputs work independently through `kb_backup` and `kb_restore` MCP tools;
read-only mode hides and rejects both. Existing `kb.v1` DTOs remain unchanged.
Results report archive, optional destination, manifest_revision, files and bytes.
The manifest revision is SHA-256 of its exact canonical JSON bytes.

Archive paths must be direct children of `.kb/backups`; restore destinations
must be direct children of `.kb/restores`. Both storage directories must be
private (no group/other access). Names cannot start with a dot. Restore reserves
a **new** destination; even an existing empty directory is refused. No overwrite,
root override or implicit server-root switch is available. These restrictions
keep MCP's fixed-root authority intact. To use the restored base, explicitly
select that root in a new CLI invocation or restart the MCP server against it.

Same-disk storage is a staging/verification mechanism, **not disaster recovery**.
Copy the archive to separate protected storage with an independently selected
mechanism. To restore an externally retained archive, explicitly place a copy
in `.kb/backups` of an initialized writable staging base, then run restore there.
KB does not perform remote transfer, encryption, signing, retention or Git sync.
Treat archives as sensitive as the source: receipts can contain prior diffs.

## Included and excluded state

- Includes all regular root content, including out-of-scope Markdown and binary
  source artifacts, not only the configured knowledge scope. Comments, unknown
  metadata, line endings, revisions and configuration bytes remain exact.
- Includes `.kb/config.toml` and `.kb/state/receipts/*.json` without expiring or
  rewriting receipts. Identical successful request retries replay after restore,
  before checking now-stale preconditions.
- Excludes disposable `.kb/cache`, `.kb/state/lock`, `.kb/backups`, `.kb/restores`,
  Git, dependency and agent/editor directories. Actual excluded paths are listed
  in the manifest. The fixed directory exclusions are `.git`, `node_modules`,
  `vendor`, `.claude`, `.codex`, `.agents`, `.obsidian`, `.venv`, `venv`,
  `__pycache__` and `.idea`, at any depth.
- Refuses unknown `.kb` files, symbolic links, hard-linked included files and
  special files rather than silently dropping potentially authoritative state.
  Empty directories, original permission bits, ownership, timestamps, ACLs and
  extended attributes are not preserved. Restored files/directories are private
  (`0600`/`0700`, subject to the process umask).
- Files outside the selected root are not read or protected. Review exclusions
  and separately protect any external or excluded authoritative artifacts.

If `.kb/state/pending.json` exists, both operations return `RECOVERY_REQUIRED`
without recovering or discarding it. A partial multi-file snapshot is not a
safe recovery-state backup. Inspect `kb recover --dry-run`, perform authorized
`kb recover`, then retry backup. Pending journals are never archived. The archive
therefore represents a settled operation boundary with receipts, not a portable
pending transaction. Never delete durable state to make backup succeed.

## Archive format and limits

Version 1 is an uncompressed standard-library tar archive. The first regular
entry is `kb-backup-manifest.json`, canonical compact JSON with `version: 1`,
`kind: "kb-operational-backup"`, sorted `files` entries (`path`, `size`, `revision`)
and sorted `excluded` paths. Following regular files must match the manifest's
order, exact sizes and SHA-256 revisions. Configuration is required. The format
requires tar end markers and allows only zero padding after them. PAX `path`
records support long/Unicode names; other extensions are rejected.

Limits: 100,000 included files, 200,000 visited entries, 64 MiB per file, 1 GiB
content total, 16 MiB manifest, 1,152 MiB archive, 1,024-byte paths, maximum 32
path separators and 128-byte storage names. Configuration still obeys the normal
base-open limits. The restore parser rejects traversal, absolute/Windows paths,
case-fold collisions, file/directory conflicts, links, devices, sparse files,
unknown state, duplicates, missing/extra entries, malformed manifests, unsupported
versions, truncation, corrupt checksums and oversized inputs. JSON checksums
prove integrity against the manifest, **not authenticity against an attacker who
can replace both**. A restored configuration remains data, not action authority.

## Publication, failures and limits

Backup reads under the exclusive lock, verifies each file against its initial
hash and rechecks the full inventory before publication. It writes and syncs a
private temporary file, then publishes via a no-overwrite hard link and syncs
the directory. A failure before publication leaves no final archive; a failure
or process interruption after publication can leave a complete archive even if
no success response was delivered. Choose a fresh name or inspect the existing
archive; output conflicts are not durable request-replay receipts. An interruption
may leave a private `.stage-*` file in excluded backup storage.

Restore reserves a nonexistent directory, uses confined handles and exclusive
file creation, verifies every entry, detects destination identity changes and
rechecks extracted content, then publishes configuration last. Files and directory
metadata are synced. Before configuration publication an interruption leaves an
uninitialized destination; after it, the verified root may be complete despite a
missing response. Ordinary failures remove only unchanged files created by this
operation; unexpected external edits are retained and can prevent full cleanup.
A new attempt never reuses any existing destination. Inspect incomplete output,
retain needed evidence and choose a fresh destination; do not run `init` into it.

Advisory locks coordinate KB clients, not arbitrary editors. Hash/identity checks
catch observed races but cannot isolate malicious same-user directory changes
between filesystem calls. Keep storage private and coordinate external editing.
There is no atomic multi-file directory publication or claim of universal crash,
power-loss, disk/controller or filesystem durability. A successful restore is
not a semantic validation or an end-to-end disaster-recovery certification.

Operational size also grows with receipt history. Receipts retain full-file diffs
without expiry; repeatedly extending the event log can cause quadratic aggregate
receipt storage and eventually exceed archive limits. Monitor/protect that state;
do not delete receipts to fit a backup. No automatic compaction or receipt-format
change is part of this contract.

## Verification scope

Synthetic regressions cover CLI/MCP parity, exact content/config/receipt bytes,
revision fidelity, replay after restore, continued reads/search/cache rebuild and
writes, source immutability, hostile archives, concurrent output selection,
external edits/swaps, failure cleanup and fresh retries. Bounded mixed workloads
exercise create/preview/patch/conflict/move/replay with periodic backup/restore.
Real child processes are killed after observed transaction and operational
publication boundaries; recovery and restore retries are exercised afterward.
Archive and path parsers have fuzz targets. These are tested boundaries, not a
claim that all possible interruptions or hardware failures are covered.

D02 additionally requires two separately reported candidate profiles: at least
30 minutes across three seeds/four writers with log maintenance disabled but
indexes maintained and at least 10,000 acknowledged mutation/replay outcomes;
and at least 60 seconds total across three seeds/four writers with default log
maintenance enabled, at least 250 acknowledged mutation/replay outcomes combined
and backup/restore. Report unique effects and replays separately. Both use setup
creates followed by sustained patch/CAS/replay/read/search and periodic archives;
reads/search do not count toward the minima. Deterministic regressions separately
cover create/move/archive/apply/preview and multi-file publication. Both profiles
verify receipt replay and continued writes after synthetic restoration.
These extended observations and their independent review remain pending here;
the long profile must not be presented as default-log-enabled evidence.
