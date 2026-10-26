# 0001 — Go and quality foundation before application code

Status: adopted for the repository foundation on 2026-10-24. Historical
bootstrap context below; runtime dependencies were subsequently selected in
[0002](0002-v0.1.md). Published production remains experimental v0.3.1;
integrated source is preparing v0.4.0 qualification after all four roadmap
increments. See the [roadmap](../roadmap.md) and [exit checklist](../stability.md).

The product needs one local binary, a CLI and MCP over one core, deterministic
file operations, and straightforward distribution. Use Go as the implementation
baseline, with the exact toolchain recorded in `go.mod`. The version was checked
against the official Go download manifest when this repository was created.

At foundation time, no runtime dependency was selected. Cobra and the official
MCP SDK were the leading adapter choices; parser and lexical index required
prototypes. Explicit constructors and standard library logging were preferred.
Zap was an option; Fx was not justified by the composition needs.

Bootstrap quality tooling uses Python's standard library so repository checks
can run before the first Go package exists. Go source must automatically activate
formatting, vet, lints, tests, coverage, vulnerabilities, and build checks; there
is no manually maintained switch to leave those checks disabled.

At foundation time, application acceptance criteria were planned until tests and
evidence existed. The current [acceptance inventory](../acceptance.json) records
later validation. Do not add a dummy application or test to imply readiness.

Instructions are distributed with the product in `skills/` when implemented.
The workdir establishes mandatory consultation and points to the configured
base. The knowledge bundle itself does not require an AGENTS.md.
