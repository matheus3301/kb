# 0006 — A separate static Astro website

Status: adopted for the landing and canonical documentation renderer. Deployment remains separate.

## Decision

Keep the marketing site in `site/`, separate from the Go runtime. Use Astro static output, TypeScript and small browser scripts for keyboard-operable installation tabs and clipboard feedback. No React/Solid runtime, backend, analytics or network-loaded assets are required. The no-UI runtime scope in the specification remains unchanged.

The default deployment base is `/kb/`, with `KB_SITE_BASE` as a build-time override. Import the existing root logo rather than maintaining another copy. Do not set a canonical site URL until a real publication URL is selected.

The original V0 linked documentation on GitHub. The documentation increment
adds `/docs/` without a second source tree: Astro's native `glob()` content
loader selects an explicit allowlist in `site/src/docs/catalog.mjs` from root
`docs/`, and `render()` emits static articles, headings and tables. Missing
selected sources fail generation. The overview renders `docs/README.md`, not a
second product introduction. CLI plus the `knowledge-base` skill leads the
navigation; MCP remains an independent secondary interface. Installing a skill
does not establish policy activation.

The reading surface retains Midnight blue, the shared mascot and self-hosted
fonts: 240px grouped sidebar, bounded article measure, narrow desktop TOC and
native mobile disclosure navigation. Articles and navigation work without
JavaScript. Small browser scripts add local search, keyboard selection,
focus-restoring dialogs and clipboard feedback; there are no account, AI,
rating, analytics or server-search controls.

Native Sätteri MDAST hooks remove raw HTML and image requests and rewrite links
before HTML generation. Fenced examples remain text. Malformed encodings,
unsafe schemes and encoded traversal become inert text, not fake `#` actions.
Published relative targets resolve to base-aware routes with queries/fragments
preserved; other repository targets link explicitly to GitHub. Native GitHub
slugging supplies stable heading anchors, including Unicode and duplicates.
Mermaid is labeled source text, not a rendered diagram.

Search publishes only selected document bodies and titles in a static JSON
index, fetched from the same origin on the first query. It uses case-insensitive
all-term matching with title ranking and excerpts. It is not semantic search
and does not search a user's collection. Markdown copying includes the canonical
body, not frontmatter. No raw Markdown downloads, evidence trees or local
filesystem metadata are emitted.

`site/Dockerfile` keeps its repository-root build context. It explicitly copies
the selected root documents; its matching Docker ignore allowlist excludes
evidence and unrelated source. Build assertions require docs HTML and search
JSON. Additions to the publication catalog must update both Docker input lists.
Use `docker build -f site/Dockerfile .`; there are no provider-specific settings.

The setup-lifecycle increment retains this static design and animation. The
human path presents `brew install` then `kb setup` with a visible source-only
availability warning. The agent prompt checks help/version and, only when source
is selected, builds a reviewed checkout into a separate executable without
changing the intended project cwd. It uses explicit root/scope/agents/learning,
dry-run/apply/check and fresh-session activation evidence instead of manual
configuration copies. Published v1.0.0 does not include setup; no merge or release
is implied by this landing update.

## Dependencies

Exact package versions are recorded in `site/package.json` and
`site/pnpm-lock.yaml`. Root `mise.toml` owns the current Node and pnpm pins;
`site/package.json`'s `packageManager` is a checked duplicate of the pnpm pin.

Use mise's explicit [`npm:pnpm` backend](https://mise.jdx.dev/dev-tools/backends/npm.html)
with the declared Node toolchain. The short `pnpm` key selects an Aqua backend
whose asset mapping in mise 2026.4.9 cannot install the selected pnpm release.
Changing only the backend preserves the tool version without requiring a mise
upgrade. npm is an internal tool installer; project development and CI use pnpm.

The pnpm migration preserves exact direct dependency declarations and retains
package integrity verification. A fresh resolution under pnpm's default release-age policy selects
`magic-string` 1.4.3 instead of the npm lock's newly published 1.4.4; no age
exception, audit exclusion, or direct dependency was added to force the result.
The resolver also omits an unused optional Vite peer and its platform packages.

- Astro 7.4.1: static HTML generation, asset processing and component authoring. Its Node `>=22.22.0` engine requirement constrains toolchain updates; Astro 7.4.2 requires a newer Node patch. The initial 6.4.8 selection was rejected after `npm audit` reported critical/high advisories; no audit suppression was added.
- `@astrojs/check` 0.9.12 and TypeScript 5.9.3: Astro and browser-script type checking.
- `@types/node` 22.x: types for the Node-based browser-test configuration; the exact patch is locked.
- `@playwright/test` 1.64.1 and `@axe-core/playwright` 4.14.0: real browser behavior, responsive layout and accessibility checks. They are development-only dependencies.
- `@fontsource-variable/instrument-sans` 5.3.0 and `@fontsource/ibm-plex-mono` 5.3.0: self-hosted fonts, licensed under OFL-1.1. The static output includes their license notices.
- `@astrojs/markdown-satteri` 0.4.4: explicit customization of Astro 7.4.1's
  already-selected native Markdown processor. No framework or Astro upgrade;
  this promotes the existing transitive version rather than adding a parallel
  parser. Current official documentation was checked through Context7 and the
  Astro/Sätteri references; pinned declarations govern API differences.

Official Astro configuration, imports and client-script documentation were consulted for the original decision, including Context7 resolution and documentation retrieval. Astro does not type-check during `build`; `pnpm --dir site run check` remains separate.

## Consequences

Site work adds Node build dependencies but none to the CLI or MCP runtime; neither requires mise or pnpm. Local setup uses `mise install`, followed by `mise exec -- make tools`. With mise activated, run `pnpm --dir site install --frozen-lockfile`, `pnpm --dir site audit --audit-level=high`, `pnpm --dir site run build`, `pnpm --dir site run check`, `pnpm --dir site exec playwright install chromium`, and `pnpm --dir site test` from the repository root; alternatively prefix each command with `mise exec --`. The test command is `playwright test`. Frozen installation rejects manifest/lockfile disagreement. Audit includes development dependencies and fails on high/critical advisories or audit-service errors, without exclusions. CI installs Chromium with `--with-deps`. Generated dependencies, build output and browser evidence are ignored. Local preview uses `http://127.0.0.1:4321/kb/`. Public `npx skills` setup commands are unchanged.

The browser suite uses `astro preview --ignore-lock` so the test runner owns a foreground server even when Astro detects an agent environment. Build-tool telemetry is disabled in the website CI job and `make site-check`, without changing global settings.

Website CI and `make site-check` explicitly build and test `/kb/`, then rebuild
and test `/`. `pnpm test` remains the single-base `playwright test` command.
Policy regressions reject removing or conditionally skipping either required
root step. Desktop/mobile checks cover every published route, all local
link/heading fragments, table containment, no-JavaScript reading, keyboard
navigation, real clipboard round trips, local search lifecycle and
serious/critical axe findings. Screenshots, traces and reports stay in separate
ignored `.build/site/kb/` and `.build/site/root/` trees.

Examples are labeled illustrative and do not claim benchmark results or user adoption. Source excerpts are not generated answers. The setup lifecycle is a separate explicit CLI feature; the site neither runs it nor configures agents. Browser verification and screenshots establish only the exercised site behavior, not universal accessibility or native agent adherence.
