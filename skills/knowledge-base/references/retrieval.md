# Retrieval

## Confirm the collection

Use the configured root consistently. Examples assume `KB_ROOT` already selects
it; add `--root ./example-kb` when an explicit root is appropriate.

```sh
kb version
kb status
kb list --limit 10
```

Inspect `data.root`, diagnostics, `degraded`, `index_stale`, and the response's
`meta`/`warnings`. Do not silently initialize, change roots, or rebuild state to
hide an access problem. With MCP, use `kb_status` and `kb_list` against the server's
startup root; calls cannot choose another root.

## Search narrowly, then expand

```sh
kb search "project decisions" --limit 5
kb search "deployment" --type Reference --tags operations --has-sources
kb search "retention" --prefix 03_Resources --include-archives --include-deprecated
kb search "quoted phrase" --literal
```

Search is lexical, not semantic. Filters combine across fields with AND and
within one multi-value field with OR. `--type`, `--tags`, `--status`, `--trust`,
and `--para` accept comma-separated values. Use `--stale=true` or `--stale=false`
to request a staleness filter. Defaults omit archives/deprecated notes unless
configuration enables them; broaden deliberately. A negative search is not proof
of absence: try alternate terms, paths, and filters, or inspect `kb list`.

`--literal` still operates on lexical candidates; it is not an unrestricted
substring scan. Arbitrary CJK substrings and emoji-only queries have known limits.
Do not promise semantic retrieval or unrestricted Unicode substring search.

Continue list/search with the returned `data.cursor`, unchanged query, filters,
and limit. A generation/query mismatch invalidates the cursor: restart the query
rather than mixing pages from different snapshots. Do not invent cursors.

## Read the evidence

```sh
kb read 03_Resources/decisions.md --include-links
kb read 03_Resources/decisions.md --section "Project decisions"
kb read 03_Resources/decisions.md --line-start 1 --line-end 5 --raw
kb links 03_Resources/decisions.md --direction incoming
kb context "project decisions" --max-tokens 1000 --max-bytes 16384 --include-links
```

Read returns `data.note.revision`, `markdown`, and the returned physical line
range; line numbers include frontmatter. Sections select headings; missing or
ambiguous selections are errors, not empty evidence. `--raw` does not remove the
response envelope. Avoid requesting ranges past the note's actual length.

Context returns `text`, `sources` (paths, revisions, headings, lines), token
estimates, `truncated`, and `omitted`. Preserve citations when summarizing. The
token budget is an estimate, not the model's exact tokenizer. Narrow the query or
read individual notes if excerpts omit the evidence needed for a conclusion.
Context has its own budgets; it does not accept list/search pagination flags.

MCP equivalents are `kb_search`, `kb_read`, and `kb_context`; request fields use
underscores, with query filters nested under `filters`. There is no `kb_links`
tool: use `kb_read` with `include_links: true` for a bounded incoming/outgoing neighborhood, or an
authorized CLI workflow for the full incoming/outgoing link view.

## Evaluate and finish

Inspect linked source evidence, stale flags, `basis_outdated`, and active trust.
Stale or changed basis means review is needed, not that the claim is false.
Unverified, machine-confirmed, and human-reviewed are different states. State
uncertainty and any degraded/truncated retrieval. Cite the scoped path, observed
revision, and relevant lines; never present an old stored claim as newly verified.

Answer-only retrieval ends here. If nothing new was learned, do not create a
note, duplicate a source, renew verification, or append a log entry. For a durable
correction authorized by the task/workdir policy, read [Writing](writing.md).
