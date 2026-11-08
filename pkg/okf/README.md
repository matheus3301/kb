# OKF for Go

`github.com/matheus3301/kb/pkg/okf` is KB's shared format implementation, extracted
from its concept codec and metadata editor. It parses documents, exposes detached
metadata, reports structured findings, creates documents and preserves existing
presentation during targeted edits. It also checks supported reserved index/log
syntax. It does not import KB internals or start any runtime services.

**Introduced in KB v1.2.0.** Earlier v1.1.0 does not contain this package.
Install the versioned package with:

```sh
go get github.com/matheus3301/kb/pkg/okf@v1.2.0
```

To evaluate an untagged source checkout from another Go module, use a local
replacement (substitute your checkout path):

```sh
go mod edit -require=github.com/matheus3301/kb@v0.0.0
go mod edit -replace=github.com/matheus3301/kb=../kb
```

The placeholder requirement is resolved by the local replacement; it is not a
published library version. Use the Go version pinned by the checkout's
[`mise.toml`](https://github.com/matheus3301/kb/blob/main/mise.toml). No additional dependency or toolchain upgrade is
needed for the extraction.

## Read and edit

```go
package main

import (
    "fmt"
    "log"

    "github.com/matheus3301/kb/pkg/okf"
)

func main() {
    document, err := okf.Parse([]byte("---\ntype: Reference\ntitle: Original\n---\nBody\n"))
    if err != nil {
        log.Fatal(err)
    }
    for _, finding := range document.Validate() {
        fmt.Printf("%s %s at %s:%d: %s\n", finding.Kind, finding.Code,
            finding.Field, finding.Line, finding.Message)
    }
    out, err := document.Edit(okf.Edit{Set: map[string]any{"title": "Revised"}})
    if err != nil {
        log.Fatal(err)
    }
    fmt.Print(string(out))
}
```

See [compiling external-package examples](https://github.com/matheus3301/kb/blob/main/pkg/okf/example_test.go),
[behavior tests](https://github.com/matheus3301/kb/blob/main/pkg/okf/document_test.go)
and the [implementation contract](CONFORMANCE.md). Source-code links point to the
repository so this guide also works inside a binary archive without Go sources.

- `Version` / `Specification`: upstream v0.2 and an immutable source reference.
- `Parse`: bounded syntax parsing, not certification of a concept or bundle.
- `Document.Metadata`, `Body`, `Bytes`: detached values and original presentation.
- `Document.Validate`: required `type` and supported optional-field guidance.
- `Document.Edit`: top-level metadata set/remove and optional body replacement;
  no-op edits preserve exact bytes. Start nested edits from `Metadata()` to retain
  unknown children. Removal wins over setting the same key. A new nil value adds
  an explicit null; it does not mean removal.
- `Marshal`: create deterministic frontmatter with LF delimiters. Pass exact
  numeric values as `encoding/json.Number`, not binary floating-point numbers.
- `ValidateReserved`: pure validation for `RootIndex`, `Index` or `Log` roles.
- `Error`: inspect `Code` with `errors.As`; error prose is explanatory.
- `Diagnostic`: branch on `Code`/`Kind`, with `Field` and physical `Line` when known.

The only non-standard runtime dependency is the already pinned
`go.yaml.in/yaml/v3`. The package does not read files, confine paths, resolve
Markdown links, derive PARA categories, update provenance or verification,
search, execute computations, or publish transactions. The application controls
those operations and decides whether optional guidance is blocking.

`make okf-check` runs a real consumer from a temporary unrelated module and checks
`go list -deps` against the format dependency boundary. `make check` includes this
gate, external examples, parser/editor fuzz smoke, existing KB fidelity tests and
CLI/MCP acceptance. These checks support the documented subset, not universal OKF
conformance or attestation authenticity.
