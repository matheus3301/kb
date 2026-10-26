# 0006 — A separate static Astro marketing site

Status: adopted for the landing-page V0 implementation. Publication and deployment require separate authorization.

## Decision

Keep the marketing site in `site/`, separate from the Go runtime. Use Astro static output, TypeScript and small browser scripts for keyboard-operable installation tabs and clipboard feedback. No React/Solid runtime, backend, analytics or network-loaded assets are required. The no-UI runtime scope in the specification remains unchanged.

The default deployment base is `/kb/`, with `KB_SITE_BASE` as a build-time override. Import the existing root logo rather than maintaining another copy. Do not set a canonical site URL until a real publication URL is selected.

V0 has one page. Its documentation index links to canonical repository Markdown on GitHub, not a second renderer or copied documentation tree. CLI plus the `knowledge-base` skill is the primary setup path. MCP is an optional lower-page documentation link. The agent setup prompt points to the existing public setup guide and `llms.txt`, preserves existing bases/configuration, and requests explicit activation and verification. Installing a skill is not policy activation.

## Dependencies

Exact versions are recorded in `site/package.json` and `site/package-lock.json`:

- Astro 7.4.1: static HTML generation, asset processing and component authoring. Its Node `>=22.22.0` requirement supports the selected Node 22.22.3 environment; Astro 7.4.2 requires a newer Node patch. The initial 6.4.8 selection was rejected after `npm audit` reported critical/high advisories; no audit suppression was added.
- `@astrojs/check` 0.9.12 and TypeScript 5.9.3: Astro and browser-script type checking.
- `@playwright/test` 1.64.1 and `@axe-core/playwright` 4.14.0: real browser behavior, responsive layout and accessibility checks. They are development-only dependencies.
- `@fontsource-variable/instrument-sans` 5.3.0 and `@fontsource/ibm-plex-mono` 5.3.0: self-hosted fonts, licensed under OFL-1.1. The static output includes their license notices.

Current official Astro configuration, imports and client-script documentation were consulted, including Context7 resolution and documentation retrieval. Astro does not type-check during `build`; `npm run check` remains separate.

## Consequences

Site work adds Node build dependencies but none to the CLI executable. `npm ci`, `npm run build`, `npm run check` and `npm test` run from `site/`; the test command is `playwright test`. Generated dependencies, build output and browser evidence are ignored. Local preview uses `http://127.0.0.1:4321/kb/`.

Examples are labeled illustrative and do not claim benchmark results or user adoption. Source excerpts are not generated answers. No new demo corpus or automatic installer is introduced. Browser verification and screenshots establish only the exercised site behavior, not universal accessibility or agent adherence.
