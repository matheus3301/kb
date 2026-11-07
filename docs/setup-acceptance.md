# Setup acceptance

This is the acceptance contract for `kb setup`, introduced in v1.1.0. The
source-candidate observations below retain their original identities and do not
qualify later artifacts automatically. Runtime A01–A26 remain in
[acceptance.json](acceptance.json). The complete S01–S44 requirements, named
fixtures, observed scopes and evidence references are in
[setup-acceptance.json](setup-acceptance.json).

## Status and verification boundary

**All 44 criteria have bounded verified observations.**
The five review gaps were closed with the identified native and process checks below.
`verified` applies to the identified fixtures and observed scopes, not every host/build.
The review `make check` passed again with **220 named top-level Go tests**,
including race/shuffle,
**86.5% aggregate statement coverage**, all **17 fuzz targets**, four cross-build
platforms, security scans, executable docs, retrieval checks and concurrent soak.
See [review qualification](evidence/setup-review.json) for the current gate and
[earlier qualification](evidence/setup-qualified.json) for the executable hash,
actual PTY wizard, real Git worktrees, 19 lifecycle rows and post-fix native MCP run.

The JSONC comma-only-container defect discovered during fuzzing is fixed by
**b9883e6**. Corpus `10c2b389efa4b531` remains a regression; the stronger fuzzer
passed 9,623,675 inputs in 120 seconds before the complete integrated rerun.
No quality gate or threshold was relaxed. Earlier native observations retain
explicit source identities rather than being relabeled as final-binary runs.

No original criterion is expanded into universal native qualification: S03
requires the explicitly selected create/PARA behavior, S20 the expected-hash
conflict behavior, S32 the named actual-host retrieval and S34 the observed
disposable revision/provenance exercise. Other-host capability qualification is
tracked separately. Compaction/resume and mixed-scope hooks now have actual
OMP 18.16.0 observations; they are not inferred from resolver or package tests.

## Review qualification

- [Hook qualification](evidence/setup-hook-qualification.json): OMP 18.16.0
  and 18.17.0 disabled hooks, no-hit, failure and timeout delivery; retained
  real-process regression and explicit unknown-build inspection boundary.
- [Native lifecycle](evidence/setup-native-lifecycle.json): actual committed
  compaction, same-session resume after changing roots, both mixed CLI/MCP scope
  directions, global fallback, byte-pure collections and confirmed teardown.
- [Compact site prompt](evidence/setup-site.json): complete GitHub Markdown guide,
  short copyable handoff and desktop/mobile browser evidence.

## Evidence index

- **L — [CLI lifecycle matrix](evidence/setup-lifecycle.json):** 19 supported
  project/global CLI rows passed on the initial candidate, 8b1c5b6 and the final bfcd143 binary.
  Cursor global is explicitly unsupported; initial invalid `vscode` project/global
  IDs were rejected before mutation. Local executable presence is not native
  activation. Missing native executables are explicitly nonready, not skipped.
- **B — [Executable-doc and ownership probes](evidence/setup-cli-docs.json):**
  43 existing + 12 setup real CLI requests passed with native harnesses deliberately
  absent; six guard regressions passed. A compiled Claude/Copilot shared MCP
  ownership and moved-collection uninstall probe passed. No native model harness
  was simulated.
- **N — [Actual native observations](evidence/setup-native.json):** initial
  OMP 18.16.0 project CLI/hooks read cited facts twice without collection changes;
  a separate authorized-learning session used revision-checked preview/apply/
  readback, preserved metadata, added provenance and repeated a byte-pure query.
  Codex 0.166.0-alpha.5.1 project CLI/hooks-off retrieval and purity passed.
  Claude 2.1.212 native session/prompt hook commands exited 0, but its model answer
  failed because login was unavailable: **partial hook activation, not successful
  Claude model retrieval**. No authentication changes were made.
