# Review, maintenance, and recovery

## Choose the check

```sh
kb status
kb validate --profile okf
kb validate --profile health
```

`okf` checks format/policy conformance. `health` adds structural/reference and
synthesis-dependency findings. A validation request may succeed (`ok: true`) with
`data.conformant: false` and CLI exit 1. Inspect diagnostics, not only `ok`.

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

Maintenance, recovery, index rebuild, and export have no MCP tools. Explain the
missing capability and use a separately authorized CLI workflow only when its
permissions allow it. A read-only server is not permission to bypass restrictions
through another transport. MCP retains startup root/configuration: restart it
after config or binary changes. See [Results and MCP](results.md) for the actual
tool set and error handling.
