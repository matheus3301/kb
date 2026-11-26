# Draft: OKF v0.3 compatibility and additive root-index metadata

Status: **draft first-PR proposal with a narrowly implemented candidate**. No release,
merge, general v0.3 compatibility claim or migration is authorized by this document.
CLI plus the knowledge-base skill remains primary; MCP is independently usable.
All examples and regression inputs are synthetic.

## References and baseline

- Product baseline: KB **v1.2.2**, commit
  `e89485259a3c936da179dae1286bacf701a307cc`.
- Primary normative source: [OKF v0.3 SPEC.md, immutable commit
  `527e81aff55992be21507e97004286789126f88c`](https://github.com/open-knowledge-format/open-knowledge-format/blob/527e81aff55992be21507e97004286789126f88c/SPEC.md).
  Sections below refer to this pin, not mutable upstream main.
- Baseline implementation reference: [OKF v0.2 SPEC.md,
  `ad30107c31c06aec8a7d5636e0d1058118604e6f`](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md).
- Local contracts: [product specification](../spec.md), [quality gates](../quality.md),
  [A01–A26 acceptance](../acceptance.json), [public package boundary](../../pkg/okf/CONFORMANCE.md),
  [format-library decision](../decisions/0007-public-okf-library.md).

The installed v1.2.2 executable rejected a synthetic declared-v0.3 index
with four `INVALID_RESERVED` findings. Baseline `pkg/okf/reserved.go` rejects every root metadata key
except `okf_version`, even when that value is `"0.3"`. The core codec forwards
these as `INVALID_RESERVED` conformance findings. `write.Service.conform` then
blocks normal writes; export excludes invalid paths. This explains both write
rejection and omission of the root index from a degraded export. Historical
acceptance evidence is not relabeled as v0.3 qualification.

## Candidate contract: one additive slice

Only `pkg/okf.ValidateReserved` changes production behavior. No global switch,
new dependency, DTO, filesystem permission or alternate parser is introduced.

| Root declaration | Candidate behavior |
| --- | --- |
| No frontmatter | Accepted, as before. |
| Frontmatter without `okf_version` | Existing v0.2 metadata restrictions remain. Optional v0.3 bundle fields do not opt a file into v0.3. |
| `okf_version: "0.2"` or another nonempty string other than exactly `"0.3"` | Only `okf_version` permitted, as before. Unknown versions alone remain accepted best-effort, not certified. No numerical comparison or future-version inheritance. |
| `okf_version: "0.3"` | Also accept optional string `title`, `description`, `license`, and all other keys as opaque bundle metadata, including unprefixed extensions. |
| Numeric, null, empty or container `okf_version` | Existing string validation diagnostic. No numeric coercion. |

For the three optional v0.3 fields, null, empty strings and empty lists mean
absent (§11.2). Other non-string shapes receive deterministic
`INVALID_RESERVED` findings in source order; they remain blocking in KB. This
is a supported-schema restriction, **not a claim that malformed optional values
make an upstream v0.3 bundle nonconformant**. Full permissive consumer semantics
are deferred. Existing YAML duplicate-key, cycle, depth, size, UTF-8 and
string-key restrictions still apply recursively to extensions. Recognizing
`x-` is not an allowlist: §4.1 permits unknown keys without that prefix.

Non-root indexes and all logs still reject frontmatter. Reserved files never
become concepts, even if their metadata includes `type` or `verified`. Bundle
metadata is not inherited by notes. License values are opaque strings: no SPDX
expression checker, path resolution, file read, network fetch or code execution.
Unknown values and comments are not normalized by validation.

`okf.Version` and `okf.Specification` remain v0.2. Initialization still creates a
v0.2 declaration. A v0.3 declaration enables **only this root metadata slice**;
it does not select v0.3 concept, path, trust, date or conformance semantics.
Unknown-version handling deliberately retains the baseline's narrower metadata
policy instead of guessing an unknown schema. Future widening needs an explicit
contract and regression cases.

### Preservation and interfaces

- `write.Service.indexes` edits the managed body region, preserving the original
  root frontmatter bytes. The candidate needs no new serializer or index writer.
  Preview, unmanaged-index adoption, subsequent normal creates, LF/CRLF headers,
  comments, numeric spellings, nested extensions and aliases are exercised.
- `Store.Export` copies accepted scoped Markdown bytes. The root index is no
  longer excluded solely for its valid declared-v0.3 metadata. The manifest still
  reports scoped findings/exclusions, **not full upstream conformance**.
- Export remains Markdown-only. `license: /LICENSE.txt` can point at an excluded
  asset; asset shipping and complete exchange bundles are deferred. Do not use
  KB export's `conformant` flag as proof of a self-contained v0.3 exchange bundle.
- CLI `validate`, `create`, `patch`/normal write validation and MCP tools share the
  codec. MCP has no maintain/export tools: no parity is claimed for nonexistent
  tools. CLI `maintain`/`export` are tested directly; MCP `kb_validate`, `kb_create`
  and `kb_read` exercise the independent protocol on the same roots.
- The CLI skill needs no new invocation syntax or dependency. Its existing
  maintain/write workflow now works with this declared metadata subset. No
  setup, agent configuration or skill activation is performed during checks.

## Compatibility matrix against actual implementation

“Static” rows below are source observations/inferences, not new executable
interoperability results. “Deferred” means intentionally unchanged in this PR.

| v0.3 rule or change | Actual KB v0.2 implementation and risk | First PR / next evidence |
| --- | --- | --- |
| Root bundle title/description/license, extensions; preserve regenerated header (§8.1) | Baseline `ValidateReserved` rejects additional keys. `navigation.go` already preserves frontmatter; `maintenance.go` exports accepted bytes. | Implement declared-only additive validation; public-package, core caller and CLI/MCP fidelity regressions. |
| Unknown keys at every level, `x-` reservation (§4.1) | Concept parser/editor preserves unknown keys and numeric spelling. Baseline root index is exceptional. | Remove that exception only for declared v0.3; no rejection merely for an unprefixed extension. |
| Nonconcept `.md` files do not harm conformance (§3.2, §11.1) | KB diagnoses missing frontmatter/type and blocks normal writes (A04). A declared-v0.3 index currently does not change that. | Deferred: requires inventory/status/search/write-policy redesign, not silently weakening A04. |
| Non-Markdown assets and dot-path exclusion (§3.2) | KB indexes scoped Markdown; selected operational paths are excluded by application config. Export excludes assets. No claim of every v0.3 dot-path rule. | Deferred: asset inventory/export and a scoped policy matrix. |
| YAML 1.2 core, exact delimiters, duplicates, bounded aliases (§4) | Public parser uses the pinned YAML decoder and explicit interoperability/complexity limits; timestamps have decoder-specific scalar classification. | Existing syntax remains. Test duplicate/cyclic extension rejection; do not claim arbitrary YAML 1.2 conformance. |
| CommonMark + GFM tables + footnotes; claim/source IDs (§4.2, §5.1) | Bodies are opaque in `okf`; KB has a bounded Markdown/link scanner, not a complete dialect or footnote attribution verifier. | Deferred; no new renderer or attribution claim. |
| Date-only timestamps mean UTC midnight (§5) | `Document.Validate` requires RFC3339Nano timezone-bearing instants; core freshness logic follows that contract. Native timestamp parsing does not implement v0.3 date-only semantics. | Deferred: timestamp normalization plus read/search/filter/write and API cases. |
| Only current verification contributes trust (§5.3) | Core derives trust from `verified.by` without comparing `verified.at` to `generated.at`; meaningful KB edits remove active verification. A stale external human event can still yield human-reviewed. | Deferred, high priority: needs version-aware or explicitly revised trust policy and all caller assertions. No trust upgrade from root metadata. |
| Actor schemes including `team:` (§7) | Core recognizes only `human:` as human; other actors are non-human. Actor validation is not authentication. | Static overlap, not a new universal trust guarantee. |
| `superseded_by` and retire/move guidance (§5.4) | Unknown field survives, but graph/navigation does not follow it as a standardized replacement edge; move changes paths and repairs supported links rather than leaving deprecated stubs. | Deferred: do not change existing move contract. |
| Unknown status excluded from current-knowledge filters; malformed stale value never fresh (§5.4–5.5) | Optional validation blocks malformed values for writes; derivation and filter semantics remain KB's existing contract, not selected by declaration. | Deferred: explicit raw-status/freshness behavior and malformed/external-edit cases. |
| Link and path-field resolution from containing file, root form, URI escaping (§6) | KB's `resolveLink` handles relative/root links and percent decoding. Graph supports selected provenance fields and KB `basis_revisions`. There is no complete v0.3 path-field contract or version-selected fallback. | Deferred: do not redefine v0.2 binding. Need nested ambiguity/fallback/encoded escape tests before changing graph/move semantics. |
| Out-of-root and symlink safety (§3.2, §6) | KB is stricter: file operations reject symlinks; links are data, not permission to read. | Retain. Add root metadata containing outside license references and CLI/MCP traversal/symlink rejection witnesses. |
| Broken links do not invalidate bundle; missing indexes accepted (§6.1, §11) | OKF/health layers are separate; broken links are health advisories; normal writes may maintain missing indexes. | Existing behavior retained. No automatic external read from a broken target. |
| Absent/null/empty and singleton family forms (§11.2) | Singleton `verified` accepted; scalar tags/sources and many null optional concept values are diagnosed by the narrower KB policy. | Only optional root fields' absent/null/empty forms in this PR; general family coercion deferred. |
| Attested computation parameters, inline locator, verdict and runtime (§10) | Validation checks a subset; KB never executes computations or attesters or proves runtime attestation. | Deferred; metadata validation cannot establish attestation or semantic correctness. |
| Optional index/log structures (§8–9) | Pure reserved validator checks placement/log dates; KB adds managed-marker policy. Neither validates the full index-entry grammar. | Unchanged outside metadata; root YAML strings containing marker text must not become managed-body markers. |
| Best-effort unknown versions; pre-1.0 breaking minors (§12–13) | Unknown string alone accepted; no version-dispatched interpretation. | Preserve exact behavior with an explicit narrower metadata contract. Full best-effort future schema interpretation untested/deferred. |

## Security and review boundaries

Metadata is untrusted data. A root `verified: {by: 'human:synthetic'}` is not a
review, does not grant write authority and cannot change a concept's trust.
Likewise a description containing an instruction, HTML, Markdown links or managed
marker strings is not executable. Existing note-title/description escaping and
percent-encoded filenames remain in generated navigation; bundle titles are not
newly rendered into it. Validation does not follow a `license` or extension path.
Actual file reads retain traversal/symlink rejection through both interfaces.

No new parser loops over extension values after parsing or materializes a detached
metadata map just to validate known root keys. Existing parser bounds apply.
This PR is not a transaction, filesystem or sandbox redesign. Existing confinement,
revision checks, journals, recovery and idempotency remain necessary.

## Reproducible verification

Run from a public source checkout with its pinned Go toolchain and Python. Roots,
synthetic home and exports are temporary and removed by the harness. `KB_ROOT` is
cleared; every product call binds an explicit root. No private KB, credentials,
agent harness or global instruction files are needed.

```sh
gofmt -l pkg/okf/reserved.go pkg/okf/reserved_test.go cmd/kb/okf_v03_test.go
go test ./pkg/okf -count=1
go test ./internal/core -run 'TestReservedValidation|TestCodecPatchPreservesSpansNoOpAndTrust|TestRebuildExportNoOverwriteAndSourcePreservation' -count=1
go test ./internal/write -run 'TestIndex|TestMalformedStandaloneIndexMarkers|TestMaintain|TestMalformedManagedIndex|TestNestedReservedLog|TestReservedNames' -count=1
go test ./cmd/kb -run 'TestV03RootMetadata|TestRealProviderCLIAndMCPParity|TestRealProviderWriteReplayParity|TestHealthChecksLocalAssetPresenceWithoutIndexingOrFollowingSymlinks' -count=1
make okf-check
python3 -m unittest discover -s scripts -p test_okf.py
go test ./pkg/okf -run '^$' -fuzz '^FuzzReserved$' -fuzztime=1s -parallel=2
CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local go build -trimpath -o .build/kb-okf-v03 ./cmd/kb
python3 scripts/check_okf_v03.py --binary .build/kb-okf-v03
python3 scripts/quality.py repo
```

The public bundle tooling exercised here is KB's public-package consumer gate and
actual CLI maintain/export/manifest paths. The real stdio MCP process uses
independent initialization and `tools/call`, not the CLI skill or a mock server.
The Go parity tests additionally compare envelopes in process. The harness prints
only sanitized counts, binary hash and elapsed time, not temporary roots or content
from any operator collection.

### Evidence state

- **Measured on 2026-11-26:** Go 1.27.4, Python 3.14.4, macOS arm64, source-built
  candidate. The listed Go checks passed: public package **0.425 s**, selected core
  callers **0.370 s**, write/navigation callers **2.401 s**, CLI/MCP callers
  **2.625 s**. `make okf-check` passed its unrelated-module consumer and dependency
  boundary; the three `test_okf.py` tooling regressions passed in **0.024 s**.
  Reserved-file fuzz smoke passed **33,454 executions**, two workers, requested
  one-second duration (Go reported **2.314 s** including startup/cleanup).
  These are single-run check durations, not latency benchmarks or SLAs.
- **Measured public executable paths:** `check_okf_v03.py` passed **4 synthetic
  fixtures** (CLI/MCP writes × LF/CRLF), **36 validation parity comparisons**,
  **4 write flows**, **4 exact exports** and **8 escape-read rejections** in
  **1,238.482 ms** total. Header/manual/unchanged-note fidelity, manifest scope,
  escaping, invalid-base write blocking and non-inherited trust are asserted.
  The source-built executable SHA-256 was
  `ecede23f4a810edebbb953607517624147f9e1cad66c90419f4cc9ce65263937`.
  These cases exercise the KB executable through CLI and stdio MCP.
- **Measured upstream validator round trip:** the pinned
  [upstream validator](https://github.com/open-knowledge-format/open-knowledge-format/blob/527e81aff55992be21507e97004286789126f88c/skills/okf-generate/scripts/validate.py)
  accepted a separately authored synthetic fixture with three Markdown files,
  `okf_version: "0.3"`, title, description, an SPDX license identifier and a
  nested extension. KB v1.2.2 rejected its index with four findings. The candidate
  reported zero findings, preserved the root YAML bytes through `maintain --apply`,
  and copied the resulting index byte for byte into the export. The same upstream
  validator reported zero errors and zero warnings for the input and export
  (three files checked in each).
- **Inferred from source:** the matrix's baseline and deferred behavior; forwarding
  of the shared validator into unmodified write/maintenance/export callers.
- **Still untested:** bundles from third-party producers, upstream worked-example
  conformance, broader v0.3 semantics, installed release artifacts, foreign
  operating systems/architectures, universal agent activation and arbitrary
  filesystem interruption.

Targeted checks do not replace the repository's required CI `make check` gate.
This first PR deliberately does not run broad performance/reliability qualification
or change A01–A26's existing statuses. Review as a source candidate only.
