# Diagnose and repair an existing Knowledge Base

Use this workflow when the user authorizes adopting an existing collection or
correcting diagnosed format/health issues. A retrieval request alone does not
authorize a collection-wide migration. Work in the selected root, retain a
backup for a collection-wide conversion, and preserve existing paths, bodies,
unknown metadata, comments, sources, and concurrent work.

## Diagnose before editing

```sh
kb validate --profile okf
kb validate --profile health
kb validate --profile health --strict
```

The JSON envelope can have `ok: true` while validation fails. Inspect
`data.conformant`, `data.clean`, `data.summary.errors/warnings`, and diagnostics:
`path`, stable `rule`, `severity`, `hint`, `revision` when readable, and
`field`/physical `line` when known. `code` remains the compatibility category;
for example `INVALID_NOTE` can have the precise rule `MISSING_FRONTMATTER`.
Default exit 1 means format/policy errors; `--strict` also exits 1 for warnings.
A strict failure does not redefine warnings as normative OKF errors.

Search indexes only valid concepts. Read invalid files directly by their reported
path; an empty search cannot establish absence. Use `kb read PATH` to inspect
complete source and current `data.note.revision`. Large reads may be truncated;
request physical line ranges before constructing a complete replacement. If
inspection is blocked by a scan/size/access error, report the actual blocker;
do not exclude files or change permissions to claim a clean collection.

## Choose an explicit correction

A missing header requires a deliberate type choice. Use a neutral type such as
`Reference` when justified; never infer dates, sources or verification. Preserve
incompatible workflow status under an appropriate extension field instead of
pretending it is an OKF lifecycle status. Ambiguous or conflicting values require
review. Do not discard source material to satisfy lint.

For parseable headers, prefer intended metadata changes. Missing/malformed
frontmatter can require a full `document` including corrected header and exact
original body. Ordinary `patch` cannot bypass degraded-base restrictions.

```json
{
  "actor": "process:agent",
  "summary": "Add an explicit type while preserving the existing note",
  "idempotency_key": "repair-example-001",
  "operations": [
    {
      "operation_id": "fix-example",
      "kind": "patch",
      "path": "03_Resources/example.md",
      "expected_revision": "sha256:REPLACE_WITH_CURRENT_READ_REVISION",
      "document": "---\ntype: Reference\n---\n# Example\n\nExisting body preserved exactly.\n"
    }
  ]
}
```

Replace the example revision and document with the actual inspected bytes. Then:

```sh
kb apply --repair --incremental --input repair.json --dry-run
kb apply --repair --incremental --input repair.json
kb validate --profile okf
```

Each batch accepts at most 50 existing-file patches/4 MiB. Every changed file,
including managed navigation, must be conformant; unrelated invalid files retain
exact bytes. Existing format/policy errors must decrease while any remain.
Creates, moves and deletes are rejected in incremental mode. Access, size,
revision, dependency, verification, idempotency and recovery rules still apply.
Without `--incremental`, explicit repair retains the stricter whole-base final
conformance requirement. Normal writes require whole-base conformance throughout.

Inspect `data.repair_progress`: `errors_before`, `errors_remaining`, and
`conformant`. This describes that transaction, including preview/replayed
receipts; run validation again for the current inventory. If invalid reserved
files prevent navigation maintenance, a `NAVIGATION_DEFERRED` warning explains
what to fix. The durable receipt still records the batch. After correcting those
files and all format errors, run managed maintenance to reconcile navigation.

Repeat diagnose/read/preview/apply/revalidate until errors reach zero. Stop on a
conflict, unchanged findings, a blocked source, or an ambiguous correction; do not
blindly loop. Reconcile revision conflicts from fresh content with a new key.
Retry uncertain outcomes with the identical request and key.

## Resolve health findings with evidence

After format conformance, preview/apply `kb maintain` for navigation. Read source
and target before fixing broken links; do not delete links just to hide warnings.
Review stale evidence and changed synthesis dependencies before updating dates
or recorded revisions. Preserve original evidence. Add relationships only when
meaningful. Run health validation again; `--strict` gives a zero-findings exit
criterion. Report remaining warnings requiring a decision or external evidence.
Clean lint is structural evidence, not factual verification.

Health checks regular local asset existence within scope without indexing or
reading its bytes. This does not validate asset content or fragment identifiers.
