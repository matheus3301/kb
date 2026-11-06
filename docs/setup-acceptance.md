# Setup acceptance

This is the source-candidate acceptance contract for `kb setup`, not evidence
that published v1.0.0 includes setup. Runtime A01–A26 remain in
[acceptance.json](acceptance.json); the setup inventory is
[setup-acceptance.json](setup-acceptance.json).

## Evidence rules

Every S01–S44 row below is required. `in_progress` means the requirement is
tracked, not passed. Record `verified` only after the referenced checks actually
pass and their evidence exists. Partial source coverage is not full criterion
closure. A rejected unsupported capability can establish a diagnostic boundary,
not the unsupported feature itself.

Native evidence must identify date, actual host build, KB binary/asset identity,
OS/architecture, scope/profile, synthetic fixture, effective root, commands or
events, before/after hashes, outcome, pending trust/restart state and teardown.
Source inspection, mocked host delivery, rendered files and `setup check` do not
establish native activation. **S32 and S40 remain pending until actual native
retrieval/event observations are supplied.** Never test learning in a personal
collection. Historical v1.0.0 consumers do not qualify this installer.

See [lifecycle](agent-setup.md), [engine design](setup-design.md),
[adapter/native compatibility](setup-compatibility.md) and
[quality requirements](quality.md). Cross-file publication is recoverable, not
globally atomic; finite tests do not prove universal durability or adherence.

## Required scenarios

| ID | Scenario | Observable requirement |
| --- | --- | --- |
| S01 | Existing executable, bare setup | Short wizard installs a usable integration without manual file editing or Node. |
| S02 | Existing collection outside project | The chosen canonical root is used and collection bytes remain unchanged. |
| S03 | Explicit new collection | Only the selected new root is initialized; PARA is optional. |
| S04 | Invalid or uninitialized existing collection | Diagnose without migration, repair or creating another base. |
| S05 | Project installation | Only the selected activation scope and disclosed bookkeeping change. |
| S06 | Global installation | Only selected user agents are configured; unrelated files remain unchanged. |
| S07 | Project override | One effective root/interface overrides the global default; verify in a native session. |
| S08 | No binding or disabled binding | Deterministic diagnostic or no-op without guessing collections. |
| S09 | Shared artifacts | Track consumers and preserve shared assets when removing one integration. |
| S10 | Unchanged repeat | Byte-for-byte no-op with stable ownership and no duplicate rules/hooks. |
| S11 | Dry-run | Show intended changes without filesystem or network side effects. |
| S12 | Incomplete automation | Structured error without prompting, hanging or writing. |
| S13 | Conflicting flags | Reject contradictory choices before any operation. |
| S14 | Customized skill or rule | Preserve custom content; --yes does not authorize replacement. |
| S15 | Malformed ownership markers | Report conflict, never false success. |
| S16 | Text presentation | Preserve supported LF/CRLF, BOM and final-newline forms; diagnose unsupported forms. |
| S17 | Shared JSON/TOML | Preserve unrelated values, comments and formatting within supported parser shapes. |
| S18 | Invalid configuration | Preflight failure does not partially install artifacts. |
| S19 | Links and aliases | Reject unsupported symlink/hardlink/case traversal without escaped writes. |
| S20 | Concurrent user edit | Expected-hash conflict prevents a lost update. |
| S21 | Publication failures | Every boundary returns nonzero with precise journal state and safe recovery. |
| S22 | Permission, disk or lock failure | No false completion or broad permission repair. |
| S23 | CLI to MCP transition | Explicit independent mode change leaves no contradictory rules. |
| S24 | MCP to CLI transition | Preserve chosen root/policy and safely remove unused owned artifacts. |
| S25 | Selective uninstall | Remaining modes/scopes/agents continue working. |
| S26 | Repeated uninstall | Safe no-op preserves collection, binary, authentication and unrelated configuration. |
| S27 | Customized uninstall target | Preserve with actionable conflict; never recursive name-based deletion. |
| S28 | Missing collection on uninstall | Remove integration using its manifest without opening the collection. |
| S29 | Binary and asset upgrade | Full version-matched skill references; customization conflicts preserved. |
| S30 | Unsupported manifest version | Actionable error without guessed destructive migration. |
| S31 | Shadowing and profiles | Diagnose effective instruction files and custom profiles, including Codex overrides. |
| S32 | Fresh native agent session | Observe external-root retrieval and source citations in the actual host. |
| S33 | No-learning repeat query | Notes and log remain byte-identical. |
| S34 | Authorized learning | Exercise provenance and expected revisions on a disposable synthetic collection. |
| S35 | Read-only integration | Configured path prevents writes; no OS-isolation claim. |
| S36 | Offline setup | Bundled assets install without network, account or runtime downloads. |
| S37 | Portable team configuration | No personal path/secret publication; test any supported relative anchors. |
| S38 | Worktrees and nested repositories | Respect repository boundaries and select the correct binding. |
| S39 | Pending or disabled hook trust | Explicit pending/fallback status without bypass or false activation. |
| S40 | Native hook delivery | Observe valid bounded JSON/context delivery for actual prompt/session events. |
| S41 | Hook timeout, no result or failure | Bounded documented behavior without indefinite blocking. |
| S42 | Compaction, resume and concurrency | No stale routing or unsafe exactly-once assumption. |
| S43 | Mixed global/project hooks | One effective root with bounded duplicates or qualified deduplication. |
| S44 | Unknown agent/version/platform | Explicit unsupported diagnostic; no silent scope promotion. |
