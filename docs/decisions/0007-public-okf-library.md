# 0007 — Share the OKF format implementation as a public Go package

Status: implemented. The package is introduced in KB v1.2.0; earlier v1.1.0
does not contain it. Actual publication and installation evidence are tracked
separately in the [compatibility policy](../compatibility.md).

## Decision

Expose `github.com/matheus3301/kb/pkg/okf` in the existing module. Reusable
public Go packages live under `pkg/`; application-private packages remain under
`internal/`. The `pkg/` directory is an organizational convention, while Go
applies import restrictions to `internal/`.

Move the YAML syntax parser, exact metadata representation, concept checks and
presentation-preserving header editor from `internal/core` into the public package. Core and write flows
consume that implementation through the existing internal codec contract; there
is no parallel parser, public alias of an internal DTO or legacy implementation.

The public surface is `Parse`, `Marshal`, immutable `Document` accessors, explicit
`Edit`, `Validate`, structured `Error`/`Diagnostic` and format identity constants.
Pure `ValidateReserved` accepts root-index/index/log roles and implements the
existing supported upstream reserved-file checks. See the package
[usage guide](../../pkg/okf/README.md) and pinned
[implementation/conformance contract](../../pkg/okf/CONFORMANCE.md).

The dependency direction is KB adapters → `okf` → standard library + the existing
`go.yaml.in/yaml/v3` pin. No new runtime dependency or API lookup is needed for
this extraction. CLI/MCP, filesystem storage, logging backends, search,
transactions, PARA and managed-index markers remain outside the package.

## Boundaries

- Parsing checks bounded supported syntax; validation distinguishes the required
  concept type from optional upstream-family guidance. Limits/string-only mapping
  keys are implementation restrictions, not universal OKF requirements.
- Upstream reserved syntax is shared, but KB's managed marker pairs remain policy.
  This is not a complete bundle walker, Markdown/link engine or attestation runtime.
- Format editing makes only explicit changes. Core retains actor requirements,
  generated metadata refresh, verification invalidation, path checks and wire-error
  translation. `internal/write` retains publication and transaction semantics.
- Caller-created Go metadata is a new public boundary: reject invalid numbers and
  cyclic/deep/oversized trees before recursive comparison/encoding. New null keys
  are distinct from absent keys. Accessors and edits do not share mutable trees.
  A bounded 4 MiB metadata processing budget is separate from the unchanged
  1 MiB accepted-document limit. This permits the existing scalar-header versus
  final-document rejection paths without accepting a larger format language;
  core adapts early structural-limit errors to its legacy validation envelope.
- Existing CLI/MCP DTOs, codes, messages and optional-field blocking policy remain
  application contracts. Public diagnostics do not import `internal/api`.
- The package is introduced in v1.2.0. Internal Go packages remain private;
  binary/wire qualification does not establish library conformance, and the
  supported format subset is documented independently.

## Consequences and checks

A small shared format layer is reusable without KB runtime dependencies. A
consumer controls policy and safe persistence. Presentation fidelity remains
bounded: structural changes can reformat affected entries; unsupported YAML is
reported rather than silently widened during extraction.

External-package examples/regressions, parser/editor/reserved fuzz targets and
existing core/CLI/MCP fidelity tests exercise observable behavior. `make okf-check`
runs an unrelated-module consumer and checks the transitive import boundary;
`make check` includes it without reducing existing gates. Historical qualification
records remain unchanged. No third-party specification is copied wholesale;
upstream attribution and the immutable source reference are in the package contract.
