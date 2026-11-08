package okf_test

import (
	"strings"
	"testing"

	"github.com/matheus3301/kb/pkg/okf"
)

func TestEditRetainsUnchangedAliasValuesBeyondRenderingBudget(t *testing.T) {
	// The document stays below the parse limit; its detached metadata view is
	// larger because sixteen permitted aliases expand the same scalar.
	raw := "---\ntype: Reference\ngenerated:\n  by: process:old\n  template: &shared " + strings.Repeat("x", 350000) + "\n  copies: [" + strings.TrimSuffix(strings.Repeat("*shared, ", 16), ", ") + "]\n---\nBody\n"
	document := parse(t, raw)
	generated := document.Metadata()["generated"].(map[string]any)
	generated["by"] = "process:new"
	body := "Revised\n"
	out, err := document.Edit(okf.Edit{Set: map[string]any{"generated": generated}, Body: &body})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(strings.Replace(raw, "process:old", "process:new", 1), "Body\n", body, 1)
	if string(out) != want {
		t.Fatal("editing the actor/body changed retained anchor or alias spans")
	}
}
