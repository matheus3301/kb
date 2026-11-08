package okf_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/matheus3301/kb/pkg/okf"
)

func TestEditReplacesWhitespaceBlockScalarContent(t *testing.T) {
	for _, shape := range []struct {
		name, header, key string
		replacement       any
	}{
		{"top-level", "field: BLOCK\n\n  \t\n", "field", "0"},
		{"nested-map", "custom:\n  path: BLOCK\n\n    \t\n", "custom", map[string]any{"path": "0"}},
		{"nested-sequence", "custom:\n  - path: BLOCK\n\n      \t\n", "custom", []any{map[string]any{"path": "0"}}},
		{"sequence-scalar", "custom:\n  - BLOCK\n\n    \t\n", "custom", []any{"0"}},
	} {
		for _, style := range []string{"|2-", ">2+"} {
			for _, tail := range []string{"", "after: keep\n", "# Footer\n"} {
				for _, newline := range []string{"\n", "\r\n"} {
					t.Run(shape.name+style+tail+newline, func(t *testing.T) {
						raw := "---\ntype: Reference\nuntouched: 'literal' # Keep\n" + strings.ReplaceAll(shape.header, "BLOCK", style) + tail + "---\nBody\n"
						raw = strings.ReplaceAll(raw, "\n", newline)
						document := parse(t, raw)
						out, err := document.Edit(okf.Edit{Set: map[string]any{shape.key: shape.replacement}})
						if err != nil {
							t.Fatal(err)
						}
						fresh := parse(t, string(out))
						if value := fresh.Metadata()[shape.key]; !reflect.DeepEqual(value, shape.replacement) {
							t.Fatalf("old block content survived replacement: %#v", value)
						}
						if fresh.Body() != "Body"+newline || !strings.Contains(string(out), "untouched: 'literal' # Keep"+newline) {
							t.Fatal("unrelated document content changed")
						}
						if strings.Count(string(out), "# Footer") != strings.Count(raw, "# Footer") {
							t.Fatal("footer annotation changed")
						}
					})
				}
			}
		}
	}
}