- **N, identified 8b1c5b6 recheck:** native Codex CLI project-over-global precedence
  passed with both synthetic collections unchanged and temporary auth linkage
  removed. Native OMP project MCP status/search/read, hook context and repeated
  collection-byte purity passed with settled teardown and no survivors. OMP's
  `read`/`write` calls targeted deferred MCP devices; they were not filesystem
  writes. The initial observer's direct-tool-name assumption was corrected and
  the rerun passed; that false negative is not a product failure.
- **S — [Source verification boundary](evidence/setup-source-tests.json):** actual
  earlier engine/adapter package runs, named fixtures and the passing final
  integrated gate, with the retained JSONC regression and repair history. Synthetic host callbacks and direct-core
  consumers are not native model sessions.

Initial observations identify candidate **5b75df2**, SHA-256
`dfd8b0c0fe750a55221345a496562c9500a253de123e7f279a24ac2c5cd7fbeb`.
The identified recheck is **8b1c5b6**, SHA-256
`48f50ecf6415a36760b02a929e50ae474ee55f17c10c21509941fd0f45a203ba`,
on Darwin arm64. The final post-fix candidate is **bfcd143**, SHA-256
`f2865c84f348bb28e1868da97846977f5cd3b1581fe23e98fb07788a9b168a23`;
[final qualification](evidence/setup-qualified.json) records its full gate,
real PTY/worktree checks and repeated native OMP MCP retrieval. [Static-site screenshots](evidence/setup-site.json) remain
presentation evidence, not installer or native activation proof.

## Limits and safety

Named native observations do not qualify every other host/build, global/project
combination or failure transition beyond the exact recorded runs. OMP
learning does not establish every host/interface's learning behavior. Source
inspection, generated files and `setup check` do not establish activation.
`activation_observed` is never inferred from installed artifacts. Unsupported
capability rejection proves the diagnostic boundary, not support for that mode.

Use disposable synthetic collections only. Keep private paths, credentials, raw
sessions and research snapshots out of public evidence. Record actual identity,
scope, root/citation checks, file-byte comparisons, trust/restart state and
teardown. Cross-file publication is recoverable, not globally atomic; finite
tests do not prove universal power-loss safety or model adherence. See
[lifecycle](agent-setup.md), [engine design](setup-design.md),
[host compatibility](setup-compatibility.md) and [quality](quality.md).

## S01–S44 traceability

The JSON inventory retains every observable requirement. Each row names actual
source fixtures and the bounded observation or remaining gap. `L`, `B`, `N`, `S`
refer to the linked evidence records above.

