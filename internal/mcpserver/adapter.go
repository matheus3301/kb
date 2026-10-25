// Package mcpserver adapts fixed-root knowledge services to MCP and shared wire results.
package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/matheus3301/kb/internal/api"
)

// Reader is the read service consumed by both front ends.
type Reader interface {
	Root() string
	Config() api.Config
	Status(context.Context) (api.Status, error)
	List(context.Context, api.SearchRequest) (api.SearchResult, error)
	Search(context.Context, api.SearchRequest) (api.SearchResult, error)
	Read(context.Context, api.ReadRequest) (api.ReadResult, error)
	Context(context.Context, api.ContextRequest) (api.ContextResult, error)
	Links(context.Context, string, string) ([]api.Link, error)
	Validate(context.Context, string) (api.ValidationReport, error)
	Rebuild(context.Context) (api.Status, error)
	Export(context.Context, string) (api.ExportResult, error)
}

type Writer interface {
	Apply(context.Context, api.Plan) (api.Receipt, error)
	Maintain(context.Context, bool) (api.Receipt, error)
	Recover(context.Context, bool) (api.Receipt, error)
}

type Meta struct {
	Root       string `json:"root"`
	Generation string `json:"generation"`
	IndexStale bool   `json:"index_stale"`
}

type Envelope struct {
	SchemaVersion string           `json:"schema_version"`
	OK            bool             `json:"ok"`
	Data          any              `json:"data,omitempty"`
	Error         *api.DomainError `json:"error,omitempty"`
	Warnings      []api.Diagnostic `json:"warnings"`
	Meta          Meta             `json:"meta"`
}

// SingleInput is a single operation flattened alongside its transaction identity.
type SingleInput struct {
	api.Operation
	IdempotencyKey string `json:"idempotency_key"`
	Summary        string `json:"summary"`
	Actor          string `json:"actor"`
	DryRun         bool   `json:"dry_run,omitempty"`
}

type ValidateInput struct {
	Profile string `json:"profile,omitempty"`
	Strict  bool   `json:"strict,omitempty"`
}
type LinksInput struct {
	Path      string `json:"path"`
	Direction string `json:"direction,omitempty"`
}
type DryRunInput struct {
	DryRun bool `json:"dry_run,omitempty"`
}
type ExportInput struct {
	Destination string `json:"destination"`
}

// Adapter owns no mutable request state and can be used by concurrent sessions.
type Adapter struct {
	Reader   Reader
	Writer   Writer
	ReadOnly bool
}

func Invalid(message string) error { return &api.DomainError{Code: "INVALID_INPUT", Message: message} }

