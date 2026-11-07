package write_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/internal/core"
	"github.com/matheus3301/kb/internal/write"
	"github.com/matheus3301/kb/okf"
)

func TestTransactionUsesPublicFormatWithoutLosingMetadata(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	store, err := core.Init(ctx, root, false)
	if err != nil {
		t.Fatal(err)
	}
	files, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := files.Close(); err != nil {
			t.Error(err)
		}
	})
	raw := "---\r\ntype: 'Reference' # Keep\r\ncustom: {number: 1234567890.123456789, unknown: 'literal'} # Extension\r\ngenerated: {by: process:original, custom: keep}\r\nverified: {by: human:reviewer, at: 2026-01-01T00:00:00Z}\r\n# Footer\r\n---\r\nOriginal body\r\n"
	name := filepath.Join(root, "note.md")
	if err := os.WriteFile(name, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	plan := api.Plan{Actor: "process:integration", Summary: "Revise synthetic body", IdempotencyKey: "format-integration", Operations: []api.Operation{{OperationID: "edit", Kind: "patch", Path: "note.md", ExpectedRevision: api.Revision([]byte(raw)), BodyEdits: []api.TextEdit{{OldText: "Original body", NewText: "Revised body"}}}}}
	service := write.New(store)
	if _, err := service.Apply(ctx, plan); err != nil {
		t.Fatal(err)
	}
	out, err := files.ReadFile("note.md")
	if err != nil {
		t.Fatal(err)
	}
	document, err := okf.Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	metadata := document.Metadata()
	if metadata["custom"].(map[string]any)["number"] != json.Number("1234567890.123456789") || metadata["generated"].(map[string]any)["custom"] != "keep" {
		t.Fatal(metadata)
	}
	if _, exists := metadata["verified"]; exists {
		t.Fatal("KB did not clear active verification")
	}
	if metadata["generated"].(map[string]any)["by"] != plan.Actor || document.Body() != "Revised body\r\n" {
		t.Fatal(metadata, document.Body())
	}
	for _, span := range []string{"type: 'Reference' # Keep\r\n", "custom: {number: 1234567890.123456789, unknown: 'literal'} # Extension\r\n", "# Footer\r\n"} {
		if !strings.Contains(string(out), span) {
			t.Fatalf("lost span %q: %s", span, out)
		}
	}
	replayed, err := service.Apply(ctx, plan)
	if err != nil || !replayed.Replayed {
		t.Fatal(replayed, err)
	}
	unchanged, err := files.ReadFile("note.md")
	if err != nil || string(unchanged) != string(out) {
		t.Fatal("replay changed note", err)
	}
}
