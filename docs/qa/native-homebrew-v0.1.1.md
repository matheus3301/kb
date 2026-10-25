# Native Homebrew QA of v0.1.1

**Decision: keep the experimental status.** The installed release passed the
tested retrieval, concurrency, recovery, and CLI/MCP scenarios, but moving a
source silently changes precise decimal metadata and removes YAML comments in
dependent notes. Fix both fidelity defects before considering graduation.

**Correction candidate:** both defects now pass the source regression checks
described under [correction verification](#unreleased-correction-verification).
The installed and published v0.1.1 still reproduces them. The historical
assessment and counts below describe that release, not the corrected candidate.

This is a post-release black-box assessment dated **2026-10-25**, not a claim
that the existing test suite or format validator proves safe preservation.
The [structured record](../evidence/native-homebrew-v0.1.1.json) includes case
results, measurements, and independent reproduction observations. No runtime
fix or release is included in this assessment.

## Installation and isolation

The product was installed once on macOS using its public distribution:

```sh
brew install matheus3301/tap/kb
kb version
HOMEBREW_DEVELOPER=1 brew test matheus3301/tap/kb
```

Installation and the formula test passed. The installed version reports
`v0.1.1`, commit `9ed4f17894f094cfdb5a169aea9c147789e93de3`.
Its locally built executable SHA-256 is
`cec0360c0c1a61f84a47caf69a0360f7270880d1c79a1d1f35347855ee663a1c`;
this identifies this installation, not a reproducible-build guarantee or the
checksum of a release archive. Homebrew installed Go 1.27.3 as a build dependency.
Developer mode remained disabled after the formula test.

Three independent QA agents owned separate, initially empty workspaces for
retrieval, editing, and concurrency. The integration owner used a fourth
workspace for MCP, scope checks, and independent reproductions. All calls used
the installed executable, not `go run`, a development binary, or mocked services.
Each knowledge base contained synthetic notes; an existing collection was never
used as a product test target. Manual fixture edits represent normal external
editor workflows. The installed binary and local QA evidence were retained.

Host: Apple M4, 16 GiB RAM, macOS 26.6.2 arm64. This assessment does not cover
native Linux installation, other hardware, power loss, or filesystem failures.
Previous release and CI evidence remains separate from this run.

## Results

| Lane | Passed checks | Failed checks | Other outcomes |
| --- | ---: | ---: | --- |
| Retrieval | 126 | 0 | 4 limitation checks, representing CJK and emoji queries in two modes |
| Editing and coordinated plans | 47 | 3 | 2 distinct fidelity defects; 129 native CLI commands |
| Concurrency and recovery | 45 | 0 | 2 explicitly untested durability/load limits; 420 CLI invocation logs |
| Integration and live MCP | 32 | 0 | Includes independent CLI/MCP data comparisons and freshness checks |

These **250 passing checks** are assertions and scenario checks, not 250 unique
user journeys. Three failed checks identify two defects; independent
reproductions confirm them without inflating the totals. The retrieval count
includes 30 result-total oracle assertions from timed calls. Timing values do
not add further checks; additional filename/title probes are outside these totals.

### Behavior that passed

- Search: isolated title/tag/description/path weighting, deterministic ranking,
  no-match results, Portuguese accents/case/decomposed Unicode, OR within filter
  lists and AND across fields, archives/deprecated opt-in, complete pagination,
  stale/changed-query cursor rejection, and external create/edit/delete freshness.
- Reads/context: physical line citations and SHA-256 revisions, section selection
  and duplicate-heading errors, token/byte budgets, explicit truncation, and a
  bounded response from a roughly 578 KiB note.
- Editing: body-only preservation of untouched YAML, UTF-8 and CRLF fixtures,
  stale-revision rejection, byte-preserving no-ops, preview purity, retries,
  successful coordinated plans and rejection before publication on invalid plans.
  Valid resolved incoming/outgoing links, anchors and source paths were updated
  by moves. Code literals remained unchanged. These passes do not cover the
  nested metadata defects below.
- Concurrency: 48 disjoint creates with up to 16 processes; exactly one winner
  among 16 same-revision patches; 16 identical-key requests with one effect;
  changed-payload key conflicts; paired plans versus 200 CLI read snapshots
  without partial pairs; and 12 moves competing for one destination with losing
  source notes preserved. These observations concern cooperating `kb` clients.
- Recovery: actual SIGKILL interruptions with a prepared journal after 0, 5 and
  40 of 50 notes were published, SIGTERM during publication, a real committed
  marker interruption, and interruption during recovery. Pending reads blocked,
  dry-runs preserved state, rollback/rollforward matched expected bytes, and
  retries did not duplicate effects. Lock-wait cancellation and automatic
  recovery before a new mutation passed. Supplemental crafted malformed,
  traversal and symlink journals were rejected while preserving sentinels.
- External editor conflict: a controlled concurrent edit was detected and its
  bytes preserved; recovery refused to overwrite divergent data. After explicitly
  reconciling that synthetic edit, rollback completed.
- CLI/MCP: root precedence/discovery, initialization preservation, matching
  search/read/context data, writes visible across interfaces, dynamic resources,
  live refresh after same-size/restored-mtime edits, read-only enforcement,
  unknown-field rejection and root confinement passed. Reads preserved Markdown
  and configuration. Malicious multiline titles were escaped in navigation;
  supported unusual filenames produced encoded links.

## Confirmed defects

### QA-01 — P1: moves silently round unknown decimal metadata

Moving `source.md` to `nested/source.md` updates dependent `sources.resource`
and `basis_revisions.path` values, but also changes unrelated numeric values:

```yaml
# Before
unknown_decimal: 1234567890.123456789
unknown_tiny: 0.1234567890123456789

# After
unknown_decimal: 1234567890.1234567
unknown_tiny: 0.12345678901234568
```

The operation returns exit 0 and `ok: true`; subsequent OKF validation remains
conformant. Both `sources` and `basis_revisions` extension fields were affected.
This is a silent change to user data, even though the fields and the Markdown
body still exist. Body-only preservation tests passing does not establish the
safety of automatic reference rewrites.

Read-only diagnosis points to scalar conversion through ordinary decoded YAML
values in [the codec](../../internal/core/codec.go), followed by replacement of
whole metadata collections during [reference rewriting](../../internal/write/links.go).
The observed output is consistent with binary floating-point conversion.
Acceptance criterion A03 was reopened after this assessment; earlier passing
fixtures did not cover unknown decimal precision during reference rewrites.
The corrected source now includes the required regressions and new evidence.
Fix the conversion and rewrite path; do not merely special-case these two values.

### QA-02 — P2: automatic reference rewrites discard nested YAML comments

The same move removes source/basis annotations and inline comments in the
dependent note. Unrelated quote styles and mapping order also change. The
comment loss violates the explicit preservation contract; changing presentation
is secondary to losing human annotations. Valid updated links and successful
format validation do not establish fidelity.

Update only the relevant scalar nodes/spans in the original YAML tree. Preserve
unknown children, exact numeric meaning, comments and unaffected presentation.
Cover both incoming dependency rewrites and changes caused by relocating a note.

### Reproduce both defects

From a checkout, with the installed `kb` on `PATH`:

```sh
python3 scripts/qa/reproduce_move_fidelity.py
```

The probe creates a disposable base, writes two synthetic notes, computes the
actual source revision, performs a real move, and prints before/after data.
It removes only its temporary directory. **Exit 1 on v0.1.1 is the expected
observed preservation failure**, not a successful quality gate. Exit 0 means
all included preservation and path-update checks pass; exit 2 means setup failed.
`KB_BIN` can select a different installed executable for a later comparison.

When using the affected release, avoid automated moves or archives of linked notes whose dependent
metadata contains precise decimals or important comments. Review all affected
files in previews and retain backups. Format validation alone will not detect
these losses. Reverting a filename alone does not restore lost metadata.

## User-visible performance

Scale fixtures contain exactly **1,917 UTF-8 bytes per note**, created directly
as compliant Markdown after initialization. Setup is excluded. `retrievalprobe`
matches every note; `unique00042` matches one; `no-match-nonce-zxqv` matches none.
All timed responses were checked for the expected total.

```sh
kb --root ./benchmark search retrievalprobe --limit 10 --json
```

| Notes | First CLI call ms | CLI subsequent p50/p95 ms | Live MCP p50/p95 ms |
| ---: | ---: | ---: | ---: |
| 100 | 13.77 | 13.51 / 13.72 | 4.38 / 5.34 |
| 1,000 | 81.53 | 81.88 / 87.63 | 36.82 / 38.63 |
| 5,000 | 410.24 | 389.57 / 400.67 | 190.54 / 202.51 |

CLI uses one first call and seven subsequent **fresh processes** per size; MCP
uses one separately recorded first search and 20 subsequent calls on one server.
Both request limit 10. p95 uses nearest rank; with seven CLI samples it is the
maximum. First call does not mean a cold OS disk cache. MCP initialization and
its first search are recorded separately in the structured evidence.

At 5,000 notes, rare/no-match CLI medians were 391.99/389.15 ms. Similar costs
suggest inventory work dominates this workload; that is an inference, not a
profile. Keeping MCP alive improves latency but does not remove the scaling
cost. The lanes shared a host, samples are small, and MCP measurements followed
the other lanes; comparisons are indicative rather than controlled benchmarks.
This run did not measure peak RSS or independently time reindexing. The earlier
[10,000-note core benchmark](../performance.md) and its unmet 150 ms/150 MiB
targets remain separate evidence; a fresh-process CLI has no documented 150 ms
service guarantee.

## Next-version work, in priority order

| Priority | Work | Completion evidence |
| --- | --- | --- |
| P1 | Fix decimal fidelity across YAML parsing and automatic move/archive reference rewrites. | This probe passes against the installed candidate; add decimal/exponent/unknown nested scalar regressions and CLI/MCP parity. |
| P2 | Preserve source/basis comments and unaffected YAML presentation during rewrites. | Byte-level regressions for nested/inline comments, quote styles, anchors/aliases, and combinations of moves in one plan. |
| P2 | Profile and improve real CLI/MCP scaling without weakening freshness or CAS. | Reproducible 100/1,000/5,000/10,000-note cold-process/live-server measurements, p50/p95, RSS, reindex and write costs; keep existing targets visible. |
| P3 | Clarify lexical and literal search behavior. | Document that `東京` within unsegmented `東京で知識を検索する。` and emoji-only `🚀` return no hits in both modes, or implement explicitly tested alternatives. These are currently limitations, not promised substring/segmentation features. |
| P3 | Improve context readability and match explanation. | Always separate excerpt/citation blocks when a source lacks a final newline; show why metadata-only matches were selected. Citation metadata was correct in this run. |
| P3 | Make installed-binary QA repeatable across release platforms. | Retain a small synthetic corpus and crash/concurrency harness; exercise actual packaged binaries and formula installation in addition to development builds. |

For a v0.1.2 candidate, first fix QA-01 and QA-02 with regressions, run the full
existing gates, install the candidate through the intended distribution path,
and rerun the fidelity, transaction, retrieval and MCP workflows. A graduation
decision should also establish scale budgets with measurements, test installed
packages on supported platforms, and document backup/recovery operation. Green
CI alone does not satisfy these requirements.

## Unreleased correction verification

The [source correction record](../evidence/move-fidelity-fix.json) captures
before/after output from the same disposable probe. It exits 1 against installed
v0.1.1 and 0 against a binary built from the corrected source. No Homebrew binary
was replaced and no new release was published during this fix.

Finite decimal YAML scalars are converted directly to exact JSON numbers,
including exponent forms, without a binary floating-point intermediate. Path
changes replace scalar spans when collection shape is unchanged, retaining the
original numeric lexemes, quote styles, comments, indentation and line endings.
Structural edits retain existing YAML nodes; editing an alias expands the edited
value without modifying its original template or duplicating anchors.

Regression coverage includes CLI and MCP against real providers with block,
flow, CRLF and alias fixtures; moving a synthesis, coordinated moves and archive;
unknown nested decimals, scientific notation and comments; structural metadata
edits; collection, adjacent-key and document comment ownership without duplicate
annotations; and a new bounded metadata-patch fuzz target. CLI acceptance also checks
the original failure pattern against a real built executable. `make check`
passed with eight fuzz targets and 88.6% application statement coverage.

A03 is verified again for these tested source cases. The two move fidelity
findings are resolved in source. For packaged corrections, use v0.1.2 or later
when available in the published releases. This verification preceded publication;
performance priorities and the broader graduation requirements remain separate.

## Limits of this assessment

Process termination is not power loss or a failed `fsync`. Sampled schedules
cannot prove starvation freedom, exhaustive race coverage or universal
durability. Crafted journal cases supplement actual interruptions and are
identified separately. This run uses uniform synthetic notes on one host, not
a representative multilingual relevance benchmark or semantic consistency
audit. External applications do not participate in `kb` advisory locking and may
observe transient multi-file states. The two defects were reproduced by both a
QA agent and the integration owner; no claim of additional unobserved data loss
or security exploitation is made.
