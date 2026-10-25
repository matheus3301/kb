# v1.0 exit criteria

**Experimental; v1.0 readiness is not established.** This checklist is assessed
against experimental v0.3.1 as of **2026-10-25**, not against unmerged workstreams.
It extends, rather than replaces, the 26 original criteria in the
[acceptance inventory](acceptance.json). The [roadmap](roadmap.md) sets scope;
the [compatibility policy](compatibility.md) distinguishes current behavior from
proposed v1.x commitments. No item authorizes a release.

## Evidence and status rules

- **Satisfied baseline** means the cited released evidence supports the bounded
  claim. It does not mean the eventual v1.0 candidate has passed it.
- **Pending** means required implementation, measurement or independent review is
  missing. An unchecked item stays pending after a related branch or CI run exists
  until evidence for its actual acceptance conditions is linked.
- **Proposed decision** means a threshold or support commitment needs explicit
  maintainer review. It is not an implemented guarantee or an approved SLA.

For the final candidate, record exact commit/version, commands, result, platform,
synthetic fixture/corpus identity, tool versions, sample/workload counts and known
limits. Link inspectable public evidence and a review by someone other than the
implementation author for each exit item. Do not copy private collections or
transcripts. Existing acceptance labels establish traceability, not independent
v1.0 sign-off. Coverage is not evidence of isolation, durability or relevance.

## Satisfied baseline, not final-candidate sign-off

- [x] **B01 — Local core acceptance.** All A01–A26 have linked tests/evidence in
  [acceptance.json](acceptance.json): CLI/MCP parity on equivalent fixtures,
  unknown metadata/presentation preservation, CAS, idempotency, recovery,
  confinement, query purity, bounded context and explicit integration. Historical
  [full validation](evidence/validation.json) is dated 2026-10-24; later fixes and
  [interface evaluation](qa/independent-interfaces.md) extend that evidence.
