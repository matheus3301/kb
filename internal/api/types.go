// Package api defines the shared, transport-independent v1 contract.
package api

import (
	"context"
	"time"
)

type DomainError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	Retryable bool   `json:"retryable"`
}

func (e *DomainError) Error() string { return e.Code + ": " + e.Message }

type Config struct {
	Version int `json:"version" toml:"version"`
	Content struct {
		Include []string `json:"include" toml:"include"`
		Exclude []string `json:"exclude" toml:"exclude"`
	} `json:"content" toml:"content"`
	PARA struct {
		Inbox     string `json:"inbox" toml:"inbox"`
		Projects  string `json:"projects" toml:"projects"`
		Areas     string `json:"areas" toml:"areas"`
		Resources string `json:"resources" toml:"resources"`
		Archive   string `json:"archive" toml:"archive"`
	} `json:"para" toml:"para"`
	Write struct {
		Enabled         bool `json:"enabled" toml:"enabled"`
		MaintainIndexes bool `json:"maintain_indexes" toml:"maintain_indexes"`
		MaintainLog     bool `json:"maintain_log" toml:"maintain_log"`
	} `json:"write" toml:"write"`
	Search struct {
		AccentFolding     bool `json:"accent_folding" toml:"accent_folding"`
		IncludeArchives   bool `json:"include_archives" toml:"include_archives"`
		IncludeDeprecated bool `json:"include_deprecated" toml:"include_deprecated"`
	} `json:"search" toml:"search"`
}

type Diagnostic struct {
	Path     string `json:"path,omitempty"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Layer    string `json:"layer"`
	Line     int    `json:"line,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Severity string `json:"severity,omitempty"`
	Field    string `json:"field,omitempty"`
	Hint     string `json:"hint,omitempty"`
	Revision string `json:"revision,omitempty"`
}

type Note struct {
	ID            string         `json:"id"`
	Path          string         `json:"path"`
	Title         string         `json:"title"`
	Description   string         `json:"description,omitempty"`
	Type          string         `json:"type"`
	Tags          []string       `json:"tags,omitempty"`
	Status        string         `json:"status,omitempty"`
	PARA          string         `json:"para,omitempty"`
	Revision      string         `json:"revision"`
	Metadata      map[string]any `json:"metadata"`
	Body          string         `json:"body,omitempty"`
	Trust         string         `json:"trust"`
	Stale         bool           `json:"stale"`
	BasisOutdated bool           `json:"basis_outdated"`
	Raw           []byte         `json:"-"`
}

type Link struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Anchor   string `json:"anchor,omitempty"`
	Text     string `json:"text,omitempty"`
	Kind     string `json:"kind"`
	Line     int    `json:"line"`
	Resolved bool   `json:"resolved"`
	External bool   `json:"external"`
}

type Snapshot struct {
	Generation  string            `json:"generation"`
	Notes       []Note            `json:"notes"`
	Diagnostics []Diagnostic      `json:"diagnostics"`
	Files       map[string][]byte `json:"-"`
	Links       []Link            `json:"links"`
	Degraded    bool              `json:"degraded"`
}

type Filters struct {
	PARA              []string `json:"para,omitempty"`
	Prefix            string   `json:"prefix,omitempty"`
	Types             []string `json:"type,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Statuses          []string `json:"status,omitempty"`
	Trust             []string `json:"trust,omitempty"`
	Stale             *bool    `json:"stale,omitempty"`
	HasSources        *bool    `json:"has_sources,omitempty"`
	IncludeArchives   bool     `json:"include_archives,omitempty"`
	IncludeDeprecated bool     `json:"include_deprecated,omitempty"`
}

type SearchRequest struct {
	Query   string  `json:"query"`
	Filters Filters `json:"filters,omitempty"`
	Limit   int     `json:"limit,omitempty"`
	Cursor  string  `json:"cursor,omitempty"`
	Literal bool    `json:"literal,omitempty"`
}

type Snippet struct {
	Heading   string `json:"heading"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Text      string `json:"text"`
}

type Hit struct {
	Note
	Score       float64   `json:"score"`
	Snippets    []Snippet `json:"snippets"`
	ParseStatus string    `json:"parse_status"`
	OKF         bool      `json:"okf"`
}

type SearchResult struct {
	Hits       []Hit        `json:"hits"`
	Cursor     string       `json:"cursor,omitempty"`
	Generation string       `json:"generation"`
	Warnings   []Diagnostic `json:"warnings"`
	Total      int          `json:"total"`
}

type ReadRequest struct {
	Path         string `json:"path"`
	Section      string `json:"section,omitempty"`
	LineStart    int    `json:"line_start,omitempty"`
	LineEnd      int    `json:"line_end,omitempty"`
	Raw          bool   `json:"raw,omitempty"`
	IncludeLinks bool   `json:"include_links,omitempty"`
}

