package okf

import (
	"bytes"

	"go.yaml.in/yaml/v3"
)

// Version is the upstream format version targeted by this implementation.
const Version = "0.2"

// Specification is the immutable upstream specification reference, not a claim
// that every feature of that specification is implemented.
const Specification = "https://github.com/GoogleCloudPlatform/open-knowledge-format/blob/ad30107c31c06aec8a7d5636e0d1058118604e6f/SPEC.md"

// MaxDocumentBytes bounds parsing and editing to 1 MiB, including the body.
// This is an implementation limit, not an upstream OKF requirement.
const MaxDocumentBytes = 1 << 20

// MaxMetadataBytes bounds new or changed keys, strings and number literals
// rendered by Marshal/Edit. Unchanged parsed scalars do not consume this budget,
// but all supplied containers still undergo depth/node/cycle checks. Accepted
// output is limited to MaxDocumentBytes, independently of this rendering budget.
const MaxMetadataBytes = 4 << 20

// MaxDiagnostics bounds optional-field validation findings per document.
const MaxDiagnostics = 64

// Error describes an unsupported or malformed input. Code is machine-readable;
// Message is explanatory. Use errors.As to inspect it. Parsing errors do not
// establish whether a document outside the supported subset conforms upstream.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Kind distinguishes required concept structure from optional-field guidance.
type Kind string

const (
	// Requirement identifies required concept or reserved-file structure.
	Requirement Kind = "requirement"
	// Guidance identifies checks on optional upstream field conventions.
	// KB chooses to make these blocking; other consumers need not do so.
	Guidance Kind = "guidance"
)

// Diagnostic describes a concept validation finding. Code and Kind are suitable
// for branching; Message is descriptive. Field is a metadata field when known;
// Line is a physical one-based document line, or zero if unknown. Nested findings
// currently locate their top-level key rather than the exact nested scalar.
type Diagnostic struct {
	Code    string
	Message string
	Kind    Kind
	Field   string
	Line    int
}

// Document holds a parsed frontmatter mapping and an opaque Markdown body.
// It owns its input bytes and retains presentation for Edit. Treat Document as
// immutable: accessors return independent data and Edit does not mutate it.
// A zero Document is not parsed; its Validate and Edit report INVALID_DOCUMENT.
type Document struct {
	raw   []byte
	front frontmatter
}

// Parse reads one UTF-8 Markdown document beginning with a YAML mapping between
// standalone --- delimiters (LF or CRLF). It does not require type: call Validate
// for concept checks. This also allows applications to inspect root index metadata.
//
// The supported subset retains KB's existing limits: MaxDocumentBytes, nesting
// depth 32, 50,000 visited nodes, 16 alias traversals, no alias cycles, duplicate
// keys or non-string mapping keys. The last restriction is JSON interoperability,
// not an upstream OKF mandate. No filesystem access or Markdown execution occurs.
func Parse(raw []byte) (*Document, error) {
	if len(raw) > MaxDocumentBytes {
		return nil, fault("LIMIT_EXCEEDED", "note exceeds 1 MiB")
	}
	owned := bytes.Clone(raw)
	f, err := splitFront(owned)
	if err != nil {
		return nil, err
	}
	return &Document{raw: owned, front: f}, nil
}

// Bytes returns an independent copy of the original document, byte for byte.
func (d *Document) Bytes() []byte { return bytes.Clone(d.raw) }

// Body returns everything after the closing frontmatter delimiter, unchanged.
func (d *Document) Body() string { return string(d.front.body) }

// Metadata returns an independent tree of map[string]any, []any and scalar
// values. Integers and finite YAML decimals use encoding/json.Number without a
// binary-float conversion; timestamps are strings. YAML aliases are expanded in
// this view but retained in the document. Merge keys are unsupported and rejected
// by Parse. Other scalars follow the YAML decoder (including nonfinite floats).
func (d *Document) Metadata() map[string]any {
	if d.front.node == nil {
		return map[string]any{}
	}
	return nodeValue(d.front.node).(map[string]any)
}

