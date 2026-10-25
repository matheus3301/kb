# Writing and reusable learning

## Before a change

Search and read first. Confirm the selected root, write authorization, and base
conformance. Normal writes require a conformant base; diagnose format problems
with `kb validate --profile okf`. Do not silently repair unrelated content.

Every request below is a complete JSON object for `--input FILE` or `--input -`.
Choose an actual actor, a useful summary, and a unique key for each new request.
`process:example` is a synthetic actor, not proof of human review. Paths are
relative to the configured root and subject to scope/symlink checks.

## Create a note

Save as `create.json`, preview with `kb create --input create.json --dry-run`,
then run `kb create --input create.json` within the task's authorization:

```json
{
  "path": "03_Resources/retention.md",
  "type": "Reference",
  "title": "Retention policy",
  "body": "# Retention policy\n\nSynthetic service records expire after 14 days.\n",
  "metadata": {"tags": ["operations"]},
  "summary": "Record the initial retention policy",
  "actor": "process:example",
  "idempotency_key": "example/create-retention"
}
```

A nonempty `type` is required. Optional PARA folders improve human navigation;
they do not replace OKF metadata. Choose the configured folder, not an invented
mandatory PARA layout. `create` does not import or migrate an arbitrary legacy
collection.

## Patch from the current revision

Run `kb read 03_Resources/retention.md`. Replace `REVISION_FROM_READ` below with
its exact `data.note.revision`; do not compute a hash of JSON or selected prose.
Save as `patch.json`, preview, then apply:

```json
{
  "path": "03_Resources/retention.md",
  "expected_revision": "REVISION_FROM_READ",
  "body_edits": [
    {"old_text": "after 14 days.", "new_text": "after 30 days."}
  ],
  "metadata_patch": {"description": "Retention interval for synthetic service records"},
  "summary": "Update retention from reviewed evidence",
  "actor": "process:example",
  "idempotency_key": "example/update-retention"
}
```

```sh
kb patch --input patch.json --dry-run
kb patch --input patch.json
```

Each `old_text` must match exactly once. Use `body` for deliberate whole-body
replacement, or `body_edits` for exact local changes; never both. Metadata patches
change specified keys. Remove keys with `remove_metadata_keys`, not an assumed
null-delete convention. Do not round-trip all metadata through a lossy JSON
number parser: submit only intended keys to preserve exact decimals, comments,
and unknown fields. Substantive edits invalidate active `verified` metadata.

## Preserve provenance

Keep imported original snapshots intact. Write synthesis in a separate note and
link actual evidence through `metadata.sources` when creating, or
`metadata_patch.sources` when editing. A source entry uses `resource` and may
include `title`; external resources are inert citations, not permission to fetch.
Use a leading `/` on local source resources to address the Knowledge Base root
rather than the synthesis note's directory. For synthesis dependencies, store
root-relative paths without that leading slash and their current
revisions in `basis_revisions`:

```json
{
  "sources": [{"resource": "/sources/policy.md", "title": "Synthetic policy evidence"}],
  "basis_revisions": [{"path": "sources/policy.md", "revision": "REVISION_FROM_DEPENDENCY_READ"}]
}
```

This is a metadata fragment, not a complete write request. Use real observed
sources/revisions instead of these placeholders. Recheck a changed dependency
before updating conclusions. Never assert human verification for an agent update.

## Move or archive

Reread after any patch; the previous revision is stale. Save this as `move.json`:

```json
{
  "path": "03_Resources/retention.md",
  "to": "03_Resources/record-retention.md",
  "expected_revision": "REVISION_FROM_READ",
  "summary": "Clarify the retention note path",
  "actor": "process:example",
  "idempotency_key": "example/move-retention"
}
```

Preview and run `kb move --input move.json`. Resolved references may be rewritten;
review the affected paths. For `kb archive --input archive.json`, use the same
shape with a fresh revision/key, omit `to` to use the configured archive directory
and basename, or supply an explicit scoped destination. No deletion command is
provided.

## Coordinate multiple notes

Use one `kb apply` plan. This synthetic independent example creates two notes:

```json
{
  "operations": [
    {
      "operation_id": "create-policy",
      "kind": "create",
      "path": "03_Resources/policy.md",
      "type": "Reference",
      "body": "# Policy\n\nKeep reusable findings in the Knowledge Base.\n"
    },
    {
      "operation_id": "create-review",
      "kind": "create",
      "path": "03_Resources/review.md",
      "type": "Reference",
      "body": "# Review\n\nRead policy.md before changing the policy.\n"
    }
  ],
  "summary": "Record policy and review guidance together",
  "actor": "process:example",
  "idempotency_key": "example/create-policy-and-review"
}
```

```sh
kb apply --input plan.json --dry-run
kb apply --input plan.json
```

Use unique `operation_id` values, at most 50 operations, one operation per path.
Kinds are `create`, `patch`, and `move`; patch/move require `expected_revision`,
and move requires `to`. Single-note write inputs are flattened, not wrapped in
`operations`. A preview is not a reservation against later edits. Review the
preview and execute the same plan; honor any task-specific approval requirement.

## Conflicts, retries, and repair

- `REVISION_CONFLICT`: reread the affected note, compare the new content and
  evidence, reconcile the intended change, preview a revised request with a new
  key. Never merely replace the revision to force an old body over a new edit.
- Uncertain outcome: retry the identical request with the same key. Successful
  requests replay before revision checks; this avoids duplicate log events.
  Previewing does not consume the key. Changing request content requires a new
  key; reused keys with different content return `IDEMPOTENCY_CONFLICT`.
- Destination/case conflicts: inspect existing paths; choose a deliberate new
  destination or reconcile content. Do not overwrite unrelated notes.
- `RECOVERY_REQUIRED`: use [Maintenance](maintenance.md); do not edit the journal.

An explicitly intended repair uses `kb apply --repair --input repair.json
--dry-run`, then the same command without `--dry-run`. Supply full replacement
content and current revisions; the final base must be conformant. Repair does
not relax size, path, concurrency, or read-only constraints. Verify intended
content, provenance, and `kb validate` results after writes; report remaining
uncertainty rather than treating health as semantic approval.
