# Security

Do not put secrets or private knowledge content into issues, fixtures, logs,
or pull requests. Report vulnerabilities using this repository's private
vulnerability reporting facility on GitHub when enabled.

The application is not implemented yet. The security properties to test include
path confinement, symlink and traversal rejection, YAML resource limits,
read-only enforcement, protocol output isolation, and recovery after interrupted
writes. Details and acceptance criteria are in [the spec](docs/spec.md).

CI has no deployment credentials. Pull-request workflows use a read-only token,
checkout does not persist credentials, and external actions use commit SHAs.
There is no `pull_request_target` execution path for untrusted branch content.
