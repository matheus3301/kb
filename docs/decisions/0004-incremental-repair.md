# 0004 — Actionable diagnostics and incremental repair

Status: adopted for v0.3.0.

Current status: this behavior remains implemented in the integrated source
preparing v0.4.0 qualification; published production remains experimental v0.3.1.
The [roadmap](../roadmap.md) and [exit checklist](../stability.md) distinguish
merged additions from candidate evidence. The decision below is unchanged.

An existing Markdown collection can contain more invalid notes than a bounded
transaction can repair. Requiring whole-base final conformance for every repair
makes agent-driven adoption impossible through the product interface.

Keep normal writes and default repair strict. Add explicit `incremental: true`
with `repair: true` to coordinated apply plans. Restrict this mode to patches of
existing files. Each changed file must conform, unchanged invalid files remain
byte-identical, and each batch must reduce blocking errors while any remain.
Validate the full readable inventory, all paths and changed dependency revisions;
never treat scan failures as repairable note errors or bypass transaction limits.
Index/log maintenance runs when the corresponding reserved files are valid;
otherwise defer it with a warning and preserve their original bytes. Normal
maintenance remains unavailable until the whole base conforms.

Persist transaction progress in the same durable receipt as publication and
replay. Counts describe that transaction, not future inventory. The caller
revalidates between batches. Existing journal/recovery/CAS and read-only rules
are unchanged; no embedded model or automatic conversion is introduced.

Validation exposes stable specific rules, severity, correction hints, readable
file revisions and known field/line locations while retaining existing diagnostic
codes/layers. It reports error/warning counts and clean separately from
conformant. Optional strict validation makes advisory warnings fail the CLI exit
criterion too. CLI and MCP consume the same requests/reports; the skill teaches
CLI operation, and MCP descriptions independently teach the repair loop.
