# Security

Do not put secrets or private knowledge content into issues, fixtures, logs,
or pull requests. Report vulnerabilities using this repository's private
vulnerability reporting facility on GitHub when enabled. Include a minimal
synthetic reproduction, affected version, and observed impact.

v0.1.0 is an experimental local application. Its tests exercise path confinement,
symlink/traversal rejection, case collisions, YAML resource limits, revision
conflicts, read-only enforcement, protocol output isolation, idempotent retries,
and interrupted-write recovery. See [recorded validation](docs/evidence/validation.json)
and the [contract](docs/interfaces.md). These observations do not establish the
absence of vulnerabilities or guarantee semantic truth.

Treat knowledge notes and imported sources as data rather than instructions or
permission for external actions. The core does not fetch linked resources,
execute declared computations, run a model, or synchronize Git. The external
agent remains responsible for its own tools and authorization. Original source
snapshots belong outside the writable knowledge root.

The MCP server uses stdio and stays bound to its startup root. Read-only mode
hides write tools and rejects direct mutation calls. Tool annotations are not
access control. Configuration is loaded per process; restart MCP after changing
`.kb/config.toml`. Installation and initialization do not register MCP or change
global agent instructions.

Writes use an advisory per-base lock and a durable prepared/committed journal.
Readers encountering pending recovery return `RECOVERY_REQUIRED`; explicit
recovery or a subsequent mutation resolves it. External readers can observe
intermediate multi-file states, and external editors do not obey the lock.
Recovery refuses to overwrite an unrelated external edit. Preserve `.kb/state/`:
its journal and receipts are durable state, unlike the rebuildable cache.

Resource limits are application policy, not requirements of OKF: notes over
1 MiB are invalid, scoped source files over 4 MiB abort inventory explicitly,
and read/context responses have a 256 KiB ceiling. Non-string YAML mapping keys
produce policy findings. Explicit repair remains subject to operational size
limits; it is not an unrestricted recovery tool or legacy migration facility.

CI has no deployment credentials. Pull-request workflows use a read-only token,
checkout does not persist credentials, and external actions use commit SHAs.
There is no `pull_request_target` execution path for untrusted branch content,
automatic merge, or automatic release. See the [quality policy](docs/quality.md)
for review settings and their limits.
