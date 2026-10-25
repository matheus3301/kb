# Verification footer regression after v0.1.2

Historical v0.1.2/pre-v0.1.3 report from 2026-10-25. The correction subsequently
shipped; [v0.3.1 release verification](https://github.com/matheus3301/kb/releases/tag/v0.3.1)
records later macOS arm64 installed-package results. The source-candidate counts
and publication instructions below preserve the state at the time of this report,
not a current pending-release claim. See [installation](../installation.md) for
the affected releases and [stability](../stability.md) for current readiness.

Post-publication QA of the real Homebrew v0.1.2 package found a regression:
body edits and metadata patches that invalidate `verified` can remove a
standalone YAML comment immediately after that field. The original precision
and nested provenance findings are fixed, but this separate case was missing
from the newly added regressions.

The unchanged original editing harness caught three failures among 50 checks:
body-only presentation, metadata presentation, and a CRLF edit all lost the
same trailing annotation. Retrieval passed 126 checks with four documented
query limitations, concurrency/recovery passed 45 with two untested limits,
and CLI/MCP integration passed 32. Do not represent this run as a clean release
validation merely because its total passing count matched an earlier run.

## Cause and correction

The YAML parser assigns adjacent trailing comments to the preceding mapping
key. Structural editing retained that ownership to avoid duplicate comments,
but removing the verification field then consumed its foot comment as well.
The correction retains the original standalone comment span when removing the
field. Active verification is still removed and the write history still records
the substantive change; retaining an inert annotation does not preserve a
verification claim.

The parser can also attach a final CRLF comment to the document with a terminal
newline. Comment-boundary calculation now trims that terminal separator before
counting lines, avoiding consumption of a preceding metadata line.

Tests cover adjacent and separated comments, multiple annotations, a following
metadata field, indentation and LF/CRLF. The binary acceptance workflow seeds
synthetic verification and asserts that it is cleared without losing its footer.
The [disposable fidelity probe](../../scripts/qa/reproduce_move_fidelity.py)
now includes this case as well as the original precision/provenance cases.

The [source-candidate verification](../evidence/verification-footer-fix.json)
records 253 passing original harness checks with zero failures: retrieval 126,
editing 50, concurrency/recovery 45 and CLI/MCP integration 32. Four query
limitation checks and two untested durability/load limits remain explicit.
The expanded probe passes all 11 checks on the candidate and fails on installed
v0.1.2. `make check` passed with eight fuzz targets and 88.6% statement coverage.
These are pre-publication source results, not installed-v0.1.3 evidence.

## Publication handling

v0.1.2 and its assets remain unchanged: that version is already public and
recorded by the Go module proxy/checksum database. Its GitHub release notes
carry a warning. The correction is intended for v0.1.3, after source-candidate
QA, the normal CI/main checks, and a fresh installed-package QA run. Do not claim
the new release exists or passed those checks until they actually finish.

Installed-package QA must repeat the full original harnesses in fresh workspaces
as well as any new focused probes. Keep the failing release evidence distinct
from source-candidate and installed-correction evidence. Coverage, green CI and
one passing focused reproducer do not replace those user workflows.
