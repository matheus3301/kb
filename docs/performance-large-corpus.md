# Large-corpus allocation reduction

This increment follows v0.3.1 without changing lexical ranking, query scope,
format policy, source bytes, revision identity or publication durability.
It streams the existing canonical generation records into SHA-256 instead of
materializing a growing inventory-sized byte buffer and allocating a revision
string for every file. Snapshot construction reserves note storage once and
sizes the previous-note lookup from the previous valid inventory.

The canonical record remains `path + NUL + sha256:<hex> + LF`, sorted by path.
Frozen digest fixtures cover empty inventories, multiple paths, Unicode paths,
line endings, empty files and arbitrary source bytes. Existing generations,
CAS comparisons, pagination and receipts retain their byte-level identity.
Every relevant operation still reads all scoped current file bytes, including
same-size/same-mtime changes. No timestamp cache, daemon, database or shared
mutable source/body storage is introduced. Journal, receipt, fsync, confinement
and recovery code is unchanged.

## Quiet paired CLI/MCP measurements

The [complete report](evidence/performance-large-corpus.json) records a quiet
paired run on Apple M4, 10 logical CPUs, macOS 25.6/arm64 and Go 1.27.3.
Both binaries used `GOMAXPROCS=8`: installed v0.3.1 at `2ca104f`, followed by
the candidate built with `go build -trimpath`. The report fingerprints the two
changed runtime files; the candidate reported `dev/unknown`, not a release.

The whole matrix held one shared execution slot without concurrent builds,
tests or profiles. Both binaries used identical harness arguments and separately
generated identical synthetic corpora, five samples per read operation and three
per repair operation. Fresh CLI process RSS comes from `wait4` for that child,
excluding Python and corpus creation. MCP uses one reused stdio process per
corpus; first-request latency is separate from the five warm samples, and its
RSS covers the entire session. OS page caches were not evicted. Percentiles use
nearest rank; a five-sample p95 is not a portable tail-latency guarantee.

For **50,000 notes of 2 KiB**:

| Operation | v0.3.1 p50 / p95 | Candidate p50 / p95 |
| --- | --- | --- |
| CLI search | 2502.4 / 2604.3 ms | 2217.4 / 2460.4 ms |
| CLI read | 1735.9 / 1872.8 ms | 1419.7 / 1437.7 ms |
| CLI validate | 1717.6 / 1728.2 ms | 1428.6 / 1671.2 ms |
| Warm MCP search | 1374.4 / 1385.2 ms | 1093.5 / 1172.6 ms |
| CLI repair preview | 3337.5 / 3426.8 ms | 3233.9 / 3238.7 ms |
| CLI committed repair | 6097.4 / 8119.1 ms | 6397.6 / 7774.7 ms |

50k CLI search peak RSS fell from 499.1 to 488.4 MiB, read from 425.5 to
396.1 MiB, and committed repair from 954.4 to 844.2 MiB. Preview RSS increased
from 745.8 to 751.4 MiB; reused MCP session RSS fell from 579.2 to 570.9 MiB.
Committed repair includes journal/receipt/fsync publication. First apply,
which creates managed indexes, took 8119.1 versus 7774.7 ms.

At **10,000 notes**, CLI search medians were 320.6 versus 327.4 ms, read
170.7 versus 171.9 ms, and warm MCP search 98.4 versus 99.4 ms. Search RSS was
114.0 versus 115.2 MiB. Preview was 497.9 versus 500.8 ms; committed repair was
1205.3 versus 873.0 ms, with first applies 1205.3 versus 1201.0 ms.

This is a modest allocation/startup improvement at 50k, not a uniform speedup:
10k read/search did not improve, some RSS values increased, and 50k committed
repair median was about 4.9% slower. One baseline-then-candidate matrix cannot
establish statistical significance or eliminate order/filesystem variance.
All operations remain materially expensive at 50k. Historical v0.3.1 numbers
in [the earlier report](performance.md) used a different run; they are not
substituted for this paired baseline.

