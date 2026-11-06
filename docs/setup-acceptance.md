# Setup acceptance

This is the source-candidate acceptance contract for `kb setup`, not evidence
that published v1.0.0 includes setup. Runtime A01–A26 remain in
[acceptance.json](acceptance.json). The complete S01–S44 requirements, named
fixtures, observed scopes and evidence references are in
[setup-acceptance.json](setup-acceptance.json).

## Status and verification boundary

**12 criteria have bounded verified observations; 32 remain in progress.**
`verified` applies only to the identified candidate, fixture and observed scope,
not every host/build or a final release. A test name maps an actual source
fixture; it is not an invented individual final-test result. Source package
records and integrated-stage outcomes are separated in
[setup-source-tests.json](evidence/setup-source-tests.json).

The **8b1c5b6** static, race/test, coverage and cross-build stages passed; the
subsequent JSONC fuzzer found a real edit-invalid-configuration defect, retained
as corpus `10c2b389efa4b531`. **The full gate did not pass.** Parser repair, rebuild
and full rerun are pending. S17 is not verified. Earlier concrete lifecycle and
native observations remain true for their named binaries, but are not silently
relabeled as post-fix qualification. No coverage threshold is waived and no
final coverage percentage is inferred.

No original criterion is expanded into universal native qualification: S03
requires the explicitly selected create/PARA behavior, S20 the expected-hash
conflict behavior, S32 the named actual-host retrieval and S34 the observed
disposable revision/provenance exercise. Other-host capability qualification is
tracked separately. Conversely, untested compaction/resume and mixed-scope hooks
remain genuine gaps, not inferred passes from a resolver or package test.

## Evidence index

- **L — [CLI lifecycle matrix](evidence/setup-lifecycle.json):** 19 supported
  project/global CLI rows passed on the initial candidate and again on 8b1c5b6.
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
  earlier engine/adapter package runs, later named fixtures, passing integrated
  stages and the real JSONC fuzz failure. Synthetic host callbacks and direct-core
  consumers are not native model sessions.

Initial observations identify candidate **5b75df2**, SHA-256
`dfd8b0c0fe750a55221345a496562c9500a253de123e7f279a24ac2c5cd7fbeb`.
The identified recheck is **8b1c5b6**, SHA-256
`48f50ecf6415a36760b02a929e50ae474ee55f17c10c21509941fd0f45a203ba`,
on Darwin arm64. Further parser/source changes require separately identified
qualification. [Static-site screenshots](evidence/setup-site.json) remain
presentation evidence, not installer or native activation proof.

## Limits and safety

