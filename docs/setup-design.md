# Explicit agent setup design

Status: adopted implementation. Setup is an explicit opt-in integration operation,
not an extension of collection initialization or ordinary root discovery. Installing
the binary, `kb init`, and data commands never configure agents. Only a requested
`kb setup install` changes the selected project or user-global agent profile.

## Commands and onboarding

`kb setup` and `kb setup install` share the same installer. The interactive path
asks no more than four normal questions: project/global scope, selected agents,
existing or explicitly new collection, and learning on/off. A separate apply review
lists the destinations and activation diagnostics. Detection suggests agents; it
never selects or installs them automatically. A configured bare rerun inspects
existing artifacts, collection and executable rather than repeating the wizard.

Automation selects `--project PATH` or `--global`, repeated `--agent`, `--root`,
`--learning on|off`, and `--yes`. Existing matching settings may supply omitted
root/policy/mode choices; explicit changed flags are not ignored. Divergent prior
choices across selected agents require an explicit selection. Advanced flags are
`--mode cli|mcp`, `--consult always|relevant`, `--hooks auto|off|required`,
`--portable`, `--create`, `--para`, `--dry-run`, and `--operation-id`.
`--para` requires `--create`; create refuses a nonempty destination. Setup does
not initialize, migrate, or repair an existing collection.

`setup uninstall` (alias `remove`) selects scope and agents explicitly. Optional
`--mode` removes only a matching installed interface. Omitting it removes the
selected installation's current mode. Uninstall uses ownership state and does not
open the collection; a moved/deleted collection is not an obstacle. Binary removal
and knowledge deletion are not setup operations.

`setup check`, `setup resolve`, `setup recover`, `setup hook`, and `setup agents`
provide inspection, routing, interrupted-publication recovery, native hook event
handling, and capability descriptions. `resolve` and `check` accept exactly one
agent and a routing workdir. `--global` inspects the global fallback in isolation.
Setup has concise human output; `--json` emits the existing `kb.v1` envelope.
Native hook stdout is host JSON only, never that envelope.

## Architecture and scope

`internal/agentsetup` renders host-specific typed intentions and translates hook
protocols. `internal/setup` owns planning, state, byte-preserving edits and
publication. `internal/cli` composes those packages with collection validation.
Collection storage does not depend on either integration package. The complete
canonical CLI skill and five references are embedded from `skills/knowledge-base`;
setup performs no network fetch or runtime download.

Default bindings and ownership are private user state:

- `$XDG_CONFIG_HOME/kb/agent-setup.json`: routing bindings; fallback config home is
  `$HOME/.config`.
- The sibling `kb/setup-state/manifest.json`: installation identities, selected
  policies, intended artifacts, references, borrowed/owned status and baselines.
- `kb/setup-state/pending.json`, `lock`, and `receipts/`: transaction recovery,
  advisory coordination and explicit operation-identity receipts.

State directories are created with mode `0700`; state/journal/receipt files use
`0600`. Before-images can contain unrelated private host settings. Public results
contain path, action, hashes, mode and selected bindings, never full configuration
snapshots, credentials, or base64 embedded skills. Native configuration remains
personal even though the registry is private: fixed-root MCP settings and default
absolute executable/profile references can still occur inside project files.
Setup discloses and manages local `.git/info/exclude` entries for such files. A
plain non-Git project uses disclosed `.gitignore` patterns. Already tracked
personal destinations are rejected: an exclude cannot make tracked content private.
No Git commit, push, synchronization or trust-store change is performed.

Resolution searches the workdir and its ancestors, stopping at the nearest Git
repository/worktree boundary. The most-specific project binding wins; an explicit
disabled portable binding stops global fallback. Otherwise the agent's global
binding applies. Canonical project identities mean copying a worktree does not
copy its private binding. This does **not** imply that copied native host files
cannot activate a host: personal activation files must remain untracked/private.

The saved integration binding is not displaced by ambient `KB_ROOT`; check/resolve
report the mismatch. CLI instructions pass the resolved root explicitly to data
commands. Ordinary data commands retain `--root`, then `KB_ROOT`, then ancestor
`.kb/config.toml` discovery. There is no new implicit data-root fallback.

`--portable` is an explicit project CLI mode. Its public `.kb-agent/setup.json`
contains roots and skill paths relative to the project anchor, with no personal
absolute paths or private registry identity. Generated commands use `kb` on PATH;
setup checks that prerequisite and all consuming sessions need it. A second user
can resolve this binding without the first user's registry. Global and MCP portable
modes are rejected until their native launch-cwd contracts are qualified; there is
no silent scope promotion or guessed relative MCP root.

