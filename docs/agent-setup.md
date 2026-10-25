# Agent setup

The product is **KB — Knowledge Base**. Agents can use its CLI or a registered
stdio MCP server. The `knowledge-base` skill supplies task-specific operating
instructions. A workdir policy establishes when the agent should use them.

## Install the skill

Install the `kb` binary using [installation](installation.md), then run from
the agent's intended workdir:

```sh
npx skills add matheus3301/kb --skill knowledge-base
```

Skills installs into the project by default. Choose the intended agent in its
interactive installer, or specify it explicitly, for example:

```sh
npx skills add matheus3301/kb --skill knowledge-base --agent codex
```

The skill includes its own operational references. No repository checkout is
needed to read those references after installation. Node/npx is needed for this
installation channel, not to run `kb`. [Skills documentation](https://github.com/vercel-labs/skills/blob/main/docs/commands.md)
describes supported agents and scope options.

## Configure the Knowledge Base and consultation policy

For CLI use, supply `--root` or `KB_ROOT`, or work beneath an initialized
Knowledge Base. Do not assume an environment variable set in one terminal is
available to a separately launched agent or MCP client.

Add an instruction to the agent workdir's `AGENTS.md` or equivalent, preserving
existing instructions. Adapt this example to the intended scope:

```text
Use the knowledge-base skill to consult the configured KB_ROOT or associated
Knowledge Base MCP server before answering questions or making decisions its
contents may inform. Search and read existing notes first. Record reusable
learning with provenance and current revisions when writing is enabled and
authorized by this policy. A query with no new learning requires no write.
```

If consultation should cover every question, state that explicitly in the
workdir policy. Installing a skill alone does not establish that policy or
guarantee automatic invocation. An `AGENTS.md` inside the Knowledge Base is
optional; opening an external note does not load that directory's instructions.

## Register MCP

Use this generic stdio client entry, adapting the outer configuration to the
client's format:

```json
{
  "mcpServers": {
    "kb": {
      "command": "kb",
      "args": ["--root", "./example-kb", "mcp", "--read-only"]
    }
  }
}
```

The executable must be on the client's `PATH`. Use an absolute root when the
client cannot select the intended working directory. The server retains its
startup root and configuration; restart it after configuration or binary changes.
Remove `--read-only` only when deliberately enabling the authorized write workflow.
Read-only operation must not be bypassed by switching to a different transport.

MCP exposes `kb_status`, `kb_list`, `kb_search`, `kb_read`, `kb_context`,
`kb_validate`, `kb_create`, `kb_patch`, `kb_move`, and `kb_apply`. Read-only mode
hides the last four and rejects direct calls to them. CLI-only operations include
maintenance, recovery, export, and index rebuild; no corresponding MCP tool is
implied. `kb_read` with `include_links` provides a note's bounded neighborhood.

## Upgrade from knowledge-memory

Earlier releases shipped the skill as `knowledge-memory`. Its replacement is
`knowledge-base`; the CLI executable and wire schema remain `kb` and `kb.v1`.
The old name in dated research or consumer evidence describes that historical run.

Review customized old instructions first. To retire the old name for all agents
in the same project installation scope:

```sh
npx skills remove knowledge-memory
npx skills add matheus3301/kb --skill knowledge-base
npx skills list
```

If removing only selected agent bindings, a shared canonical copy can remain
while another agent uses it. Check the listing instead of assuming a successful
removal message means the old name is unavailable everywhere.

Use global-scope options only if the previous installation was global. Update
workdir instructions that name the old skill. Updating the binary alone does not rename an
already installed skill, install new instructions, or edit agent configuration.
Release archives from v0.1.3 and earlier retain their original skill name.

## Verify the integration

In a disposable Knowledge Base, ask the target agent to retrieve a known fact
with its source, update an existing note from supplied evidence, and repeat a
question without adding learning. Check actual revisions/provenance and that
the final query does not change notes or log. Also test a revision conflict and
the configured read-only mode. Verify observed behavior rather than assuming
that discovery or installation implies activation on every model/session.

See [the controlled skill validation](qa/knowledge-base-skill.md) for observed
installation/consumer results and the limits of that evidence.
