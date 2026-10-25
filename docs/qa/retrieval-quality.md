# Retrieval and context quality evaluation

`make retrieval-eval` builds the actual CLI and runs the bounded, deterministic
[standard-library harness](../../scripts/retrieval_eval.py). It is additive to
`make check` and to both Linux/macOS CI test jobs. CI retains the detailed report
at `.build/retrieval-quality.json` with the other test artifacts. Run from the
repository root; no installed binary, skill, account, network, model, credentials,
or existing knowledge collection is needed.

## Corpus and judgments

The public [synthetic corpus](../../testdata/retrieval/corpus.json) contains 30
concept notes plus two reserved navigation/log files, 31 search judgments and
eight context cases. All content is invented.
Portuguese and other Unicode strings are intentional retrieval test data, not
instructions or private source material. JSON metadata is rendered as valid YAML
values without an external parser dependency. Each run initializes a disposable
root and isolated home, copies the same exact fixture bytes, and fixes `--as-of`
to `2026-01-01T00:00:00Z`. Expiry is therefore deterministic across machines.

Judgments come from the [documented lexical contract](../spec.md#7-search-and-reading),
not from snapshotting observed rankings. Review the query, eligible `matches`,
relevant paths, and optional pairwise `before` constraints together:

- Accented, folded, case-varied and decomposed Portuguese must find the same
  intended note. Full CJK and Cyrillic tokens exercise Unicode, not arbitrary
  substring semantics. Exact `s3`, `k8s` and `crossplane` must not match longer
  identifiers.
- English river/financial bank notes and coffee/memory notes expose lexical
  ambiguity. An explicit tag filter resolves the financial sense; the suite
  does not pretend an unqualified ambiguous query has one semantic answer.
- Controlled title/path/tag/description versus body pairs require the documented
  field preference. Equal-content pairs require deterministic path ordering.
  No exact floating-point BM25 value or total ordering of unequal-length general
  prose is pinned.
- Eligible sets cover field-list OR, cross-field AND, prefix, PARA, type, tags,
  status, trust, source presence and stale booleans. Expired evidence remains
  available and marked. Archives/deprecated notes require explicit opt-in.
- Six relevant recovery notes exercise a genuine top-five cutoff. Exact misses
  and contradictory filters must return no hits.

`matches` is the complete eligible lexical set, including distractors. Each
search requests five results and verifies total count, allowed membership,
unique hits, fresh-byte revisions, OKF indicators, and the requested stale flag.
A case fails if any relevant document that can fit in five is displaced, if the
first hit is irrelevant, or if a declared field/tie ordering reverses. These
small controlled cases intentionally have demanding per-case thresholds;
aggregate scores never allow one failing case to hide behind another.

### Unsupported semantics

Search is whole-token lexical intersection. `--literal` additionally requires
the folded phrase inside a lexical candidate; it is not an unrestricted byte or
Unicode substring scan. `title:orchid` is not a field operator, `s3 OR bank` is
not Boolean OR, and emoji-only or partial CJK token queries have no candidates
in this corpus. Those cases check the declared limitations, not an invented
query grammar. There is no semantic paraphrase, stemming, embedding, translation,
implicit negation, regex or LLM relevance claim.

## Metrics

For a nonempty relevant set $R_q$ and ordered result list $L_q$, the harness uses:

- **Recall@5** = $|R_q \cap L_q[1:5]| / |R_q|$. The denominator is all relevant
  documents, not five and not the returned count. Six relevant documents with
  five retrieved give $5/6$, not 1.
- **RR@5** = $1/r$ for the first relevant rank $r \le 5$, otherwise 0. This is
  cutoff-censored reciprocal rank, not reciprocal rank over an unseen full list.
- **Macro Recall@5** and **MRR@5** are arithmetic means across positive judgments.
  No-relevant cases have undefined (`null`) recall/RR and are excluded from both
  means. They pass only with exactly zero returned hits and zero total.

Duplicate results or relevance judgments are errors, never additional credit.
[Metric and validator regressions](../../scripts/test_retrieval_eval.py) cover
fractional recall/rank, cutoff, misses, empty relevance, macro averaging, duplicate
inflation, incorrect ranking, UTF-8 byte budgets, fabricated excerpts, bad source
revisions/ranges, transport mismatch and authoritative-file mutations.

## Independent interfaces and context invariants

Every search/context request runs as a fresh CLI subprocess and through a
separately launched, persistent stdio MCP process on the same fixture. The MCP
consumer performs initialize/initialized and tools/list, checks independently
supplied guidance and descriptive search/context/read schemas, then calls tools
without loading or consulting the CLI skill. It checks full data parity,
including result order, metadata, revisions, snippets, context bytes and source
ranges. Score comparison permits only floating-point summation roundoff
(`1e-12` relative/absolute tolerance), not ranking differences.

Context cases cover complete Unicode excerpts, independent byte/token ceilings,
partial excerpts, an insufficient citation budget, multiple/omitted sources, no results,
stale eligibility, and exclusion of root navigation/log content. Every returned
citation must identify the exact SHA-256 revision and physical source lines;
every excerpt must be original source bytes, with only a terminal partial
excerpt allowed when truncated. There can be no uncited/generated text. The
runtime counts a trailing LF's zero-byte final line in its physical-line model.
The token estimator is explicitly `utf8-bytes/4-ceiling`, not a model tokenizer.
Truncation and omitted counts are checked separately; a partially returned note
is not an omitted note. Both interfaces also read the eight selected source
notes and compare their exact Markdown/revisions against the fixture bytes.

Queries must preserve all authoritative file hashes, including notes, config,
log and any journal/receipts. Only derived `.kb/cache/` files and the advisory
`.kb/state/lock` may differ. No files may appear in the isolated home. The harness
closes MCP stdin, requires normal process exit, and bounds each subprocess/RPC
response; CI additionally bounds the enclosing job. It never mutates an existing
collection or installs tools.

## Baseline and limits

The [focused evidence report](../evidence/retrieval-quality.json) records the
unchanged v0.3.1 baseline and candidate measurements. The output contains the
corpus hash, clock, per-case rankings/context observations, metrics, parity and
purity results. A nonzero exit lists action-oriented `search/<case>` or
`context/<case>` failures with expected/actual evidence. Do not update a judgment
just to make observed output pass: justify changes against the contract first.

This is a compact regression corpus, not a representative multilingual benchmark
or proof of arbitrary semantic truth. Trust coverage here is unverified notes;
verification invalidation and synthesis dependencies remain covered by their
existing application tests. This suite does not replace parser/path fuzzing,
recovery/durability/concurrency tests, pagination tests, large-corpus performance
measurements, one-hop relationship tests, installed-package validation or human
review. It makes no release-readiness or v1.0 claim.