// Edit specifies top-level metadata replacements/removals and an optional whole
// body replacement. Remove wins over Set. Nil Body leaves the body unchanged.
// Nested maps/lists replace values, not merge them; start with Metadata to retain
// unknown children. Values use Metadata's tree types; use json.Number for exact
// numbers. Edits do not add provenance, clear verification or enforce KB policy.
type Edit struct {
	Set    map[string]any
	Remove []string
	Body   *string
}

// Edit returns document bytes with only the requested changes. Semantic no-ops
// return exact original bytes. Unchanged top-level spans and scalar-only nested
// edits retain their bytes; structural edits re-encode affected entries, retaining
// unchanged child nodes, comments, styles and anchors where representable. This
// is not an arbitrary byte-preserving YAML transformation: edits that leave
// dangling aliases fail, and structural edits may change affected whitespace.
// The result is syntax-checked, not concept-validated; call Parse then Validate.
// If the assembled result fails syntax checks, the attempted bytes accompany
// the error for inspection; callers must not publish a result with a non-nil error.
func (d *Document) Edit(edit Edit) ([]byte, error) {
	if d.front.node == nil {
		return nil, fault("INVALID_DOCUMENT", "document is not parsed")
	}
	current := d.Metadata()
	if err := checkInput(edit.Set, current); err != nil {
		return nil, err
	}
	if edit.Body != nil && len(*edit.Body) > MaxDocumentBytes {
		return nil, fault("LIMIT_EXCEEDED", "note exceeds 1 MiB")
	}
	changes := map[string]any{}
	removes := map[string]bool{}
	for key, value := range edit.Set {
		old, exists := current[key]
		if !exists || !sameValue(old, value) {
			changes[key] = value
		}
	}
	for _, key := range edit.Remove {
		if _, exists := current[key]; exists {
			removes[key] = true
		}
		delete(changes, key)
	}
	originalBody := d.Body()
	body := originalBody
	if edit.Body != nil {
		body = *edit.Body
	}
	if len(changes) == 0 && len(removes) == 0 && body == originalBody {
		return d.Bytes(), nil
	}
	// Header editing mutates syntax nodes; use a fresh tree to keep d immutable.
	f, err := splitFront(d.raw)
	if err != nil {
		return nil, err
	}
	header, err := patchHeader(f, changes, removes)
	if err != nil {
		return nil, err
	}
	out := assemble(header, body, f.newline)
	if _, err := splitFront(out); err != nil {
		return out, err
	}
	return out, nil
}

// Marshal creates a document with LF delimiters and deterministically ordered
// metadata keys. It accepts Metadata's value types and syntax-checks the result,
// without requiring type or adding generated metadata. Use Document.Edit when
// preserving existing presentation matters; use Validate for concept checks.
// As with Edit, an invalid assembled result may accompany a non-nil error.
func Marshal(metadata map[string]any, body string) ([]byte, error) {
	if err := checkInput(metadata, nil); err != nil {
		return nil, err
	}
	if len(body) > MaxDocumentBytes {
		return nil, fault("LIMIT_EXCEEDED", "note exceeds 1 MiB")
	}
	node := &yaml.Node{}
	if err := encodeValue(node, metadata); err != nil {
		return nil, err
	}
	header, err := encodeNode(node)
	if err != nil {
		return nil, err
	}
	out := assemble(header, body, "\n")
	if _, err := splitFront(out); err != nil {
		return out, err
	}
	return out, nil
}

func assemble(header []byte, body, newline string) []byte {
	out := make([]byte, 0, len(header)+len(body)+8+2*len(newline))
	out = append(out, "---"+newline...)
	out = append(out, header...)
	out = append(out, "---"+newline...)
	return append(out, body...)
}
