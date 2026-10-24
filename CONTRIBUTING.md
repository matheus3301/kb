# Development

Use the Go version in `go.mod`, Python 3.12 or newer, Git, and Make. Development
tooling is separate from the eventual standalone binary.

```sh
make tools
make check
```

`make repo` runs repository validation and its regression tests without Go.
`make lint`, `make test`, `make build`, `make fuzz`, and `make security` expose
the individual CI checks. Tools are versioned in `scripts/tools.json` and
`.golangci-lint-version`; their local binaries are ignored by Git.

Open a PR with the problem, resulting behavior, validation, affected acceptance
criteria, and limitations. A speculative requirement is not an implemented
feature. For architecture changes, update or add a decision under `docs/decisions`.

Keep code and policy changes reviewable. Do not weaken checks as an incidental
part of implementing a feature. Changes to workflows, policy, and agent
instructions are covered by CODEOWNERS, with the same required CI as other PRs.

See [quality policy](docs/quality.md) for the exact gates, bootstrap behavior,
coverage threshold, and current limits of branch protection.
