# CLI guide

`kb` operates on a local OKF Markdown Knowledge Base. This guide describes the
current source; use `kb version` and `kb COMMAND --help` for the installed build.
For a first workflow, follow the [README](../README.md) and
[quickstart](quickstart.md). For agent activation and MCP, see
[agent setup](agent-setup.md). The [interface contract](interfaces.md) defines
the underlying DTOs and transaction semantics.

## Root, output, and input

```text
kb [--root ROOT] [--json] [--read-only] [--as-of RFC3339] COMMAND
```

| Global option | Behavior |
| --- | --- |
| `--root ROOT` | Select the Knowledge Base; overrides `KB_ROOT`. |
| `KB_ROOT` environment variable | Root when `--root` is absent. |
| Ancestor discovery | Otherwise find the nearest ancestor with `.kb/config.toml`. |
| `--json` | Emit JSON; this is already the default for data commands. |
| `--read-only` | Reject write-class operations and hide MCP write tools. |
| `--as-of TIMESTAMP` | Evaluate staleness at an RFC3339 timestamp with timezone. |

`init` is the discovery exception: absent `--root` and `KB_ROOT`, it initializes
the current directory. It does not discover an ancestor. Note paths are scoped
relative paths, not arbitrary filesystem paths. Configuration lives at
`.kb/config.toml`; a running MCP server retains its startup configuration.

Data commands accept `--input FILE` or `--input -` for a JSON object, maximum
4 MiB. Unknown fields, invalid shapes, and trailing JSON are
rejected. JSON property names use underscores. Explicit flags override their
corresponding JSON values; a positional argument conflicting with JSON is an
error. Write commands require an input containing `actor`, `summary`, and
`idempotency_key` in addition to operation fields.

Data commands emit a `kb.v1` envelope with `ok`, `data` or `error`, `warnings`,
and `meta`. Diagnostics go to stderr. Help is text; MCP stdout is JSON-RPC.
See the [envelope schema](../schemas/kb.v1.envelope.schema.json). Always inspect
warnings and command-specific diagnostics; JSON success does not imply exhaustive
retrieval or semantic truth.

## Initialize and inspect

```sh
kb --root ./example-kb init --para
kb --root ./example-kb status
kb version
```

`init` creates missing reserved files and configuration. `--para` adds initial
PARA navigation. Initialization is separate from importing a legacy collection,
installing a skill, and registering MCP; none happens implicitly.

`status` reports root, generation, note/invalid counts, diagnostics, excluded
paths, degraded inventory, and derived-index staleness. `version` reports build
version and commit; Go source installs can report `dev`/`unknown`.

## List and search

```sh
kb list --limit 10
kb search "project decisions" --limit 5
kb search "deployment" --type Reference --tags operations --has-sources
kb search "retention" --include-archives --include-deprecated
kb search "exact phrase" --literal
```

| Option for list/search | Behavior |
| --- | --- |
| `--limit N` | Default 10, maximum 100 hits per page. |
| `--cursor VALUE` | Continue the same effective query on the same generation. |
| `--literal` | Literal matching within lexical candidates; not unrestricted substring scanning. |

Filters shared by `list`, `search`, and `context`:

| Filter | Behavior |
| --- | --- |
| `--prefix PATH` | Scoped path prefix. |
| `--para`, `--type`, `--tags`, `--status`, `--trust` | Comma-separated values: OR within a field, AND across fields. |
| `--stale=true` / `--stale=false` | Select stale/non-stale notes. |
| `--has-sources=true` / `--has-sources=false` | Select presence/absence of sources. |
| `--include-archives`, `--include-deprecated` | Include normally omitted notes; configuration can change defaults. |

Search is lexical ranking with snippets and physical line ranges, not semantic
retrieval. Arbitrary CJK substrings and emoji-only queries have known limits.
Try alternate terms and filters before concluding that knowledge is absent.

Results contain `hits`, `total`, `generation`, warnings, and an optional `cursor`.
Reuse a returned cursor only with the same effective request. A changed inventory
returns `STALE_CURSOR`; a different query returns `INVALID_CURSOR`. Restart rather
than combining pages from incompatible snapshots.

