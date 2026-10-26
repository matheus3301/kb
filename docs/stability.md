# v1.0 exit criteria

**Shipping-candidate evidence is reviewable; stable publication remains pending.**
Experimental [v0.4.0](https://github.com/matheus3301/kb/releases/tag/v0.4.0),
source `74f0d19`, remains the observed published/Homebrew-installed release.
The shipping candidate uses Go/module source
`b71c0763c87313666b3dd79da08da12b7b3512c1`, version `v1.0.0-candidate.3`,
executable SHA-256
`8aecbf6cf22d2c4bc8c97e1d8b63fea725dfa325b2e5f632f2ed97fc815815ac`
and `CGO_ENABLED=0`. Its 69 tracked Go/module files match protected main
`5c94c993aa8ec97295bdf6bcdb1d5cd2f052a2f7`; the final documentation/release
commit and rebuilt executable identity must be bound separately.

The [candidate qualification report](qa/v1-qualification.md) and
[structured manifest](evidence/v1-qualification.json) retain fresh retrieval,
prior-version compatibility, default-log concurrency, 102 interruption cases
and one complete passing local resource matrix. The shipping 1,800-second
observation **failed a restore timeout**, despite sufficient counts/time.
**V08 publication-authorization
review, stable tag publication and actual published-v1/Homebrew/installed
consumer verification remain pending.** Factual documentation is published for
review; V08 is not a hidden prerequisite for making the evidence reviewable.

All three historical `7ceed` resource matrices and the first source-b71 CGO=1
matrix remain [failed attempts](evidence/v1-qualification-history.json).
The historical `7ceed` long pass does not qualify this shipping executable.
The original pre-fix backup-timeout endurance also remains failed; no lost
confirmed state was demonstrated. No threshold, sample selection or old passing
cell was used to erase a failure. No CGO-causality claim follows.

## Evidence and status rules

- **Satisfied baseline** is historical evidence for a bounded claim, not sign-off
  on the eventual candidate.
- **Implemented; qualification pending** means source and focused evidence exist,
  but the exact integrated/installed candidate still needs the listed checks.
- **Policy selected** means D01–D04 are delegated coordinator acceptance choices,
  with rationale below. They are not measured results, independent review or
  human approval. Changing a threshold requires a recorded reason and review,
  never silently repinning it to an observed failure.

For each V item, record exact commit/version and binary identity, actual commands
and outcomes, platform/filesystem/toolchain, synthetic fixture hash, samples,
duration/seeds/counts, known limits and an inspectable evidence link. Record the
reviewer's role and scope separately from implementation/self-checks. Do not copy
private collections or transcripts. Coverage, model obedience and green CI are
not proofs of relevance, isolation, durability or readiness.

## Satisfied baseline, not final-candidate sign-off

- [x] **B01 — Local core acceptance.** A01–A26 link tests and historical evidence
  for CLI/MCP parity, metadata preservation, CAS, replay/recovery, confinement,
  purity, bounded context and explicit integration.
  [Original validation](evidence/validation.json) is dated 2026-10-24;
  [independent interface evaluation](qa/independent-interfaces.md) and later
  regression reports extend it without making it final-v1 evidence.
- [x] **B02 — Original local core targets.** The
  [v0.3.1 report](performance.md#original-targets) records 117.7 ms warm p95,
  134.0 MiB RSS and 179.0 ms rebuild p95 against 150 ms/150 MiB/10 s targets:
  Apple M4/macOS arm64, Go 1.27.3, 10,000 synthetic 2 KiB notes. Core RSS and
  end-to-end child-process RSS are distinct; no portable SLA follows.
- [x] **B03 — Released package proof, macOS arm64 only.**
  [v0.3.1 release verification](https://github.com/matheus3301/kb/releases/tag/v0.3.1)
  records extracted binary/Homebrew source installation, formula test, acceptance,
  43 documentation requests and installed MCP on synthetic data. Other archive
  targets are not installed-tested by this record.
- [x] **B04 — Required gates without a readiness bypass.**
  [Quality policy](quality.md) and [CI](../.github/workflows/ci.yml) require
  repository/static checks, Linux/macOS tests, race/fuzz, at least 80% coverage,
  four cross-builds, scans and release snapshot. Original counts of 88.4%,
  102 tests, 314 pass events and seven fuzz targets remain dated observations.

## Candidate acceptance — observed evidence and remaining gates

- [x] **V01 — Exact frozen runtime candidate.**
  [Source/build manifest](evidence/v1-qualification.json) binds the 69 matching
  Go/module files, measured executable and CGO=0 build protocol.
  [Exact PR CI](https://github.com/matheus3301/kb/actions/runs/40263657403) and
  [exact merged-main CI](https://github.com/matheus3301/kb/actions/runs/40264275136)
  passed. Source gates recorded 91 tooling tests, 88.7% Go coverage, race/scans,
  12 fuzz targets, four cross-builds and 43 documentation requests; committed
  native/archive-source snapshots passed. Final docs-only integration and
  published executable identity remain separate; unchanged source is not an
  executable-hash claim.
- [x] **V02 — Retrieval usefulness and parity.** The
  [fresh shipping report](evidence/v1-retrieval.json) passed 47 CLI/MCP pairs and
  17 category validations on unchanged corpus
  `562af0124822b42e72125ae9307e5bc5a5fe9fb0f1164be13a8811aa6d49cc4c`:
  31 searches, eight context cases and eight source reads. All 24 positive and
  seven negative judgments retain authored membership/order/filter labels,
  measured outputs and category membership. Recall@5 is 0.9930555555555555;
  MRR@5 is 1.0. Negative-only means are null, excluded from positive means.
  D01 is satisfied for this finite lexical corpus, not arbitrary semantic truth.
- [ ] **V03 — Operational reliability.** The
  [fresh shipping evidence](evidence/v1-reliability.json) records a passing
  default-log companion: 610 mutations, 610 identical-request replays,
  366 expected CAS conflicts, six backup checks and zero unexpected errors.
  Three deterministic interruption repetitions passed all 102 observed cases:
  33 archive/recovery, 54 publication/recovery-position and 15 restore-position
  cases, with no failures/skips, in 24.901 package seconds. These are test
  children, not public-CLI or hardware power-loss proof. **The shipping long
  profile failed seed 31415's final restore at the unchanged 30-second timeout.**
  It reported 16,840 mutations, 16,840 replays, 10,104 expected conflicts and 24
  completed checkpoints, but sufficient counts/time do not offset that failure.
  No lost confirmed state was demonstrated; final failed-checkpoint fidelity did
  not complete. D02 remains failed, not qualified.
- [ ] **V04 — Operational backup/restore.**
  [Actual prior-producer archive/upgrade](evidence/v1-compatibility.json),
  companion restorations and positional interruption evidence passed exact
  authority/configuration/receipt/source fidelity, replay and continued writes.
  Existing [archive/parity/failure tests](../cmd/kb/backup_test.go) and full source
  gates cover pending refusal, new-only destinations, unsafe/corrupt archives,
  read-only enforcement, faults and interrupted retries. The shipping final
  restore timeout blocks closure; its failed checkpoint did not complete
  fidelity/replay/continued-write checks. Retain that failure and qualify any
  changed runtime afresh.
  Pending journals are refused, not archived; same-disk staging is not disaster
  recovery and disposable caches are not authoritative backup state.
- [x] **V05 — Local resource envelope.** One complete
  [shipping matrix](evidence/v1-resources.json) passed all 16 unchanged D03 rows
  and three core targets. It used 20 read, three repair, five independent rebuild
  and 20 core samples, with eight fresh all-worker native quiet-state proofs.
  Core warm p95 was 113.4 ms / 131.4 MiB; fresh 10k CLI rebuild p95 was
  179.690 ms. Both binaries' 10k/50k rebuilds and all drift are retained,
  including candidate 50k rebuild p95 +16.9%. Independent automated actual-result
  audit passed this bounded D03 claim, not V08. No whole-machine idleness,
  OS cache eviction, uniform improvement or portable latency/RSS SLA is claimed.
- [x] **V06 — Compatibility and upgrades.**
  [Actual preserved executable evidence](evidence/v1-compatibility.json) passed
  eight v0.3.1-to-shipping checks and nine v0.4.0-to-shipping checks. The latter
  includes an archive actually produced by v0.4.0, exact restored bytes/receipts,
  replay and subsequent writes. Prior-producer archive testing is explicitly
  not applicable to v0.3.1, which has no operational archive tools.
  Pending journals were recovered with the prior binary before upgrade.
  No automatic downgrade or blanket older-version compatibility is promised.
- [ ] **V07 — Published packages and installed consumers.** Historical installed
  v0.4.0 and current native/archive-source snapshots are not actual installed-v1
  proof. Under D04 and [release operations](releases.md), verify real published
  macOS arm64 Homebrew/native artifacts and Linux amd64 archive/archived-source
  workflows, formula test, CLI acceptance, separately installed skill consumer
  and independently initialized/discovered stdio MCP consumer.
  Record actual release identity/checksums and client/channel limits.
- [ ] **V08 — Independent publication-authorization review.** Review this
  completed public evidence, V01–V07, current docs/catalogs/skill, retained failures
  and bounded D01–D04 scope. This review is distinct from authorship and from
  the completed automated D03 audit; **the final V08 decision remains pending**.
  It is not a prerequisite for publishing factual evidence for review. Do not
  label automated review as human approval. Actual stable-tag publication,
  Homebrew upgrade and installed verification remain separately authorized and
  separately evidenced.


## Qualification policy — D01–D04

These choices bound what must be observed before stable v1, rather than promise
absence of bugs. They are selected for candidate qualification by delegated
coordination, subject to evidence and independent readiness review.

### D01 — Ceiling-aware lexical retrieval floor

Keep the [existing authored corpus](../testdata/retrieval/corpus.json) and metric
definitions; do not relabel expected documents to fit the output. All exact
membership/order/no-match, source/revision/citation, budget/truncation, purity
and CLI/MCP parity assertions must pass. No category may regress without an
explained, reviewed contract change.

For every positive query with relevant set R, require Recall@5 at least
`min(5, |R|) / |R|`, and RR@5 at least its recorded baseline. Thus the current
24 positive judgments require macro Recall@5 **at least 0.9930555555555555**
(within numeric roundoff) and MRR@5 **at least 1.0**. The six-relevant case has
a **5/6** floor, not an impossible 1.0; the other positive cases fit in five.
All seven negative judgments require zero hits/total and are excluded from
macro means. Report category and per-case values, not aggregates alone.
Rationale: retain the demonstrated lexical contract without inventing semantic
recall or a cutoff-incompatible target. A changed corpus needs a new baseline
and explicit review of floors, not silent comparison across different labels.

### D02 — Bounded extended reliability observation

Require **at least 30 minutes total across three distinct recorded seeds**, at
least ten minutes per seed, with **four concurrent writers per seed** on local
supported filesystems. This extended profile explicitly sets
**`write.maintain_log = false` and `write.maintain_indexes = true`**.
It is not a 30-minute default-log-enabled observation. Across these runs require
**at least 10,000 acknowledged successful mutations and identical-request replays
combined**; report unique committed effects and replay acknowledgments separately.
Previews, expected conflicts and failed attempts do not count toward that minimum.
The qualification harness must expose an explicit `--maintain-log` control and
record effective configuration; this is not a new `kb` command flag.

Complement it with a **default-log-enabled profile of at least 60 seconds total
across three seeds, four writers per seed and at least 250 acknowledged successful
mutations and identical-request replays combined**, retaining index maintenance.
Exercise backup/restore and restored replay/continued writes in that profile too.
Report unique committed effects, replays and elapsed time separately for both
profiles. The [shipping companion](evidence/v1-reliability.json) passed; the
extended profile completed **failed** with a real final-restore timeout.

Use setup creates to establish a fixed corpus, then sustain **patches, conflicting
CAS, identical-key replay, reads/search and periodic backup/restore**. Reads/search
do not count toward either mutation/replay minimum. Retain exact outcomes and
immutable-source hashes; verify receipts/replays, final inventory and continued
writes after restoration. Timed move/preview loops are not claimed. Deterministic
acceptance/crash regressions separately cover create, move, archive, apply,
preview and coordinated multi-file publication.

Repeat all **34 current deterministic leaf cases at least three times each**
(102 observed kills), including the original 11 scenarios plus multi-file
publication/recovery positions, committed roll-forward interruption and restore
positions. Retain injected publication/recovery faults and concurrency tests.
Report timeouts, skips or unobserved boundaries as failures, not successful kills.

Rationale: receipt count has no automatic expiry, retention or rotation.
Receipts store before/after full-file diff text while it fits a **256 KiB aggregate
diff-text budget per receipt** (`api.MaxReadBytes`); excess diff text is truncated.
Repeatedly extending the retained log can therefore accumulate roughly quadratic
diff storage **while the growing images fit that cap**, not indefinitely.
Serialized receipts also include revisions and other fields outside the diff
budget. Existing log/note/archive bounds still apply and rejected operations must
report explicit errors; the diff cap is not permission to bypass those bounds.

The extended profile disables logging, not receipt durability or index
maintenance. The shorter default-log profile and deterministic default-log
publication/recovery tests complement it; independent readiness review must assess
the actual observations and this bounded scope. No receipt-format/expiry change
or automatic compaction is approved. Monitor/protect event-log and receipt growth;
deleting receipts breaks replay. Neither profile proves universal crash/power-loss,
network-filesystem, hostile same-user or arbitrary-editor isolation.

### D03 — Measured local resource budgets, not a portable SLA

Use the [paired report](evidence/performance-large-corpus.json)'s Apple M4,
macOS arm64, Go 1.27.3, `GOMAXPROCS=8`, synthetic 2 KiB-note method: serialized
10k/50k baseline/candidate runs, at least five read and three repair samples,
fresh CLI children, one reused MCP process, child-only RSS, no overlapping
build/test/soak/profile work and no claimed OS cache eviction.

The local candidate ceilings below are **110% of the worse recorded
baseline/candidate p95 or RSS, rounded up to the next 10 ms or 10 MiB**.
Ten percent is explicit qualification headroom, not an empirically established
noise interval. It accommodates the disclosed trade-offs without claiming every
operation improved. MCP RSS is the whole-session peak, not per-request memory.

| Operation | 10k p95 ceiling (ms) | 10k RSS ceiling (MiB) | 50k p95 ceiling (ms) | 50k RSS ceiling (MiB) |
| --- | ---: | ---: | ---: | ---: |
| CLI search | 380 | 130 | 2870 | 550 |
| CLI read | 200 | 110 | 2070 | 470 |
| CLI validate | 190 | 120 | 1910 | 470 |
| CLI repair preview | 730 | 200 | 3770 | 830 |
| CLI committed repair | 1460 | 230 | 8940 | 1050 |
| Warm MCP search | 300 | 150 | 1530 | 640 |
| Warm MCP read | 110 | 150 | 1540 | 640 |
| Warm MCP validate | 110 | 150 | 1690 | 640 |

Report every operation's baseline/candidate p50/p95, RSS, first request/apply and
absolute/percentage drift, even inside budget. Disclose the existing 10k
search/read non-improvements, 50k repair median +4.9% and preview RSS increase.
An exceedance blocks V05 until a quiet repeat and explicit evidence-backed
review; do not silently change ceilings. Results on different hardware are
informational, not pass/fail against these machine-specific numbers.
The original **core-only 10k 150 ms warm p95 / 150 MiB / 10 s rebuild targets
remain unchanged** and need separately identified measurements. The paired
report did not measure a new rebuild improvement. No Linux timing, broader SLA
or unmeasured retained-memory improvement is claimed.

### D04 — Conditional v1.x compatibility and support

Adopt the [documented CLI/JSON/MCP commitments](compatibility.md) for v1.x only
after V06–V08 qualify them. Scope includes native pinned OKF v0.2, configuration
version 1, settled version-1 operational state/receipts and version-1 backup
archives. Qualify v0.3.1 → v0.4.0 → v1.0.0 on synthetic fixtures, preserving
unknown metadata and exact revisions; pending journals need the prior binary's
recovery first. No blanket older-version upgrade or downgrade promise.

Required native channels: **macOS arm64 Homebrew** (including formula and
installed CLI/MCP) and **Linux amd64 native archive/archived source**.
Linux arm64 and macOS amd64 remain **cross-build/package-only**, not
installed-qualified. No Windows, remote transport, network-filesystem support,
LTS lifetime, response SLA or automatic backport pledge. Rationale: commit to
interfaces and channels that can be exercised, not infer support from packaging.

## Finalization checklist

The [public snapshot](qa/v1-qualification.md) is available for independent review.
Do not wait for V08 to make factual documentation reviewable; do not infer stable
publication from that documentation.

- [x] Identify the frozen candidate source/build/executable, actual v0.4.0 release
  and exact-source CI/full-check/snapshot provenance without private paths.
- [ ] Resolve/dispose the actual shipping extended D02 restore-timeout failure
  and obtain fresh passing qualification; retain all failed attempts and current
  retrieval/default-log/interruption/resource/prior-upgrade observations.
- [ ] Attach V07 installed Homebrew/native Linux and independent consumer evidence;
  state channel/client limits and distinguish source/snapshot/installed checks.
- [ ] Record V08 reviewer identity/role, scope, findings, disposition and explicit
  readiness decision. Do not invent independent human approval.
- [ ] Reconcile README, llms.txt, spec, interfaces, installation, quickstart,
  roadmap, quality, compatibility, decision summaries and release guidance;
  preserve historical reports and update status only to observed facts.
- [ ] If stable publication is authorized, select unused v1.0.0, qualify any
  intervening changes, publish under the normal protected workflow and verify
  actual installed artifacts. Add only real release/evidence URLs, then update
  current status with observed version/commit and remaining support limits.

