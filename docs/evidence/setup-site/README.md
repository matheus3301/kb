# Setup website screenshots

Actual built-preview browser captures, visually reviewed on 2026-11-06.
The [manifest](manifest.json) records source identity, viewports and file hashes.
The [website evidence](../setup-site.json) records the automated browser checks.
These images show public onboarding, not proof of native agent activation.

- [Full desktop page](desktop-page.png)
- [Human setup, desktop](desktop-human.png)
- [Agent setup, desktop](desktop-agent.png)
- [Human setup, mobile](mobile-human.png)
- [Agent setup, mobile](mobile-agent.png)

The installation prompt is one source line (112 characters), rendered as two
desktop lines and three mobile lines at 390px. Text remains complete and uses
the unchanged 13px font with a 24.7px line height; no clipping or truncation.
Actual desktop/mobile prompt cards measured approximately 131/161 CSS pixels,
with zero horizontal page overflow. The full installer policy lives in the
linked Markdown guide; its redirected raw bytes matched the immutable committed
guide. Human screenshots were freshly captured and remain byte-identical because
their rendered content did not change.
