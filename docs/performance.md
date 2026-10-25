# Performance measurements

The performance work shipped in experimental
[v0.3.1](https://github.com/matheus3301/kb/releases/tag/v0.3.1) on 2026-10-25,
optimizing v0.3.0 without changing ranking, format policy, revision checks or
durable publication. Markdown remains authoritative; there is no new daemon,
database, network dependency or timestamp-only cache. The paired report below
retains its original **candidate** labels and pre-release binary identity; the
release notes separately record subsequent macOS arm64 installed-package proof.
The [roadmap](roadmap.md) and [exit checklist](stability.md) keep additional
large-corpus work and v1.0 readiness pending.

## Paired CLI and MCP measurements

Measured on Apple M4/macOS arm64, 10 logical CPUs, Go 1.27.3, on 2026-10-25.
Baseline is the installed Homebrew v0.3.0; candidate was the pre-release
performance branch subsequently shipped as v0.3.1. Runs were serialized baseline
then candidate using the same synthetic corpora and script. Each read has 10
samples; each repair has three. The
[machine-readable report](evidence/performance-optimization.json) includes all
1,000/10,000/50,000-note results, exact method, source fingerprint and limits.

For **10,000 notes of 2 KiB**:

| Operation | v0.3.0 p50 / p95 | Candidate p50 / p95 | Median speedup |
| --- | --- | --- | --- |
| CLI search | 895.5 / 985.1 ms | 330.1 / 331.7 ms | 2.7× |
| CLI read | 572.5 / 693.8 ms | 176.2 / 177.6 ms | 3.2× |
| CLI validate | 574.9 / 576.9 ms | 175.7 / 178.2 ms | 3.3× |
| MCP search | 445.0 / 450.8 ms | 100.2 / 101.1 ms | 4.4× |
| CLI repair preview | 6801.9 / 6888.0 ms | 508.7 / 511.5 ms | 13.4× |
| CLI repair apply | 7541.1 / 7695.5 ms | 891.1 / 1203.4 ms | 8.5× |

CLI search process peak RSS fell from 162.6 to
116.2 MiB; reused MCP session peak fell from
185.4 to 134.9 MiB.
These are end-to-end process measurements, not the historical core-only peak.

At 1,000 notes, candidate CLI search/read medians are 38.4/22.8 ms and warm
MCP search is 9.5 ms. At 50,000 notes, fresh CLI search still takes
2.19 s median (2.45 s p95), warm MCP search 1.08 s, and committed
repair 6.38 s. CLI search peaks at 483.1 MiB there. This is a substantial
improvement, not an instant large-corpus search guarantee. Exact full-file
freshness and durable writes still have real costs.


## Original targets

The unchanged core-only benchmark was rerun quietly on the same machine with
20 warm queries per binary. Its RSS includes fixture creation and calibration;
these values must not be substituted for end-to-end process RSS above.

| Core measurement | v0.3.0 | Candidate | Original target |
| --- | --- | --- | --- |
| Cold search | 620.4 ms | 331.6 ms | No cold target |
| Warm search p95 | 169.3 ms | 117.7 ms | Below 150 ms |
| Process peak RSS | 181.6 MiB | 134.0 MiB | Below 150 MiB |

The candidate meets the original warm-search and RSS targets in this local run.
A separate fresh-process `index rebuild` run over 10,000 notes, with three
samples per binary, measured 585.0/594.7 ms p50/p95 for v0.3.0 and
176.1/179.0 ms for the candidate, below the original 10-second target. Rebuilding
refreshes derived inventory state and the generation marker; it does not persist
a trusted parsed database. These are local observations, not CI latency gates.

## Reproduce

Build the candidate with the pinned Go toolchain, then run:

```sh
go build -trimpath -o .build/kb ./cmd/kb
python3 scripts/benchmark.py --binary .build/kb --output .build/performance.json
```

The harness defaults to 1,000/10,000/50,000 synthetic notes of exactly 2 KiB,
20 samples for each read operation, and three repair samples. For a comparison,
run the same script first against the released binary and then the candidate,
with identical arguments and without concurrent benchmark/build/test workloads.
The full matrix can take several minutes, especially for the baseline repair.
The compact CI invocation is `make benchmark-smoke`.

Measure the separate rebuild target with:

```sh
python3 scripts/benchmark.py --binary .build/kb --rebuild-only --sizes 10000 --samples 3 --output .build/rebuild.json
```

Each CLI sample starts a new process. Each corpus uses one reused stdio MCP
process, with its first request reported separately. File contents may be warm
in the OS page cache; the harness does not claim to evict caches or measure
power-on disk latency. CLI RSS comes from each child process, excluding Python
and corpus creation. MCP RSS is the process peak across the tested session.
Percentiles use nearest rank; small sample tails are observations, not SLAs.

Repair preview includes validation, navigation planning and response generation.
Committed repair includes durable journal/receipt publication and fsync. Its
first sample also creates missing managed indexes; later samples reuse them.
Write sample keys are unique, and the fixture restores one invalid source before
each repair. Those destructive setup steps occur only in disposable synthetic
collections. Product startup, result rendering and IPC remain inside the latency
measurement; CLI output is spooled to temporary files to avoid pipe deadlocks.

## What changed

- Avoid a second full inventory for response metadata; bind it to the result's
  generation and check the disposable marker separately.
- Read fresh bytes with bounded parallel workers and short-lived confined parent
  handles. Compare exact bytes, including same-size/same-mtime external edits.
- Reuse unchanged parsed notes/generations and build relationship graphs only
  for operations that need them.
- Stream word normalization and store compact term keys instead of retaining
  entire normalized bodies through tiny token substrings.
- Share immutable plan byte images, memoize validation by path/revision only
  within a request, and inspect shared parents once per planning phase.

The [architecture decision](decisions/0005-performance-and-freshness.md) and
[correctness regressions](qa/performance-regressions.md) describe retained safety
checks. CI executes a small benchmark corpus and keeps metrics alongside the
existing race, fuzz, recovery, cross-build and security gates. Shared-runner
latency is not a fabricated portable threshold.

## Historical core-only baseline

The original v0.1 measurement was 594.4 ms cold core search, 162.9 ms warm core
search p95 and 180.3 MiB peak RSS. Environment: Apple M4, macOS arm64, Go 1.27.3,
10,000 synthetic 2 KiB notes and 20 warm queries. The initial targets were
150 ms warm p95, 150 MiB RSS and 10 seconds full reindex. The original warm/RSS
targets were not met; reindexing was not independently timed in that report.

That benchmark measures one core process, not fresh CLI startup or MCP transport.
Its peak RSS includes fixture creation and benchmark calibration. Do not compare
it directly with child-process RSS from the new end-to-end harness. The unchanged
[historical evidence](evidence/performance.json) remains available.

```sh
go test ./internal/core -run '^$' -bench BenchmarkReadCore10K -benchtime=20x -count=1
```

`index rebuild` still refreshes derived inventory state and the generation marker;
it does not persist a trusted parsed database. Results are local measurements,
not guarantees of arbitrary-machine performance or external-editor isolation.
