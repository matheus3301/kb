# Installation

`kb` remains experimental. Published production currently provides **v0.3.1**;
integrated source is preparing **v0.4.0 qualification** with all four roadmap
increments, including backup/restore absent from v0.3.1. No v0.4.0 publication or
stable-v1 result is claimed here. Use the
[releases page](https://github.com/matheus3301/kb/releases), Homebrew, or a source
checkout; binary, skill, MCP setup and workdir activation are separate steps.

[v0.3.1 installed-package verification](https://github.com/matheus3301/kb/releases/tag/v0.3.1)
covers real Homebrew source installation/formula testing, extracted binary and
installed CLI/MCP workflows on **macOS arm64 only**. All four Linux/macOS
amd64/arm64 archives are built; this is not installed runtime proof for the
other three targets. The selected stable-v1 qualification channels are macOS
arm64 Homebrew and Linux amd64 native archive/archived source; those candidate
results remain pending. Linux arm64/macOS amd64 remain cross-build/package-only.
See [compatibility/support](compatibility.md) and [v1.0 exit criteria](stability.md).

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

Current source includes merged additions that the installed release may lack.
Use source help and the [backup contract](backups.md) only with a binary exposing
those commands; source `dev`/`unknown` is not exact released-artifact proof.

## Homebrew

The stable release workflow publishes a source formula to
[matheus3301/homebrew-tap](https://github.com/matheus3301/homebrew-tap):

```sh
brew install matheus3301/tap/kb
kb version --json
```

Homebrew compiles the checksummed release source with Go as a build dependency.
The formula targets macOS and Linux without a cask or Gatekeeper changes; actual
installed-package evidence above is macOS arm64 only. The installed binary needs
neither Go nor Homebrew at runtime. A non-prerelease publication is not itself a
stable-v1 support guarantee.

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
content, configuration and `.kb/state/` together. v0.3.1 has no backup command:
use a reviewed operator backup procedure, not `export` alone. For qualified newer
source use the [operational archive workflow](backups.md), verify a fresh restore
and copy retained archives off disk. Never overwrite a live newer base with old
content/receipts or assume reverting a binary proves safe state downgrade.
The intended v0.3.1 → v0.4.0 → v1.0.0 path still needs exact synthetic upgrade,
Homebrew/formula and CLI/MCP evidence under V06/V07 before being marked qualified.

To remove the executable:

```sh
brew uninstall kb
```

Uninstalling the executable does not delete your knowledge directory.

## Prebuilt binaries

The [releases page](https://github.com/matheus3301/kb/releases) contains
CGO-free Linux/macOS archives for amd64 and arm64, a source archive, and
`checksums.txt`. Binary archives include documentation, skill, MIT license, and
third-party notices.

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
