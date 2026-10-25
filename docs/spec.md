---
type: Specification
title: KB — Knowledge Base specification
description: Normative specification v0.3 for the experimental Knowledge Base, OKF notes, maintenance, and CLI/MCP interfaces.
tags: [okf, para, cli, mcp, memory, specification]
status: stable
implementation_version: "0.1.3"
implementation_status: experimental
spec_version: "0.3"
sources:
  - id: evaluation
    resource: research.md
    title: Practical evaluation of candidate tools
  - id: okf
    resource: https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md
    title: Referenced OKF v0.2 specification
  - id: mcp
    resource: https://modelcontextprotocol.io/specification/2025-11-25/server/tools
    title: MCP tools contract
  - id: llm-wiki
    resource: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f
    title: LLM Wiki by Andrej Karpathy
generated:
  by: codex/gpt-6
  at: 2026-10-24T20:58:27Z
---

# KB — Knowledge Base specification

Specification v0.3, with implemented v0.1 interface clarifications. The initial design
was informed by five tool evaluations and the LLM Wiki pattern on 2026-10-24.
The project repository is https://github.com/matheus3301/kb.

**The v0.1 implementation remains experimental.** Use v0.1.3 or later for the
[metadata-preservation fixes](installation.md). The commands in this document are
implemented. The [v0.1 interface contract](interfaces.md), shared DTOs in
`internal/api/types.go`, and [implementation decision](decisions/0002-v0.1.md)
clarify the wire format and current choices. Normative requirements and all 26
acceptance criteria below remain the product contract; they are not universal
safety guarantees. [Validation evidence](evidence/validation.json) records tested
behavior, and [performance measurements](performance.md) disclose unmet targets.

## 1. Problem and expected outcome

An agent must be able to find context, read its source, record learning, and
continue in another session using OKF-conforming Markdown concepts. A user
must be able to open and edit the same files in any editor.

Memory also accumulates syntheses: incorporating a source or answering a
question can improve existing concepts. The product provides operations that
let an agent maintain this wiki; it is not limited to storing independent notes.

The tool must work without a UI, browser, cloud service, account, language model,
or persistent server. The same core serves a single-invocation CLI and an MCP
stdio process. A local cache is allowed and must always be rebuildable.

OKF defines the knowledge format; PARA is a configurable human organization
profile, separate from the representation core. Search, transactions,
concurrency, and protocols are application choices. They must not be presented
as requirements of the OKF standard.

### MVP scope

Lexical search, navigation, partial reading, creation, editing, moving,
validation, indexes, history, and context composition over a native OKF base,
with MCP integration. Compatibility with notes without frontmatter, wikilinks,
and migration of an existing collection are outside the MVP. It does not include
a UI, CRDTs, remote collaboration, automatic Git sync, execution of declared
computations, binary document ingestion, web search, or knowledge generation
through an embedded LLM.

Embeddings, remote HTTP, and multiple simultaneous bases are future extensions.

## 2. Fundamental rules

1. Markdown files are the source of truth. Deleting the cache does not lose memory.
2. Reading, searching, and validating never modify notes, Git, or agent configuration.
3. Installation and initialization do not register MCP, skills, or global configuration.
4. The tool does not select facts or invent sources or human review.
5. Note content is data; instructions within it do not grant permission.
6. The agent remains responsible for consulting and updating memory according to
   the workdir instructions and product skill. An `AGENTS.md` inside the base is
   not required. An MCP server alone does not guarantee this behavior.
7. The base accumulates reusable syntheses and relationships; a new source may
   require revising several existing concepts as well as creating a reference note.
8. Original sources preserve evidence; wiki notes record revisable interpretations.
   A synthesis never silently replaces its source material.

## 3. Base, scope, and configuration

Each invocation operates on one explicit root: `--root PATH` or `KB_ROOT`.
Without either, search for `.kb/config.toml` from the current directory through
its ancestors. Never guess a base by searching the entire home directory.
Include the resolved root in the result.

`kb init` is explicit and idempotent. It creates only local configuration and
missing files selected by the user; it refuses to overwrite existing indexes
or notes. `kb init --para` creates PARA directories when needed. It does not
require Git. For first init, an explicit root or `KB_ROOT` may name a directory
that does not yet exist. With neither, init uses the current directory rather
than ancestor discovery. The default content include is all Markdown (`**`),
with operational, dependency, and agent directories excluded; the scoped
include list below is an optional customization.

