# Supported OKF v0.2 implementation contract

## Reference and attribution

The reference is GoogleCloudPlatform's **Open Knowledge Format, version 0.2**, at
commit **`ad30107c31c06aec8a7d5636e0d1058118604e6f`**:
[upstream SPEC.md](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md).
`okf.Version` and `okf.Specification` expose that identity. They do not claim
complete implementation or verify a bundle's declared version.

This document summarizes KB's implementation; it does not reproduce the upstream
standard. The upstream project is attributed to its contributors and distributed
under [Apache-2.0](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/LICENSE.md).
The Go implementation is extracted from KB's existing code under its
[MIT license](../LICENSE); no upstream source code or full specification is copied.

## Three distinct layers

| Layer | Contract |
| --- | --- |
| Upstream concept requirements (§§4,11) | UTF-8 Markdown with delimited YAML frontmatter and a nonempty string `type`. Unknown types/fields and missing optional families are tolerated. Bodies have no required sections. |
| Supported implementation boundary | One document at a time; explicit limits and the YAML subset below. `Parse` checks syntax, not `type`. `Validate` separates required concept structure (`Requirement`) from checks on optional-field conventions (`Guidance`). |
| KB application policy | Optional-field findings block KB writes; KB confines paths, derives IDs/titles/trust/PARA/staleness, updates generated actors, removes active verification on meaningful edits, checks managed marker pairs and coordinates transactions. These are not all universal OKF requirements and are not applied by the format editor. |

Passing `Parse` or having no findings is **not** full OKF conformance. Upstream
bundle conformance also includes reserved files (§11); this package has no tree
walker. No attestation, receipt execution, semantic correctness or authenticity is
established by parsing metadata.

## Supported concept syntax and values

- `---` opening delimiter at byte zero followed by LF or CRLF; closing standalone
  `---` may end at EOF. A UTF-8 BOM, leading prose, `...` closing delimiter and
  frontmatter-free concepts are not accepted. Existing KB delimiter behavior is
  unchanged; this extraction does not add YAML or Markdown language support.
- Exactly one YAML mapping document. Duplicate keys and non-string keys anywhere
  in the metadata tree are rejected. String-only keys are an implementation
  interoperability restriction inherited from KB's JSON API, not an upstream OKF
  mandate. YAML merge keys are unsupported; aliases are not a merge mechanism.
- Limits: **1 MiB** whole document, nesting depth **32**, **50,000** visited YAML
  nodes and **16** alias traversals; cycles are rejected. These are supported
  implementation bounds, not limits imposed by upstream OKF. Parse checks size
  before copying input. Caller-created metadata trees are checked before
  comparison/encoding, with matching depth/node bounds and a **4 MiB** aggregate
  string/key/number processing budget (`MaxMetadataBytes`); cycles terminate
  with a bounded error. This rendering budget does not widen accepted document
  syntax or the 1 MiB output limit. It preserves the distinction between
  scalar-header failures and final-document failures for transport adapters.
  Edit's unchanged parsed scalar values do not consume the rendering budget:
  permitted aliases can expand a detached metadata view beyond the raw file's
  size. All supplied containers still undergo depth/node/cycle checks, and
  `json.Number` values always undergo strict numeric validation.
- `Metadata()` returns detached `map[string]any` / `[]any` trees. YAML aliases are
  expanded in this view; original syntax remains available to the editor.
  Integers and recognized finite decimal YAML scalars use `json.Number`, without
  binary rounding; timestamp scalars retain their original string values.
  Implicit scalar classification remains that of the pinned YAML decoder (an
  out-of-range hexadecimal token, for example, may remain a string). No promise
  is made that arbitrary YAML tags produce JSON-compatible values; nonfinite
  floats retain the decoder's existing behavior.
- Marshal/Edit accept string-keyed maps, `[]any`, nil, strings, bools, built-in
  integer/float types and valid `json.Number` numeric literals. They reject other
  Go objects, cyclic/deep trees and nonnumeric `json.Number` literals such as
  `null`, `true`, arrays and quoted strings. Use `json.Number` for exact numeric
  input; a float already rounded by its caller cannot be recovered.
- The Markdown body is opaque. No parsing/rendering, link resolution, footnote
  attribution, heading grammar, directory access or declared computation runs.

## Implemented validation, not a complete schema

`Document.Validate` checks:

- Nonempty string `type` (`Requirement`, `MISSING_TYPE`).
- String shapes for `title`, `description`, `resource`, `status`, `stale_after`,
  `runtime`, `computation`; string-list tags; the three conventional statuses.
- `generated` actor mapping with nonempty `by` and an optional timestamp;
  `verified` mapping or list of mappings, each with nonempty `by` and required
  `at`. Singleton verification is accepted as upstream requires.
- RFC3339Nano timezone-bearing timestamps for the supported timestamp fields,
  including `sources[].last_modified`. This is narrower than all ISO 8601 forms.
- Sources as mappings with nonempty `resource`; Attested Computation's nonempty
  `runtime` and any provided executor/attester mapping's `resource`.

