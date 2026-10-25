# Compatibility and support

Status as of **2026-10-25**: experimental **v0.3.1**. This policy records the
released contract and the proposed commitments to review before v1.0. Product
version, wire version, knowledge format, configuration and operational state are
separate axes. `kb.v1` does **not** mean the product has reached v1.0.

The [specification](spec.md), [CLI guide](cli.md), [interface contract](interfaces.md),
[MCP guide](mcp.md) and version-specific
[release notes](https://github.com/matheus3301/kb/releases) remain authoritative
for shipped behavior. Use documentation from the installed version's tag;
`main` may describe newer merged code. A [roadmap](roadmap.md) item is not an
available interface. The [exit checklist](stability.md) tracks missing evidence.

## Released contract and version axes

| Surface | v0.3.1 behavior and boundary |
| --- | --- |
| Product/binary | `kb version --json` identifies release version/source commit. Source installs may report `dev`/`unknown`; they are not release identity proof. Go internals under `internal/` are not a supported external library API. |
| CLI commands/flags | Installed help describes available commands. Root precedence is `--root`, `KB_ROOT`, then nearest ancestor config; `init` without an explicit root initializes the current directory instead. Explicit flags override matching JSON inputs. Unknown input fields, invalid shapes and trailing JSON are rejected. No legacy flag aliases are promised. |
| CLI exit status | 0 success; 1 validation failure; 2 invalid request or otherwise unmapped domain failure; 3 missing resource; 4 revision/idempotency conflict, `DESTINATION_EXISTS` or `CASE_COLLISION`; 5 access/path/symlink/read-only/write-disabled restriction; 6 I/O/lock/cancellation/recovery failure. See the [exact mapping caveats](cli.md#exit-codes-and-errors), not broad error-family guesses. |
| `kb.v1` JSON | Data commands emit one envelope with `schema_version`, `ok`, `data` or `error`, `warnings` and `meta`; logs go to stderr. Read the [envelope schema](../schemas/kb.v1.envelope.schema.json). Validation can return `ok: true`, `conformant: false` and CLI exit 1; strict validation also fails on warnings. Domain error codes are machine signals; error prose is not a parsing API. |
| MCP tools/stdio | Six read-only tools, ten when writes are enabled, in the [runtime-checked catalog](../schemas/kb.v1.tools.json). The server uses a fixed startup root/configuration; stdout is JSON-RPC only. Results carry `kb.v1` in `structuredContent`; compact text is not the data contract. Tool failures set `isError`; validation findings can be successful report data. Actual access checks, not annotations, enforce read-only. |
| Knowledge format | Native OKF v0.2 as pinned in the [specification sources](spec.md); optional root `okf_version` is knowledge-format metadata, not the product or JSON version. Unknown types/metadata and YAML comments are preserved by supported targeted edits. PARA is optional. Application size/path/JSON policies are distinct from OKF conformance. No legacy-note migration or universal future-OKF compatibility is promised. |
| Configuration | `.kb/config.toml` uses `version = 1`. Configuration is loaded when the base opens; restart a long-lived MCP process after changing it. Config version is independent of `kb.v1` and OKF. Do not change it manually to suggest compatibility with an unknown version. |
| Durable operational state | `.kb/state/` contains advisory locking, the version-1 prepared/committed journal and durable receipts. The journal's version is not a public backup format. Receipts retain request identity/replay semantics and currently have no expiry. Do not delete, hand-edit or reconstruct these files to bypass recovery. No cross-version downgrade or external state-file editing API is promised. |
| Derived state | `.kb/cache/` is disposable and rebuildable; it is not authoritative content or durable receipts. Cache layout is internal, not a client integration API. |
| Export | A validated scoped knowledge snapshot with an inclusion/exclusion report. It excludes operational state and is not an operational backup. v0.3.1 has no backup/restore CLI command or MCP tool. |

CLI and MCP share domain semantics on their overlapping operations, not identical
command inventories: v0.3.1 MCP has no init, maintenance, recovery, export or
index-rebuild tool. Inspect discovery instead of translating every CLI command
into an invented tool name. Restart the MCP server to load a new executable.

Search is lexical. Exact ranking scores, pagination cursor bytes, generated
identifiers and textual diffs are not portable persistent handles. Bind cursors
to the same effective query/configuration and generation; restart on stale or
invalid cursors. Store exact revisions for CAS, not timestamps. Trust/health
signals do not certify factual truth, and advisory locks do not isolate external
editors that ignore them.

## Experimental change and upgrade policy

Before v1.0, compatibility is evaluated per release; a pre-1.0 minor release may
change interfaces and must disclose breaking behavior and any migration in its
release notes. Patch releases target compatible corrections, not silent schema
or data migrations. There is no blanket promise that every old client, skill,
configuration or journal will work with every future experimental binary.
The v0.3.1 patch required no knowledge-format migration and left the v0.3.0 CLI
skill unchanged; this is a release-specific fact, not a general upgrade rule.

Before upgrading a base with pending recovery, finish recovery with the current
supported binary or obtain a reviewed version-specific procedure. Preserve notes,
configuration and durable state together using an operator-reviewed backup
procedure; `export` alone cannot preserve idempotency receipts. Retain the old
binary and verify the new version/commit and synthetic workflow before production
use. Reverting the executable is not evidence that newer state can be downgraded.
Do not restore old content over live newer journals/receipts.

v0.1.0 was withdrawn and must never be reused. v0.1.1/v0.1.2 have documented
metadata-fidelity defects; use v0.1.3 or later for those corrections. An upgrade
cannot recover already lost metadata; reconcile with original files/backups.
[Installation guidance](installation.md) preserves the affected cases.

## Proposed v1.x commitments — require exit review

These are intended compatibility rules, **not current v1.0 guarantees**. Approve
scope and upgrade origins in D04 of the [exit checklist](stability.md) first.

- Preserve documented CLI command/flag meanings and exit mappings within v1.x.
  Additive opt-in commands/flags are allowed; removals, changed defaults with
  incompatible effects or repurposed exit codes need a major-version decision
  and migration guidance. Bug fixes must disclose consumer-visible corrections.
- Preserve `kb.v1` field names/types, required semantics and existing domain error
  codes. Add optional result fields only with updated schemas and contract tests;
  clients must tolerate unknown result fields and handle unknown error codes
  safely. Request schemas remain closed: clients must not send new controls to
  older servers without checking their supported surface. An incompatible wire
  contract needs a new wire version and explicit migration, not reused `kb.v1`.
- Preserve documented MCP tool names/argument semantics and structured results
  within the approved scope. Discover tools/capabilities at initialization;
  optional additions do not imply support in older clients. Negotiate the MCP
  protocol through the SDK rather than treating product version as protocol
  version. No remote transport is promised.
- Version knowledge/configuration/operational-state changes independently.
  Document accepted version ranges, upgrade steps, pending-journal handling and
  downgrade limits before changing a persisted format. Never reinterpret unknown
  durable state, discard receipts or require a silent destructive migration.
  A future backup manifest needs its own explicit version/integrity contract.
- No aliases, deprecated compatibility layer or dual implementation is implied.
  A breaking cutover requires an explicit versioned decision and migrated callers,
  docs, schemas, skill/client guidance and contract evidence.

## Platform and maintenance support

The [CI workflow](../.github/workflows/ci.yml) runs source tests, race/fuzz,
acceptance and benchmark smoke on **ubuntu-24.04** and **macos-15** runner labels.
Cross-builds and release archives target **linux/amd64, linux/arm64,
darwin/amd64 and darwin/arm64**, with CGO disabled. A runner label is not an
installed-package matrix or a promise to test every kernel/filesystem version.
The release snapshot job on Ubuntu inspects all archives, executes its native
artifact and builds/exercises archived source; it does not execute foreign binaries.

[v0.3.1 installed verification](https://github.com/matheus3301/kb/releases/tag/v0.3.1)
is **macOS arm64 only**, including a real Homebrew source installation/formula
test, extracted binary, CLI acceptance/documentation requests and installed MCP.
Linux amd64/arm64 and macOS amd64 packages are built, not claimed installed-tested
by that record. Apple signing/notarization is not configured. Windows, network
filesystems, remote multi-host writers and universal power-loss durability are
not validated support claims. Failpoint and interruption tests have bounded scope.

No LTS branch, support lifetime, response-time SLA or automatic backport schedule
is promised. Report supported, reproducible defects through the repository;
follow [SECURITY.md](../SECURITY.md) for security-sensitive reports. Include the
version/commit, platform, synthetic reproduction and relevant redacted diagnostics,
not a private base. Package upgrades remain explicit operator actions.

## Binary, skill, client and operator responsibilities

- **Binary:** enforce scope, input limits, revisions, write permissions, recovery,
  receipts and protocol separation. It does not install skills, register MCP,
  edit global agent settings, select factual truth or authorize note instructions.
- **CLI skill:** teach CLI operation only. Install/update separately and compare
  its guidance with installed help/version. Installation alone does not activate
  mandatory consultation; obsolete skill names are not runtime compatibility APIs.
- **MCP server/client:** the server supplies independent initialization/tool
  guidance without skill files or shell commands. The client registers the binary,
  selects root/access policy, negotiates protocol, uses discovery and reads
  structured results/errors. A client needs restart/reinitialization after upgrade.
- **Operator/agent instructions:** choose the intended root, authorize mutations,
  establish consultation policy, preserve provenance and reconcile conflicts.
  Note/source text remains untrusted data. A successful controlled consumer run
  does not prove every client/model will activate or follow the guidance.

Release, installation and v1.0 readiness remain separate decisions under
[release operations](releases.md); none is authorized by this policy alone.
