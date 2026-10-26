# Roadmap toward v1.0

This roadmap records the implemented increments and their qualification sequence,
not a release announcement. The [README project status](../README.md#project-status),
[compatibility policy](compatibility.md) and [exit checklist](stability.md)
centralize current published/installed versions, candidate outcomes and the
readiness decision. Integration, qualification, publication and installed upgrade
remain separate steps.

## Implemented increments

| Increment | Integrated capability and evidence | Required candidate qualification |
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

The dated v0.3.1 [historical report](performance.md) met the original local
10k-note core targets: 117.7 ms warm-search p95, 134.0 MiB RSS and 179.0 ms
rebuild p95. It is not fresh CLI/MCP evidence or a portable SLA.
v0.3.1 has no operational backup command; v0.4.0 introduced the version-1
archive workflow. Export remains a scoped knowledge bundle, not receipt backup.

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

The original v0.4.0 extended attempt failed on backup latency as durable
receipts accumulated. The [receipt-history report](qa/backup-performance-v1.md)
retains the failed observation, reproduced repeated-directory-scan cause and
the candidate correction without changing formats, expiry or the qualification
deadline. On one host, three samples at 2,000 receipts measured backup median
24.45 s to 0.47 s and restore 23.88 s to 9.93 s. That repair does not qualify
arbitrary histories or a later binary's reliability/resource envelope.

## Candidate sequence

1. **Intermediate baseline:** retain the actual v0.4.0 executable and its immutable
   source/checksum provenance. Earlier macOS arm64 Homebrew/formula, asset,
   CLI/documentation and separate CLI + skill/MCP observations are release-specific,
   not stable-v1 approval.
2. **Integrated candidate:** require exact protected-main CI and both D02 profiles:
   1,800 seconds across three seeds/four writers with log maintenance disabled and
   indexes maintained, plus the separate shorter default-log profile. Do not
   present the long profile as 30 minutes of default-log behavior. A Go runtime
   change restarts full qualification; preserve earlier passing runs as history.
3. **Resource envelope:** use the predeclared complete quiet matrix and unchanged
   D03 ceilings, only after concurrent reliability/build/test work ends. Review
   drift and every required row. Preserve failures; do not waive a budget because
   the candidate is faster than its baseline or repeat until green.
4. **Exact compatibility and package evidence:** require both pinned v0.3.1 and
   v0.4.0 CI origins, actual prior-produced archive restoration where supported,
   pending recovery with the prior binary, replay and continued writes. Final
   macOS arm64 Homebrew and Linux amd64 native/archive-source qualification must
   identify the intended candidate. Configured published-asset verification,
   snapshots and temporary-prefix execution do not replace real installed checks.
5. **Readiness before publication:** independently review V01–V08 and record the
   explicit decision in the [finalization checklist](stability.md#finalization-checklist).
   Only then publish the separately authorized new tag and verify actual assets
   and installed identities. Source changes require an impact assessment and
   appropriate requalification; current outcomes stay in the central status records.

The delegated qualification choices are documented with rationale in D01–D04,
not represented as independent human approval. Missing evidence remains pending,
even after green CI. No calendar deadline, portable latency SLA or LTS lifetime
is promised.

Out of scope: UI, remote transport, automatic Git sync, telemetry, embedded LLM,
embeddings, legacy compatibility layer and simultaneous multi-base operation.
Health and conformance do not establish factual truth; advisory locks do not
isolate arbitrary external editors.

