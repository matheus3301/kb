# Compatibility and support

Published and installed production is experimental
**[v0.4.0](https://github.com/matheus3301/kb/releases/tag/v0.4.0)** at
`74f0d196408c1354899802e18ce569b0eab33adf`. Current source is being qualified
for stable v1.0.0; readiness and publication remain pending.
Product version, wire version, knowledge format, configuration, operational
state and backup format are separate axes. `kb.v1` is not product v1.0.

The [specification](spec.md), [CLI guide](cli.md), [interface contract](interfaces.md),
[MCP guide](mcp.md) and version-specific
[release notes](https://github.com/matheus3301/kb/releases) remain authoritative
for their respective surfaces. Use the installed version's tagged documentation
for shipped behavior; `main` may contain unreleased corrections. The
[exit checklist](stability.md) tracks exact-candidate evidence and review.

## Current qualification status

**Stable v1 is not ready.** The predeclared 20-read quiet matrix failed V05 only
on 10k CLI search: **384.470 ms > 380 ms**, despite improving on its 487.272 ms
baseline. The other 15 rows and core targets passed. Preserve that failure and
the two earlier failed attempts; do not waive the ceiling, combine passing rows
or repeat until green. Causal cold-CLI profiling/optimization is pending.

The receipt-scale-fixed `7ceed676bed3c1218563f9534a1c095dd3ce4cec` binary completed
the 1,800-second log-disabled/index-enabled profile with 32,850 acknowledged
mutations/replays. It remains passing **historical evidence for that binary**,
separate from the shorter default-log profile. A subsequent Go runtime change
requires full qualification again. The original v0.4.0 backup-timeout failure
and bounded paired repair measurements remain in the
[scaling report](qa/backup-performance-v1.md); no deadline or threshold was raised.

Current source includes the receipt-history repair, bounded compatibility-report
publication guardrails and the configured read-only published-asset verification
matrix. The matrix's **first actual release execution is pending**; configured
jobs, snapshots and previous installed releases are not final artifact evidence.
The [exit checklist](stability.md) owns candidate outcomes and the final decision.
Readiness, authorized publication and installed qualification remain separate.

## Current source contract and version axes

| Surface | Integrated source behavior and boundary |
| --- | --- |
| Product/binary | `kb version --json` identifies release version/source commit. Source installs may report `dev`/`unknown`; they are not release identity proof. Go internals under `internal/` are not a supported external library API. |
| CLI commands/flags | Installed help describes available commands. Root precedence is `--root`, `KB_ROOT`, then nearest ancestor config; `init` without an explicit root initializes the current directory instead. Explicit flags override matching JSON inputs. Unknown input fields, invalid shapes and trailing JSON are rejected. No legacy flag aliases are promised. |
| CLI exit status | 0 success; 1 validation failure; 2 invalid request or otherwise unmapped domain failure; 3 missing resource; 4 revision/idempotency conflict, `DESTINATION_EXISTS` or `CASE_COLLISION`; 5 access/path/symlink/read-only/write-disabled restriction; 6 I/O/lock/cancellation/recovery failure. See the [exact mapping caveats](cli.md#exit-codes-and-errors), not broad error-family guesses. |
| `kb.v1` JSON | Data commands emit one envelope with `schema_version`, `ok`, `data` or `error`, `warnings` and `meta`; logs go to stderr. Read the [envelope schema](../schemas/kb.v1.envelope.schema.json). Validation can return `ok: true`, `conformant: false` and CLI exit 1; strict validation also fails on warnings. Domain error codes are machine signals; error prose is not a parsing API. |
| MCP tools/stdio | Six read-only tools, twelve with writes enabled, in the [runtime-checked catalog](../schemas/kb.v1.tools.json). v0.4.0 includes backup/restore; the prior v0.3.1 release exposes ten write-enabled tools. Fixed startup root/configuration; stdout is JSON-RPC only. Results carry `kb.v1` in `structuredContent`; compact text is not the data contract. Tool failures set `isError`; validation findings can be successful report data. Access checks, not annotations, enforce read-only. |
| Knowledge format | Native OKF v0.2 as pinned in the [specification sources](spec.md); optional root `okf_version` is knowledge-format metadata, not the product or JSON version. Unknown types/metadata and YAML comments are preserved by supported targeted edits. PARA is optional. Application size/path/JSON policies are distinct from OKF conformance. No legacy-note migration or universal future-OKF compatibility is promised. |
| Configuration | `.kb/config.toml` uses `version = 1`. Configuration is loaded when the base opens; restart a long-lived MCP process after changing it. Config version is independent of `kb.v1` and OKF. Do not change it manually to suggest compatibility with an unknown version. |
| Durable operational state | `.kb/state/` contains advisory locking, version-1 prepared/committed journal and durable receipts with no expiry. Journal version is not backup version. Do not delete, hand-edit or reconstruct state to bypass recovery. No external state-file editing API or untested downgrade is promised. |
| Derived state | `.kb/cache/` is disposable and rebuildable; it is not authoritative content or durable receipts. Cache layout is internal, not a client integration API. |
| Export | A validated scoped knowledge snapshot with an inclusion/exclusion report. It excludes operational state and is not an operational backup. |
| Operational backup/restore | Available since v0.4.0: version-1 checksummed tar under the [backup contract](backups.md), preserving exact included files, config and settled receipts. Pending journals are refused, not archived; cache/locks and declared exclusions are omitted. Restore reserves a new confined destination and publishes config last. The running server never changes root. Integrity is not authenticity or disaster-recovery certification. |

CLI and MCP share domain semantics on their overlapping operations, not identical
command inventories: MCP has no init, maintenance, recovery, export or
index-rebuild tool. Inspect discovery instead of translating every CLI command
into an invented tool name. Restart the MCP server to load a new executable.

Search is lexical. Exact ranking scores, pagination cursor bytes, generated
identifiers and textual diffs are not portable persistent handles. Bind cursors
to the same effective query/configuration and generation; restart on stale or
invalid cursors. Store exact revisions for CAS, not timestamps. Trust/health
signals do not certify factual truth, and advisory locks do not isolate external
editors that ignore them.

Literal search still filters lexical candidates; it is not unrestricted
substring matching. Arbitrary CJK substrings and emoji-only queries have known
limits. No semantic, embedding, remote-search or language-model retrieval is
provided. The [retrieval report](qa/retrieval-quality.md) defines tested scope.

Receipt count has no automatic expiry, retention or rotation. Each receipt's
aggregate raw diff text is capped at **256 KiB**; full before/after file images
are represented only while they fit, and excess diff text is truncated.
Extending a retained event log can accumulate roughly quadratic diff storage
while the growing images fit that cap, **not unbounded quadratic growth forever**.
Revisions and other serialized receipt fields are outside that diff-text budget.
Log/note/archive bounds still apply and rejected operations report explicit
errors. Monitor/protect growing state without deleting durable replay receipts.
D02's extended log-disabled/index-enabled profile, shorter default-log profile
and deterministic default-log boundary tests do not change the receipt format or
replay contract. There is no automatic compaction.

## Experimental change and upgrade policy

Before v1.0, compatibility is evaluated per release; a pre-1.0 minor release may
change interfaces and must disclose breaking behavior and any migration in its
release notes. Patch releases target compatible corrections, not silent schema
or data migrations. There is no blanket promise that every old client, skill,
configuration or journal will work with every future experimental binary.
The v0.3.1 patch required no knowledge-format migration and left the v0.3.0 CLI
skill unchanged; this is a release-specific fact, not a general upgrade rule.

Before upgrading a base with pending recovery, finish recovery with the prior
supported binary or follow a reviewed version-specific procedure. Preserve notes,
configuration and durable state together; `export` alone loses receipts.
The v0.3.1 origin needs an operator-reviewed backup procedure because it lacks
the backup command. v0.4.0 and compatible newer binaries provide the
[backup workflow](backups.md); verify a fresh restore and retain the archive
on separate protected storage. v0.4.0 has a reproduced
[receipt-history scaling problem](qa/backup-performance-v1.md); candidate
corrections preserve existing formats and do not expire receipts. Keep the
prior executable: reverting it does not prove a safe state downgrade. Never
restore old content over live newer journals/receipts.

v0.1.0 was withdrawn and must never be reused. v0.1.1/v0.1.2 have documented
metadata-fidelity defects; use v0.1.3 or later for those corrections. An upgrade
cannot recover already lost metadata; reconcile with original files/backups.
[Installation guidance](installation.md) preserves the affected cases.

## Conditional v1.x commitments

D04 selects these rules for stable-v1 qualification, **not as guarantees of the
current experimental release**. They take effect only after V06–V08 in the
[exit checklist](stability.md) pass and stable v1.0.0 is explicitly released.

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
- Preserve native pinned OKF v0.2 knowledge and configuration version 1 semantics
  within the documented scope; unknown metadata/presentation remains preserved
  by supported edits. Preserve settled version-1 operational receipts/replay and
  version-1 backup archives. Knowledge/config/state/archive versions remain
  independent. A future format change must declare accepted versions, upgrade
  procedure, pending-journal handling and downgrade limits; never reinterpret
  unknown state, discard receipts or silently migrate destructively.
- No aliases, deprecated compatibility layer or dual implementation is implied.
  A breaking cutover requires an explicit versioned decision and migrated callers,
  docs, schemas, skill/client guidance and contract evidence.

The selected upgrade path is **v0.3.1 → v0.4.0 → v1.0.0**, conditional on
synthetic V06 qualification at each transition. Require exact note/config/receipt
fidelity, stale-revision conflict behavior, prior-request replay and continued
writes, plus version-1 archive restoration where available. Recover pending
journals with the prior binary before upgrade. Older experimental origins,
unknown format versions and downgrade after newer writes are not blanket support
claims; publish a reviewed migration procedure if a future change needs one.

Required compatibility CI builds both immutable prior sources:
**v0.3.1 / `2ca104f48cc4a6551f4bbc4fae12d51432e42dff`** and
**v0.4.0 / `74f0d196408c1354899802e18ce569b0eab33adf`**.
It checks tag-to-commit and candidate-HEAD identities and retains separate
artifacts; failure, cancellation or missing required outcomes cannot pass the
aggregate gate. An archive-capable prior binary must itself produce the tested
archive before candidate operations touch the fixture. The candidate restores
those bytes in separate staging and verifies exact content/config/receipts,
old-request replay, continued writes and producer-root purity. v0.3.1's archive
edge is explicitly not applicable, not a fictional pass. The
[consumer recipe](qa/compatibility-v1.md) separates source-build, actual Homebrew
and final-artifact evidence. No persisted format changes are required by the
current candidate corrections.

## Platform and maintenance support

The [CI workflow](../.github/workflows/ci.yml) runs source tests, race/fuzz,
acceptance and benchmark smoke on **ubuntu-24.04** and **macos-15** runner labels.
Cross-builds and release archives target **linux/amd64, linux/arm64,
darwin/amd64 and darwin/arm64**, with CGO disabled. A runner label is not an
installed-package matrix or a promise to test every kernel/filesystem version.
The release snapshot job on Ubuntu inspects all archives, executes its native
artifact and builds/exercises archived source. The configured tag-release flow
also downloads actual published assets in read-only **ubuntu-24.04 / macos-15**
matrix arms with independent tag/checkout identity expectations, then exercises
each host's native artifact and archived source. Tap publication requires the
whole matrix to succeed. Neither flow executes foreign binaries; temporary-prefix
execution is not a real Homebrew installation. See [release operations](releases.md).

[v0.4.0 installed verification](https://github.com/matheus3301/kb/releases/tag/v0.4.0)
includes actual **macOS arm64 Homebrew** source installation/upgrade, formula,
CLI/documentation requests and separate CLI + skill and MCP consumers. D04's
other required channel is **Linux amd64 native archive and archived-source
execution**, including CI release snapshots. Every final candidate needs
exact-identity evidence on both required channels; an earlier release does not
qualify later artifacts automatically.

**Linux arm64 and macOS amd64 are package/cross-build-only**, not native-qualified.
Linux native archive/source execution does not imply a tested Linux Homebrew
installation. Apple signing/notarization, Windows, network filesystems, remote
multi-host writers and universal power-loss durability are not validated support
claims. Failpoint and interruption tests cover finite observed boundaries.

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
