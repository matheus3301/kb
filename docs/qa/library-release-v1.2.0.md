# v1.2.0 publication, installation and Go consumer

This record covers observed release artifacts and synthetic consumers. The
[structured record](../evidence/library-release-v1.2.0.json) preserves exact
identities and completed job timestamps. Private collection data, workstation
paths and raw sessions are absent from public evidence.

## Publication

[PR33](https://github.com/matheus3301/kb/pull/33) added the canonical docs portal;
[PR34](https://github.com/matheus3301/kb/pull/34) added `pkg/okf` and prepared its
versioned usage. Both were squash-merged with Conventional Commit subjects.
Release source is `a039eb5c070c70625942b340bbcc78679e88c244`. Its exact protected-main
[CI](https://github.com/matheus3301/kb/actions/runs/42432765160) passed before the
new annotated tag was pushed; no prior version was reused or moved.

[v1.2.0](https://github.com/matheus3301/kb/releases/tag/v1.2.0) was published at
**2026-11-08T01:20:41Z**. All five
[release jobs](https://github.com/matheus3301/kb/actions/runs/42433217642) passed.
Actual downloaded native archives and archived source ran on Linux amd64 and
macOS arm64 before the
[tap update](https://github.com/matheus3301/homebrew-tap/commit/5bd3e2dc926d86374ce6109797e18e0d63021204).
Both runner architectures were confirmed in their setup-go logs. Foreign
archives were inspected, not executed.

## Downloads and Homebrew

Anonymous downloads of all four binary archives and the source archive matched
the published SHA-256 checksums. Archive privacy/metadata and required package
guide/source-file inspection passed. The extracted Darwin arm64 executable
reported the expected version and full source commit; its SHA-256 is
`b43c46d683188251d29984a5da99bc1bd919010bb10f05b514b73df14b5d8c84`.

An actual **macOS arm64 Homebrew source upgrade from v1.1.0 to v1.2.0** and
`brew test matheus3301/tap/kb` passed. Installed identity matches the release;
its executable SHA-256 is
`9e4fd87190786e78d14eb059e3880c772b19ec23c5a64aa39ebcbc8ced002e78`. Source-built and prebuilt binaries are distinct artifacts.

The installed binary passed 14 CLI acceptance workflows, 43 data documentation
requests, 12 setup lifecycle requests and independent CLI/MCP operational
workflows, including backup/restore. A preserved v1.1.0 executable also passed
the declared prior-version compatibility workflow against the installed v1.2.0.
Every product collection used for these checks was disposable and synthetic.

## Published Go module

A separate module fetched `github.com/matheus3301/kb/pkg/okf@v1.2.0` with the
public proxy and checksum database enabled, without a local replacement. Its
external consumer passed parsing, exact numbers, validation, editing, explicit
nulls and reserved-file checks. All 13 Go package files matched the published
source archive byte for byte. Runtime dependencies remained standard library
plus the pinned YAML package. The module checksum is:

```text
h1:UnkGE4o++umn/2226sKssRjNdrq9eBmlSraVTa5CVac=
```

The first ordinary request failed with `404 unknown revision` from the official
proxy/checksum database. A version probe made before tagging had left a cached
negative response, observed to expire at **2026-11-08T01:28:39Z**. The failed
attempt remains a failure; the later checksum-verified retry passed. The tag
and assets stayed immutable, and checksum authentication remained enabled.
See the [publication procedure](../releases.md#verify-the-published-go-package)
for post-tag indexing and retry guidance.

## Scope

This release introduces the public Go package and a searchable canonical docs
portal. CLI + skill remains primary; MCP is an independent secondary interface.
Library conformance limits remain in the
[package contract](../../pkg/okf/CONFORMANCE.md).

These checks do not rerun the earlier long resource profiles, model-consumer
qualification, native hook trust or fresh-session activation. Earlier records
retain their original identities. Linux arm64 and macOS amd64 remain
package/cross-build-only; no universal durability or authenticity guarantee is
inferred from coverage or parser success.