```toml
version = 1

[content]
include = ["00_Inbox/**", "01_Projects/**", "02_Areas/**", "03_Resources/**", "04_Archive/**", "06_Metadata/**"]
exclude = [".git/**", ".kb/**", "node_modules/**", ".claude/**", ".obsidian/**"]

[para]
inbox = "00_Inbox"
projects = "01_Projects"
areas = "02_Areas"
resources = "03_Resources"
archive = "04_Archive"

[write]
enabled = true
maintain_indexes = true
maintain_log = true

[search]
accent_folding = true
include_archives = false
include_deprecated = false
```

This example is an optional base profile, not an OKF-imposed taxonomy. Root
`index.md` and `log.md` also participate in navigation under their reserved rules.
`.kb/config.toml` holds local configuration; `.kb/cache/` contains only derived,
rebuildable data. `.kb/state/` holds the durable operation journal and receipts:
it is not cache and must not be removed during cleanup or recovery. Deleting
only the cache loses neither notes nor receipts. Deleting durable state can
lose request deduplication and compromise pending recovery. These operational
directories are excluded from an exported knowledge bundle; exporting knowledge
is not a backup of the installation's operational state. A scope configuration
does not make an entire repository conformant: the report identifies exactly
what it examined and excluded.

### Agent integration and skill distribution

The product distributes its skill at `skills/knowledge-base/SKILL.md` in the
code repository as `knowledge-base`. It teaches consulting
before answering, searching before creating, preserving provenance, reviewing
affected concepts, and applying changes with expected revisions. It contains no
fixed personal paths; its root comes from explicit integration configuration.

The workdir's `AGENTS.md`, or equivalent agent mechanism, establishes the duty
to use the skill and identifies the base. The consultation policy always belongs
in that entry instruction, without relying on automatic topic selection.
Installing a skill does not prove activation on every question. Integration must
be validated in the target agent, including with a base outside the workdir.

An `AGENTS.md` inside the knowledge base is optional and useful for working in
it directly. It is neither an OKF requirement nor a CLI/MCP dependency. Do not
assume an agent loads instructions from an external directory merely because
it read notes there. Base-specific conventions are documented/configured data;
search results do not gain authority over session instructions.

