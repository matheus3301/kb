# v0.1 implementation contract

This document closes implementation gaps in spec v0.3. The public wire version
is `kb.v1`; the first release is `v0.1.0`. `internal/api/types.go` is the shared DTO
contract. Coordinate changes through the integration owner rather than changing
shared types independently in feature branches.

`api.Revision` is SHA-256 of complete file bytes, prefixed `sha256:`. Shared
`api.Generation` hashes sorted scoped Markdown paths joined as path, NUL,
revision, newline. Config and operational files are excluded. Pagination also
binds the effective query/config so a filter change cannot reuse an old cursor.
Use the shared limits in api rather than duplicating transport limits.

## Package ownership

- `internal/api`: shared types, owned by integration.
- `internal/filelock`: advisory shared/exclusive locks, owned by the read-core agent.
- `internal/core`: scope, config, codec, inventory, search, links, context, validation,
  cache and export, owned by the read-core agent.
- `internal/write`: transactions, repair, maintenance and recovery, owned by the
  write-core agent. Depends on `api.Provider`, `api.Codec`, and `filelock`, not core.
- `internal/cli`, `internal/mcpserver`, `cmd/kb`, `skills/knowledge-base`: owned by
  the adapters agent. Wires `core.Store` and `write.Service` together.
- Integration owns release packaging, end-to-end acceptance, privacy/link
  guardrails and final public documentation.

## Read core API

`core.ResolveRoot(explicit string) (string, error)` uses the explicit flag first,
then KB_ROOT, then an ancestor `.kb/config.toml`. It never scans the home directory.
For `init`, an explicit root may not exist yet and is created; without an explicit
root or KB_ROOT, init uses the current directory rather than discovery.

`core.Init(ctx context.Context, root string, para bool) (*Store, error)` creates
only missing base files and local config, never overwrites existing notes or
instructions, and is idempotent. The default include is all Markdown `**`; PARA
only changes initial folders/navigation. `.kb`, `.git`, dependency folders and
agent operational folders are always excluded. Optional user includes/excludes
are honored. The default config enables writes and index/log maintenance.

`core.Open(root string) (*Store, error)` opens an initialized base. Store exposes:

```go
Root() string
Config() api.Config
Engine() api.Codec
Snapshot(ctx context.Context) (api.Snapshot, error)
SnapshotUnlocked(ctx context.Context) (api.Snapshot, error)
Status(ctx context.Context) (api.Status, error)
List(ctx context.Context, request api.SearchRequest) (api.SearchResult, error)
Search(ctx context.Context, request api.SearchRequest) (api.SearchResult, error)
Read(ctx context.Context, request api.ReadRequest) (api.ReadResult, error)
Context(ctx context.Context, request api.ContextRequest) (api.ContextResult, error)
Links(ctx context.Context, path, direction string) ([]api.Link, error)
Validate(ctx context.Context, profile string) (api.ValidationReport, error)
Rebuild(ctx context.Context) (api.Status, error)
Export(ctx context.Context, destination string) (api.ExportResult, error)
```

Read operations acquire the base shared lock. If `.kb/state/pending.json` exists,
return `RECOVERY_REQUIRED` without changing notes. SnapshotUnlocked is called only
by a caller already holding the exclusive lock, and does not acquire another
lock or reject the pending marker. No other public method may bypass locking.
An immutable in-process snapshot can be shared among read workers; changes must
invalidate it by checking the actual file inventory, not by trusting the cache.

`filelock.Acquire(ctx context.Context, root string, exclusive bool) (*Lock, error)`
uses `.kb/state/lock` and a context-aware OS advisory lock on supported Linux and
macOS targets. `(*Lock).Close() error` releases resources. Creation of operational
lock/cache files is allowed during reads; notes, config and Git are not changed.
All filesystem operations remain confined to the base. Reject symlinks, traversal,
absolute file arguments and case-fold destination collisions.

The codec's Parse returns diagnostics instead of inventing missing metadata.
Patch supports body replacement, unique exact text edits, metadata patch and
explicit key removal, preserving unrelated presentation. Body-only edits retain
untouched frontmatter spans; generated metadata is refreshed on meaningful edits,
and trust invalidation removes verified. Changes
must never add a human verification claim. Preserve unknown keys and YAML comments.
Create calls Patch with empty input and a create operation. Document replacement
is only for an explicit repair plan; the writer enforces that precondition.

## Write API and recovery