A selected project may itself be a collection. Setup consults the actual collection
scope before adding Markdown. New raw instruction/skill Markdown in included paths
is a preflight conflict; existing valid scoped instruction notes retain their
metadata and bytes outside the managed block. Excluded paths remain excluded under
the existing profile. In particular `.pi` and `.omp` are not implicitly added to
core exclusions. Setup never edits a collection configuration to make installation
appear valid. Collection inspection is observational and creates no operational
lock/cache files, including during dry-run.

## Ownership and byte preservation

An artifact is a whole owned file, marked block, JSON/JSONC object entry, individual
hook-array item, TOML table, or local Git-exclusion block. Shared physical assets
require byte-identical intentions and explicit references; differing policies do
not get silently combined under a shared marker. Host-specific guarded policies
use distinct identifiers. Removing one consumer preserves remaining references.

A byte-identical preexisting asset is borrowed, never deleted by uninstall.
Install/check detect borrowed drift or a borrowed old-version mismatch instead of
claiming the current skill/policy is installed. Customized owned regions conflict;
`--yes` does not authorize overwriting them. No recursive same-name directory
deletion occurs. Entirely owned files may be removed; otherwise only exact owned
regions/entries are removed. A clean final uninstall restores the original file
bytes. Once unrelated edits occur, those edits are preserved and only the owned
regions are removed.

Markdown blocks retain foreign bytes, BOM, LF/CRLF, permissions, and missing-final-
newline presentation. Orphaned/duplicate/reversed markers conflict. JSON edits use
validated byte spans rather than decoding/reserializing the document, preserving
unrelated values, decimal presentation and whitespace. JSONC is accepted only for
adapters declaring that format; same-length comment/trailing-comma masking preserves
spans and string literals. A comma without a preceding member/element is invalid,
not a trailing-comma extension; masking must not turn `{,}` or `[,]` into valid
empty containers. Duplicate keys are rejected in both formats. Whole-document
validation runs after JSON/JSONC entry and array edits as well as before them.
TOML is validated before and after edits; the conservative section scanner respects quoted
and multiline values. Unsupported/ambiguous structures fail before mutation.

## Publication and recovery

Planning reads and validates every destination and records existence, bytes,
permissions and hashes. `--dry-run` only plans: no collection initialization,
manifest, lock, journal, credential access, or external configuration write.

Apply takes one private per-user nonblocking advisory lock, checks preconditions,
persists a journal, stages adjacent after-images, checks hashes again, and publishes
one atomic same-directory rename per file. Registration removal precedes helper
removal; routing and ownership state follow external files. Each boundary is
journaled. The complete multi-file operation is **not globally atomic**, and
uncooperative external editors do not honor the advisory lock.

Before the durable commit marker, recovery rolls back only files still equal to
the owned after-image. It also inspects unrecorded rename boundaries, because a
process can stop between rename and journal progress. External edits are preserved
as actionable conflicts. After the marker, recovery verifies committed images and
finishes cleanup. Recovery is repeatable; unresolved pending state blocks normal
setup resolution/mutation. An explicit operation id replays the same completed
request and rejects changed inputs.

Explicit collection creation uses the existing independent `init` transaction
before agent publication. If later setup publication fails, the intentionally
created collection is preserved, not deleted as rollback cleanup. The agent
transaction still rolls back/recovers independently. This is a documented
cross-resource boundary, not a claim of atomic collection-plus-agent creation.

## Interfaces and activation claims

CLI integrations use the complete skill and scoped resolver. Learning-off policies
emit read-only data calls. MCP integrations register the independent existing stdio
server with the selected fixed root and optional `--read-only`; model consumers
need neither shell commands nor the CLI skill. Host/scope guards select one
interface rather than mandating inherited CLI and project MCP simultaneously.

Hooks accept bounded native JSON, resolve the event's cwd binding, and perform
bounded cooperative-timeout direct-core read-only retrieval for both CLI and MCP
integrations. Lifecycle machinery invokes the binary; the MCP model consumer still
needs no shell, resolver, or CLI skill. Model-facing continuation and unavailable
retrieval fallback name only the selected interface. Input/output ceilings are
64 KiB/16 KiB. Hook failures produce bounded native unavailability context and a
nonzero result; no automatic note write, prompt/transcript log, hidden retry loop,
trust bypass or strict exactly-once enforcement is claimed. Rules remain present.
Native trust, restarts, reloads and version qualification remain host responsibilities.

