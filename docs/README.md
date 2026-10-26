# Documentation

KB keeps knowledge in local Markdown files. Start with the CLI; add the
`knowledge-base` skill when you want your agent to use the same workflow.
The static marketing site is a separate presentation of this product, not a
browser UI for a collection. These Markdown guides remain canonical.

## Get started

- [Installation](installation.md): Homebrew, verified release archives, source
  installs, upgrades, and prerequisites.
- [Quickstart](quickstart.md): create, search, read, edit, and preview a coordinated
  plan using a disposable example base.
- [Agent setup](agent-setup.md): install the CLI and skill, deliberately select
  agent/project/global scope and a root, preserve existing collections and
  instructions, activate a workdir policy, and verify actual use.
- [CLI guide](cli.md): command syntax, JSON requests, root selection, errors,
  budgets, and operating limits. Check installed command help for your version.
- [llms.txt](../llms.txt): concise public Markdown discovery index for agents.

## Use and maintain a collection

- [knowledge-base skill](../skills/knowledge-base/SKILL.md): CLI workflow selection
  and references that travel with the installed skill.
- [Retrieval](../skills/knowledge-base/references/retrieval.md): search, filtering,
  source reads, citations, and no-write queries.
- [Writing](../skills/knowledge-base/references/writing.md): provenance, expected
  revisions, previews, idempotent retries, and multi-note changes.
- [Diagnose and repair](../skills/knowledge-base/references/repair.md): validation
  findings and explicitly reviewed incremental repair.
- [Maintenance](../skills/knowledge-base/references/maintenance.md): navigation,
  recovery, export, and backups.
- [Operational backups](backups.md): exact settled-state archives, new-only
  restore, exclusions, and local-staging limits. Export is not an operational
  backup; `.kb/state/` is not disposable cache.

## Optional integration

- [MCP guide](mcp.md): independent stdio setup, discovery, tool schemas, access
  boundaries, and workdir activation. MCP does not require the CLI skill.
- [Interface contract](interfaces.md) and [tool catalog](../schemas/kb.v1.tools.json):
  wire-level behavior for integration authors.

## Versions, evidence, and limits

- [Project status](../README.md#project-status) and
  [compatibility/support](compatibility.md): published and installed identities,
  supported version axes, upgrade paths, and qualified versus cross-build-only
  platforms.
- [Release operations](releases.md): packaging, verification, and explicit
  publication authorization; merging code does not publish a release.
- [Quality gates](quality.md) and [acceptance inventory](acceptance.json): enforced
  checks, observable criteria, evidence paths, and limitations.
- [Retrieval evaluation](qa/retrieval-quality.md): synthetic lexical judgments,
  citations, budgets, read purity, and CLI/MCP parity.
- [Performance](performance.md) and [large-corpus results](performance-large-corpus.md):
  dated local measurements, disclosed regressions, and reproduction methods;
  not portable SLAs.
- [v1.0 exit checklist](stability.md), [qualification](qa/v1-qualification.md), and
  [delivery](qa/v1-delivery.md): bounded readiness and publication evidence,
  including failed attempts and remaining limits.

## Understand and contribute

- [Project overview](project-overview.md), [specification](spec.md), and
  [architecture decisions](decisions/0002-v0.1.md): domain, adapters, persistence,
  trust, and design choices.
- [Roadmap](roadmap.md): implemented increments and explicit exclusions.
- [Contributing](../CONTRIBUTING.md): runtime checks, static site commands,
  synthetic-fixture policy, and PR expectations.
- [Security policy](../SECURITY.md) and [MIT license](../LICENSE).
