# Knowledge Base site design

## Intent

The landing is a quiet, readable introduction to a local Markdown knowledge base. Its signature connects a readable note to a cited terminal excerpt. The CLI and knowledge-base skill lead the installation flow; MCP remains a lower-page alternative link.

## Tokens

| Role | Value | Use |
| --- | --- | --- |
| Background | `#071724` | Page, terminal, code in tinted sections |
| Surface | `#102940` | Markdown source, install commands, workflow section |
| Accent | `#A8C7FF` | Main action, selected tab, source citations, focus |
| Text | `#ECF3FB` | Headings and primary content |
| Muted | `#A4B5C9` | Supporting text and file chrome |
| Border | `#2B415B` | Dividers and artifact boundaries |

The pinned Midnight blue palette keeps code and prose in the same tonal family. Do not add neon status colors, glow or gradient text. Status messages use text and explicit wording rather than color alone.

## Typography

Instrument Sans Variable is an agent design choice, not a human-selected font. Its compact counters and restrained 500-weight display keep the exact three-part headline readable at desktop and mobile sizes. The mechanical detector flagged it as frequently used; it is retained for the observed readable composition, not treated as a user-approved exception. IBM Plex Mono is self-hosted only for commands, Markdown, paths and workflow numbering, rather than using monospace as a general technical costume.

Desktop display: fluid 42–62px with 1.13 line height and `-.04em` tracking. Section headings: 32–43px, 1.15 line height. Body: 16px, 1.6 line height; hero description: 19px. Mobile headings remain 34px and body 16px. Actionable commands remain 13px across widths; essential source captions and copy feedback remain at least 12px. Overflowing commands scroll inside their own focusable blocks instead of shrinking or wrapping shell syntax.

Only the Latin font subsets needed for this English page are imported. Font licenses and attribution are shipped in `public/licenses/` and linked in the footer. Root MIT attribution is unchanged.

## Layout and components

- Container: max 1280px, desktop side margins 56px, intermediate 32px, mobile 20px.
- Major sections: 108px vertical padding desktop, 64px mobile. Adjacent heading and body remain tighter than unrelated sections.
- Hero: balanced columns. Markdown source shifts right; terminal overlaps its bottom edge and shifts left. At 760px and below, the promise, actions and source demonstration stack in reading order.
- Corners: 10px for source/terminal artifacts, 7px command blocks, 6px primary button. No generic card grid.
- `CopyBlock`: a label, explicit copy action, selectable code and live success/error feedback. Clipboard denial instructs manual copying; never report success before the browser promise resolves.
- Installation tabs: automatic activation with Left/Right/Home/End, roving tab stop, linked tabpanels. Without JavaScript both setup paths remain visible and text can be copied manually.
- Sequential workflow: numbers encode the actual search/read/context order. Benefits use open rows. Documentation links pair task titles with practical descriptions.

## Accessibility and interaction

Use semantic landmarks, one h1, ordered headings, descriptive links and decorative SVGs hidden from assistive technology. Every interactive control has a visible 2px accent focus outline with 5px offset. Keep primary actions at least 48px tall and mobile copy controls at least 44px tall. Copy controls retain focus and announce results with an atomic polite live region.

Smooth anchor scrolling is the only page motion. Reduced-motion preference disables it and transitions. No entrance animations, typewriter effects or fake live-terminal playback. Selection and scrollbars use the same palette. Assets are bundled locally; no analytics, external fonts or runtime fetches.

## Content rules

Documentation is canonical in the existing repository Markdown. V0 links from `#docs`; it does not duplicate a renderer. Illustrative source/terminal content must be labeled and must not imply captured output, usage statistics, performance guarantees or generated answers. Skill installation, root selection and workdir activation remain distinct actions.