Reproduce with the existing harness, serially and without overlapping workloads:

```sh
GOMAXPROCS=8 python3 scripts/benchmark.py --binary ./baseline-kb --sizes 10000 50000 --samples 5 --repair-samples 3 --output baseline.json
GOMAXPROCS=8 python3 scripts/benchmark.py --binary .build/kb --sizes 10000 50000 --samples 5 --repair-samples 3 --output candidate.json
```


## Profiling

Source-level exploratory profiles used synthetic 10,000/50,000-note collections
from `scripts/benchmark.py`, exactly 2,048 bytes per note, on Apple M4/macOS
arm64 with Go 1.27.3 and `GOMAXPROCS=8`. A separate process generated the corpus.
The probe opened a fresh store, searched `quartzbeam recovery`, forced GC while
keeping the store live, and read `runtime.MemStats`. Each value below is **one
profiling sample**, including test/runtime overhead; these are not quiet CLI
latencies or child-process RSS and are not a confidence interval.

| Notes | v0.3.1 retained heap | Candidate retained heap | v0.3.1 total allocated | Candidate total allocated |
| --- | --- | --- | --- | --- |
| 10,000 | 54.165 MiB | 53.657 MiB | 208.949 MiB | 193.007 MiB |
| 50,000 | 264.579 MiB | 261.990 MiB | 1054.114 MiB | 966.583 MiB |

The 50k retained-heap change is only about 1%; the allocation reduction is about
8.3%. CPU samples were dominated by confined filesystem syscalls. Heap profiles
showed independently owned source bytes and parsed body strings dominating
retained space. Raw profiles and local profiling scaffolding are not public
fixtures. The probe was removed after measurement.

A separate five-iteration digest microbenchmark compared the prior materialized
algorithm with the candidate under `GOMAXPROCS=8`. At 10k, allocations fell from
7,484,280 to 164,080 bytes/op (30,035 to five allocations), with 10.54 versus
8.03 ms/op. At 50k, 37,709,688 to 803,056 bytes/op (150,042 to five), with
46.39 versus 42.84 ms/op. Fixture construction is outside benchmark timing and
allocation counters. These are digest-only measurements, not complete inventory
or transaction latency. The candidate benchmark remains in
`internal/api/generation_test.go`; the throwaway prior-algorithm probe was removed.


## Remaining design limit

Exact inventory reads remain proportional to total scoped bytes and file count.
The current public snapshot model carries both mutable source byte slices and
body strings. Aliasing them with unsafe conversions would jeopardize image
isolation and is not an acceptable memory shortcut. Reserving note capacity uses
all scoped files as an upper bound; invalid/reserved-file-heavy collections can
reserve more note slots than they use. The measured corpora are conformant,
not evidence for lower memory on every distribution of malformed files.

A larger retained-memory reduction needs a separately reviewed internal immutable
inventory representation: own each source once, retain validated body offsets,
and materialize public body strings only at the API boundary. Search could scan
an ephemeral body string while building its generation-bound index, rather than
retaining that string with every cached note. Such a change must migrate all
read/context/graph/write consumers together, preserve physical citations and
snapshot ownership, and prove same-byte freshness and concurrent snapshot
isolation before adoption. It is a design proposal, not implemented behavior.

## Safety verification

`make check` passed with 89.3% aggregate statement coverage, including the
existing race, fuzz, confinement, source-fidelity, CAS, idempotency, recovery,
real CLI acceptance, documentation examples and CLI/MCP benchmark smoke gates.
The new frozen canonical-digest cases passed alongside the existing freshness
and deterministic inventory-error regressions.

An additional actual-process smoke compared installed v0.3.1 and candidate CLI
and reused MCP sessions on the same disposable 100-note root. Generations,
ordered hits, scores, citations and revisions matched before and after an
external same-size edit with restored nanosecond mtime. Queries for the removed
and replacement tokens returned the expected 99/1 split; unmatched queries
returned zero. All query operations preserved filesystem bytes. This is bounded
observed parity, not universal external-editor isolation or semantic correctness.
