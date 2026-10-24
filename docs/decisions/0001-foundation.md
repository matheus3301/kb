# 0001 — Go and quality foundation before application code

Status: adopted for the repository foundation.

The product needs one local binary, a CLI and MCP over one core, deterministic
file operations, and straightforward distribution. Use Go as the implementation
baseline, with the exact toolchain recorded in `go.mod`. The version was checked
against the official Go download manifest when this repository was created.

No runtime dependency is selected yet. Cobra and the official MCP SDK are the
leading adapter choices; parser and lexical index require prototypes. Use
explicit constructors and standard library logging initially. Zap is an option;
Fx is not justified by the current composition needs.

Bootstrap quality tooling uses Python's standard library so repository checks
can run before the first Go package exists. Go source must automatically activate
formatting, vet, lints, tests, coverage, vulnerabilities, and build checks; there
is no manually maintained switch to leave those checks disabled.

Application acceptance criteria are tracked as planned until tests and evidence
exist. Do not add a dummy application or test to imply product readiness.

Instructions are distributed with the product in `skills/` when implemented.
The workdir establishes mandatory consultation and points to the configured
base. The knowledge bundle itself does not require an AGENTS.md.
