# Hook failure qualification

The Pi/OMP generated extension now sends `SIGKILL` at its existing 3,500 ms
subprocess timeout and on output-buffer overflow. The previous default `SIGTERM`
could leave a child that ignored that signal running, keeping the callback and
native event pending indefinitely. The wrapper executes the KB process directly,
without a shell; this is not a general descendant-process supervisor or a hard
real-time guarantee.

`TestNativeExtensionProcessFailures` runs the generated extension with real child
processes. Before the fix, the ignored-`SIGTERM` case exceeded the independent
8-second test deadline. After the fix it returned unavailable context after
3,510 ms and the child was gone. Nonzero exit (even with valid success-shaped
stdout), malformed JSON and oversized output produce unavailable context;
valid no-hit context remains a successful empty retrieval, not a process failure.
These Node runtime fixtures do not establish native host event delivery.

The regression requires the repository-pinned Node development tool. A missing
Node executable fails with setup instructions; it is never counted as a skipped
qualification pass. The installed KB binary has no new Node dependency.

Native qualification and its exact host/build boundary are recorded in
[evidence/setup-hook-qualification.json](evidence/setup-hook-qualification.json).
Artifact presence and `setup check` local `ready` status are not native trust,
version support or observed activation. Hooks disabled by the native host remain
disabled; setup does not alter trust settings or install another execution path.

## Observed native scope

OMP 18.16.0 and 18.17.0 on Darwin arm64 passed project CLI/portable dispatch with
hooks required and learning off. Actual native processes loaded the generated extension:

| Scenario | Observed result |
| --- | --- |
| Native `--no-extensions` | Zero hook-helper invocations or KB context messages; inspection stayed activation pending. |
| No lexical hit | Actual KB retrieval delivered a 730-byte no-hit context, not invented evidence. |
| Failed process | Synthetic helper exited 7 with success-shaped stdout; native delivery was a 107-byte unavailable context. |
| Timeout | Synthetic helper ignored `SIGTERM`; generated wrapper terminated it and native delivery was a 107-byte unavailable context. |

All four native turns on each build completed and settled, all collection file
bytes remained unchanged, and every native process teardown confirmed no
survivors. The official checksum-verified 18.16.0 executable used the matching
SDK and the parent combined candidate; its hook helper processes were also
confirmed gone before native teardown. The same 18.16.0-source Python SDK was
exercised empirically with 18.17.0, not relabeled as a matching-version run.
A controlled failing helper establishes wrapper failure handling, not that the
actual KB binary hung or failed.

Unknown agent and unsupported Cursor global installation also returned
`INVALID_INPUT`/exit 2 without changing fixture files or silently promoting scope.
The unknown-build inspection regression confirms that mere executable presence
does not run the host or change `activation_observed` to true. Inspection reports
native version/platform compatibility as unobserved; these observations do not
qualify other operating systems, native builds or trust-policy combinations.
