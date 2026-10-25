# Roadmap toward v1.0

Status as of **2026-10-25**: all four initial roadmap increments are implemented
and integrated at `a320dea`. **Published production remains experimental
[v0.3.1](https://github.com/matheus3301/kb/releases/tag/v0.3.1)**, source commit
`2ca104f48cc4a6551f4bbc4fae12d51432e42dff`. Current source is being prepared for
**v0.4.0 qualification**, not presented as an already published version.
Stable v1.0.0 remains conditional on the [exit checklist](stability.md).
Integration, qualification, publication and installed upgrade are separate steps.

## Implemented increments

| Increment | Integrated capability and evidence | Remaining candidate qualification |
| --- | --- | --- |
| Retrieval quality | [Synthetic corpus and evaluator](qa/retrieval-quality.md): 30 concepts, 31 search judgments, eight context cases and eight source reads through actual CLI/MCP. [Recorded baseline](evidence/retrieval-quality.json): macro Recall@5 0.9930556, MRR@5 1.0, exact citations, budgets, parity and query purity. No ranking change. | V02: rerun on the exact candidate and review category results against D01. |
| Reliability and operational backup/restore | [Version-1 backup contract](backups.md), `backup`/`restore` CLI and `kb_backup`/`kb_restore` MCP tools. Exact content/config/receipt preservation, settled-state archives, hostile-input rejection, injected failures and actual process interruptions have synthetic regressions. | V03/V04: extended multi-seed reliability observation, repeated interruption runs and candidate-specific restoration evidence under D02. |
| Larger collections | [Quiet paired 10k/50k report](performance-large-corpus.md) and [measurements](evidence/performance-large-corpus.json): streamed generation hashing and smaller inventory allocations, without weakening exact freshness or publication. | V05: exact integrated candidate paired rerun, rebuild and drift review against local D03 budgets. |
| Stability and current documentation | This roadmap, [compatibility policy](compatibility.md), [quality policy](quality.md) and [V01–V08 checklist](stability.md) distinguish source, released and installed behavior. D01–D04 select qualification policy; they are not passing evidence. | V01/V06/V07/V08: integrated gates, upgrades, real packages/consumers and independent readiness review. |

## Released baseline and limits

The local CLI and self-contained stdio MCP share native OKF parsing, lexical
retrieval, bounded cited context, revision-checked writes, durable receipts and
recovery. Incremental repair is explicit; normal writes remain strict. Binary
installation, CLI skill installation, MCP registration and workdir activation
remain separate. The [specification](spec.md), [interface contract](interfaces.md)
and [acceptance inventory](acceptance.json) retain the original product contract.
Use documentation from the installed release's tag for shipped behavior.

v0.3.1 met the original local 10k-note core targets: 117.7 ms warm-search p95,
134.0 MiB peak RSS and 179.0 ms rebuild p95. The [historical report](performance.md)
is not a fresh CLI/MCP measurement or portable SLA. Its installed-package evidence
is macOS arm64 only. v0.3.1 has **no operational backup/restore interface**;
its export is a scoped knowledge bundle, not configuration/journal/receipt backup.

The newer paired matrix measured 50k CLI search at 2502.4 → 2217.4 ms median,
read at 1735.9 → 1419.7 ms and warm MCP search at 1374.4 → 1093.5 ms.
This is not a uniform speedup: 10k search/read did not improve, 50k committed
repair median rose 6097.4 → 6397.6 ms (about 4.9%), and preview RSS increased
745.8 → 751.4 MiB. Exact inventory reads still scale with scoped bytes and files;
retained-heap reduction was only about 1%. One local small-sample matrix does not
establish statistical significance or results on other hardware.

Backup and restore are local staging/verification, not off-disk disaster recovery.
They refuse pending recovery and never archive a partial journal. Restore creates
a new confined destination, publishes configuration last and never changes the
running MCP root. Preserve excluded/external artifacts separately; retain archives
off disk with an explicitly chosen mechanism. No encryption, retention, remote
transfer or universal power-loss guarantee is implied.

## Candidate sequence

1. Reconcile the exact integrated source and run all V01–V06 checks, including
   D01 retrieval assertions, D02 reliability/restoration and a quiet D03 matrix.
   Preserve historical reports; publish new evidence with candidate identity.
2. Qualify **v0.4.0** through the normal release process when authorized, including
   actual macOS arm64 Homebrew installation and Linux amd64 native archive/source
   execution. Record CLI + skill and independent MCP consumer limits under V07.
   A v0.4.0 non-prerelease tag still does not establish stable-v1 readiness.
3. Independently review V01–V08 against one identified candidate and resolve
   remaining findings. Update the [finalization checklist](stability.md#finalization-checklist)
   in a later docs-only change with exact evidence, not anticipated results.
4. Consider stable **v1.0.0** only after qualification and readiness approval;
   separately authorize publication and verify installed artifacts. Any source
   change after qualification requires impact assessment and affected reruns.

The delegated qualification choices are documented with rationale in D01–D04,
not represented as independent human approval. Missing evidence remains pending,
even after green CI. No calendar deadline, portable latency SLA or LTS lifetime
is promised.

Out of scope: UI, remote transport, automatic Git sync, telemetry, embedded LLM,
embeddings, legacy compatibility layer and simultaneous multi-base operation.
Health and conformance do not establish factual truth; advisory locks do not
isolate arbitrary external editors.