The current [Skills CLI documentation](https://github.com/vercel-labs/skills/blob/main/docs/commands.md)
uses `npx skills add matheus3301/kb --skill knowledge-base`; installation is
scoped to the project by default. Its [documented layout](https://github.com/vercel-labs/skills/blob/main/docs/creating-skills.md)
recognizes `skills/<name>/SKILL.md`. The skill is included in this repository;
installation still does not activate it automatically for every agent.

Documented setup separates installing the binary, installing the skill,
configuring the root, registering MCP if desired, and adding instructions to
the selected workdir. Node/npx is needed only for that installation channel,
not to run the CLI/MCP. The CLI can also be used without MCP. `kb init` remains
limited to the base. Integration setup is a separate, explicit action: preserve
existing instructions, show changes, and never implicitly register agents or
edit global configuration. Do not implement a custom skill installer in the
core. In v0.1, the [quickstart](quickstart.md) documents explicit binary and
skill installation, root selection, MCP configuration, and workdir instructions;
there is no combined automatic integration installer.

### Human organization and concept hierarchy

OKF does not prescribe an ideal tree by domain, type, or productivity method.
Section 3 of OKF permits producers to group concepts according to their
knowledge. The tool's core understands concepts, metadata, links, and indexes;
it does not require PARA. The `para` profile adds human-readable navigation
and classification.

A base using PARA should group each project, area, or topic cohesively, with
focused concepts and per-directory indexes. A possible layout, not an
already-performed reorganization:

```text
example-knowledge/
  index.md
  log.md
  01_Projects/
    index.md
    example-project/
      index.md
      context.md
      decisions/
        index.md
        backup-strategy.md
  02_Areas/
    index.md
    operations/
      index.md
      service-objectives.md
  03_Resources/
    index.md
    kubernetes/
      index.md
      backup-restoration.md
  04_Archive/
    index.md
```

The context, objectives, and decision files are concepts with appropriate
`type` values; indexes are reserved files. Links connect a project decision to
a reusable resource without duplicating content to fit two categories. The
tool also works with another folder profile without changing the OKF model.

## 4. OKF model

### Identity and paths

- Canonical Concept ID: root-relative path using `/`, without the `.md` suffix.
- No UUID, database, or `id` field is required. Existing additional fields are
  preserved. Moving a file changes its Concept ID.
- `path` identifies the file including its extension; `revision` is SHA-256 of
  all bytes read. The index hash is a separate field, `generation`.
- Accept existing names with spaces, Unicode, and uppercase letters. Do not
  automatically rename them to `kebab-case`; recommend that style for new files.
- Concept links use standard Markdown, relative to the document or absolute to
  the bundle root. Preserve anchors. Wikilink resolution is not required.
  Finding wikilinks may produce a portability advisory without implicit conversion.

### Parsing

Every new or revised note requires parseable YAML and a nonempty string `type`.
Unknown types and keys are valid. `title`, `description`, `sources`, `generated`,
`verified`, `status`, and `tags` are optional for OKF conformance.

`index.md` and `log.md` are reserved. Indexes do not receive concept frontmatter;
only the root index may declare `okf_version`. Validate the specific index and
log structures without requiring `type`. Log dates use `YYYY-MM-DD`.

Preserve the bytes of unchanged notes. Targeted body edits must leave
untouched frontmatter spans, comments, field ordering, and line-ending style
intact. Meaningful edits refresh `generated` metadata and substantive changes
remove active `verified`. Metadata
changes require a parser retaining presentation information or a patch
implementation preserving untouched fields. Whole-document serialization that
loses comments must not be the default. Duplicate YAML keys, excessive aliases,
oversized documents, and invalid UTF-8 must produce bounded diagnostics.

### Native OKF base

There is no `mixed` mode or legacy-note adapter. Invalid files are integrity
problems to diagnose, not a second supported format. Search indexes valid
concepts; status and validate enumerate all invalid files and mark the base as
degraded, never silently claiming full coverage. Diagnostic reading by path,
including `read --raw`, allows inspecting and repairing an invalid file without
presenting it as a valid concept.

Normal writes require a conformant base before and after the operation. An explicit
`apply --repair` plan permits fixing a degraded base with preview and revalidation;
it does not invent metadata or automatically convert notes. Migrating an
existing collection may be a separate task but does not constrain the core design.

Separate `conformance_errors`, `policy_findings`, and `advisories`. A broken
link or missing description, for example, is not a normative OKF error. A local
profile may elevate these findings while making the rule's origin explicit.

## 5. PARA and the memory lifecycle

PARA supports human reading and maintenance; a category derives from the path
and configured mapping without imposing folders or types on the OKF core.
`status` remains the OKF lifecycle (`draft`, `stable`, `deprecated`); project
progress uses a different key such as `workflow_status`. Archiving changes the
folder without implying that knowledge is false or `deprecated`.

Expected agent workflow:

1. Use `context` or `search` before answering; use `read` to confirm the source.
2. Search before creating to avoid duplicates; similar results are suggestions,
   never automatic merges.
3. Classify as project, area, resource, or archive. If uncertain, use Inbox with
   `type: Capture` rather than inventing a definitive classification.
4. Create or edit with a summary and real provenance; update indexes and log
   in the same operation.
5. Record decisions, reasons, corrections, and open questions. Do not archive
   full conversations by default or require a note per question when nothing was learned.

### Three responsibilities: sources, wiki, and instructions

Adopt the LLM Wiki pattern's distinction between original evidence, curated
knowledge, and maintenance rules. One possible layout for a new installation:

```text
workspace/
  AGENTS.md           # only if this directory is also an agent workdir
  sources/            # original snapshots, outside the OKF concept root
  example-knowledge/  # --root: OKF bundle; PARA organizes human navigation
    index.md
    log.md
    01_Projects/
    02_Areas/
    03_Resources/
    04_Archive/
```

The CLI writes only inside the configured bundle. It does not automatically
move or transform an existing collection into this tree. Raw Markdown in
`sources/` is not a concept missing OKF frontmatter: it is outside the validated
tree. Original sources are immutable snapshots; a source edit creates another
version while retaining the previous one. Do not store originals in disposable
`.kb/cache/`.

The agent uses its own reading tools to consult an explicitly supplied original
source. Web acquisition, downloads, OCR, and PDF extraction are outside the
core. Do not expand `kb_read` permissions to cross the root. Inside the bundle,
a source note may use `type: Source`, identify the original URL in `resource`,
and record a snapshot digest in an extension field. Other concepts refer to
that note through `sources[].resource`. `Source`, `Synthesis`, and `Question`
are suggested profile types, not mandatory OKF types.

### Incorporating knowledge

The agent executes the ingestion workflow with support from the tool:

1. Read the supplied source and identify its version/digest when a snapshot exists.
2. Consult indexes, search existing concepts, and inspect incoming and outgoing
   relationships before deciding which notes to create or update.
3. Produce a plan: source notes, affected concepts, syntheses, links,
   disagreements, and questions still lacking evidence. Each note to edit has
   an expected revision.
4. Apply the validated set in one `apply` operation, including index and log
   maintenance. Original sources are never in the writable file list.
5. Return a receipt listing created/changed concepts and preserved open questions.

Do not assume one source equals one page or that a summary replaces integration
with existing concepts. For conflicting sources, retain both provenances and
make the disagreement explicit; the agent does not fabricate consensus.

### Answers that become memory

`context` continues to return original excerpts. The agent produces the answer
and may persist reusable comparisons, decisions, and connections as concepts
linked to sources actually used. This is an explicit workflow write, without
automatically storing every question and answer in the log.

A synthesis may declare the extension `basis_revisions`, a list of
`{path, revision}` objects for local concepts on which it depends. A changed
revision makes the synthesis a review candidate. The core reports
`basis_outdated: true` without declaring its text false or rewriting it. This
signal is distinct from `stale_after` and is not an OKF-standard field. Moving
a concept updates known references and these paths in the same plan.

### Structural maintenance and semantic review

`validate --profile okf` checks format. `validate --profile health` adds
deterministic checks: broken links, orphans, pages missing from indexes, source
coverage, syntheses with changed bases, and temporal validity. Types that need
no external source can be configured; missing provenance does not prove falsity.
Health warnings do not block writes solely because they exist.

The agent analyzes contradictions, conceptual gaps, and research questions using
exposed context and relationships. The core does not claim BM25, link counts,
or an infallible LLM prove semantic consistency. A review produces a report with
evidence and suggestions; applying fixes is a separate operation. Do not start
web searches, review schedules, or edits merely because a diagnostic appeared.
Workdir instructions and the skill document authorized maintenance triggers.

## 6. CLI interface

Read commands offer `--json`. In JSON mode stdout contains exactly one JSON
document without progress messages, ANSI, or logs; stderr is for operational
diagnostics. MCP stdio stdout contains only protocol messages.

| Command | Purpose |
| --- | --- |
| `kb status` | Scope, integrity, counts, ignored files, and index freshness. |
| `kb list --para resources --limit 20` | Paginated navigation without dumping bodies. |
| `kb search QUERY --limit 10` | Search with snippets, lines, and metadata. |
| `kb read PATH --section HEADING` | Whole note or section with `revision`. |
| `kb context QUERY --max-tokens 2000` | Bounded context with sources and relationships. |
| `kb links PATH --direction both` | Resolved incoming/outgoing relationships to identify affected notes. |
| `kb create --input request.json` | Create a note at an explicit path; a front end may suggest a path first. |
| `kb patch PATH --input request.json` | Edit body and/or metadata with an expected revision. |
| `kb move PATH NEW_PATH --input request.json` | Move, check conflicts, and update known links. |
| `kb archive PATH --input request.json` | Shortcut for moving to the Archive category. |
| `kb apply --input plan.json --dry-run` | Validate and plan creations, patches, and moves; without dry-run, apply a transaction. |
| `kb validate --profile okf` | Normative diagnostics with separate policies. |
| `kb index rebuild` | Rebuild the search cache without changing notes. |
| `kb maintain --dry-run` | Show index, link, and log repairs; apply only with `--apply`. |
| `kb recover --dry-run` | Preview pending recovery; `kb recover` performs it explicitly. |
| `kb export DIR` | Export a validated scope snapshot with an inclusion/exclusion manifest. |
| `kb mcp` | Stdio server; `--read-only` removes and rejects write tools. |

`create`, `patch`, `move`, `archive`, `apply`, and `maintain` accept `--dry-run`.
The input file may be `-` for stdin. Do not interpolate Markdown content into
shell commands. Export is implemented in v0.1 with a validated snapshot and
inclusion/exclusion manifest.

CLI envelope, shared by the core:

```json
{
  "schema_version": "kb.v1",
  "ok": true,
  "data": {},
  "warnings": [],
  "meta": {"root": "./example-knowledge", "generation": "sha256:...", "index_stale": false}
}
```

The example root is illustrative; real results contain the resolved root.
Failure uses `ok: false` and `error: {code, message, details, retryable}` without
returning secrets or external file content. Exit codes: `0` success, `1` failed
validation, `2` usage/configuration, `3` not found, `4` revision conflict,
`5` access denied, `6` I/O/recovery. Domain error codes are stable
(`REVISION_CONFLICT`, `INVALID_FRONTMATTER`, `PATH_OUTSIDE_ROOT`, etc.).

## 7. Search and reading

v0.1 uses a local weighted lexical index with BM25 scoring by note and
section-aware excerpts, cached in-process with on-disk freshness checks.
Title, path, description, and tags have greater weight than body text.
Portuguese search must accept both `memoria` and `memória`, preserving original
text in results. Do not call embeddings or an LLM. Test tokenization of
identifiers and terms such as `s3`, `k8s`, and `crossplane`.

Filters: `para`, path prefix, `type`, tags, `status`, derived trust, `stale`, and
presence of sources. Values in one list are OR; different fields are AND.
Including archives and deprecated notes is opt-in, with exclusion indicators.
Expired notes remain available and marked; they do not automatically become
deprecated. Literal search and empty-query inventory have distinct documented
semantics; regex operators are not implicit.

Each hit contains `id`, `path`, `title`, `type`, `para`, `revision`, `score`,
`parse_status`, `okf`, and snippets `{heading, line_start, line_end, text}`.
Ties are ordered deterministically by path. Pagination uses a cursor bound to
the query and generation: an index change produces an explicit warning/restart.

`read` returns Markdown, frontmatter, revision, line ranges, and OKF indicators.
Section reading must handle repeated headings without silently picking one:
return section identifiers or an ambiguity error.

`context` packages original excerpts with sources and lines, not generated
answers. It respects a global byte limit and estimated token budget; identifies
the estimator, truncation, and omitted items. Optional relationships are limited
to one hop. Do not fill context with indexes and logs merely because they
contain many keywords.

External updates must become visible after a freshness check. No daemon is
required: an incremental cache uses a file inventory, mtime/size, and hashes.
Reading a note and every write check its current hash; a stale cache never
authorizes overwriting a different revision.

`links` returns incoming/outgoing edges with origin (Markdown or provenance),
target, location, and resolution state. It does not infer semantic relationships
from filenames. `read` may include this bounded neighborhood so MCP can support
the same operation without a separate tool.

## 8. Writing, concurrency, and recovery

Every write operation requires an explicit path. Creation requires a
nonexistent destination. `patch` and `move` require
`expected_revision`. Missing or mismatching revisions change no files. The
agent must reread and reapply; never blindly retry a whole-document replacement.

Example patch, also accepted through MCP:

```json
{
  "expected_revision": "sha256:...",
  "idempotency_key": "session-123/decision-cache-ttl",
  "summary": "Correct TTL after the expiration test",
  "actor": "process:agent-session",
  "body_edits": [{"old_text": "TTL of 300 seconds", "new_text": "TTL of 600 seconds"}],
  "metadata_patch": {"status": "draft"}
}
```

Each `old_text` must occur exactly once; zero or multiple matches fail. Section
or whole-body replacement still requires an expected revision. Metadata removal
uses the explicit `remove_metadata_keys` operation; do not give `null` an
implicit destructive meaning.

### Coordinated operations across notes

`apply` takes `operations`, `idempotency_key`, `summary`, and `actor`. Each item
uses the same grammar and preconditions as `create`, `patch`, or `move`: `kind`
is `create`, `patch`, or `move`; `path` is explicit, and moves use `to`. A
single-operation CLI input supplies operation fields alongside nonempty
`summary`, `actor`, and `idempotency_key`; the CLI wraps it into an apply plan.
Multi-operation plans require unique `operation_id` values; a single operation
may have its ID generated. `body` and `body_edits` are mutually exclusive.
Creation uses `metadata`; patches use `metadata_patch` and explicit
`remove_metadata_keys`. A complete-file `document` replacement is accepted only
by an explicit `repair: true` plan (`apply --repair`); repair permits an invalid
pre-state but requires a conformant final state. Initial limit: 50 operations and 4 MiB per plan.
`--dry-run` returns affected files, bounded diffs, diagnostics, and revisions
without writing.

Apply uses the entire plan as one logical transaction: a conflicting revision,
occupied destination, or invalid note cancels the set before publication.
Revalidate synthesis dependencies declared in the plan as well. Indexes and log
are derived from the transaction, not operations the agent must enumerate.
Receipts distinguish creations, changes, moves, and files kept unchanged.

References between notes created in one plan must resolve against the proposed
final state. Multiple operations on one file or overlapping destinations are
rejected in the MVP; the agent must consolidate patches. Moving and editing
one concept requires separate calls in the MVP, with an updated revision
between them. The isolation and recovery limits below also apply to `apply`;
do not promise an atomic transaction to external readers.

A logical operation includes the note, link changes, index regions, and log
entry. The core acquires a per-base lock, revalidates revisions, plans the diff,
and validates affected files before writing. It records a durable journal in
`.kb/state/`, writes temporary files on the same filesystem, synchronizes, and
replaces files by rename. After a crash, reads return `RECOVERY_REQUIRED` without changing notes.
`kb recover` explicitly recovers or rolls back; a new mutation recovers pending
state before accepting its request. Creating operational lock/cache files during
reads is permitted; notes, configuration, and Git remain unchanged. The cache publishes a new generation only
after success. Failures must state whether recovery is needed; never report
success for a partial application.

There is no atomic multi-file rename: external readers can observe a transient
state. Tool clients use the lock/generation. External editors that ignore the
lock remain subject to races; detect changes until the latest possible point
and document this limit without promising absolute isolation.

`idempotency_key` plus request hash identifies an effect. Repeating the same
request returns the previous result without duplicating notes or log entries;
reusing the key with another request fails. Store receipts in the journal and
reconstruct them during recovery so a crash between publication and response
does not duplicate effects. In v0.1 receipts do not expire. Request hashing
excludes `dry_run`; dry-run persists no receipt. Check an existing receipt
before retry preconditions so a successful retry can replay after revisions change.

The v0.1 journal is `.kb/state/pending.json`, with version, request digest/key,
prepared/committed state, before/after bytes, and receipt. It is atomically
written and fsynced before publication. Synchronize files and directory metadata,
then persist the durable commit marker. Recovery rolls back before that marker
and rolls forward after it, reconstructing receipts under
`.kb/state/receipts/` before durably removing the pending journal. Interrupted
recovery is itself idempotent; failures at every boundary require tests.

No implicit Git commit, pull, push, hook, or branch changes. Git is an optional,
separately operated layer. The MVP exposes no permanent deletion: archiving and
moving support the memory lifecycle.

## 9. Provenance, trust, and validity

- `sources` is a list of objects with `resource`; preserve unknown fields.
  External references are inert and never downloaded automatically.
- `generated.by` identifies the real actor; `generated.at` receives a timestamp
  with timezone on a meaningful change. Do not rewrite the timestamp on a no-op.
- `verified` accepts a list or a single mapping. Absent: unverified; only
  nonhuman actors: machine-confirmed; a `human:` actor present: human-reviewed.
- An agent write cannot create a claim of human review. Do not infer human
  identity from a token or actor parameter supplied by the model.
- A substantive change to the body or claims removes active `verified` and
  records the reason in the log. A human-review tool may be added later with
  confirmation outside the model's control. These controls are product policy
  beyond OKF's minimum requirements.
- `stale_after` follows the timezone-bearing ISO 8601 instant in the official
  revision pinned by this spec. Date-only values receive a diagnostic and must
  not silently become midnight in a guessed timezone. `--as-of` enables
  deterministic tests. Absence means unspecified validity.
- Attested Computation is read and statically validated; do not execute code,
  contact executors, or claim a receipt has been attested.

## 10. Indexes and log

Search uses the on-disk inventory and does not depend on a complete `index.md`.
Indexes support progressive navigation: direct files and links to subdirectory
indexes, with descriptions when available.

To preserve existing manual indexes, maintenance uses generated regions bounded
by HTML comments. Preserve allowed frontmatter, introductions, and manual links.
Taking over an entire index as generated requires `maintain --dry-run` and an
explicit takeover action. Detect duplicates between manual and generated
content. Use deterministic ordering and escape Markdown from titles.

The global log records a concise line per logical operation under the date in
UTC in v0.1, newest first, with links and summary. Metadata instants
still use explicit timezones. If an existing reserved file is invalid or repair
requires deleting manual text, the write fails before changing notes and offers
a repair proposal. Never replace the entire existing log.

An event type (`ingest`, `synthesis`, `maintenance`, or `edit`) may appear in
the line to aid reading. Preserve previous events; a correction is a new event.
Keep OKF's `YYYY-MM-DD` date sections even if the gist uses another format.
Do not create an event for every query without an effect.

## 11. MCP interface

Use the same core and DTOs as the CLI. Negotiate the version supported by the
official SDK; use MCP `2025-11-25` as the interoperability baseline demonstrated
in the experiments. Do not implement an ad hoc JSON-RPC variant. Future
transports must reuse the core and declare authentication requirements separately.

| Tool | Main input | Result |
| --- | --- | --- |
| `kb_status` | no arguments | Scope, index, and capabilities. |
| `kb_list` | folder/category, cursor, limit | Paginated inventory. |
| `kb_search` | query, filters, cursor, limit | Structured hits. |
| `kb_read` | path, section/range, optional relationships | Source, revision, and bounded neighborhood. |
| `kb_context` | query, filters, budget | Bounded citable excerpts. |
| `kb_create` | explicit path, type, body, metadata, summary, key | Receipt and revision. |
| `kb_patch` | path, expected_revision, edits, summary, key | Summarized diff and new revision. |
| `kb_move` | path, destination, expected_revision, summary, key | Mapping and changed links. |
| `kb_apply` | operations, summary, key, actor, dry_run | Plan or coordinated-application receipt. |
| `kb_validate` | scope and okf/health profile | Report with distinct layers. |

Closed input schemas, short descriptions, explicit limits. Do not expose a
generic shell or dozens of UI tools. In read-only mode, write tools disappear
from `tools/list` and remain blocked on direct calls.

Return `structuredContent` with the domain envelope and compact textual content
for compatible clients. Tool failure uses `isError: true` plus a domain code
rather than hiding `ok: false` within a protocol success. Base-validation errors
are report data, not necessarily tool-execution failures. Set `readOnlyHint`
and other annotations correctly without treating them as access control.

Expose paginated resources at `kb://local/...` URIs with percent-encoded paths
and revision in the result. This URI is a transport extension, not the OKF
Concept ID. The server is confined to its startup root: a model argument cannot
switch it to another directory. Prompts and subscriptions are not MVP requirements.

## 12. Operational limits

Reject out-of-root paths, `..`, absolute file-operation paths, symlinks, and
normalization escapes. Knowledge links outside the bundle may be shown as
references but never opened by the file layer. Handle filesystem case collisions
and existing destinations before moving.

Initial values: notes up to 1 MiB; reads up to 256 KiB per response; queries up
to 4,096 characters; 10 hits by default, maximum 100; context of 2,000 estimated
tokens by default, maximum 16,000 and a 256 KiB ceiling; limits also apply to
YAML depth and patch operation count. Configurable limits have documented ceilings.
These are application policies, not additional OKF format requirements. Scoped
source files over 4 MiB abort inventory explicitly; notes over 1 MiB are invalid.
Non-string YAML mapping keys produce policy findings for the JSON interface.
Explicit repair remains subject to operational size limits, including limits
on the existing file; it is not an unrestricted large-file repair facility.

Configuration is loaded when a process opens the base. CLI invocations reopen
it; restart a long-lived MCP process after editing `.kb/config.toml` to apply
changes to scope, search, and write permissions.

No telemetry or network access by default. Errors do not include the entire
environment, credentials, or out-of-scope files. Cache is local, written with
user-appropriate permissions, and does not change original file access policy.

## 13. Architecture and language

Layers: filesystem/scope → parser and model → inventory/search/relationships →
read and transaction services → CLI and MCP adapters. No business rule lives
only in an adapter. Model and parser do not depend on a running MCP server.

Adopted implementation language: **Go**, for binary distribution and nearby Go
references, as recorded in the foundation decision. Rust was a suitable design
alternative, but v0.1 uses Go. The experiments did not compare custom Rust and
Go implementations and do not prove either superior.

The runtime uses Cobra 1.10.2, the official MCP Go SDK 1.8.0,
BurntSushi/toml 1.6.0, go.yaml.in/yaml/v3 3.0.5, and x/text 0.43.0. Versions
are pinned in [go.mod](https://github.com/matheus3301/kb/blob/main/go.mod).
Composition uses explicit constructors and standard library diagnostics; Fx is not a dependency. Search is implemented in
the core without SQLite, CGO, embeddings, or a model runtime. The
[implementation decision](decisions/0002-v0.1.md) records tradeoffs and consequences.

Future dependency changes must be evaluated against current documentation and
the same revision, durability, confinement, and content-preservation contract.
The [research summary](research.md) records earlier candidates, not additional
runtime dependencies. Do not embed models weighing tens of megabytes in the
base binary.

## 14. Acceptance criteria and delivery plan

| ID | Required evidence |
| --- | --- |
| A01 | Search/read work without a UI, Git, model, credentials, or network. |
| A02 | CLI and MCP return the same IDs, revisions, filters, and errors on equivalent fixtures. |
| A03 | A minimal concept with only `type` conforms; unknown types and fields survive. |
| A04 | Notes without YAML or type are reported as invalid; normal writes reject a nonconformant base. Legacy search or migration is not required. |
| A05 | Invalid YAML never silently hides notes; status reports how many files could not be parsed. |
| A06 | A single mapping in `verified` is accepted; a content change invalidates active verification. |
| A07 | A timezone-bearing `stale_after` instant is interpreted correctly; incompatible formats are diagnosed. |
| A08 | Search with and without accents retrieves the same expected Portuguese result set. |
| A09 | Body edits preserve YAML comments and untouched fields; a no-op preserves bytes and produces no log entry. |
| A10 | Of two writes with the same revision, only the first applies; a conflict changes no note, index, or log. |
| A11 | A retry with the same key never duplicates effects, including after failure between publication and response. |
| A12 | Create/edit update the note, index, and log; failures injected at every stage permit recovery. |
| A13 | Move updates resolved Markdown links and anchors without improperly changing ambiguous or external references. |
| A14 | Traversal, symlinks, collisions, and out-of-root files permit neither reading nor writing. |
| A15 | `--read-only` hides and rejects mutations; MCP stdout contains no logs. |
| A16 | Context respects the global ceiling, reports truncation, and provides source lines/revisions. |
| A17 | External-editor changes invalidate cache and revision at the next relevant operation. |
| A18 | Export includes a report and exclusion list; it never claims conformance for unexamined files. |
| A19 | Installation/init never change global configuration or register agents automatically. |
| A20 | Replacing PARA navigation with another hierarchy does not change parsing, identity, or OKF validity of the same files. |
| A21 | A synthetic ingestion creates a source note and updates two existing notes through apply; a conflict in either publishes no changes. |
| A22 | Original snapshot hashes remain unchanged after wiki creation, editing, moving, and plan application. |
| A23 | A synthesis with basis_revisions for two concepts is flagged for review when either changes; it is never automatically rewritten or declared false. |
| A24 | Health identifies an orphan, missing link, concept absent from an index, and changed dependency without claiming deterministic contradiction detection. |
| A25 | An agent-persisted comparison retains provenance; repeating a query without learning creates no note or event. |
| A26 | CLI/MCP operate normally without AGENTS.md inside the base. In the target agent, workdir instructions and the installed skill lead to external-base consultation before answering; reusable learning is updated and a query without new information does not write. Record observed evidence without promising universal model adherence. |

The original delivery increments were: (1) parser/scope/validate/read;
(2) index/search/context and reference queries; (3) create/patch/apply/transactions/log/index; (4) move/archive, links,
and health; (5) MCP, ingestion/synthesis/review workflows, export, and
distribution. These capabilities are implemented in experimental v0.1;
subsequent changes still require behavior evidence. Workflows are
instructions for an external agent, with no LLM calls added to the core.

Measure on macOS arm64 and Linux amd64/arm64: binary size, RSS, cold start,
reindexing, and search p50/p95, declaring hardware and corpus. Initial targets
for 10,000 synthetic notes of 2 KiB each: warm search p95 ≤ 150 ms and RSS ≤
150 MiB; full reindex ≤ 10 s. These remain targets, not requirements implied by
OKF. The [recorded local benchmark](performance.md) on Apple M4/macOS arm64,
Go 1.27.3, 10,000 synthetic 2 KiB notes, and 20 warm samples measured 594.4 ms
cold core search, 162.9 ms warm search p95, and 180.3 MiB peak process RSS.
The warm-search and RSS targets were not met. This is not fresh CLI startup or
MCP transport latency; reindexing has not been independently benchmarked here.

### Implementation review — 2026-10-24

The v0.1 implementation includes the CLI, official SDK MCP stdio adapter, native
OKF parser and validation, local lexical search, context, revision-checked
writes, journal/recovery, maintenance, export, and installable skill. Versioned
`kb.v1` DTOs and the [interface contract](interfaces.md) settle operation shapes.
Explicit-path writes and unique IDs for multi-operation plans remain required.

The journal uses prepared/committed states and a durable commit marker. Recovery
rolls back before commit and rolls forward afterward; receipts do not expire.
Reads return `RECOVERY_REQUIRED` without changing notes; recovery is explicit
or precedes a new mutation. `apply --repair` permits an invalid pre-state but
requires a conformant final state within operational limits. First init selects
an explicit root, `KB_ROOT`, or the current directory. Workdir integration
remains explicit, with no required AGENTS.md inside the knowledge bundle.

[Recorded full validation](evidence/validation.json),
[CLI workflow evidence](evidence/cli-workflow.json), and the
[controlled consumer observation](evidence/consumer/consumer-evidence.json)
provide observed results. The consumer explicitly loaded the installed skill;
automatic activation across agents is not established. The
[acceptance inventory](acceptance.json) retains the authoritative criterion
labels and evidence references. Coverage and successful tests do not establish
absence of bugs, absolute isolation from external editors, or semantic truth.
Performance limitations above remain open. See the [project overview](project-overview.md)
for the product and architecture.
