# Performance measurements

The v0.1 functional workflow passes, but the initial search and memory targets
are not yet met. This is a local measurement, not a performance guarantee.

| Measurement | Observed | Initial target |
| --- | --- | --- |
| Warm core search p95 | 162.9 ms | 150 ms |
| Peak process RSS | 180.3 MiB | 150 MiB |
| Cold core search | 594.4 ms | No cold-search target |

Environment: Apple M4, 10 logical CPUs, macOS arm64, Go 1.27.3. Corpus: 10,000
synthetic notes of 2,048 bytes each. Twenty measured warm queries followed the
initial query. Peak RSS includes fixture creation and benchmark calibration.
The benchmark measures the core in one process, not fresh CLI startup or MCP
transport overhead. Reindexing has not been independently benchmarked here.

Reproduce from a checkout with the pinned Go toolchain:

```sh
go test ./internal/core -run '^$' -bench BenchmarkReadCore10K -benchtime=20x -count=1
```

The lexical index is cached in-process; freshness checks inspect on-disk files.
An MCP process can reuse the index. A new CLI process loads its own snapshot.
Avoid interpreting a warm core number as a fresh-process CLI latency result.
`index rebuild` refreshes derived inventory state and the generation marker;
it does not create a required database or start a background daemon.

Machine-readable results are in [performance evidence](evidence/performance.json).
Future optimization must preserve revision checks, external-edit detection,
confinement, and metadata correctness. The existing quality thresholds remain in
force; this measurement does not relax them.
