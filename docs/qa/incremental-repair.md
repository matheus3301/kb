# Incremental repair verification

Existing collections can exceed the 50-operation transaction limit. v0.3.0
adds explicit incremental repair and actionable diagnostics so an agent can
work through the collection without making a separate compatibility format.

[Targeted regression evidence](../evidence/incremental-repair.json) covers a
53-note synthetic collection through real CLI handlers and MCP calls, preview
purity, repeated requests, final retrieval, strict health exits, rejected unsafe
plans, concurrent writers and interrupted publication/recovery boundaries.
Normal writes and default repair retain whole-base conformance requirements.

The [agent workflow](../../skills/knowledge-base/references/repair.md) teaches
reading invalid paths directly, supplying explicit corrections with current
revisions, previewing, applying, and revalidating between bounded batches.
MCP supplies equivalent guidance independently in initialize/tools discovery.
Diagnostic counts describe the examined scope; receipt progress describes its
transaction and can be historical on replay. Neither is semantic verification.

Incremental mode allows only existing-file patches, with every changed file
valid and a strict decrease in existing format/policy errors while any remain.
Untouched invalid files retain exact bytes. Missing inventory, invalid changed
files, moves/creates, stale revisions and read-only restrictions remain blockers.
Invalid reserved navigation is preserved and maintenance deferred explicitly.

A synthetic asset regression also checks that existing scoped JSON evidence is
not called a broken link. Missing, excluded, escaping and symlink targets remain
findings. Presence checks do not read or index assets, validate their contents
or fragments, or expand note-read permissions. Asset presence is checked afresh
even when Markdown generation has not changed.
