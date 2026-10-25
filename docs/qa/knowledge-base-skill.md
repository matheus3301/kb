# Knowledge Base documentation and skill validation

The candidate replaces the `knowledge-memory` installation identifier with
`knowledge-base`, without changing the `kb` executable or `kb.v1` wire schema.
[Agent setup](../agent-setup.md#upgrade-from-knowledge-memory) documents the scope
and workdir instruction changes required for an existing installation.

## Observed on 2026-10-25

- Installed the local candidate through Skills CLI into a disposable project for
  Codex using `--skill knowledge-base --agent codex --copy --yes`. The copied
  skill and its four references matched the source and passed local link checks
  with the skill folder as the only allowed root. Skill Creator's frontmatter
  validator also passed. A project-scope upgrade from the old name was checked
  through the final installed listing; filtering removal by one agent can retain
  a shared copy for others. No global skill or MCP registration was performed.
- Ran 43 real CLI requests from README, CLI guide, and skill JSON examples. This
  checked creation, previews, idempotent replay, revision conflict rejection,
  patch/move, coordinated plans, source/basis metadata, archive, structural health,
  and retrieval without Markdown/log changes. The same workflow runs in
  `make acceptance` and against native/source-built release snapshots.
- An independent consumer explicitly loaded the installed skill, consulted an
  external synthetic base, reconciled a new 30-day policy bulletin with an older
  14-day synthesis, preserved the original source, and recorded provenance and
  current basis revisions. It previewed writes and asserted no human verification.
- The same consumer answered a repeated question without new learning. Every
  Markdown note, managed index, and log remained byte-identical afterward.
- A separate consumer audited another synthetic base using the read-only CLI.
  It distinguished the stored 14-day claim from the newer 30-day bulletin and
  explained why passing health checks cannot resolve that semantic discrepancy.
  Every Markdown file remained unchanged. The evaluator initially enumerated
  filenames with `rg`; all base content reads used the read-only CLI.

The [sanitized evidence](../evidence/knowledge-base-skill.json) records scope and
outcomes. Test fixtures and installed copies were disposable; no private
collection or globally installed Homebrew binary was a test target.

## Limits

Explicit skill loading in two controlled sessions does not prove automatic
activation across models or workdirs. The new instructions preserve the default
implicit-discovery policy; a deliberate workdir instruction establishes when
consultation is required. Deterministic request checks verify examples, not
universal agent behavior. CAS/recovery durability and performance retain the
limits documented by the existing application QA and benchmarks.