Named native observations do not qualify every other host/build, global/project
combination, mixed CLI/MCP scope, compaction/resume or failure transition. OMP
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
| S01 / in_progress | Existing executable, bare setup | [TestSetupCLIWizardReviewAndHumanOutput](../internal/cli/setup_test.go) | **S, L, N.** Wizard review/buffer and concise output fixture is mapped; compiled lifecycle and named native sessions are separate observations. Final integrated wizard/TTY qualification remains pending. Post-parser-fix integrated qualification is pending. |
| S02 / verified | Existing collection outside project | [TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | **L, B, N.** Verified for disposable external-root CLI lifecycles and the recorded native project sessions: selected canonical root and collection bytes preserved. No all-host activation claim. |
| S03 / verified | Explicit new collection | [TestSetupCLICreateReplayAndModePolicyInheritance](../internal/cli/setup_test.go)<br>[TestSetupPortableAndTildeCreation](../internal/cli/setup_regression_test.go) | **S.** Verified for the selected create/PARA and replay fixtures in the passing integrated test/race stage on 8b1c5b6. This criterion adds no broader native-onboarding requirement; post-parser-fix full qualification remains separate. |
| S04 / in_progress | Invalid or uninitialized existing collection | [TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go)<br>[TestSetupCLIRedactionAndCollectionRootConflict](../internal/cli/setup_test.go) | **S.** Invalid collection and collection/project raw-Markdown conflicts have source fixtures. Final exact run, including uninitialized-root diagnosis, remains pending; no migration/repair is authorized. Post-parser-fix integrated qualification is pending. |
| S05 / in_progress | Project installation | [TestHostConfigurationContracts](../internal/agentsetup/adapter_test.go)<br>[TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go)<br>[TestPrivacyNoGitAndFatalRepositoryCommands](../internal/setup/regression_test.go) | **L, S.** All ten supported project CLI configuration rows passed. Private registry/exclusion and unrelated-file boundaries are source-mapped; native usability is limited to the named native runs. Post-parser-fix integrated qualification is pending. |
| S06 / in_progress | Global installation | [TestHostConfigurationContracts](../internal/agentsetup/adapter_test.go)<br>[TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go) | **L, S.** Nine supported global CLI rows passed; Cursor global is explicitly rejected. Global configuration does not establish native global activation or cross-project model behavior. Post-parser-fix integrated qualification is pending. |
| S07 / verified | Project override | [TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go)<br>[TestSharedInstructionsKeepDistinctHostOwners](../internal/agentsetup/adapter_test.go) | **S, N.** Verified within native Codex CLI/hooks-off global-plus-project scope on 8b1c5b6: the project root/value overrides the distinct global root/value, and both collections remain byte-pure. Mixed CLI/MCP and other host combinations are separately unqualified. |
| S08 / in_progress | No binding or disabled binding | [TestDisabledPortableBindingStopsGlobal](../internal/setup/routing_test.go)<br>[TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go) | **S.** Disabled portable binding stops global fallback in a source fixture. Final no-binding/disabled-path execution evidence is pending; no home-directory collection guessing is supported. Post-parser-fix integrated qualification is pending. |
| S09 / verified | Shared artifacts | [TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go)<br>[TestSetupSharedMCPEntryRemovalRetainsOtherConsumer](../internal/cli/setup_regression_test.go) | **B, S.** Verified in the compiled Claude+Copilot shared project MCP probe: one server entry survives removal of the first consumer and disappears after the last. Shared/borrowed rule coverage is separately source-mapped. |
| S10 / verified | Unchanged repeat | [TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go)<br>[TestOwnedHookUpgradePreservesForeignHandler](../internal/setup/regression_test.go) | **L, B, S.** Verified byte-stable repeated CLI installation in all 19 supported scope rows and the executable-doc workflow. This is filesystem/configuration evidence, not repeated native activation. |
| S11 / in_progress | Dry-run | [TestPreflightPureAndAllOrNothing](../internal/setup/setup_test.go)<br>[TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go) | **L, B, S.** Dry-run filesystem purity passed the CLI matrix and executable docs, including source create-dry-run coverage. Setup uses bundled assets without network downloads; exact final integrated checks remain pending. Post-parser-fix integrated qualification is pending. |
| S12 / in_progress | Incomplete automation | [TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go) | **S.** Incomplete automation/no-prompt argument fixtures are mapped. Final exact CLI regression execution remains pending. Post-parser-fix integrated qualification is pending. |
| S13 / in_progress | Conflicting flags | [TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go)<br>[TestRejectedRequestsProduceNoPartialIntents](../internal/agentsetup/adapter_test.go) | **S.** Conflicting scope/create/PARA and unsupported option fixtures are mapped. Final exact CLI/adapter regression execution remains pending. Post-parser-fix integrated qualification is pending. |
| S14 / in_progress | Customized skill or rule | [TestBorrowedDriftAndUpgrade](../internal/setup/setup_test.go)<br>[TestOwnedConflictAndOutsideEdit](../internal/setup/setup_test.go) | **S.** Borrowed/custom owned asset preservation and conflict fixtures are mapped; --yes is not overwrite authority. Final exact source run remains pending. Post-parser-fix integrated qualification is pending. |
| S15 / in_progress | Malformed ownership markers | [TestJSONCAndMarkerRejection](../internal/setup/setup_test.go)<br>[FuzzManagedBlockParsing](../internal/setup/fuzz_test.go) | **S.** Orphaned, reversed and duplicate markers plus bounded fuzz seeds are mapped. Final integrated fuzz/test evidence remains pending. Post-parser-fix integrated qualification is pending. |
| S16 / in_progress | Text presentation | [TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go)<br>[TestJSONAndTOMLPreserveForeignBytes](../internal/setup/setup_test.go)<br>[FuzzManagedBlockParsing](../internal/setup/fuzz_test.go) | **S.** BOM, CRLF and missing-final-newline preservation fixtures are mapped. This is supported-format coverage, not arbitrary encoding compatibility; final exact run pending. Post-parser-fix integrated qualification is pending. |
| S17 / in_progress | Shared JSON/TOML | [TestJSONAndTOMLPreserveForeignBytes](../internal/setup/setup_test.go)<br>[TestJSONCAndMarkerRejection](../internal/setup/setup_test.go)<br>[TestTOMLEscapedMultilineHeadersAndUpgrade](../internal/setup/regression_test.go)<br>[FuzzJSONCEntryPreservation](../internal/setup/fuzz_test.go) | **S.** Preservation/upgrade fixtures exist, but the 8b1c5b6 full gate found a real JSONC edit-invalid-config defect (retained corpus 10c2b389efa4b531). Parser correction and post-fix regression/fuzz/full-gate rerun are pending; this criterion is not verified. Post-parser-fix integrated qualification is pending. |
| S18 / in_progress | Invalid configuration | [TestPreflightPureAndAllOrNothing](../internal/setup/setup_test.go)<br>[TestJSONCAndMarkerRejection](../internal/setup/setup_test.go) | **S.** Malformed config and duplicate-key preflight fixtures require no artifact/state publication. Final exact regression evidence remains pending. Post-parser-fix integrated qualification is pending. |
| S19 / in_progress | Links and aliases | [TestPreconditionRaceAndSymlinkHardlink](../internal/setup/setup_test.go)<br>[TestUnsupportedManifestAndCaseCollision](../internal/setup/routing_test.go)<br>[FuzzConfinedSetupPath](../internal/setup/fuzz_test.go)<br>[FuzzAdapterProfilePaths](../internal/agentsetup/adapter_test.go) | **S.** Symlink, hardlink, case-alias and path/profile confinement fixtures are mapped. Final exact fuzz/platform run remains pending; no hostile-editor isolation guarantee follows. Post-parser-fix integrated qualification is pending. |
| S20 / verified | Concurrent user edit | [TestPreconditionRaceAndSymlinkHardlink](../internal/setup/setup_test.go)<br>[TestConcurrentInstallAndLockFailure](../internal/setup/routing_test.go) | **S.** Expected-hash lost-update rejection and competing-plan fixtures passed the integrated test/race stage on 8b1c5b6 (86.5 seconds). This is bounded conflict/concurrency evidence, not arbitrary-editor isolation or post-parser-fix full-gate qualification. |
| S21 / in_progress | Publication failures | [TestRollbackAtEveryPublicationBoundary](../internal/setup/setup_test.go)<br>[TestRecoveryCommittedAndExternalEdit](../internal/setup/setup_test.go)<br>[TestRecoveryUnrecordedRenameAndInterruptedRollback](../internal/setup/routing_test.go) | **S.** Prepared/publication/commit and interrupted-recovery fixtures are mapped. Final exact run is pending; per-file publication is not globally atomic or universal power-loss proof. Post-parser-fix integrated qualification is pending. |
| S22 / in_progress | Permission, disk or lock failure | [TestConcurrentInstallAndLockFailure](../internal/setup/routing_test.go)<br>[TestReadOnlyDestinationPreflightAndPrivateState](../internal/setup/regression_test.go)<br>[TestRollbackAtEveryPublicationBoundary](../internal/setup/setup_test.go) | **S.** Lock refusal, read-only destinations, private state modes and injected publication errors have fixtures. Final exact run and any broader disk-exhaustion claim remain pending. Post-parser-fix integrated qualification is pending. |
| S23 / in_progress | CLI to MCP transition | [TestSetupCLICreateReplayAndModePolicyInheritance](../internal/cli/setup_test.go) | **S.** The engine checkpoint records a compiled CLI-to-MCP smoke removing unused skills. Native mixed-scope/interface transitions and final exact regression run remain pending. Post-parser-fix integrated qualification is pending. |
| S24 / in_progress | MCP to CLI transition | [TestSetupCLICreateReplayAndModePolicyInheritance](../internal/cli/setup_test.go) | **S.** MCP-to-CLI removal of unused config and inherited root/policy passed the earlier source fixture. Final frozen-source execution is pending; this criterion does not require every host to perform a new model session. Post-parser-fix integrated qualification is pending. |
| S25 / in_progress | Selective uninstall | [TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go)<br>[TestSetupSharedMCPEntryRemovalRetainsOtherConsumer](../internal/cli/setup_regression_test.go) | **B, S.** Compiled selective removal preserved the shared MCP entry and remaining configured artifacts. Final exact source checks are pending; native model-adherence qualification is separate from ownership retention. Post-parser-fix integrated qualification is pending. |
| S26 / verified | Repeated uninstall | [TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go)<br>[TestLifecycleSharedBorrowedAndPristine](../internal/setup/setup_test.go) | **L, B, S.** Verified byte-stable repeated uninstall in the 19 supported CLI lifecycle fixtures and executable docs; collection bytes and unrelated fixture state were preserved. Not an audit of every host credential store. |
| S27 / in_progress | Customized uninstall target | [TestOwnedConflictAndOutsideEdit](../internal/setup/setup_test.go)<br>[TestBorrowedDriftAndUpgrade](../internal/setup/setup_test.go) | **S.** Modified owned regions conflict; borrowed customizations and outside edits survive uninstall in source fixtures. Final exact regression run remains pending. Post-parser-fix integrated qualification is pending. |
| S28 / verified | Missing collection on uninstall | [TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | **B, S.** Verified by compiled Pi uninstall after moving the collection: integration removed without opening or changing the moved collection. Missing-root source regression is also mapped. |
| S29 / in_progress | Binary and asset upgrade | [TestKnowledgeBaseReferenceDocumentsResolve](../skills/embed_test.go)<br>[TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go)<br>[TestBorrowedDriftAndUpgrade](../internal/setup/setup_test.go)<br>[TestOwnedHookUpgradePreservesForeignHandler](../internal/setup/regression_test.go) | **S.** Complete bundled references, borrowed-version conflict and owned hook upgrade fixtures are mapped. Exact final candidate/asset checks are pending; future released-package qualification is a separate distribution concern. Post-parser-fix integrated qualification is pending. |
| S30 / in_progress | Unsupported manifest version | [TestUnsupportedManifestAndCaseCollision](../internal/setup/routing_test.go)<br>[TestOperationReceiptReplayConflictsAndPending](../internal/setup/regression_test.go) | **S.** Unsupported manifest and operation receipt conflicts have source fixtures. Final exact execution/version-boundary evidence remains pending; no guessed migration is promised. Post-parser-fix integrated qualification is pending. |
| S31 / in_progress | Shadowing and profiles | [TestProfileAndInstructionPrecedence](../internal/agentsetup/adapter_test.go)<br>[TestSetupProfileChangesAndExistingFallbackInstructions](../internal/cli/setup_regression_test.go)<br>[TestSetupBareRerunChecksArtifactRootAndCommand](../internal/cli/setup_regression_test.go) | **S, B, N.** Custom profiles, nonempty fallbacks, new shadowing files and missing executables have source fixtures. Missing-host check behavior was exercised without claiming readiness. Final exact diagnosis regressions are pending; model-adherence qualification is separate. Post-parser-fix integrated qualification is pending. |
| S32 / verified | Fresh native agent session | [TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | **N.** Verified within initial OMP project CLI and Codex project CLI/hooks-off retrieval, plus the 8b1c5b6 native Codex global/project CLI and OMP project MCP rechecks. Claude model retrieval remains login-blocked. Other host/build qualification and post-parser-fix rechecks remain separate. |
| S33 / verified | No-learning repeat query | [TestHookRetrievesCitableReadOnlyContext](../internal/agentsetup/hook_test.go)<br>[TestGeneratedMCPLaunchUsesIndependentServer](../internal/agentsetup/mcp_integration_test.go) | **N, S.** Verified for repeated native OMP CLI queries on 5b75df2 and MCP queries on 8b1c5b6: full collection file bytes remain unchanged. Codex single-query/root-precedence purity also passed; no all-host repeated-query guarantee is inferred. |
| S34 / verified | Authorized learning | [TestCASMultiNoteConflictAndConcurrentWriters](../internal/write/service_test.go)<br>[TestMoveRewritesResolvedLinksProvenanceAndBasisPreservingAnchors](../internal/write/policy_test.go)<br>[exercise](../scripts/check_docs.py) | **N, B, S.** Verified within native OMP 18.16.0 project CLI learning on candidate 5b75df2: revision-checked patch preview/apply/readback, metadata preservation, URN provenance and byte-pure repeat retrieval. Other host/interface learning remains unqualified separately; no universal expansion of this criterion is implied. |
| S35 / in_progress | Read-only integration | [TestReadOnlyCancellationAndPendingDryRun](../internal/write/policy_test.go)<br>[TestGeneratedMCPLaunchUsesIndependentServer](../internal/agentsetup/mcp_integration_test.go)<br>[TestSetupCLIHookNativeOutputAndReadOnlyPolicy](../internal/cli/setup_test.go) | **N, S.** Generated MCP read-only tools/direct-core no-write fixtures passed scoped tests; native OMP MCP status/search/read used only deferred MCP dispatch with no model shell/filesystem calls and preserved collection bytes. Local read-only paths are not an OS sandbox; post-parser-fix source qualification remains pending. Post-parser-fix integrated qualification is pending. |
| S36 / in_progress | Offline setup | [TestKnowledgeBaseReferenceDocumentsResolve](../skills/embed_test.go)<br>[TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | **B, S.** Version-matched bundled skill/reference installation has source fixtures and real CLI smoke; setup has no account/runtime download path. Final exact source checks remain pending. Native host qualification is separate from offline asset installation. Post-parser-fix integrated qualification is pending. |
| S37 / in_progress | Portable team configuration | [TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go)<br>[TestPortableCLIArtifactsContainNoPersonalPaths](../internal/agentsetup/adapter_test.go)<br>[TestSetupPortableAndTildeCreation](../internal/cli/setup_regression_test.go) | **S.** Relative project binding, second-user resolution and no personal paths have source fixtures. Final exact portable regressions are pending. Portable MCP/global are explicitly unsupported; this criterion does not require every native host/team session. Post-parser-fix integrated qualification is pending. |
| S38 / in_progress | Worktrees and nested repositories | [TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go) | **S.** Synthetic nested-repository/copied-worktree boundaries and anchored portable resolution have source fixtures. Final exact routing run remains pending; no additional native multi-worktree session requirement is imposed. Post-parser-fix integrated qualification is pending. |
| S39 / in_progress | Pending or disabled hook trust | [TestHookRegistrationUsesHostEventAndUnits](../internal/agentsetup/adapter_test.go)<br>[TestRejectedRequestsProduceNoPartialIntents](../internal/agentsetup/adapter_test.go)<br>[TestSetupCLIInstallResolveCheckAndUninstall](../internal/cli/setup_test.go) | **L, N, S.** Pending activation and unsupported required-hook diagnostics have source coverage. Named native runs do not cover disabled/trust-pending settings on every host; no trust or auth bypass was performed. Post-parser-fix integrated qualification is pending. |
| S40 / verified | Native hook delivery | [TestHookHostProtocols](../internal/agentsetup/hook_test.go)<br>[TestSetupCLIHookNativeOutputAndReadOnlyPolicy](../internal/cli/setup_test.go)<br>[TestSetupHookMCPUsesNativeContextWithoutSkill](../internal/cli/setup_regression_test.go)<br>[TestMCPHookRetrievesWithoutCLIContinuation](../internal/agentsetup/hook_test.go) | **N, S.** Verified for native OMP session/prompt context delivery in CLI on 5b75df2 and MCP on 8b1c5b6. Claude session/prompt commands exited 0 but its model answer was login-blocked. The corrected OMP MCP observer recognizes deferred MCP dispatch, not filesystem writes; other native hosts remain separately unqualified. |
| S41 / in_progress | Hook timeout, no result or failure | [TestHookEmptyFailureAndCancellation](../internal/agentsetup/hook_test.go)<br>[TestHookRejectsAmbiguousAndUnboundedInput](../internal/agentsetup/hook_test.go)<br>[TestSetupCLIHookNativeOutputAndReadOnlyPolicy](../internal/cli/setup_test.go) | **S.** Bounded input/output, no-hit, invalid input and cooperative cancellation have source fixtures. Native timeout/kill/failure behavior across hosts remains pending; no hard realtime SLA is claimed. Post-parser-fix integrated qualification is pending. |
| S42 / in_progress | Compaction, resume and concurrency | [TestConcurrentHookConsultationDoesNotWriteNotes](../internal/agentsetup/hook_test.go)<br>[TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go) | **S, N.** Concurrent read-only source coverage and settled native OMP teardown exist. Compaction/resume routing and exactly-once behavior are not qualified; no exactly-once guarantee is made. Post-parser-fix integrated qualification is pending. |
| S43 / in_progress | Mixed global/project hooks | [TestRoutingBoundaryWorktreeAndPortable](../internal/setup/routing_test.go)<br>[TestSharedInstructionsKeepDistinctHostOwners](../internal/agentsetup/adapter_test.go) | **S, N.** Scope/host policy guards and resolver routing are source-mapped. Actual mixed global/project hooks and CLI/MCP precedence have not been exercised in native sessions. Post-parser-fix integrated qualification is pending. |
| S44 / in_progress | Unknown agent/version/platform | [TestRejectedRequestsProduceNoPartialIntents](../internal/agentsetup/adapter_test.go)<br>[TestSetupCLIPureDryRunAndValidation](../internal/cli/setup_test.go)<br>[SetupDocumentationTests.test_missing_host_must_have_the_exact_nonzero_inspection_status](../scripts/test_docs.py) | **L, B, S.** Cursor global and invalid vscode ID were explicitly rejected before mutation; missing native hosts are explicitly nonready. Unknown native versions/platforms and every harness build are not qualified by descriptors or cross-builds. Post-parser-fix integrated qualification is pending. |
