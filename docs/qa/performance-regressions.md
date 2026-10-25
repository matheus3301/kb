# Performance correctness regressions

The optimization retains exact on-disk freshness instead of trusting timestamps.
Synthetic regressions exercise added/removed empty notes, same-size edits with
restored mtime, refreshed relationship graphs, directory replacements before and
after traversal, deterministic parallel scan errors, byte-image immutability,
path/revision validation-cache invalidation, bulk/per-path confinement parity,
and result-bound metadata without a second inventory. Existing race, CLI/MCP
parity, metadata fidelity, CAS and injected recovery tests remain required.

Unicode differential fuzzing compares word-stream normalization against the
existing whole-string lexical contract. A malformed-UTF-8 counterexample found
during development is retained as a synthetic fuzz seed; invalid text uses the
reference path. No search stemming, fuzzy matching, ranking or filter change is
introduced by this optimization.

The Python harness tests large stdout spooling, source corpus byte sizes, declared
percentile calculation, actual stdio framing/shutdown and explicit rebuild
command execution. `make benchmark-smoke`
executes search, read, validation and preview/committed repair against the built
binary in disposable synthetic directories. A metric file is not a performance
SLA: larger paired runs declare hardware, samples, OS cache conditions and scope.
No private Knowledge Base is a benchmark fixture.
