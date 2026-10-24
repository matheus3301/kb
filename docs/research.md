# Research informing the specification

This public summary describes design inputs, not evidence that `kb` is
implemented. No private source collection or snapshots are distributed here.

- [OKF specification v0.2](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md):
  Markdown concepts, YAML metadata, source provenance, reserved indexes and logs.
  PARA is an optional navigation profile; it is not required by OKF.
- [Karpathy's LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f):
  original evidence stays separate from a wiki that accumulates connected
  learning. In `kb`, the external agent synthesizes; the core validates and
  applies deterministic operations.
- [Cobra](https://github.com/spf13/cobra) is the leading CLI adapter candidate.
  Viper, Afero, Fileflow, and Pathologize are conditional options, not assumed
  solutions for YAML preservation, path confinement, or crash durability.
- [Zap](https://github.com/uber-go/zap) is a valid structured logging option;
  start with standard library slog and explicit constructors. Fx/Dig are not
  needed for the current architecture. Diagnostic logging is not the journal.
- Five existing tools were evaluated for headless use, MCP, metadata
  preservation, revisions, and writes. Those observations informed the spec;
  they are not reproducible public benchmark results or validation of this
  application. Future acceptance evidence must use synthetic public fixtures.
- [Skills distribution](https://github.com/vercel-labs/skills/blob/main/docs/creating-skills.md)
  supports a `skills/<name>/SKILL.md` layout. Installation uses
  [skills add](https://github.com/vercel-labs/skills/blob/main/docs/commands.md).
  The planned command is `npx skills add matheus3301/kb --skill knowledge-memory`,
  usable once the skill is published. Skill installation, binary installation,
  and MCP registration are separate actions.

Pin implementation dependencies only after checking current official
documentation and validating them against the acceptance criteria. The adopted
language is Go; the foundation decision and v0.1 interface contract govern
implementation choices, rather than these candidate recommendations.

## Public source and link audit — 2026-10-24

All existing Markdown documents were inspected for inline and reference-style
links, including frontmatter source URLs. Local links resolve inside this
repository. No reference-style link definitions currently exist. Public
repository links point to `matheus3301/kb`; upstream references point to public
author or project sources.

The pinned OKF v0.2 specification, MCP tools specification, Skills CLI command
and layout references, Cobra and Zap repositories, and this repository's public
entry and CI workflow were reachable during the audit. The author's canonical
LLM Wiki gist was reachable. Its old revision-specific page repeatedly failed
to load, so references now use the verified canonical author URL above. This
avoids depending on an inaccessible pinned-page URL while retaining attribution.
Upstream pages can change; this manual audit is not a network-dependent CI gate.