- [x] **B02 — Original local core performance targets.** The 2026-10-25
  [performance report](performance.md#original-targets) records 117.7 ms warm
  p95 against 150 ms, 134.0 MiB RSS against 150 MiB and 179.0 ms rebuild p95
  against 10 seconds, on Apple M4/macOS arm64, 10,000 synthetic 2 KiB notes.
  Core RSS and end-to-end child-process RSS are different measurements. No
  Linux or arbitrary-machine latency guarantee follows.
- [x] **B03 — Released installed-package proof, macOS arm64 only.**
  [v0.3.1 release verification](https://github.com/matheus3301/kb/releases/tag/v0.3.1)
  records exact-commit extracted binary/Homebrew source installation, formula
  test, acceptance, 43 documentation requests and installed MCP checks on
  synthetic data. Checksums/archive inspection cover all release assets; this
  is not installed runtime proof for Linux or macOS amd64.
- [x] **B04 — Required gates exist without a readiness bypass.**
  [Quality policy](quality.md) and [CI workflow](../.github/workflows/ci.yml)
  require repository checks, Linux/macOS tests, race/fuzz, coverage of at least
  80%, four cross-builds, scans and a release snapshot. A green run proves only
  its exercised checks. The 88.4%/102 tests/314 pass events/seven fuzz targets
  in the original report are dated counts, not current suite totals.

## Pending exit evidence

- [ ] **V01 — Revalidate the integrated candidate.** Run `make check` and
  `make release-snapshot` on the exact integrated commit; link passing CI gate,
  application acceptance and per-criterion regressions. Recheck A01–A26 after
  all workstreams land. Record and independently review failures and limitations;
  no skipped jobs, weakened gates or historical counts as fresh results.
- [ ] **V02 — Retrieval usefulness and interface parity.** Publish the annotated
  Portuguese/English corpus, query categories, expected results and definitions
  of top-k relevance, reciprocal rank and recall. Record baseline and candidate
  values by category, including no-match/filter/accents cases. All deterministic
  expected-result, exact citation/revision, budget/truncation, query-purity and
  CLI/MCP parity assertions must pass on the same fixtures. Link the bounded CI
  evaluation and its own behavioral regressions. Quality thresholds require D01;
  aggregate metrics alone cannot hide a failing category.
- [ ] **V03 — Operational reliability.** Link repeated concurrency and real
  process-interruption runs with declared worker count, operation mix, duration,
  seeds and completed operation count. Exercise every journal/publication/recovery
  boundary, interruption during recovery, conflicting CAS and identical-key
  retries. After recovery there must be no partial acknowledged transaction,
  duplicate logical effect, lost confirmed receipt or changed immutable source.
  Link deterministic regressions for reproducible failures. Coverage percentage
  and advisory locks are not substitutes for these observations; workload scope
  needs D02.
- [ ] **V04 — Operational backup/restore.** Link the merged contract and actual
  CLI/MCP synthetic scenarios: pending-state behavior, empty-destination restore,
  exact revisions/configuration/receipt fidelity, replay and continued writes.
  Prove rejection of overwrites, unsafe archive paths/symlinks, missing/corrupt
  manifests or bytes and read-only violations, including injected publication
  failures. Distinguish operational backup from export and disposable cache;
  document operational version/upgrade behavior. No such interface ships in
  v0.3.1, and this criterion has no passing evidence yet.
- [ ] **V05 — Large-corpus resource envelope.** Publish serialized paired
  baseline/candidate 10k/50k, 2 KiB measurements: fresh CLI search/read/validation,
  reused MCP, repair preview/apply, rebuild, p50/p95 and peak RSS. Declare hardware,
  toolchain, sample counts, cache conditions and noise; link freshness, ranking,
  confinement and durability regressions. Demonstrate the claimed startup/RSS
  reduction against v0.3.1 and disclose regressions. v0.3.1's 50k search is still
  2.19 s median/483.1 MiB peak RSS; acceptable budgets need D03.
- [ ] **V06 — Compatibility and upgrades.** Approve the v1.x commitments and
  support scope in D04. Exercise prior supported client requests against the
  candidate CLI/JSON and MCP, preserving flags, exit mappings, error codes,
  schemas, read-only enforcement and fixed-root stdio behavior. Verify content,
  config and operational-state upgrade/recovery on synthetic prior-version
  fixtures, with rollback limits disclosed. Link any breaking-change decision
  and operator migration procedure; a build alone does not prove compatibility.
- [ ] **V07 — Package and consumer verification.** For each platform/channel
  selected in D04, link actual installation, exact version/commit, CLI acceptance
  and independent MCP initialization/discovery/workflow results. Distinguish native
  archive execution and archived-source build from an installed package. Exercise
  CLI + the separately installed skill and MCP without the skill; keep client
  versions and activation limits explicit. No universal model adherence claim.
- [ ] **V08 — Independent readiness decision.** Review this checklist and all
  linked evidence against one candidate, reconcile public docs/catalogs/skill
  with merged behavior, list remaining risks and explicitly decide whether to
  remove experimental status. Release publication still needs separate authority
  under [release operations](releases.md). No author self-check or green CI alone
  satisfies this item.

## Proposed decisions, not passed criteria

- [ ] **D01 — Retrieval quality floor.** Approve metric definitions, k values,
  category-specific pass thresholds and permitted baseline/candidate changes
  after examining the annotated corpus. Proposed rule: no unexplained regression
  in any category; required citation, budget and purity assertions always pass.
- [ ] **D02 — Reliability observation scope.** Approve a repeatable extended
  workload (worker/operation counts, interruption schedule, duration and supported
  filesystem conditions) and the observation period required in addition to
  deterministic boundary tests. No arbitrary soak duration is claimed sufficient.
- [ ] **D03 — Performance support envelope.** Approve hardware/corpus-specific
  10k/50k startup/RSS budgets and a noise/regression policy after paired profiling.
  Retain the original local core targets; do not turn shared-runner timing into
  a portable SLA or silently lower correctness/quality gates.
- [ ] **D04 — Compatibility/support commitment.** Review the proposed v1.x
  compatibility rules, supported upgrade origins, OS/architecture/package/client
  matrix and maintenance scope. Current four-target packaging and two-OS source
  CI do not imply four-target installed-package proof or an LTS commitment.

After the other roadmap PRs merge, reconcile V02/V04/V05 with their actual report
paths and interface versions, then reassess every unchecked item. This document
must not predict their results or mark them passed on another branch's behalf.