## Read, context, and links

```sh
kb read 03_Resources/decisions.md --include-links
kb read 03_Resources/decisions.md --section "Project decisions"
kb read 03_Resources/decisions.md --line-start 1 --line-end 5 --raw
kb context "project decisions" --max-tokens 1000 --max-bytes 16384 --include-links
kb links 03_Resources/decisions.md --direction both
```

`read PATH` accepts `--section`, `--line-start`, `--line-end`, `--raw`, and
`--include-links`. Sections and explicit ranges are mutually exclusive. Physical
line numbers include frontmatter and must remain inside the document. For an
ambiguous heading, use an explicit `#section-id` from the diagnostic. `--raw`
still returns an envelope. The current response carries Markdown source in
`data.markdown` and its exact `data.note.revision`; it is not rendered HTML.

`context QUERY` accepts the shared filters and `--max-tokens`, `--max-bytes`,
`--include-links`. It does not accept `--limit`, `--cursor`, or `--literal`.
Inspect `text`, `sources` (paths, revisions, headings, lines), `estimated_tokens`,
`estimator`, `truncated`, and `omitted`. Token counts are estimates. Preserve
citations and report omissions when using excerpts to answer.

`links PATH --direction both|incoming|outgoing` inspects resolved and unresolved
references. `read --include-links` provides a bounded incoming/outgoing
neighborhood instead of a separate link query. Neither interface follows
external URLs or grants permission to act on their contents.

## Create and patch

Single-note requests are flattened JSON objects. Save a create request as
`create.json`, then preview and execute it:

```json
{
  "path": "03_Resources/retention.md",
  "type": "Reference",
  "title": "Retention policy",
  "body": "# Retention policy\n\nRecords expire after 14 days.\n",
  "metadata": {"tags": ["operations"]},
  "summary": "Record the retention policy",
  "actor": "process:cli-guide",
  "idempotency_key": "cli-guide/create-retention"
}
```

```sh
kb create --input create.json --dry-run
kb create --input create.json
kb read 03_Resources/retention.md
```

Copy the returned `data.note.revision` into `REVISION_FROM_READ` in `patch.json`:

```json
{
  "path": "03_Resources/retention.md",
  "expected_revision": "REVISION_FROM_READ",
  "body_edits": [{"old_text": "14 days", "new_text": "30 days"}],
  "metadata_patch": {"description": "Current record retention interval"},
  "summary": "Update retention after reviewing new evidence",
  "actor": "process:cli-guide",
  "idempotency_key": "cli-guide/update-retention"
}
```

```sh
kb patch --input patch.json --dry-run
kb patch --input patch.json
```

Create requires a nonempty `type`; title, body, and metadata are optional. Patch
requires the current `expected_revision`. Use either `body` for whole-body
replacement or `body_edits` with unique exact `old_text` matches. Change selected
metadata keys with `metadata_patch`; remove keys with `remove_metadata_keys`.
Avoid resubmitting unknown metadata through a lossy numeric parser. Substantive
edits invalidate active verification. Record real evidence instead of asserting
human review; see [writing and provenance](../skills/knowledge-base/references/writing.md).

## Move and archive

Read the current revision after any earlier edit, then save `move.json`:

```json
{
  "path": "03_Resources/retention.md",
  "to": "03_Resources/record-retention.md",
  "expected_revision": "REVISION_FROM_READ",
  "summary": "Clarify the note path",
  "actor": "process:cli-guide",
  "idempotency_key": "cli-guide/move-retention"
}
```

```sh
kb move --input move.json --dry-run
kb move --input move.json
```

`move [PATH] [TO]` rewrites resolved references; review the affected files in the
preview. `archive [PATH] --input archive.json` uses the same request shape and
revision/key rules. Omit `to` to use the configured archive directory and basename,
or provide an explicit scoped destination. No deletion command is exposed.

## Coordinated changes and retries

