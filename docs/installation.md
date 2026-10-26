# Installation

Use this guide with the selected release's tag or checked-out revision, not as
proof that `main` is installed. The [compatibility policy](compatibility.md),
[README project status](../README.md#project-status) and public
[release records](https://github.com/matheus3301/kb/releases) centralize current
versions, qualification outcomes and native versus cross-build-only scope.
Binary installation, skill installation, MCP registration and workdir activation
are separate actions.

## Source checkout

Use Git and the Go toolchain declared in
[go.mod](https://github.com/matheus3301/kb/blob/main/go.mod):

```sh
git clone https://github.com/matheus3301/kb.git
cd kb
go install ./cmd/kb
kb version --json
```

Put Go's binary installation directory on your `PATH`. Source installations use
`dev` / `unknown` version defaults; release-built binaries embed the version and
source commit. Start with the [getting started example](../README.md#getting-started)
or the [full quickstart](quickstart.md).

Source checkouts may contain unreleased fixes. Compare installed version/help
with the selected release documentation; source `dev`/`unknown` defaults are
not exact released-artifact proof.

## Homebrew

The non-prerelease publication workflow maintains a source formula in
[matheus3301/homebrew-tap](https://github.com/matheus3301/homebrew-tap):

```sh
brew install matheus3301/tap/kb
kb version --json
```

Homebrew compiles the checksummed release source with Go as a build dependency.
The formula targets macOS and Linux without a cask or Gatekeeper changes. The
installed binary needs neither Go nor Homebrew at runtime. Platform claims need
actual installed evidence under the [compatibility policy](compatibility.md);
non-prerelease publication alone is not a support guarantee.

The configured release workflow first publishes assets, then runs read-only
published-archive verification on ubuntu-24.04 and macos-15. Both matrix arms
must succeed before the tap updates. Each executes its native package from a
temporary prefix and builds/exercises archived source with independent tag/source
expectations. This is not `brew install` or `brew test`; those remain separate
installed-channel checks. Configuration is not evidence of a completed release
job. See the [release flow and failure handling](releases.md#publish-a-new-version).

Upgrade and uninstall using the normal Homebrew commands:

```sh
brew update
brew upgrade matheus3301/tap/kb
kb version --json
```

For the metadata fidelity corrections, use **v0.1.3 or later**.
v0.1.1 can silently round decimal
metadata and discard YAML comments when updating references during a move.
v0.1.2 fixes those defects but can lose a trailing annotation when clearing
`verified`; the [installed-package regression](qa/verification-footer-regression.md)
documents its correction.
The [QA assessment and correction evidence](qa/native-homebrew-v0.1.1.md)
describe the affected cases and a disposable reproducer. Updating the executable
does not restore metadata already changed by an earlier operation; use original
files or backups to reconcile those changes.

Before upgrading, settle pending recovery with the prior binary and preserve
content, configuration and `.kb/state/` together. **v0.3.1 has no backup command**:
that origin needs an operator-reviewed backup procedure, not `export` alone.
With v0.4.0 or a compatible newer binary, use the [operational archive workflow](backups.md),
verify a fresh restore and retain archives on separate protected storage.
v0.4.0 has a reproduced [receipt-history scaling limitation](qa/backup-performance-v1.md);
the v1 candidate correction does not change persisted formats or expire receipts.
Never overlay old content on live newer journals/receipts or assume reverting
the executable proves safe state downgrade.

The selected qualification path is **v0.3.1 → v0.4.0 → v1.0.0**; retain each
actual prior executable and qualify each edge against the intended consumer.
The [upgrade recipe](qa/compatibility-v1.md) verifies exact content/config/receipt
fidelity and restores an archive actually produced by a capable prior version.
Earlier installed checks do not qualify a later package automatically.

To remove the executable:

```sh
brew uninstall kb
```

Uninstalling the executable does not delete your knowledge directory.

## Prebuilt binaries

The public [releases index](https://github.com/matheus3301/kb/releases) provides
tagged CGO-free Linux/macOS archives for amd64 and arm64, a source archive and
`checksums.txt`. Select the actual release linked by the
[compatibility policy](compatibility.md); do not guess a future release URL.
Binary archives include documentation, skill, MIT license and dependency notices.

Download the archive matching your operating system and architecture and the
checksum file from the same release. Verify the checksum before extracting:

```sh
# Linux
sha256sum --check --ignore-missing checksums.txt

# macOS: calculate the chosen archive's SHA-256 and compare its exact line.
shasum -a 256 kb_VERSION_darwin_arm64.tar.gz
```

`VERSION` stands for the actual version in the downloaded filename, without its
leading `v`. Extract the verified archive and place its `kb` executable in a
directory on your `PATH`. Binary archives do not install a skill, register MCP,
or change global agent configuration. Apple code signing and notarization are
not configured; Homebrew source installation is the recommended macOS package
channel.

To install a published Go module version instead, use the exact new tag from
that release with `go install github.com/matheus3301/kb/cmd/kb@TAG`. Do not reuse
withdrawn `v0.1.0`: Go proxies and the checksum database may retain it.
