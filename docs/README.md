# Documentation

KB keeps knowledge in local Markdown files. Start with the CLI; add the
`knowledge-base` skill when you want your agent to use the same workflow.
The static website renders selected guides from these canonical Markdown files;
it is documentation, not a browser UI for a collection. Start with installation
and the quickstart, then choose CLI + skill or the independent MCP interface.

## Get started

- [Installation](installation.md): Homebrew, verified release archives, source
  installs, upgrades, and prerequisites.
- [Quickstart](quickstart.md): create, search, read, edit, and preview a coordinated
  plan using a disposable example base.
- [Agent installer guide](agent-install.md): self-contained instructions for an
  agent handling availability, choices, preview/apply and fresh-session checks.
- [Agent setup](agent-setup.md): version availability, explicit root,
  project/global scope, agents and learning, bundled skill or independent MCP,
  dry-run, check, fresh-session activation and safe uninstall.
- [Setup design](setup-design.md) and [host compatibility](setup-compatibility.md):
  binding/ownership/recovery contract, supported adapters and native limits.
- [Setup architecture by harness](setup.md): installed files, root routing,
  CLI/MCP alternatives, hooks and Mermaid installation/runtime diagrams for all
  ten adapters.
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

## Reuse OKF in Go

- [OKF package](../pkg/okf/README.md): parsing, validation and presentation-preserving
  editing in `github.com/matheus3301/kb/pkg/okf`, introduced in v1.2.0.
- [Supported format contract](../pkg/okf/CONFORMANCE.md): pinned OKF version,
  implementation limits, diagnostics and fidelity guarantees.

## Versions, evidence, and limits

- [Project status](../README.md#project-status) and
  [compatibility/support](compatibility.md): published and installed identities,
  supported version axes, upgrade paths, and qualified versus cross-build-only
  platforms.
- [Release operations](releases.md): packaging, verification, and explicit
  publication authorization; merging code does not publish a release.
- [Quality gates](quality.md) and [acceptance inventory](acceptance.json): enforced
  checks, observable criteria, evidence paths, and limitations.
- [Setup acceptance](setup-acceptance.md) and [S01–S44 inventory](setup-acceptance.json):
  installer criteria, factual source/native evidence and remaining qualification.
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
