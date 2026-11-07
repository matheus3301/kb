# v1.1.0 publication and installed setup

This record describes observed release and installation checks, not a new
universal host, model or capacity qualification. Machine paths, private source
notes and raw native sessions are intentionally absent from public evidence.
The [structured record](../evidence/setup-release-v1.1.0.json) retains artifact
identities and completed job timestamps.

## Publication

[PR 31](https://github.com/matheus3301/kb/pull/31) was squash-merged as
`8933c0e1e68318a94f2dfdd8a6fa989781e37301`, with a Conventional Commit subject.
The exact merged [main CI](https://github.com/matheus3301/kb/actions/runs/42414558125)
passed before the new annotated `v1.1.0` tag was pushed. No prior tag was reused.

[v1.1.0](https://github.com/matheus3301/kb/releases/tag/v1.1.0) was published on
**2026-11-07 at 21:25:23 UTC**. All five
[release jobs](https://github.com/matheus3301/kb/actions/runs/42415103752)
passed, including read-only native and archived-source verification on Linux
amd64 and macOS arm64. Both arms finished before the
[Homebrew tap update](https://github.com/matheus3301/homebrew-tap/commit/984416868c0bc80dac4d0305d80be0f3fb11041d)
at 21:26:21 UTC.

Anonymous downloads of all four binary archives and the source archive matched
the published SHA-256 checksums. Archive privacy/metadata inspection passed.
The extracted Darwin arm64 executable reported `v1.1.0` and the independent
expected full source commit; its SHA-256 is
`3bd5b2c430fefc2a9ea235e0d7d7d06f6c5bbe4c46b120031ef3c2f5d4bb56dd`.

## Actual Homebrew installation

A real **macOS arm64 Homebrew source upgrade from v1.0.0 to v1.1.0** and
`brew test matheus3301/tap/kb` passed. Installed version and full commit match
the release. The Homebrew executable SHA-256 is
`8d13dd6f6f950ad1908dda8313f3744201e7d00f8ca2eecc6dbef3e80e456aee`.
Source-built and prebuilt executables are distinct artifacts.

Against explicitly disposable synthetic collections, the installed binary passed:

- 14 CLI acceptance workflows, including revision conflicts, durable retries,
  coordinated edits, repair and original-source preservation.
- 43 executable data-documentation requests and 12 setup lifecycle requests,
  including dry-run/no-op preservation and missing-native-host diagnosis.
- Independent CLI/MCP operational workflows covering metadata fidelity,
  backup/restore, restored retry receipts and read-only rejection.

These are direct installed-binary observations, not a rerun of the earlier
long resource profiles or v1.0.0 model-consumer evaluation.

## Existing collection setup

An explicit dry-run was reviewed and applied using the Homebrew executable for
a user-global Codex CLI integration with `consult=always`, `learning=on` and
`hooks=auto`. The selected collection was already initialized. A prior-binary
operational backup was restored and validated separately before upgrade.

All six existing skill files matched the embedded bundle and were preserved.
Existing global instructions remained a byte-identical prefix; setup appended
its managed block, registered two native events and created its private registry.
Collection regular-file bytes remained unchanged, comparing content, cache and
operational state while excluding Git and backup/restore storage. No collection
migration, demonstration write, Git commit, push or synchronization occurred.
The global binding resolved correctly both within a repository and outside the
workspace, avoiding project-root discovery boundaries.

`setup check` returned `configured=true`, `ready=true` and
`activation_observed=false`. It correctly reports activation pending: local
inspection does not establish native trust, compatibility or model behavior.
No native hook trust store was changed and no trust bypass was used. Review
hooks through the native Codex interface before relying on automatic context
injection. CLI rules and skill discovery remain independently testable.

An initial ephemeral Codex 0.166.0-alpha.5.1 session in an unrelated temporary
directory loaded the global policy and skill and resolved the intended external
root, but `status` returned `ACCESS_DENIED` under a fully read-only filesystem
sandbox. It correctly reported consultation unavailable. This attempt remains
failed retrieval, not a successful source-read observation. The lock helper
opens `.kb/state/lock` read/write even for a shared reader lock; setup does not
silently grant external filesystem access. See the
[sandbox access guidance](../agent-setup.md#filesystem-sandbox-access).

A second fresh ephemeral session used normal `workspace-write` sandboxing with
an explicit external-collection directory grant. It loaded the global skill and
binding, ran `status`, search and a focused source read with `--read-only`, and
returned a current revision and matching physical line citation. Collection
bytes remained unchanged across setup and both sessions. This establishes one
bounded CLI-rule/skill activation observation; it does not establish trusted
hook execution. The first denied attempt remains recorded separately.

## Limits

The [setup compatibility matrix](../setup-compatibility.md) retains each host's
bounded evidence and unsupported combinations. Linux arm64 and macOS amd64
remain package/cross-build-only. Existing v1.0.0 consumer and resource evidence
retains its original identities in the [compatibility policy](../compatibility.md);
it is not automatically new v1.1.0 qualification.