Optional-family checks are `Guidance`, even where a family describes required
members. They do not redefine upstream's permissive bundle conformance (§11).
Unknown fields/types are preserved. Missing optional fields produce no finding.
An absent status remains absent; the package does not materialize upstream's
suggested stable default or KB's derived trust tier.

Not validated: resource URI existence, actor identity/authenticity, all optional
source signals or usage windows, computation parameters, receipt/attester schemas,
body computation headings, per-claim footnotes, trust/freshness semantics and
bundle-level relationships. No legacy v0.1 migration is performed. Validation is
capped at 64 concept findings; nested locations identify the top-level field.

## Reserved files (§§3.1,8,9,12)

`ValidateReserved(role, raw)` takes a logical role, not a filesystem path:

- `RootIndex`: optional frontmatter may contain only a nonempty string
  `okf_version`. Unknown version strings are tolerated, not certified.
- `Index`: no frontmatter; `Log`: no frontmatter and each line beginning `## `
  must carry a valid `YYYY-MM-DD` date in newest-first order. Equal dates remain
  accepted, matching the existing implementation. Other lines are opaque.
- All roles use the same UTF-8/size bounds. No complete index-entry, grouping,
  description, log-entry or Markdown grammar is enforced. Missing index/log files
  are not errors; callers choose whether a file exists and which role it has.
- KB's `kb:index` marker syntax is deliberately not checked here. It remains an
  application publication policy layered on the shared format validation.

The reserved validator's `Requirement` kind identifies structural findings; the
size/encoding message combines an upstream encoding requirement with an explicit
implementation size restriction. It does not turn the size bound into upstream
law. Findings retain the established messages for KB's internal adapter.

## Fidelity and editing

A Document owns its bytes. Its accessors return independent data; edits do not
mutate the document or caller maps. Repeated no-ops preserve exact original bytes,
including comments, line endings and numeric spellings. Body-only edits leave the
header exact. Scalar-only metadata edits patch original tokens when possible.
Structural edits re-encode affected entries while retaining unchanged child
nodes, comments, style and anchors where representable. Unchanged block-mapping
entry spans remain exact. Editing an alias expands only that occurrence; it must
not rewrite its template. Edits leaving dangling aliases fail.

This is not a guarantee of identical whitespace inside structurally changed
entries, nor arbitrary anchor-graph rewriting. Reparse edited bytes and inspect
findings before publishing. When assembled output fails syntax validation,
Edit/Marshal may return attempted bytes alongside an error for inspection;
never publish a result with a non-nil error. Format edits do not silently invent
`generated`, clear `verified` or authorize any action. Safe publication and
revision checks remain the consumer's responsibility.

## Diagnostic contract

`Error` supports `errors.As` and exposes `Code` and explanatory `Message`.
Parsing/encoding may report `MISSING_FRONTMATTER`, `INVALID_YAML`, `INVALID_UTF8`,
`DUPLICATE_KEY`, `METADATA_KEY_POLICY`, `YAML_LIMIT`, `LIMIT_EXCEEDED`,
`INVALID_INPUT`; editing a zero Document reports `INVALID_DOCUMENT`.
These codes include unsupported inputs, not only upstream violations.

Validation returns `Diagnostic{Code, Message, Kind, Field, Line}`. Codes are
`MISSING_TYPE`, `INVALID_METADATA`, `INVALID_STATUS`, `INVALID_TAGS`,
`INVALID_TIME`, `INVALID_ACTOR`, `INVALID_SOURCES`, `INVALID_SOURCE`,
`INVALID_COMPUTATION`, and `INVALID_DOCUMENT` for an unparsed zero value.
Reserved validation uses `INVALID_RESERVED` (`INVALID_INPUT` for an unknown role).
Line zero means unknown; otherwise lines are physical and one-based.

KB translates these independent public types into its existing `kb.v1` codes,
legacy layers, hints, paths and revisions. No public type aliases an internal DTO.
Library versioning and the CLI/MCP wire version remain separate.

## Verification and availability

External-package examples and tests cover requirements/guidance, numeric fidelity,
comments/CRLF/unknown fields/anchors, null presence, invalid caller trees, bounds
and reserved roles. Fuzz targets cover parsing, metadata editing and reserved
validation. Existing core and actual CLI/MCP fidelity regressions exercise the
same parser/editor after extraction. `make okf-check` compiles/runs a consumer in
an unrelated module and rejects runtime dependencies outside the standard library
and pinned YAML package. These are scoped checks, not a full standards test suite.

The new editor fuzz target exposed a pre-existing span-boundary defect: trailing
whitespace-only content in literal/folded YAML scalars could survive a structural
replacement as if it were inter-entry trivia. Source now replaces the entire
old scalar content, including nested final-child and final-entry cases. The
focused regression and minimized synthetic fuzz seed are retained; published
v1.1.0 does not contain this correction.

This package is source-only pending a later release. Published v1.1.0 is unchanged;
no historical qualification evidence or approval is reclassified as library proof.
