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
