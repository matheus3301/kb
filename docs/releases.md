# Releases

Release automation uses GoReleaser **2.19.0**, pinned in `.tool-versions`.
`.goreleaser.yaml` builds four CGO-free binaries, packages the public docs and
licenses, creates a source archive, calculates SHA-256 checksums, and prepares
Conventional Commit changelog groups. The configuration was informed by
[LoopForge's setup](https://github.com/matheus3301/loopforge/blob/b025dce3a2a6fa1903b45475deca8d9f27a70dee/.goreleaser.yaml).

The earlier manual `v0.1.0` tag and release were deliberately removed. Go module
proxies and the checksum database may retain that version; **never reuse
`v0.1.0`**. The first automated publication was `v0.1.1`; it must not be moved
or reused either. Select a new, unused version for subsequent publications.
Merging a change does not publish a release.

## One-time tap credential

Create a fine-grained GitHub token restricted to **matheus3301/homebrew-tap**
with **Contents: read and write**. Store it as `HOMEBREW_TAP_TOKEN` in the
[kb repository's Actions secrets](https://github.com/matheus3301/kb/settings/secrets/actions).
Keep its value out of issues, source, logs, and agent conversations. A token
already stored in another repository cannot be retrieved from GitHub; supply
an existing suitable token from your secret manager or create a new one.

The normal `GITHUB_TOKEN` publishes assets only to `kb`. The tap token is used
only by the tap publication step and the preceding presence check. Stable
publication fails before creating a release when the tap secret is absent.
Prereleases do not need the tap token and never update the stable formula.

## Validate without publishing

Install the GoReleaser version in `.tool-versions` with your tool manager or the
[official installation instructions](https://goreleaser.com/install/).
Then run:

```sh
make tools
make check
make release-snapshot
```

`make release-snapshot` validates the config, builds all targets with
`--snapshot --skip=publish`, generates a Homebrew formula, and inspects actual
archives. It verifies checksums, normalized owners, public docs/licenses, source
identity, and formula identity; executes the extracted native binary against an
isolated synthetic base; and builds the archived source and exercises it too.
It does not install globally or write to GitHub or the tap. Outputs live under
`.build/dist/` and `.build/homebrew/`, both ignored by Git.

PR CI requires the release snapshot job as part of **CI gate**, alongside the
existing application checks. Unit regressions cover missing/failed exact-commit
CI, malformed versions, the retired tag, checksum tampering, extra Ruby in a
formula, wrong-commit artifacts, duplicate publication, and tap downgrades.

## Publish a new version

Publication is an explicit maintainer action under the current authorization.
A green branch build alone does not request a release.

1. Merge the intended change through a PR with a Conventional Commit title.
2. Wait for **CI gate** to pass on the exact merged `main` commit.
3. Ensure the tap secret is configured for a stable release.
4. Fetch current main and create a new, unused semantic version tag:

```sh
git switch main
git pull --ff-only
git tag -a v0.1.3 -m "Release v0.1.3"
git push origin v0.1.3
```

`v0.1.3` illustrates the patch release following v0.1.2. Verify that the chosen
version has never been published before creating its tag; these examples do not
create a release themselves. A prerelease such as
`v0.2.0-rc.1` publishes GitHub assets but skips the stable tap.

The [Release workflow](../.github/workflows/release.yml) runs three stages:

1. **Verify**: require a valid tag, reject retired `v0.1.0`, require the tagged
   commit to belong to `origin/main`, require the latest exact-commit **CI gate**
   from the GitHub Actions app to succeed, and rerun `make tools check`.
2. **Publish**: use the sole job-scoped `contents: write` token to run pinned
   GoReleaser. Generate a formula from the checksummed source archive and retain
   it as a workflow artifact for 30 days.
3. **Tap**: download that artifact from the same workflow run, reconstruct and
   verify its expected content against the tag/commit, and update only
   `Formula/kb.rb` in `matheus3301/homebrew-tap` using its expected existing SHA.

All checkout steps disable persisted credentials; actions use full commit SHAs.
The workflow runs only on version tag pushes. PRs, forks, ordinary main pushes,
and snapshot checks receive no publishing credentials or write permissions.
The narrow policy exception for the tag publisher is tested; it does not grant
write tokens to other CI jobs. Main branch protection is unchanged.

## Homebrew design

Homebrew installs `kb` from a checksummed GoReleaser source archive and compiles
with Go. The formula sets the same version and full commit as binary releases,
uses `std_go_args`, and tests version output plus initialization/validation.
Its automation commits use a public bot identity and Conventional Commit titles.

GoReleaser's legacy `brews` generator, used by the reference setup, is
[deprecated](https://goreleaser.com/resources/deprecations/#brews).
The replacement cask integration distributes macOS binaries. A standard
[source formula](https://docs.brew.sh/Formula-Cookbook) supports macOS and Linux
and avoids requiring an unsigned-binary quarantine workaround. Prebuilt
binaries remain available directly from GitHub releases.

## Recovery and verification

If only the tap job fails after a release exists, rerun **failed jobs** in that
workflow run. This reuses the formula artifact and does not rebuild or replace
published assets. Publication is idempotent for identical formula content;
a retry of an older version cannot downgrade a newer tap formula. Different
tag runs serialize their tap jobs. Unexpected external changes cause a SHA
conflict rather than a blind overwrite.

If artifact retention has expired, regenerate the formula from the published
source archive, exact commit, and its verified checksum in a reviewed maintenance
change. Do not replace published binaries just to repair a formula update.

After the first automated publication, verify anonymous downloads and hashes,
`kb version --json` against the release source commit, and a real
`brew install matheus3301/tap/kb` / `brew test matheus3301/tap/kb` on the intended
platforms. Snapshot validation does not replace a real installation test after
publication. Update public installation guidance to match observed results.