| ID / status | Scenario | Named source fixtures | Observed coverage / remaining boundary |
| --- | --- | --- | --- |
| S01 / verified | Existing executable, bare setup | [TestSetupCLIWizardReviewAndHumanOutput](../internal/cli/setup_test.go) | Actual PTY wizard passed with all answers including final confirmation pasted together, reviewed destinations, exact selected root and unchanged collection; PATH contained no Node, Git or native harness. Separate native sessions qualify the resulting integration contracts. |
| S02 / verified | Existing collection outside project | [TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | Verified for disposable external-root CLI lifecycles and the recorded native project sessions: selected canonical root and collection bytes preserved. No all-host activation claim. |
| S03 / verified | Explicit new collection | [TestSetupCLICreateReplayAndModePolicyInheritance](../internal/cli/setup_test.go)<br>[TestSetupPortableAndTildeCreation](../internal/cli/setup_regression_test.go) | Verified for the selected create/PARA and replay fixtures in the passing integrated test/race stage on 8b1c5b6. This criterion adds no broader native-onboarding requirement; post-parser-fix full qualification remains separate. |
| S04 / verified | Invalid or uninitialized existing collection | [TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go)<br>[TestSetupCLIRedactionAndCollectionRootConflict](../internal/cli/setup_test.go) | Invalid/uninitialized roots are rejected without implicit migration or repair; exact final CLI preflight fixtures passed. |
| S05 / verified | Project installation | [TestHostConfigurationContracts](../internal/agentsetup/adapter_test.go)<br>[TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go)<br>[TestPrivacyNoGitAndFatalRepositoryCommands](../internal/setup/regression_test.go) | Ten project adapters passed disposable install/check/resolve/removal cycles with collection preservation. This verifies configuration lifecycle, not native activation for every host. |
| S06 / verified | Global installation | [TestHostConfigurationContracts](../internal/agentsetup/adapter_test.go)<br>[TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go) | Nine supported global adapters passed disposable lifecycle cycles. Cursor global is explicitly unsupported and rejected without mutation; personal global agent settings were not used. |
| S07 / verified | Project override | [TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go)<br>[TestSharedInstructionsKeepDistinctHostOwners](../internal/agentsetup/adapter_test.go) | Native Codex CLI/hooks-off project-over-global precedence passed on 8b1c5b6. Review qualification additionally passed actual OMP 18.16.0 global CLI/project MCP, inverse mixed mode and global fallback with correct effective roots and byte-pure collections. |
| S08 / verified | No binding or disabled binding | [TestDisabledPortableBindingStopsGlobal](../internal/setup/routing_test.go)<br>[TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go) | No-binding and disabled portable-binding fixtures passed; resolution does not guess a collection or fall through a disabled project binding. |
| S09 / verified | Shared artifacts | [TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go)<br>[TestSetupSharedMCPEntryRemovalRetainsOtherConsumer](../internal/cli/setup_regression_test.go) | Verified in the compiled Claude+Copilot shared project MCP probe: one server entry survives removal of the first consumer and disappears after the last. Shared/borrowed rule coverage is separately source-mapped. |
| S10 / verified | Unchanged repeat | [TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go)<br>[TestOwnedHookUpgradePreservesForeignHandler](../internal/setup/regression_test.go) | Verified byte-stable repeated CLI installation in all 19 supported scope rows and the executable-doc workflow. This is filesystem/configuration evidence, not repeated native activation. |
| S11 / verified | Dry-run | [TestPreflightPureAndAllOrNothing](../internal/setup/setup_test.go)<br>[TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go) | Pure dry-run fixtures and all 19 supported lifecycle rows passed; selected collection and fixture files remained unchanged. |
| S12 / verified | Incomplete automation | [TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go) | Incomplete automation fixtures fail with structured diagnostics instead of prompting or publishing artifacts. |
| S13 / verified | Conflicting flags | [TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go)<br>[TestRejectedRequestsProduceNoPartialIntents](../internal/agentsetup/adapter_test.go) | Contradictory scope/policy/agent requests fail preflight in the final CLI fixtures. |
| S14 / verified | Customized skill or rule | [TestBorrowedDriftAndUpgrade](../internal/setup/setup_test.go)<br>[TestOwnedConflictAndOutsideEdit](../internal/setup/setup_test.go) | Modified owned and borrowed content is preserved with conflict diagnostics; --yes does not authorize destructive replacement. |
| S15 / verified | Malformed ownership markers | [TestJSONCAndMarkerRejection](../internal/setup/setup_test.go)<br>[FuzzManagedBlockParsing](../internal/setup/fuzz_test.go) | Malformed/duplicate ownership-marker fixtures and the marker fuzzer passed; conflicts do not publish partial changes. |
| S16 / verified | Text presentation | [TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go)<br>[TestJSONAndTOMLPreserveForeignBytes](../internal/setup/setup_test.go)<br>[FuzzManagedBlockParsing](../internal/setup/fuzz_test.go) | LF/CRLF, BOM and supported final-newline fixtures passed with preservation of foreign bytes. Unsupported representations remain explicit conflicts. |
| S17 / verified | Shared JSON/TOML | [TestJSONAndTOMLPreserveForeignBytes](../internal/setup/setup_test.go)<br>[TestJSONCAndMarkerRejection](../internal/setup/setup_test.go)<br>[TestTOMLEscapedMultilineHeadersAndUpgrade](../internal/setup/regression_test.go)<br>[FuzzJSONCEntryPreservation](../internal/setup/fuzz_test.go) | Final JSON/JSONC/TOML preservation fixtures passed. The retained comma-only-container fuzz regression is fixed; edited JSON/JSONC is revalidated before publication. Extended JSONC fuzz passed 9,623,675 executions. |
| S18 / verified | Invalid configuration | [TestPreflightPureAndAllOrNothing](../internal/setup/setup_test.go)<br>[TestJSONCAndMarkerRejection](../internal/setup/setup_test.go) | Malformed/empty JSON, invalid JSONC and conflicting shared policies are rejected before writes; final preflight fixtures passed. |
| S19 / verified | Links and aliases | [TestPreconditionRaceAndSymlinkHardlink](../internal/setup/setup_test.go)<br>[TestUnsupportedManifestAndCaseCollision](../internal/setup/routing_test.go)<br>[FuzzConfinedSetupPath](../internal/setup/fuzz_test.go)<br>[FuzzAdapterProfilePaths](../internal/agentsetup/adapter_test.go) | Confined-path fuzz plus symlink/hardlink/case-collision fixtures passed for supported Unix targets. This is bounded filesystem validation, not a universal hostile-kernel guarantee. |
| S20 / verified | Concurrent user edit | [TestPreconditionRaceAndSymlinkHardlink](../internal/setup/setup_test.go)<br>[TestConcurrentInstallAndLockFailure](../internal/setup/routing_test.go) | Expected-hash lost-update rejection and competing-plan fixtures passed the integrated test/race stage on 8b1c5b6 (86.5 seconds). This is bounded conflict/concurrency evidence, not arbitrary-editor isolation or post-parser-fix full-gate qualification. |
| S21 / verified | Publication failures | [TestRollbackAtEveryPublicationBoundary](../internal/setup/setup_test.go)<br>[TestRecoveryCommittedAndExternalEdit](../internal/setup/setup_test.go)<br>[TestRecoveryUnrecordedRenameAndInterruptedRollback](../internal/setup/routing_test.go) | Every instrumented publication boundary, interrupted rollback and recovery fixture passed; failures are nonzero with journal/recovery state and external edits preserved. |
| S22 / verified | Permission, disk or lock failure | [TestConcurrentInstallAndLockFailure](../internal/setup/routing_test.go)<br>[TestReadOnlyDestinationPreflightAndPrivateState](../internal/setup/regression_test.go)<br>[TestRollbackAtEveryPublicationBoundary](../internal/setup/setup_test.go) | Read-only destinations, private-state permissions, competing setup locks and injected publication failures passed. No broad permission repair or universal power-loss/disk guarantee is claimed. |
| S23 / verified | CLI to MCP transition | [TestSetupCLICreateReplayAndModePolicyInheritance](../internal/cli/setup_test.go) | Explicit CLI-to-MCP transition fixtures passed, removing unused CLI assets. Native OMP MCP also passed status/search/read using only deferred MCP dispatch. |
| S24 / verified | MCP to CLI transition | [TestSetupCLICreateReplayAndModePolicyInheritance](../internal/cli/setup_test.go) | Mode/policy inheritance and explicit MCP-to-CLI transition fixtures passed; omitted flags retain the selected prior policy. |
| S25 / verified | Selective uninstall | [TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go)<br>[TestSetupSharedMCPEntryRemovalRetainsOtherConsumer](../internal/cli/setup_regression_test.go) | Shared Claude/Copilot MCP entry survives selective removal; actual Git main/worktree bindings and shared exclusions also survive removal of one installation. |
| S26 / verified | Repeated uninstall | [TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go)<br>[TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go) | Verified byte-stable repeated uninstall in the 19 supported CLI lifecycle fixtures and executable docs; collection bytes and unrelated fixture state were preserved. Not an audit of every host credential store. |
| S27 / verified | Customized uninstall target | [TestOwnedConflictAndOutsideEdit](../internal/setup/setup_test.go)<br>[TestBorrowedDriftAndUpgrade](../internal/setup/setup_test.go) | Customized removal targets and borrowed drift produce actionable preservation/conflict behavior in final lifecycle fixtures. |
| S28 / verified | Missing collection on uninstall | [TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | Verified by compiled Pi uninstall after moving the collection: integration removed without opening or changing the moved collection. Missing-root source regression is also mapped. |
| S29 / verified | Binary and asset upgrade | [TestKnowledgeBaseReferenceDocumentsResolve](../skills/embed_test.go)<br>[TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go)<br>[TestBorrowedDriftAndUpgrade](../internal/setup/setup_test.go)<br>[TestOwnedHookUpgradePreservesForeignHandler](../internal/setup/regression_test.go) | Owned asset upgrades, borrowed/customized conflicts and embedded complete skill-reference fixtures passed. No executable self-upgrade is implied. |
| S30 / verified | Unsupported manifest version | [TestUnsupportedManifestAndCaseCollision](../internal/setup/routing_test.go)<br>[TestOperationReceiptReplayConflictsAndPending](../internal/setup/regression_test.go) | Unknown manifest/receipt versions fail without destructive migration in final fixtures. |
| S31 / verified | Shadowing and profiles | [TestProfileAndInstructionPrecedence](../internal/agentsetup/adapter_test.go)<br>[TestSetupProfileChangesAndExistingFallbackInstructions](../internal/cli/setup_regression_test.go)<br>[TestSetupBareRerunChecksArtifactRootAndCommand](../internal/cli/setup_regression_test.go) | Re-rendered profile/shadowing checks, Codex overrides, nonempty Pi/OpenCode fallback files, and Gemini/OMP custom-profile routing fixtures passed. Native loading of every profile remains separately unqualified. |
| S32 / verified | Fresh native agent session | [TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | Verified within initial OMP project CLI and Codex project CLI/hooks-off retrieval, plus the 8b1c5b6 native Codex global/project CLI and OMP project MCP rechecks. Claude model retrieval remains login-blocked. Other host/build qualification and post-parser-fix rechecks remain separate. |
| S33 / verified | No-learning repeat query | [TestHookRetrievesCitableReadOnlyContext](../internal/agentsetup/hook_test.go)<br>[TestGeneratedMCPLaunchUsesIndependentServer](../internal/agentsetup/mcp_integration_test.go) | Verified for repeated native OMP CLI queries on 5b75df2 and MCP queries on 8b1c5b6: full collection file bytes remain unchanged. Codex single-query/root-precedence purity also passed; no all-host repeated-query guarantee is inferred. |
| S34 / verified | Authorized learning | [TestCASMultiNoteConflictAndConcurrentWriters](../internal/write/service_test.go)<br>[TestMoveRewritesResolvedLinksProvenanceAndBasisPreservingAnchors](../internal/write/policy_test.go)<br>[exercise](../scripts/check_docs.py) | Verified within native OMP 18.16.0 project CLI learning on candidate 5b75df2: revision-checked patch preview/apply/readback, metadata preservation, URN provenance and byte-pure repeat retrieval. Other host/interface learning remains unqualified separately; no universal expansion of this criterion is implied. |
| S35 / verified | Read-only integration | [TestReadOnlyCancellationAndPendingDryRun](../internal/write/policy_test.go)<br>[TestGeneratedMCPLaunchUsesIndependentServer](../internal/agentsetup/mcp_integration_test.go)<br>[TestSetupCLIHookNativeOutputAndReadOnlyPolicy](../internal/cli/setup_test.go) | Generated MCP read-only tool exposure, direct-core no-write hooks and final native OMP MCP collection-byte purity passed. These controls are not OS isolation. |
| S36 / verified | Offline setup | [TestKnowledgeBaseReferenceDocumentsResolve](../skills/embed_test.go)<br>[TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | Bundled skill/reference assets install locally; actual PTY wizard passed with an empty tool PATH (no Node/Git/harness). Setup has no account/runtime-download prerequisite. |
| S37 / verified | Portable team configuration | [TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go)<br>[TestPortableCLIArtifactsContainNoPersonalPaths](../internal/agentsetup/adapter_test.go)<br>[TestSetupPortableAndTildeCreation](../internal/cli/setup_regression_test.go) | Supported portable project CLI anchors and absence of personal root/executable paths passed. Portable MCP/global remain explicitly unsupported rather than receiving a false qualification. |
| S38 / verified | Worktrees and nested repositories | [TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go) | Routing fixtures plus actual Git repository/worktree installation passed: independent bindings, private policy exclusion, selective removal, original shared-exclude restoration and unchanged memory. |
| S39 / verified | Pending or disabled hook trust | [TestHookRegistrationUsesHostEventAndUnits](../internal/agentsetup/adapter_test.go)<br>[TestRejectedRequestsProduceNoPartialIntents](../internal/agentsetup/adapter_test.go)<br>[TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | Actual OMP 18.16.0 and 18.17.0 with native --no-extensions completed and settled with zero KB helper invocations or context messages; collection bytes remained unchanged. Local inspection stayed activation_pending with activation_observed=false. No trust or authentication bypass was performed. |
| S40 / verified | Native hook delivery | [TestHookHostProtocols](../internal/agentsetup/hook_test.go)<br>[TestSetupCLIHookNativeOutputAndReadOnlyPolicy](../internal/cli/setup_test.go)<br>[TestSetupHookMCPUsesNativeContextWithoutSkill](../internal/cli/setup_regression_test.go)<br>[TestMCPHookRetrievesWithoutCLIContinuation](../internal/agentsetup/hook_test.go) | Verified for native OMP session/prompt context delivery in CLI on 5b75df2 and MCP on 8b1c5b6. Claude session/prompt commands exited 0 but its model answer was login-blocked. The corrected OMP MCP observer recognizes deferred MCP dispatch, not filesystem writes; other native hosts remain separately unqualified. |
| S41 / verified | Hook timeout, no result or failure | [TestHookEmptyFailureAndCancellation](../internal/agentsetup/hook_test.go)<br>[TestHookRejectsAmbiguousAndUnboundedInput](../internal/agentsetup/hook_test.go)<br>[TestSetupCLIHookNativeOutputAndReadOnlyPolicy](../internal/cli/setup_test.go)<br>[TestNativeExtensionProcessFailures](../internal/agentsetup/extension_test.go) | Generated Pi/OMP wrapper regression reproduced ignored-SIGTERM blocking, then passed after SIGKILL termination at the existing 3.5-second timeout. Actual OMP 18.16.0 and 18.17.0 delivered bounded no-hit or unavailable contexts for no-hit, failed-valid-stdout and timeout cases, with unchanged collections and confirmed teardown. No hard real-time or descendant-supervisor guarantee. |
| S42 / verified | Compaction, resume and concurrency | [TestConcurrentHookConsultationDoesNotWriteNotes](../internal/agentsetup/hook_test.go)<br>[TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go) | Actual OMP 18.16.0 committed real compaction, retrieved fresh MCP evidence afterwards, stopped and resumed the same session in a new process after changing the project binding, then read the new root and distinct facts. Collection bytes stayed unchanged and all processes settled with confirmed teardown. Concurrent read-only source regression is retained; no exactly-once guarantee. |
| S43 / verified | Mixed global/project hooks | [TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go)<br>[TestSharedInstructionsKeepDistinctHostOwners](../internal/agentsetup/adapter_test.go) | Actual OMP 18.16.0 exercised global CLI/project MCP, inverse global MCP/project CLI and global CLI fallback, with fresh evidence from the effective root and one observed project context per prompt. Bounded read-only duplicates remain permitted by contract; observed deduplication is not a cross-host guarantee. |
| S44 / verified | Unknown agent/version/platform | [TestRejectedRequestsProduceNoPartialIntents](../internal/agentsetup/adapter_test.go)<br>[TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go)<br>[SetupDocumentationTests.test_missing_host_must_have_the_exact_nonzero_inspection_status](../scripts/test_docs.py)<br>[TestSetupCheckDoesNotQualifyUnknownNativeBuild](../internal/cli/setup_qualification_test.go) | Compiled CLI rejected unknown agent and unsupported Cursor global before mutation or scope promotion. Unknown-build executable inspection did not execute the host, left activation unobserved and explicitly diagnosed unobserved native version/platform. Native support remains limited to identified observations; cross-builds do not qualify other host versions. |
