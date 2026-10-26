# Cold CLI search and release-shaped resource qualification

## Result and artifact identity

The **2026-10-26 03:47:53–03:56:38 UTC** complete paired observation passed all
**16 candidate operation/size rows**, each with unchanged p95/RSS ceilings, and
all **three separately identified core/rebuild targets** in the
[D03 policy](../stability.md#d03--measured-local-resource-budgets-not-a-portable-sla).
This is a local resource result, **not stable-v1 readiness or a published v1 release**.

The [public evidence projection](../evidence/cold-cli-performance-v1.json) includes
all ordered CLI/warm-MCP/rebuild samples, available first requests, RSS, absolute
and percentage drift, budgets, sanitized isolation observations and four earlier
failed matrices. It contains numeric synthetic observations and whitelisted build
metadata, not private operational records. Original inputs are fingerprinted.

| Role | Observed version / source | Executable SHA-256 |
| --- | --- | --- |
| Preserved actual Homebrew baseline | `v0.3.1` / `2ca104f48cc4a6551f4bbc4fae12d51432e42dff` | `0f6267b05e3eb80ae053cd113aecdb61b3d6e29bd74451b9dc658578b9643238` |
| Release-shaped development candidate | `v1.0.0-candidate.3` / `b71c0763c87313666b3dd79da08da12b7b3512c1` | `8aecbf6cf22d2c4bc8c97e1d8b63fea725dfa325b2e5f632f2ed97fc815815ac` |

The narrow ASCII change is available in the
[public integrated source](https://github.com/matheus3301/kb/blob/b71c0763c87313666b3dd79da08da12b7b3512c1/internal/core/search.go).
The release-build recipe correction is in
[the integrated Makefile](https://github.com/matheus3301/kb/blob/5c94c993aa8ec97295bdf6bcdb1d5cd2f052a2f7/Makefile)
from [PR 23](https://github.com/matheus3301/kb/pull/23).
Later recipe/documentation integration does not relabel the candidate's embedded
source identity or change its frozen Go/module bytes.

## Build, hardware and measurement method

- Apple M4, 10 logical CPUs, macOS **26.6.2**, Darwin **25.6.0**, arm64;
  Go **1.27.3**, Python **3.14.4**, measured `GOMAXPROCS=8`.
- Both CLI artifacts independently reported **`CGO_ENABLED=0`**, `-trimpath=true`,
  Go 1.27.3 and Darwin/arm64 build metadata. The declared shipping environment was
  `CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local`; the candidate used stripped
  `-s -w` plus explicit version/full-source linker identities. Go build-info does
  not expose every environment/linker option; those provenance claims are distinct
  from the independently visible CGO/toolchain/target/trimpath fields and the
  version/source asserted by actual CLI execution.
- The non-race core benchmark was compiled with `go test -c` under that same
  explicit CGO0/workspace/toolchain environment, then executed with
  `GOMAXPROCS=8`. Its inspected build-info also reports CGO0; it has no recorded
  trimpath setting and is not claimed to be the stripped CLI artifact. Separate
  correctness/race gates retain the CGO1 environment they require.
- Fresh disposable synthetic corpora contained **10,000 and 50,000 notes of
  exactly 2,048 bytes**, grouped 500 per directory. The public harness's search
  is `quartzbeam recovery`. Fixture generation is outside operation timings.
- Order: baseline complete normal matrix, candidate normal matrix, baseline and
  candidate both-size rebuild matrices, then the separately identified core run.
  There were **20 read samples per operation/transport/size/binary**, **three
  preview and three committed-repair samples**, **five rebuild samples per
  size/binary**, and **20 core warm iterations**. No samples were removed, no
  extra warm-up was added and no passing cells were spliced across runs.
- The **original accepted recording adapter and measured functions** were used;
  the proposed CPU/wait4 diagnostic hook was **not used**. CLI RSS is each fresh
  child's `wait4` peak, excluding Python and fixture creation. MCP uses one reused
  initialized stdio process per corpus; its single session peak is shared by the
  three warm-operation rows, **not three per-operation or per-request measurements**.
- CLI first samples remain in their arrays. MCP first values are the first
  request for each operation after initialization, separate from its 20 warm
  samples; they do not time the initialization handshake. `Sample max` means
  sampled/warm maximum; the JSON also retains the MCP first-plus-warm maximum.
  Committed repair includes journal/receipt/fsync publication, with managed-index
  creation in its first apply.
- Original nearest-rank p95 selects rank 19 of 20 read samples. With three repair
  or five rebuild samples it is the observed maximum. The original core's integer
  index also selects rank 19 at 20 iterations. These are empirical order statistics,
  not population-tail estimates or a statistical-significance claim.
- One reserved execution slot separated application builds/tests/profiles/soaks.
  **Eight fresh native-state checks** each observed all four orchestration workers
  settled, without streaming, compaction, pending asynchronous work or queued
  messages. These checks occurred at stage boundaries, outside operation clocks;
  they do not continuously monitor the whole OS. Earlier mirror-only idle records
  were **not** equivalent fresh native proof; that limitation does not establish
  actual overlap or a culprit in the earlier runs.

“Cold CLI” means a fresh process without a reusable in-process parsed/index cache,
**not cold disk**. OS caches were not evicted, background applications were not
controlled, and baseline-first order does not remove filesystem/cache/thermal
variation. These Apple M4 observations are not a portable SLA or Linux timing proof.

## Complete paired results

Every cell below is **baseline → candidate (absolute delta; percentage delta)**.
Latency deltas are milliseconds; RSS deltas are MiB. Drift is derived from the
reported rounded summaries; the linked JSON preserves the original ordered float
samples for independent recomputation. Budgets apply to the candidate, not the
baseline comparison. A baseline outlier never changes the candidate ceiling.

### 10,000 notes

| Operation | p50 (ms) | p95 (ms) | First (ms) | Sample max (ms) | Peak RSS (MiB) |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| CLI search | 320.038 → 284.181 (-35.857; -11.204%) | 489.321 → 291.584 (-197.737; -40.410%) | 332.378 → 290.570 (-41.808; -12.578%) | 494.835 → 348.033 (-146.802; -29.667%) | 115.984 → 115.750 (-0.234; -0.202%) |
| CLI read | 174.392 → 176.592 (+2.200; +1.262%) | 177.557 → 177.750 (+0.193; +0.109%) | 173.637 → 175.218 (+1.581; +0.911%) | 177.580 → 179.629 (+2.049; +1.154%) | 100.719 → 98.984 (-1.735; -1.723%) |
| CLI validate | 173.376 → 175.992 (+2.616; +1.509%) | 178.538 → 177.666 (-0.872; -0.488%) | 173.428 → 175.640 (+2.212; +1.275%) | 179.788 → 177.718 (-2.070; -1.151%) | 100.641 → 99.719 (-0.922; -0.916%) |
| CLI repair preview | 505.558 → 522.604 (+17.046; +3.372%) | 520.282 → 524.845 (+4.563; +0.877%) | 520.282 → 522.604 (+2.322; +0.446%) | 520.282 → 524.845 (+4.563; +0.877%) | 174.156 → 178.594 (+4.438; +2.548%) |
| CLI committed repair | 892.665 → 896.412 (+3.747; +0.420%) | 1187.333 → 1207.517 (+20.184; +1.700%) | 1187.333 → 1207.517 (+20.184; +1.700%) | 1187.333 → 1207.517 (+20.184; +1.700%) | 206.516 → 212.906 (+6.390; +3.094%) |
| Warm MCP search | 100.914 → 100.892 (-0.022; -0.022%) | 102.682 → 101.835 (-0.847; -0.825%) | 330.103 → 274.410 (-55.693; -16.871%) | 116.868 → 102.620 (-14.248; -12.192%) | 131.984 → 135.531 (+3.547; +2.687%) |
| Warm MCP read | 100.164 → 100.568 (+0.404; +0.403%) | 100.810 → 101.452 (+0.642; +0.637%) | 101.135 → 99.658 (-1.477; -1.460%) | 101.303 → 101.957 (+0.654; +0.646%) | 131.984 → 135.531 (+3.547; +2.687%) |
| Warm MCP validate | 99.899 → 100.427 (+0.528; +0.529%) | 102.972 → 101.176 (-1.796; -1.744%) | 99.936 → 101.621 (+1.685; +1.686%) | 112.569 → 101.438 (-11.131; -9.888%) | 131.984 → 135.531 (+3.547; +2.687%) |

Candidate search p95 **291.584 ms** is **88.416 ms (23.267%) below** the unchanged
380 ms ceiling, with **115.750 MiB** child peak RSS. Its median improved
**35.857 ms (11.204%)** against this released baseline. The larger p95 difference
also reflects retained baseline outliers; it is not a claim that every query
became 40% faster. Candidate validation p95 is **177.666 ms** against 190 ms.

The result is not a uniform improvement. Read and validation medians rose by
2.200 and 2.616 ms. Preview median rose by 17.046 ms, with peak RSS +4.438 MiB;
committed-repair p95/first apply rose by 20.184 ms, with RSS +6.390 MiB.
The reused MCP session peak increased **3.547 MiB**, shared by all three MCP rows.
All remain disclosed rather than removed because their candidate budgets passed.

### 50,000 notes

| Operation | p50 (ms) | p95 (ms) | First (ms) | Sample max (ms) | Peak RSS (MiB) |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| CLI search | 2334.302 → 1973.854 (-360.448; -15.441%) | 2526.158 → 2227.160 (-298.998; -11.836%) | 2314.563 → 1975.493 (-339.070; -14.649%) | 2535.562 → 2233.037 (-302.525; -11.931%) | 512.750 → 491.406 (-21.344; -4.163%) |
| CLI read | 1570.203 → 1431.846 (-138.357; -8.811%) | 1734.623 → 1444.015 (-290.608; -16.753%) | 1622.724 → 1417.009 (-205.715; -12.677%) | 1737.509 → 1448.302 (-289.207; -16.645%) | 432.797 → 400.891 (-31.906; -7.372%) |
| CLI validate | 1558.623 → 1431.330 (-127.293; -8.167%) | 1777.229 → 1607.952 (-169.277; -9.525%) | 1584.293 → 1484.984 (-99.309; -6.268%) | 1863.801 → 1656.095 (-207.706; -11.144%) | 433.078 → 399.406 (-33.672; -7.775%) |
| CLI repair preview | 3604.185 → 3282.410 (-321.775; -8.928%) | 4054.806 → 3605.977 (-448.829; -11.069%) | 3398.350 → 3282.410 (-115.940; -3.412%) | 4054.806 → 3605.977 (-448.829; -11.069%) | 753.297 → 743.125 (-10.172; -1.350%) |
| CLI committed repair | 6542.525 → 6406.868 (-135.657; -2.073%) | 8538.917 → 7747.954 (-790.963; -9.263%) | 8538.917 → 7747.954 (-790.963; -9.263%) | 8538.917 → 7747.954 (-790.963; -9.263%) | 944.875 → 817.938 (-126.937; -13.434%) |
| Warm MCP search | 1196.092 → 1082.393 (-113.699; -9.506%) | 1398.657 → 1089.205 (-309.452; -22.125%) | 2320.601 → 1962.579 (-358.022; -15.428%) | 1430.948 → 1104.833 (-326.115; -22.790%) | 570.438 → 547.109 (-23.329; -4.090%) |
| Warm MCP read | 1285.851 → 1082.499 (-203.352; -15.815%) | 1407.264 → 1300.823 (-106.441; -7.564%) | 1393.795 → 1078.479 (-315.316; -22.623%) | 1455.142 → 1302.947 (-152.195; -10.459%) | 570.438 → 547.109 (-23.329; -4.090%) |
| Warm MCP validate | 1192.451 → 1081.831 (-110.620; -9.277%) | 1403.166 → 1280.394 (-122.772; -8.750%) | 1438.273 → 1076.878 (-361.395; -25.127%) | 1403.912 → 1322.017 (-81.895; -5.833%) | 570.438 → 547.109 (-23.329; -4.090%) |

All eight candidate p95/RSS rows passed. Nevertheless, fresh search still takes
about two seconds, preview about 3.3 seconds and committed repair about 6.4 seconds
at their medians. This is not instant arbitrary-scale retrieval or a broad SLA.
The shipping comparison is against v0.3.1 and includes all integrated increments;
it does **not** isolate the ASCII change from every other post-v0.3.1 improvement.

### Both paired rebuild sizes

| Operation | p50 (ms) | p95 (ms) | First (ms) | Sample max (ms) | Peak RSS (MiB) |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 10k CLI index rebuild | 176.170 → 177.421 (+1.251; +0.710%) | 177.666 → 179.690 (+2.024; +1.139%) | 177.482 → 179.690 (+2.208; +1.244%) | 177.666 → 179.690 (+2.024; +1.139%) | 100.266 → 97.031 (-3.235; -3.226%) |
| 50k CLI index rebuild | 1441.045 → 1427.882 (-13.163; -0.913%) | 1446.556 → 1691.151 (+244.595; +16.909%) | 1432.446 → 1691.151 (+258.705; +18.060%) | 1446.556 → 1691.151 (+244.595; +16.909%) | 428.188 → 399.859 (-28.329; -6.616%) |

The 50k rebuild median fell slightly, but its **p95/max increased
244.595 ms (16.909%)** and its **first increased 258.705 ms (18.060%)**.
That slow first observation is retained. No new 50k rebuild
ceiling or unmeasured rebuild improvement is invented. Fresh CLI rebuild includes
startup plus the core call; these values are not separately measured pure-core
rebuild latencies.

### Separate core observations

| Observation | Candidate | Existing ceiling |
| --- | ---: | ---: |
| 10k warm search p95, 20 iterations | 113.400 ms | 150 ms |
| Core process peak RSS | 131.400 MiB | 150 MiB |
| Fresh CLI 10k rebuild p95 | 179.690 ms | 10,000 ms |
| Core cold search | 287.300 ms | No separate cold target |

Core RSS includes that process's fixture creation/calibration; it is not
CLI-child RSS. The unchanged core benchmark reports cold time, warm p95 and process
peak only, not raw warm arrays or a warm maximum; none are fabricated here.

## Causal scope and development profiles

The implementation removes redundant work in `forEachToken`. A word already
split using the existing word predicate cannot gain a word boundary through
ASCII lowercasing. Already-lowercase ASCII words are visited directly; uppercase
ASCII is lowercased once without a second tokenizer. Every non-ASCII word keeps
full folding/normalization/accent handling and retokenization, and invalid UTF-8
keeps the existing fallback. Parsing, fresh complete inventory reads, hashes,
confinement, locking, map/posting order, float accumulation and worker counts are
unchanged. There is no new cache, parser concurrency or all-note frequency-map
materialization.

The [integrated regression source](https://github.com/matheus3301/kb/blob/b71c0763c87313666b3dd79da08da12b7b3512c1/internal/core/performance_semantics_test.go)
checks folding/boundary cases and property equivalence to the original token
reference over Unicode and invalid bytes. The implementation's focused core
race/shuffle run and **159,233** property-fuzz executions passed. Those tests are
correctness evidence, not a timing or durability guarantee.

Exploratory **CGO1 development** profiles used 12 actual fresh CLI searches per
variant over 10k synthetic 2 KiB notes. A development-only profiling entrypoint
bracketed unchanged CLI `mainCode`; a separate process generated fixtures before
profiling. Forced GC and allocation-profile serialization followed CPU stop.
Allocation samples include process/profiler overhead, never fixture generation.
No profiling entrypoint or CPU recorder was used in the shipping matrix.

| Sampled profile quantity, aggregate over 12 requests | Before ASCII change | After ASCII change |
| --- | ---: | ---: |
| Lexical construction CPU | 1.29 s | 0.89 s |
| Token-stream CPU | 1.02 s | 0.79 s |
| Parser CPU, including revision hashing | 0.31 s | 0.45 s |
| Whole-thread sampled CPU | 10.24 s | 12.03 s |
| Syscall-attributed CPU | 7.36 s | 9.46 s |
| Sampled aggregate allocations, pprof GB labels | 2.25 GB | 2.31 GB |

The targeted lexical sample fell about **31%**. Whole-thread sampled CPU and
sampled allocations **increased**; syscall attribution across workers is not a
wall-time decomposition. These short development profiles do not establish an
overall CPU, allocation, retained-memory, RSS or native-latency improvement.
They explain the narrow optimization choice, not every shipping-matrix delta.
No per-request CPU/scheduler diagnostics or external-cause attribution is claimed.

## Four earlier failed matrices remain failed

| Development matrix | Source / artifact | Failed candidate p95 versus unchanged ceiling | Whole result |
| --- | --- | --- | --- |
| 1 | `7ceed676…` / `v1.0.0-candidate` | 50k CLI search 4,191.294 > 2,870 ms; preview 3,827.358 > 3,770 ms | **FAIL** |
| 2 | Same pre-ASCII source/artifact | 10k warm MCP validate 117.925 > 110 ms | **FAIL** |
| 3 | Same pre-ASCII source/artifact | 10k CLI search 384.470 > 380 ms | **FAIL** |
| 4 | `b71c076…` / `v1.0.0-candidate.2` | 10k CLI validate 234.312 > 190 ms | **FAIL** |

The fourth matrix's two adjacent validation outliers were 289.392 and
234.312 ms; the other eighteen were near 173–179 ms. Its search and other passing
cells did not make that matrix pass. All four failure outcomes, failed-row
measurements and original summary hashes are retained in the public projection;
no best-cell combination replaces them.

Build-info inspection found the audited `candidate.2` local development binary
used **CGO1**, while the actual Homebrew baseline and release-shaped `candidate.3`
use **CGO0**. The earlier three development artifacts' CGO metadata is not
independently asserted by this report. Correcting the qualification build procedure
is **not proof that CGO caused any spike**, nor a waiver of a failure. Separate
race checks legitimately need CGO1 and were not disabled. The shipping result is
one new-artifact observation, not a repeated-until-green same-binary selection.

## Public reproduction

Use a clean checkout of the linked integrated Go source and retain the actual
released baseline separately as `./baseline-kb`. Verify both identities/build-info
and record their executable hashes. A rebuilt candidate has its own artifact hash;
do not assume it is the exact preserved binary above or relabel it as published.
The integrated `make benchmark` target provides the existing public convenience
entrypoint with scoped release-mode CLI environment. For the explicit paired
20/3/5 matrix, run the existing public harness serially:

```sh
mkdir -p .build
CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local go build -trimpath -ldflags '-s -w -X main.version=v1.0.0-candidate.3 -X main.commit=b71c0763c87313666b3dd79da08da12b7b3512c1' -o .build/cold-candidate ./cmd/kb
go version -m ./baseline-kb .build/cold-candidate
GOMAXPROCS=8 CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local python3 scripts/benchmark.py --binary ./baseline-kb --sizes 10000 50000 --samples 20 --repair-samples 3 --output .build/cold-baseline.json
GOMAXPROCS=8 CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local python3 scripts/benchmark.py --binary .build/cold-candidate --sizes 10000 50000 --samples 20 --repair-samples 3 --output .build/cold-candidate.json
GOMAXPROCS=8 CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local python3 scripts/benchmark.py --binary ./baseline-kb --sizes 10000 50000 --samples 5 --repair-samples 3 --rebuild-only --output .build/cold-baseline-rebuild.json
GOMAXPROCS=8 CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local python3 scripts/benchmark.py --binary .build/cold-candidate --sizes 10000 50000 --samples 5 --repair-samples 3 --rebuild-only --output .build/cold-candidate-rebuild.json
CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local go test -c -o .build/cold-core-bench ./internal/core
go version -m .build/cold-core-bench
GOMAXPROCS=8 .build/cold-core-bench -test.run '^$' -test.bench '^BenchmarkReadCore10K$' -test.benchtime 20x -test.count 1 -test.v
```

The harness owns fresh disposable synthetic roots; no operator collection is
needed. Keep fixture setup outside timings, preserve every observation, use one
quiet serialized window, and do not overlap with builds/tests/profiles/soaks.
Environment prefixes above apply only to these shipping/performance commands,
not to full correctness/race gates. The public harness emits aggregate reports;
the linked evidence also publishes the already-collected ordered arrays so its
summaries can be recomputed without private recording helpers. Other hardware or
newly built artifacts produce new observations, not a portable pass guarantee.

## Remaining qualification

At the **2026-10-26 03:58:31 UTC checkpoint**, the new shipping-artifact extended
reliability observation had **started and was not yet passed**. It began after
the recorded resource window ended. This report does not infer its later result
or transfer the historical pre-ASCII artifact's endurance to these bytes.

Stable-v1 readiness still requires the shipping artifact's own complete 1,800-second
and default-log companion observations, interruption/restoration evidence, final
artifact/installed-channel qualification and independent readiness disposition.
See the [stability checklist](../stability.md) and
[compatibility policy](../compatibility.md). No independent human approval,
publication or stable-v1 completion is claimed here.