// Decode rejects unknown fields, nonobjects and concatenated documents before dispatch.
func Decode(raw []byte, dst any) error {
	if len(raw) > api.MaxPlanBytes {
		return Invalid("request exceeds 4 MiB")
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if raw[0] != '{' {
		return Invalid("request must be one JSON object")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var object map[string]any
	if err := d.Decode(&object); err != nil {
		return Invalid("invalid request JSON: " + err.Error())
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return Invalid("request must contain exactly one JSON document")
	}
	if hasNullControl(object) {
		return Invalid("request fields cannot be null; omit optional fields instead")
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	d.UseNumber()
	if err := d.Decode(dst); err != nil {
		return Invalid("invalid request JSON: " + err.Error())
	}
	return nil
}

// Null is meaningful inside extensible metadata. Request controls instead use
// absence for optional fields, matching their published nonnullable schemas.
func hasNullControl(v any) bool {
	switch value := v.(type) {
	case nil:
		return true
	case map[string]any:
		for name, child := range value {
			if name == "metadata" || name == "metadata_patch" {
				if _, ok := child.(map[string]any); ok {
					continue
				}
			}
			if hasNullControl(child) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if hasNullControl(child) {
				return true
			}
		}
	}
	return false
}

// ReadInput bounds file or stdin input before JSON decoding.
func ReadInput(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, api.MaxPlanBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > api.MaxPlanBytes {
		return nil, Invalid("request exceeds 4 MiB")
	}
	return b, nil
}

func domainError(err error) *api.DomainError {
	var de *api.DomainError
	if errors.As(err, &de) {
		return de
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return &api.DomainError{Code: "CANCELED", Message: "operation canceled", Retryable: true}
	case errors.Is(err, os.ErrNotExist):
		return &api.DomainError{Code: "NOT_FOUND", Message: "requested file was not found"}
	case errors.Is(err, os.ErrPermission):
		return &api.DomainError{Code: "ACCESS_DENIED", Message: "access denied"}
	default:
		return &api.DomainError{Code: "IO_ERROR", Message: "operation failed", Retryable: true}
	}
}

func Failure(root string, err error) Envelope {
	return Envelope{SchemaVersion: "kb.v1", Error: domainError(err), Warnings: []api.Diagnostic{}, Meta: Meta{Root: root}}
}

func IsWrite(action string) bool {
	switch action {
	case "create", "patch", "move", "archive", "apply", "maintain", "recover", "rebuild":
		return true
	}
	return false
}

// Call is the shared transport dispatch. Domain validation stays in the services.
func (a *Adapter) Call(ctx context.Context, action string, raw []byte) Envelope {
	root := a.Reader.Root()
	if err := ctx.Err(); err != nil {
		return Failure(root, err)
	}
	if a.ReadOnly && IsWrite(action) {
		return Failure(root, &api.DomainError{Code: "READ_ONLY", Message: "writes are disabled"})
	}
	data, err := a.dispatch(ctx, action, raw)
	if err != nil {
		return Failure(root, err)
	}
	// The returned Markdown or snippets already carry the requested content.
	// Strip full bodies from embedded note summaries to avoid duplicate payloads.
	switch v := data.(type) {
	case api.ReadResult:
		v.Note.Body = ""
		data = v
	case api.SearchResult:
		v.Hits = append([]api.Hit{}, v.Hits...)
		for i := range v.Hits {
			v.Hits[i].Body = ""
		}
		data = v
	}
	e := Envelope{SchemaVersion: "kb.v1", OK: true, Data: data, Warnings: []api.Diagnostic{}, Meta: Meta{Root: root}}
	// Metadata is advisory. A recovery or mutation result must not be replaced by
	// a failed follow-up status query after its effect has already committed.
	if s, statusErr := a.Reader.Status(ctx); statusErr == nil {
		e.Meta.Generation = s.Generation
		e.Meta.IndexStale = s.IndexStale
	}
	switch v := data.(type) {
	case api.SearchResult:
		e.Meta.Generation = v.Generation
		e.Warnings = nonNil(v.Warnings)
	case api.ContextResult:
		e.Meta.Generation = v.Generation
	case api.ValidationReport:
		e.Meta.Generation = v.Generation
	case api.Receipt:
		e.Meta.Generation = v.Generation
		e.Warnings = nonNil(v.Warnings)
	case api.Status:
		e.Meta.Generation = v.Generation
		e.Meta.IndexStale = v.IndexStale
	case api.ExportResult:
		e.Meta.Generation = v.Generation
	}
	limit := api.MaxPlanBytes
	if action == "read" || action == "context" {
		// Reserve framing space for MCP's structuredContent and compact text.
		limit = api.MaxReadBytes - 512
	}
	b, marshalErr := json.Marshal(e)
	if marshalErr != nil {
		return Failure(root, marshalErr)
	}
	if len(b) > limit {
		if action == "read" && boundRead(&e, limit) {
			return e
		}
		return Failure(root, &api.DomainError{Code: "RESPONSE_TOO_LARGE", Message: "response exceeds transport limit; request a smaller range or page"})
	}
	return e
}

// The core bounds its result before the adapter adds a root and generation.
// Shorten only the returned source excerpt when this envelope needs more room.
func boundRead(e *Envelope, limit int) bool {
	r, ok := e.Data.(api.ReadResult)
	if !ok {
		return false
	}
	original := r.Markdown
	truncated := false
	for _, d := range r.Diagnostics {
		if d.Code == "TRUNCATED" {
			truncated = true
		}
	}
	if !truncated {
		r.Diagnostics = append(r.Diagnostics, api.Diagnostic{Path: r.Note.Path, Code: "TRUNCATED", Message: "read reached the transport byte ceiling", Layer: "advisories"})
	}
	setExcerpt := func(n int) {
		if n < len(original) {
			for n > 0 && !utf8.RuneStart(original[n]) {
				n--
			}
		}
		r.Markdown = original[:n]
		r.LineEnd = r.LineStart + strings.Count(r.Markdown, "\n")
		if strings.HasSuffix(r.Markdown, "\n") {
			r.LineEnd--
		}
		e.Data = r
	}
	low, high := 0, len(original)
	for low < high {
		middle := low + (high-low+1)/2
		setExcerpt(middle)
		encoded, err := json.Marshal(e)
		if err != nil {
			return false
		}
		if len(encoded) <= limit {
			low = middle
		} else {
			high = middle - 1
		}
	}
	setExcerpt(low)
	encoded, err := json.Marshal(e)
	return err == nil && len(encoded) <= limit && r.Markdown != ""
}

func nonNil(d []api.Diagnostic) []api.Diagnostic {
	if d == nil {
		return []api.Diagnostic{}
	}
	return d
}

func (a *Adapter) dispatch(ctx context.Context, action string, raw []byte) (any, error) {
	switch action {
	case "status", "rebuild":
		if err := Decode(raw, &struct{}{}); err != nil {
			return nil, err
		}
		if action == "rebuild" {
			return a.Reader.Rebuild(ctx)
		}
		return a.Reader.Status(ctx)
	case "list", "search":
		var r api.SearchRequest
		if err := Decode(raw, &r); err != nil {
			return nil, err
		}
		if utf8.RuneCountInString(r.Query) > api.MaxQueryRunes || r.Limit < 0 || r.Limit > api.MaxHits {
			return nil, Invalid("query or page limit exceeds allowed bounds")
		}
		if action == "list" {
			return a.Reader.List(ctx, r)
		}
		return a.Reader.Search(ctx, r)
	case "read":
		var r api.ReadRequest
		if err := Decode(raw, &r); err != nil {
			return nil, err
		}
		return a.Reader.Read(ctx, r)
	case "context":
		var r api.ContextRequest
		if err := Decode(raw, &r); err != nil {
			return nil, err
		}
		if utf8.RuneCountInString(r.Query) > api.MaxQueryRunes || r.MaxTokens < 0 || r.MaxTokens > api.MaxContextTokens || r.MaxBytes < 0 || r.MaxBytes > api.MaxReadBytes {
			return nil, Invalid("query or context budget exceeds allowed bounds")
		}
		return a.Reader.Context(ctx, r)
	case "links":
		var r LinksInput
		if err := Decode(raw, &r); err != nil {
			return nil, err
		}
		if r.Direction == "" {
			r.Direction = "both"
		}
		return a.Reader.Links(ctx, r.Path, r.Direction)
	case "validate":
		var r ValidateInput
		if err := Decode(raw, &r); err != nil {
			return nil, err
		}
		if r.Profile == "" {
			r.Profile = "okf"
		}
		report, err := a.Reader.Validate(ctx, r.Profile)
		report.Strict = r.Strict
		return report, err
	case "export":
		var r ExportInput
		if err := Decode(raw, &r); err != nil {
			return nil, err
		}
		return a.Reader.Export(ctx, r.Destination)
	case "maintain", "recover":
		var r DryRunInput
		if err := Decode(raw, &r); err != nil {
			return nil, err
		}
		if a.Writer == nil {
			return nil, Invalid("write service unavailable")
		}
		if action == "maintain" {
			return a.Writer.Maintain(ctx, r.DryRun)
		}
		return a.Writer.Recover(ctx, r.DryRun)
	case "apply":
		var p api.Plan
		if err := Decode(raw, &p); err != nil {
			return nil, err
		}
		return a.apply(ctx, p)
	case "create", "patch", "move", "archive":
		var r SingleInput
		if err := Decode(raw, &r); err != nil {
			return nil, err
		}
		kind := action
		if kind == "archive" {
			kind = "move"
		}
		if r.Kind != "" && r.Kind != kind {
			return nil, Invalid("operation kind conflicts with command")
		}
		r.Kind = kind
		if action == "archive" {
			base := strings.TrimSuffix(a.Reader.Config().PARA.Archive, "/")
			if _, err := ResourcePath(ResourceURI(base + "/placeholder.md")); err != nil || base == "" {
				return nil, Invalid("configured archive directory must be a scoped relative path")
			}
			if r.To == "" {
				if _, err := ResourcePath(ResourceURI(r.Path)); err != nil {
					return nil, Invalid("archive requires a scoped source note path")
				}
				r.To = base + "/" + path.Base(r.Path)
			}
			if base == "" || !strings.HasPrefix(r.To, base+"/") {
				return nil, Invalid("archive destination must be inside the configured archive directory")
			}
		}
		return a.apply(ctx, api.Plan{Operations: []api.Operation{r.Operation}, IdempotencyKey: r.IdempotencyKey, Summary: r.Summary, Actor: r.Actor, DryRun: r.DryRun})
	default:
		return nil, Invalid("unknown operation")
	}
}

func (a *Adapter) apply(ctx context.Context, p api.Plan) (api.Receipt, error) {
	if a.Writer == nil {
		return api.Receipt{}, Invalid("write service unavailable")
	}
	if len(p.Operations) > api.MaxOperations {
		return api.Receipt{}, Invalid("at most 50 operations are permitted")
	}
	return a.Writer.Apply(ctx, p)
}

// ExitCode maps stable domain categories to the documented CLI process status.
func ExitCode(e Envelope) int {
	if e.OK {
		if v, ok := e.Data.(api.ValidationReport); ok && (!v.Conformant || v.Strict && !v.Clean) {
			return 1
		}
		return 0
	}
	if e.Error == nil {
		return 6
	}
	switch e.Error.Code {
	case "NOT_FOUND", "NOTE_NOT_FOUND", "SECTION_NOT_FOUND":
		return 3
	case "REVISION_CONFLICT", "IDEMPOTENCY_CONFLICT", "DESTINATION_EXISTS", "CASE_COLLISION":
		return 4
	case "PATH_OUTSIDE_ROOT", "ACCESS_DENIED", "READ_ONLY", "SYMLINK", "SYMLINK_NOT_ALLOWED", "WRITE_DISABLED":
		return 5
	case "IO_ERROR", "RECOVERY_REQUIRED", "CANCELED", "LOCK_FAILED", "RECOVERY_FAILED":
		return 6
	case "INVALID_FRONTMATTER", "VALIDATION_FAILED", "NONCONFORMANT", "INVALID_NOTE":
		return 1
	default:
		return 2
	}
}
