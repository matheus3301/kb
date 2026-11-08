package core

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
)

func TestCodecPublicEditorKeepsFinalValidationErrors(t *testing.T) {
	for _, test := range []struct{ body, rule, message string }{
		{"\xff", "INVALID_UTF8", "INVALID_UTF8: note is not valid UTF-8"},
		{strings.Repeat("x", api.MaxNoteBytes+1), "LIMIT_EXCEEDED", "LIMIT_EXCEEDED: note exceeds 1 MiB"},
	} {
		for _, kind := range []string{"create", "patch"} {
			raw := []byte("---\ntype: Reference\n---\nBody\n")
			if kind == "create" {
				raw = nil
			}
			_, err := (codec{}).Patch("note.md", raw, api.Operation{Kind: kind, Type: "Reference", Body: &test.body}, "process:check", time.Unix(0, 0))
			var failure *api.DomainError
			if !errors.As(err, &failure) || failure.Code != "INVALID_NOTE" || failure.Message != "patched note failed validation" {
				t.Fatalf("%s: %v", kind, err)
			}
			ds, ok := failure.Details.([]api.Diagnostic)
			if !ok || len(ds) != 1 || ds[0].Code != "INVALID_NOTE" || ds[0].Rule != test.rule || ds[0].Message != test.message || ds[0].Layer != "conformance_errors" || ds[0].Path != "note.md" {
				t.Fatalf("details: %+v", failure.Details)
			}
		}
	}
}
