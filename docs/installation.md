# Installation

`kb` is experimental software for Linux and macOS. Use a stable version from
the [releases page](https://github.com/matheus3301/kb/releases), install through
Homebrew, or build the current source checkout. Binary, skill, and MCP setup
are separate steps.

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

## Homebrew

The stable release workflow publishes a source formula to
[matheus3301/homebrew-tap](https://github.com/matheus3301/homebrew-tap):

```sh
brew install matheus3301/tap/kb
kb version --json
```

Homebrew compiles from the checksummed release source using Go as a build
dependency. This supports macOS and Linux without distributing a cask or
changing Gatekeeper settings. The installed `kb` binary needs neither Go nor
Homebrew at runtime.

Upgrade and uninstall using the normal Homebrew commands:

```sh
brew upgrade matheus3301/tap/kb
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