```go
func write.New(provider api.Provider) *Service
func (*Service) Apply(ctx context.Context, plan api.Plan) (api.Receipt, error)
func (*Service) Maintain(ctx context.Context, dryRun bool) (api.Receipt, error)
func (*Service) Recover(ctx context.Context, dryRun bool) (api.Receipt, error)
```

`api.Operation.kind` is create, patch or move. Path is always explicit; a front end
may derive a suggested path before constructing the operation. Create requires a
new path and nonempty textual type. Patch/move require expected_revision. Move
uses `to`. Body and body_edits are mutually exclusive. `metadata` is create input;
`metadata_patch` and `remove_metadata_keys` are patch input. `document` is an
explicit complete-file replacement accepted only with `plan.repair: true`.
Repair permits invalid pre-state but requires a conformant post-state. Normal
writes require both states to conform. No automatic migration or invented type.

An apply plan requires nonempty summary, actor and idempotency_key. Missing
operation_id is generated for a single operation, but multi-operation plans need
unique IDs. Dry run does not persist a receipt, modify notes, create directories
for note targets, or update indexes/log. It returns bounded diffs and revisions.
Validate all paths, operations, existing revisions and synthesis dependencies
before publishing any note. At most 50 operations and 4 MiB input; note size at
most 1 MiB. Reject overlapping source/destination paths and multiple operations
on the same path. Update known links and basis_revisions on move, and clear
verified on substantively changed notes. Never touch raw source snapshots.

Use an exclusive lock. Mutations recover any pending journal before accepting a
new request. Check a durable receipt before revalidating retry preconditions.
Request hashing excludes dry_run; the same key and effective request replays the
receipt, while a different request conflicts. Receipts have no expiry in v0.1.

The journal is `.kb/state/pending.json`, written atomically and fsynced before
publication. It stores version, request digest/key, prepared/committed state,
before/after file bytes and the receipt. Replace affected files using same-filesystem
temporaries, sync file and directory metadata, then persist a committed marker.
Before the durable commit marker, recovery rolls back the before-state. After
the marker, recovery rolls forward and reconstructs the durable receipt. Persist
receipts under `.kb/state/receipts/` using a hash of the key, then remove the
pending journal durably. Never report success after partial publication or failed
durability. Interrupted recovery is itself idempotent. Tests must inject failures
at each boundary. Readers outside this tool may see multi-file intermediate
states; external writers do not honor advisory locking, so no absolute isolation
from them is claimed. Recheck their revisions up to the publication boundary.

Generated indexes use explicit HTML marker regions and preserve manual content.
Do not silently take over unmanaged text. Init supplies managed regions in new
indexes, and Maintain dry run previews changes. One logical write produces one
dated log event; no-op and reads produce none. Preserve older log entries and use
UTC for new dates/timestamps. Cache is disposable; journal/receipts are not.

## Front ends

CLI root flag is global. All data commands offer --json and emit one `kb.v1`
envelope with ok, data/error, warnings, and meta (root, generation, index_stale).
Logs go to stderr. Domain exit codes remain as in the spec; a validation report
can have ok=true with conformant=false and CLI exit 1. Use --input FILE or - for
JSON requests, reject unknown fields, and bound stdin. Create/patch/move/archive
wrap one operation into Apply, accepting summary, actor and idempotency_key from
the input alongside operation fields. Apply accepts the Plan directly.

CLI includes init, status, list, search, read, context, links, create, patch, move,
archive, apply, validate, index rebuild, maintain, recover, export, mcp and version.
Add --dry-run to writes; maintain applies only with --apply. Recover is explicit
and also supports dry run. Read defaults to a bounded response; section headings
must be unambiguous or addressed by an explicit section identifier.

MCP uses the official Go SDK over stdio, fixed to the startup root. Expose the ten
tools in the spec using the same DTOs, input limits, policy and errors as CLI.
Tools return structuredContent plus compact text. Read-only hides and rejects
writes. Validation findings are report data, while execution failures set isError.
Expose kb://local/ resources without granting filesystem access outside the base.
No generic shell or tool-driven root switching. Both front ends must cancel
operations and release locks when their context ends.

## Release and evidence

Distribution includes four CGO-free binaries (Linux/macOS, amd64/arm64),
checksums, MIT license, English docs and an installable knowledge-base skill.
Only synthetic public fixtures are allowed. Public URLs use this repository or
public upstream sources; local file links may not leave the checkout. Intentional
multilingual search fixtures are labeled synthetic, not user data.

Every acceptance claim references actual tests and recorded evidence. Performance
targets remain measurements to run, never claims inferred from a green CI build.
