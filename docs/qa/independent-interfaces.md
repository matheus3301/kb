# Independent CLI and MCP guidance validation

The user clarified that agent integration has two modes: **CLI + skill**, or
**MCP with its own guidance**. The `knowledge-base` skill now teaches CLI
operation only. MCP initialization instructions and tool/input/output descriptions
supply the operating context without loading skill files or product docs.

## Controlled observations — 2026-10-25

Two independent consumers received the same realistic task in separate synthetic
external Knowledge Bases: reconcile an old 14-day synthesis with a newer 30-day
policy bulletin, preserve evidence, and answer with citations. Neither base
contained `AGENTS.md`.

The CLI consumer loaded only the copied skill and its references, then used the
candidate CLI with an explicit root. It retrieved the evidence, previewed and
applied a revision-checked synthesis update, recorded sources/basis revisions,
and preserved both source snapshots. A follow-up question with no new evidence
left every Markdown note, index and log unchanged.

The MCP consumer received only the actual `initialize`/`tools/list` metadata and
a controlled stdio test client. It accessed the base exclusively through MCP,
without a skill, CLI data commands, raw file reads, or repository documentation.
It searched/read evidence, previewed/applied `kb_patch`, reread the result and
validated health. Sources remained intact and basis revisions were current. Its
follow-up answer also changed no Markdown note, index or log.

The [sanitized evidence](../evidence/independent-interfaces.json) records interface
scope and results. Both consumers distinguished stored policy from actual service
enforcement and did not fabricate human verification. Both observed a stale-index
advisory and relied on direct evidence reads.

## Regressions and limits

- Live SDK tests verify initialization guidance, described parameters (including
  nested operations), output schemas, and read-only discovery/enforcement.
- The public [tool catalog](../../schemas/kb.v1.tools.json) must match actual
  initialization and tools/list metadata. Regeneration is explicit; ordinary CI
  rejects drift rather than updating evidence silently.
- A real-service MCP workflow tests discovery-independent create/preview/replay,
  search/read/citations, revision conflicts, edits and no-write retrieval, without
  skill files or shell operations.
- The existing 43-request documentation workflow continues to exercise CLI and
  skill examples. Release snapshot checks require the public MCP guide too.

These are controlled explicit-interface evaluations, not proof of automatic
invocation or universal model adherence. MCP still exposes its current ten tools;
operator initialization/maintenance/recovery/export/rebuild capabilities are not
implied. No global setup or private collection was a product test target.