`apply --input plan.json` takes `operations`, `summary`, `actor`, and
`idempotency_key`. Each operation has a unique `operation_id`, `kind`, and `path`.
Kinds are `create`, `patch`, and `move`; patch/move require `expected_revision`,
and move requires `to`. Use at most 50 operations and one operation per path.
See the [complete plan example](../skills/knowledge-base/references/writing.md#coordinate-multiple-notes).

```sh
kb apply --input plan.json --dry-run
kb apply --input plan.json
```

Review preview diffs before applying within the task's authorization. Previewing
reserves no files. Normal writes require a conformant base; an explicitly intended
repair uses `apply --repair`, full replacement content, and current revisions.
Preview the repair too. The final base must be conformant; repair does not bypass
path or size policy.

An uncertain outcome should be retried with the **identical request and key**.
Completed requests replay without duplicate effects. A changed request needs a
new key. On a revision conflict, reread and reconcile the actual content; never
merely replace the revision token to overwrite concurrent changes.

## Validation, maintenance, recovery, and export

```sh
kb validate --profile okf
kb validate --profile health
kb maintain --dry-run
kb maintain --apply
kb index rebuild
kb recover --dry-run
kb recover
kb export ./example-export
```

| Command | Purpose and boundary |
| --- | --- |
| `validate --profile okf` | Check format/policy conformance; no semantic certification. |
| `validate --profile health` | Add structural, link, and synthesis-dependency findings. |
| `maintain` | Preview managed navigation by default; `--apply` publishes it, preserving manual content. |
| `index rebuild` | Rebuild disposable derived search state. |
| `recover` | Inspect with `--dry-run`, then recover a pending journal when authorized. |
| `export DESTINATION` | Export a validated scoped snapshot with inclusion/exclusion manifest. |

A pending journal blocks reads with `RECOVERY_REQUIRED`. Prepared recovery rolls
back; committed recovery rolls forward. A new mutation can auto-recover before
its own work. Preserve failed recovery evidence; do not edit journals/receipts
by hand. Advisory locking cannot isolate arbitrary external editors.

Export is not an operational backup: it excludes journal/receipt state and does
not certify excluded files. Choose a suitable new destination. Read-only mode
rejects mutation-class commands, including maintenance/recovery/index rebuild;
it does not prohibit exporting authorized content to a separate directory.

## MCP and limits

`kb mcp` serves stdio with a fixed startup root. Read-only mode exposes six tools:
`kb_status`, `kb_list`, `kb_search`, `kb_read`, `kb_context`, `kb_validate`.
Write-enabled mode adds `kb_create`, `kb_patch`, `kb_move`, `kb_apply`. Other CLI
commands do not automatically have MCP equivalents. See [agent setup](agent-setup.md)
for configuration and [tool schemas](../schemas/kb.v1.tools.json) for inputs.

| Limit | Bound |
| --- | --- |
| Note | 1 MiB; a scoped source file over 4 MiB aborts inventory explicitly. |
| Read/context | 256 KiB response ceiling, including metadata/citations. |
| Context tokens | 2,000 estimated by default, maximum 16,000. |
| Search/list | 10 hits by default, maximum 100 per page; query at most 4,096 runes. |
| JSON input/plan | 4 MiB and at most 50 plan operations. |

## Exit codes and errors

| Exit | Meaning |
| --- | --- |
| 0 | Success. |
| 1 | Validation/conformance failure, including `ok: true` with `conformant: false`. |
| 2 | Invalid request or otherwise unmapped domain failure. |
| 3 | Missing resource. |
| 4 | Revision/idempotency conflict, `DESTINATION_EXISTS`, or `CASE_COLLISION`. |
| 5 | Path, symlink, access, read-only, or write-disabled restriction. |
| 6 | I/O, lock, cancellation, or recovery failure. |

Inspect `error.code`, `message`, optional `details`, and `retryable`. Other
collision codes may map to 2; do not infer behavior from a broad error family.
Invalid input needs correction, stale cursors need a new query, conflicts need
reconciliation, and pending transactions need recovery. Full operational response
guidance lives in [results and MCP](../skills/knowledge-base/references/results.md).
