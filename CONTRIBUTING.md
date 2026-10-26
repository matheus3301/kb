# Development

Install [mise](https://mise.jdx.dev/getting-started.html), Python 3.12 or newer,
Git, and Make. Root `mise.toml` is authoritative for development-tool versions.
`go.mod` and `site/package.json`'s `packageManager` are checked ecosystem
duplicates, not independent version sources. Development tooling is separate
from the standalone CLI and MCP runtime; neither requires mise, Node, or pnpm.

```sh
mise install
mise exec -- make tools
mise exec -- make check
```

`make check` runs repository validation, static analysis, race-enabled application
tests with at least 80% statement coverage, four CGO-free cross-builds, fuzz smoke
tests, secret/vulnerability scans, and the real CLI workflow. See
[quality policy](docs/quality.md) for exact gates and limitations.

`make repo` runs repository validation and tooling regression tests; Go-dependent
regression cases are skipped without a Go toolchain. Such a partial run does not
validate the application. `make lint`, `make test`, `make build`, `make fuzz`,
`make security`, and `make acceptance` expose individual checks. `make tools`
installs the Go tools into ignored `.tools/bin` using the pins in `mise.toml`;
`scripts/tools.json` supplies module mappings only. Generated `.build/` artifacts
are also ignored by Git. Activate mise in your shell or prefix later commands
with `mise exec --` to use the pinned tools.

For a focused build or local experiment:

```sh
go build -o .build/kb ./cmd/kb
make acceptance
```

## Static marketing site

The `site/` package (`@kb/site`) is separate from the Go runtime. It presents the
product and links to canonical Markdown documentation; it is not a browser UI
for a Knowledge Base. Site development uses Node and pnpm pinned in root
`mise.toml`. Use the committed `site/pnpm-lock.yaml`, not an ad hoc dependency
upgrade. This development/CI choice does not change public `npx skills` setup
commands.

From the repository root:

```sh
mise exec -- pnpm --dir site install --frozen-lockfile
mise exec -- pnpm --dir site run dev --host 127.0.0.1
```

Open `http://127.0.0.1:4321/kb/`. The default static base is `/kb/`; set
`KB_SITE_BASE` deliberately when checking another hosting base. This local URL
is not a claim of a deployed homepage. Installation fetches dependencies; the
page itself uses local/self-hosted assets and must not need a runtime network.
Import the existing brand assets rather than creating divergent copies.

Run the site checks from the repository root with mise activated (or prefix
each command with `mise exec --`):

```sh
pnpm --dir site audit --audit-level=high
pnpm --dir site run check
pnpm --dir site run build
pnpm --dir site exec playwright install chromium
pnpm --dir site test
```

The scripts are `dev`, `build`, `check`, and `test`; `test` invokes
`playwright test`. Browser installation is a separate local prerequisite.
Frozen installation fails if the manifest and lockfile disagree; update both
deliberately when changing dependencies. Audit includes development dependencies
and fails on high/critical advisories or audit-service errors; do not bypass
either failure. See the [pnpm install](https://pnpm.io/cli/install) and
[audit](https://pnpm.io/cli/audit) documentation.
Exercise the actual page, keyboard focus, responsive layout, and reduced motion.
Capture full-page review screenshots at desktop 1440×1000 and mobile 390×844;
keep generated browser reports and screenshot artifacts untracked. Do not use
synthetic popularity, users, or performance claims in site copy.

Site checks supplement rather than replace `make check`. Run the integrated
repository gates before opening a PR; do not weaken a runtime gate for a
documentation or marketing change.

## Repository contributions

Use only synthetic fixtures and public sources. Keep Markdown, comments,
examples, and descriptions in English; label deliberate multilingual search
fixtures as synthetic. Do not add private knowledge, credentials, or workstation
paths to tests, logs, or public evidence.

Open a PR describing the problem, resulting behavior, validation, affected
acceptance criteria, and limitations. Add regression evidence for reproducible
bugs. Read the [interface contract](docs/interfaces.md) before changing DTOs or
transaction behavior. Record architecture changes under `docs/decisions`.
Acceptance labels require existing test and evidence paths; coverage alone does
not establish conformance, isolation, durability, or semantic correctness.

Keep code and policy changes reviewable. Do not weaken checks as an incidental
part of implementing a feature. Changes to workflows, policy, and agent
instructions are covered by CODEOWNERS and the same required CI. Independent
approval count remains zero because the sole maintainer also authors PRs;
independent review is recommended, especially for recovery and policy changes.

Release publication starts with an explicitly requested new version tag after
required main CI. Use the GoReleaser version pinned in `mise.toml`, then
validate packaging locally without publishing:

```sh
mise exec -- make release-snapshot
```

This checks the GoReleaser config, builds four binary archives and a source
archive, verifies checksums/metadata/docs/licenses, generates the Homebrew
source formula, and exercises the extracted binary and archived source build.
PR CI requires this job as part of CI gate. It does not install globally,
create a tag, publish a release, or update the tap. See
[release operations](docs/releases.md) for the dedicated tag workflow and
required tap credential. Do not reuse the withdrawn v0.1.0 version.

There is no automatic merge. Performance changes must retain existing
correctness checks and report measurements using the
[documented benchmark](docs/performance.md).