type ReadResult struct {
	// Generation binds transport metadata to the same snapshot as the source.
	Generation  string       `json:"-"`
	Note        Note         `json:"note"`
	Markdown    string       `json:"markdown"`
	LineStart   int          `json:"line_start"`
	LineEnd     int          `json:"line_end"`
	Links       []Link       `json:"links,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

type ContextRequest struct {
	Query        string  `json:"query"`
	Filters      Filters `json:"filters,omitempty"`
	MaxTokens    int     `json:"max_tokens,omitempty"`
	MaxBytes     int     `json:"max_bytes,omitempty"`
	IncludeLinks bool    `json:"include_links,omitempty"`
}

type ContextSource struct {
	Path      string `json:"path"`
	Revision  string `json:"revision"`
	Heading   string `json:"heading"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
}

type ContextResult struct {
	Text            string          `json:"text"`
	Sources         []ContextSource `json:"sources"`
	EstimatedTokens int             `json:"estimated_tokens"`
	Estimator       string          `json:"estimator"`
	Truncated       bool            `json:"truncated"`
	Omitted         int             `json:"omitted"`
	Generation      string          `json:"generation"`
}

type Status struct {
	Root        string       `json:"root"`
	Generation  string       `json:"generation"`
	Notes       int          `json:"notes"`
	Invalid     int          `json:"invalid"`
	Degraded    bool         `json:"degraded"`
	IndexStale  bool         `json:"index_stale"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Excluded    []string     `json:"excluded"`
}

type ValidationReport struct {
	Profile     string            `json:"profile"`
	Conformant  bool              `json:"conformant"`
	Generation  string            `json:"generation"`
	Diagnostics []Diagnostic      `json:"diagnostics"`
	Clean       bool              `json:"clean"`
	Strict      bool              `json:"strict,omitempty"`
	Summary     DiagnosticSummary `json:"summary"`
}

type DiagnosticSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

type ExportResult struct {
	Destination string   `json:"destination"`
	Files       []string `json:"files"`
	Excluded    []string `json:"excluded"`
	Conformant  bool     `json:"conformant"`
	Generation  string   `json:"generation"`
}

type TextEdit struct {
	OldText string `json:"old_text"`
	NewText string `json:"new_text"`
}

type Operation struct {
	OperationID        string         `json:"operation_id,omitempty"`
	Kind               string         `json:"kind"`
	Path               string         `json:"path"`
	To                 string         `json:"to,omitempty"`
	ExpectedRevision   string         `json:"expected_revision,omitempty"`
	Type               string         `json:"type,omitempty"`
	Title              string         `json:"title,omitempty"`
	Body               *string        `json:"body,omitempty"`
	BodyEdits          []TextEdit     `json:"body_edits,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
	MetadataPatch      map[string]any `json:"metadata_patch,omitempty"`
	RemoveMetadataKeys []string       `json:"remove_metadata_keys,omitempty"`
	Document           *string        `json:"document,omitempty"`
}

type Plan struct {
	Operations     []Operation `json:"operations"`
	IdempotencyKey string      `json:"idempotency_key"`
	Summary        string      `json:"summary"`
	Actor          string      `json:"actor"`
	DryRun         bool        `json:"dry_run,omitempty"`
	Repair         bool        `json:"repair,omitempty"`
	Incremental    bool        `json:"incremental,omitempty"`
}

type Change struct {
	Path           string `json:"path"`
	Kind           string `json:"kind"`
	BeforeRevision string `json:"before_revision,omitempty"`
	AfterRevision  string `json:"after_revision,omitempty"`
	Diff           string `json:"diff,omitempty"`
}

type Receipt struct {
	IdempotencyKey string          `json:"idempotency_key"`
	Generation     string          `json:"generation"`
	DryRun         bool            `json:"dry_run"`
	Replayed       bool            `json:"replayed"`
	Changes        []Change        `json:"changes"`
	Warnings       []Diagnostic    `json:"warnings"`
	RepairProgress *RepairProgress `json:"repair_progress,omitempty"`
}

type RepairProgress struct {
	ErrorsBefore    int  `json:"errors_before"`
	ErrorsRemaining int  `json:"errors_remaining"`
	Conformant      bool `json:"conformant"`
}

// Codec preserves note presentation and applies content-level policies.
// Input byte images are immutable; edits return a new image or the unchanged input.
type Codec interface {
	Parse(path string, raw []byte) (Note, []Diagnostic)
	Patch(path string, raw []byte, operation Operation, actor string, at time.Time) ([]byte, error)
	ValidateReserved(path string, raw []byte) []Diagnostic
}

// Provider is consumed by the writer while it owns the exclusive base lock.
type Provider interface {
	Root() string
	Config() Config
	SnapshotUnlocked(context.Context) (Snapshot, error)
	Engine() Codec
}
