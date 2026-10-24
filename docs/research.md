# Research informing the specification

This is a public summary. The original working knowledge base and snapshots are
not part of this repository.

- [OKF specification](https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md):
  Markdown concepts, YAML metadata, source provenance, reserved indexes and logs.
  PARA is our navigation profile; it is not required by OKF.
- [Karpathy's LLM Wiki](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f):
  preserve original evidence and integrate new learning into connected concepts.
  The agent synthesizes; the core validates and applies deterministic operations.
- [Cobra](https://github.com/spf13/cobra) is the leading CLI adapter candidate.
  Viper, Afero, Fileflow and Pathologize are conditional options, not assumed
  solutions for YAML preservation, path confinement or crash durability.
- [Zap](https://github.com/uber-go/zap) is a valid structured logging option;
  start with standard library slog and explicit constructors. Fx/Dig are not
  needed for the current architecture. Diagnostic logging is not the journal.
- Existing OKF tooling was evaluated to compare headless use, MCP, metadata
  preservation, revisions and writes. Those observations are not benchmarks or
  validation of this unimplemented application.
- [Skills distribution](https://github.com/vercel-labs/skills/blob/main/docs/creating-skills.md)
  supports a `skills/<name>/SKILL.md` layout. Installation uses
  [skills add](https://github.com/vercel-labs/skills/blob/main/docs/commands.md).

Pin exact implementation dependencies only after checking current official
documentation and validating them against the acceptance criteria.
