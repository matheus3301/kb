# 0003 — GoReleaser and a Homebrew source formula

Status: adopted for the next automated release; supersedes the manual packaging
and distribution decisions in [0002](0002-v0.1.md).

The initial CLI/MCP implementation is usable from source, but its manual tag and
release were withdrawn at the maintainer's request. Preserve the implementation,
validation evidence, and experimental limitations. Do not reuse v0.1.0 because
its Go module was already publicly resolved; cached module versions cannot be
made new by deleting a Git tag.

Use pinned GoReleaser 2.19.0 for binary/source archives, checksums and changelog.
Keep dependency notices, public documentation, SVG brand assets, and skill in
binary packages. Normalize archive owners and timestamps; strip build paths.
CI must exercise actual artifacts and the archived source without publishing.

Follow the existing matheus3301/homebrew-tap convention. Instead of adopting the
now-deprecated binary `brews` generator or a macOS-only unsigned cask, publish a
standard source formula for macOS and Linux. A small release helper verifies
GoReleaser's source checksum and renders the formula; GoReleaser owns packaging.
The Homebrew install requires Go during the build. The installed binary has no
runtime Go dependency. GitHub releases provide prebuilt binaries separately.

A dedicated tag workflow verifies protected-main ancestry and exact-commit CI,
reruns application gates, publishes GitHub artifacts, then updates the tap in a
separate retryable job. Keep CI read-only; allow job-scoped contents: write only
for this verified tag publisher, with regression checks for the exception.
Use a separate tap-scoped secret; never copy a developer's desktop login token
into CI or expose secret values in prompts. No new release is authorized merely
by changing the workflow or obtaining green checks.

See [release operations](../releases.md) and [installation status](../installation.md).