`configured` means intended artifacts match. `ready` in check additionally means
that the selected collection validates, the native host executable exists on PATH,
the recorded KB command is executable, and currently discovered profile/instruction
destinations still match the installed paths. New overriding instruction files or
changed profile homes are diagnosed. This is not proof of supported native version,
trust, actual instruction loading, model behavior or fresh-session retrieval.
`activation_pending` is reported separately;
`activation_observed` remains false without actual native-session evidence.
Unknown agents and unsupported modes/scopes/hooks fail preflight. See `kb setup
agents` for declared capability limits and primary-source references.

## Acceptance traceability and limits

The following maps all research scenarios to engine/CLI evidence, not blanket
native qualification. Test sources are synthetic/disposable and clear `KB_ROOT`.

| Scenarios | Implemented evidence / remaining observation |
| --- | --- |
| S01 | `TestSetupCLIWizardReviewAndHumanOutput`; compiled CLI lifecycle smoke. Fresh native usability remains separately observed. |
| S02 | `TestSetupCLIInstallResolveCheckAndUninstall`; external-root bytes unchanged. |
| S03 | `TestSetupCLICreateReplayAndModePolicyInheritance`; explicit create/PARA/replay. |
| S04 | `TestSetupCLIPureDryRunAndValidation`; invalid collections rejected without repair. |
| S05–S06 | Scoped/private registry planning and adapter path tests; native activation remains pending. |
| S07–S08 | `TestRoutingBoundaryWorktreeAndPortable`, `TestDisabledPortableBindingStopsGlobal`; native root selection needs session observation. |
| S09–S10 | `TestLifecycleSharedBorrowedAndPristine`; explicit sharing and byte-stable repeat. |
| S11–S13 | `TestPreflightPureAndAllOrNothing`, `TestSetupCLIPureDryRunAndValidation`; no dry-run writes or non-TTY prompts. |
| S14–S16 | `TestOwnedConflictAndOutsideEdit`, `TestBorrowedDriftAndUpgrade`, `TestJSONCAndMarkerRejection`, lifecycle BOM/CRLF fixture. |
| S17–S18 | `TestJSONAndTOMLPreserveForeignBytes`, `TestTOMLEscapedMultilineHeadersAndUpgrade`, empty/malformed-config preflight and JSONC fuzz target. |
| S19–S20 | `TestPreconditionRaceAndSymlinkHardlink`, `TestUnsupportedManifestAndCaseCollision`, confined-path fuzz target. |
| S21–S22 | `TestRollbackAtEveryPublicationBoundary`, recovery tests, `TestReadOnlyDestinationPreflightAndPrivateState`, Git operational-failure preflight, and concurrency/lock regression; no universal disk/crash-safety claim. |
| S23–S24 | `TestSetupCLICreateReplayAndModePolicyInheritance`; unused interface artifacts removed. |
| S25–S28 | Shared/borrowed/owned uninstall tests, repeated uninstall, moved collection regression. |
| S29–S30 | Embedded reference completeness, borrowed upgrade conflict, unsupported manifest regression. |
| S31 | Adapter profile/override precedence tests and `TestSetupProfileChangesAndExistingFallbackInstructions`; actual host loading remains native evidence. |
| S32 | Fresh native retrieval/citation is explicitly not established by engine tests. |
| S33–S35 | Compiled CLI collection-byte smoke and hook read-only regression; independent MCP generated-command tests exercise read-only tools. Authorized learning uses existing revision/provenance transaction coverage, not a new setup guarantee. |
| S36 | Embedded offline assets; setup has no network dependency. |
| S37–S38 | Portable relative registry, copied-worktree/nested-repository routing tests. Portable MCP remains unsupported, explicitly rejected. |
| S39 | Activation-pending diagnostics, unsupported-required-hook preflight and actual OMP disabled-extension observation; no trust approval is automated. |
| S40–S43 | Native schema/concurrent read-only tests, real-process timeout regression, and identified OMP delivery/compaction/resume/mixed-scope observations in [native lifecycle evidence](evidence/setup-native-lifecycle.json) and [hook evidence](evidence/setup-hook-qualification.json). No universal or exactly-once claim. |
| S44 | Unknown-agent/scope/required-hook rejections and unknown-build inspection regression; version/platform remain explicitly unobserved by local inspection. |

Existing source tests and source-derived adapters do not prove native activation.
Full repository gates and fresh-host observations are integration evidence and
must be reported separately rather than inferred from this design document.
