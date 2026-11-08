// Package okf parses, inspects, validates and edits the supported subset of
// Open Knowledge Format v0.2 concept documents and validates reserved index/log
// structure. Specification pins the upstream source. Parsing is not proof of
// full OKF conformance: call Document.Validate for concept requirements and
// optional-field guidance, and ValidateReserved for supported reserved checks.
// There is no bundle walker, Markdown renderer/link resolver or attestation
// runtime. Bodies and metadata are untrusted data, never executable authority.
//
// Parse requires UTF-8 and delimited YAML mapping frontmatter, with explicit
// size/complexity and string-key interoperability bounds. Those bounds are
// implementation restrictions, not universal OKF requirements. Unknown types
// and metadata are preserved. Metadata returns detached trees; Edit and Marshal
// accept bounded caller-created trees, rejecting cycles/excess depth and invalid
// json.Number literals before recursive encoding. Callers must not concurrently
// mutate trees passed to these functions.
//
// Document.Edit preserves unchanged presentation and exact finite numeric
// values. It does not implement KB's actor/verification lifecycle, filesystem
// confinement, PARA policy, managed navigation, search or transactions. A
// consumer decides whether optional Guidance findings block its operations.
// Package errors and diagnostics have no dependency on internal KB DTOs.
//
// This package is introduced in KB v1.2.0; earlier KB v1.1.0 does not contain it.
// See the package README and CONFORMANCE.md for availability,
// supported syntax, diagnostic codes and upstream attribution.
package okf
