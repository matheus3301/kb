package okf_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/matheus3301/kb/okf"
)

func parse(t *testing.T, raw string) *okf.Document {
	t.Helper()
	d, err := okf.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func code(t *testing.T, err error, want string) {
	t.Helper()
	var failure *okf.Error
	if !errors.As(err, &failure) || failure.Code != want {
		t.Fatalf("error = %v, want %s", err, want)
	}
}

func TestParseAndValidationSeparateRequirementsFromGuidance(t *testing.T) {
	for _, fields := range []string{"type: Unknown Type\n", "type: Reference\nverified: {by: human:reviewer, at: 2026-01-01T00:00:00Z}\n", "type: Reference\nverified: [{by: process:check, at: 2026-01-01T01:00:00+01:00}]\n"} {
		d := parse(t, "---\n"+fields+"custom: [a, 12]\n---\n[missing](missing.md)\n")
		if ds := d.Validate(); len(ds) != 0 {
			t.Fatal(ds)
		}
		if d.Body() != "[missing](missing.md)\n" {
			t.Fatal(d.Body())
		}
	}
	d := parse(t, "---\ntitle: Example\nstatus: unknown\n---\n")
	want := []okf.Diagnostic{{Code: "MISSING_TYPE", Message: "type must be a nonempty string", Kind: okf.Requirement, Field: "type"}, {Code: "INVALID_STATUS", Message: "status must be draft, stable, or deprecated", Kind: okf.Guidance, Field: "status", Line: 3}}
	if ds := d.Validate(); !reflect.DeepEqual(ds, want) {
		t.Fatalf("findings = %+v", ds)
	}
	for _, fields := range []string{"type: 1", "tags: x", "tags: [1]", "title: []", "stale_after: 2026-01-01", "generated: {}", "verified: [x]", "verified: {by: process:check}", "sources: no", "sources: [no]", "sources: [{}]", "sources: [{resource: source.md, last_modified: yesterday}]", "type: Attested Computation", "type: Attested Computation\nruntime: sql\nexecutor: []", "type: Attested Computation\nruntime: sql\nattester: {}"} {
		d := parse(t, "---\n"+fields+"\n---\n")
		if len(d.Validate()) == 0 {
			t.Fatalf("no finding for %s", fields)
		}
	}
	many := parse(t, "---\ntype: Reference\nverified: ["+strings.Repeat("{},", 100)+"]\n---\n")
	if len(many.Validate()) != okf.MaxDiagnostics {
		t.Fatal("diagnostic bound")
	}
}

func TestParseRejectsUnsupportedInputsWithTypedErrors(t *testing.T) {
	for _, test := range []struct{ raw, want string }{
		{"plain", "MISSING_FRONTMATTER"}, {"\ufeff---\ntype: X\n---\n", "MISSING_FRONTMATTER"},
		{"---\ntype: X\n", "INVALID_YAML"}, {"---\n[]\n---\n", "INVALID_YAML"},
		{"---\ntype: [\n---\n", "INVALID_YAML"}, {"---\ntype: X\ntype: Y\n---\n", "DUPLICATE_KEY"},
		{"---\ntype: X\ncustom: {42: value}\n---\n", "METADATA_KEY_POLICY"},
		{"---\ntype: X\na: &cycle [*cycle]\n---\n", "YAML_LIMIT"},
		{"---\ntype: X\na: " + strings.Repeat("[", 40) + "0" + strings.Repeat("]", 40) + "\n---\n", "YAML_LIMIT"},
		{"---\ntype: X\n---\n\xff", "INVALID_UTF8"}, {strings.Repeat("x", okf.MaxDocumentBytes+1), "LIMIT_EXCEEDED"},
	} {
		_, err := okf.Parse([]byte(test.raw))
		code(t, err, test.want)
	}
}

func TestExactNumbersAndPresentationSurviveEditing(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		raw := strings.ReplaceAll("---\n# Header\ntype: 'Reference' # Keep\ncustom: {label: 'café 🧪', path: 'old.md', decimal: 1234567890.123456789, huge: 900719925474099312345} # Collection\nverified: {by: human:reviewer, at: 2026-01-01T00:00:00Z}\n\n# Footer\n---\nOriginal\n", "\n", newline)
		d := parse(t, raw)
		custom := d.Metadata()["custom"].(map[string]any)
		if custom["decimal"] != json.Number("1234567890.123456789") || custom["huge"] != json.Number("900719925474099312345") {
			t.Fatal(custom)
		}
		custom["path"] = "new,path.md"
		body := "Revised" + newline
		out, err := d.Edit(okf.Edit{Set: map[string]any{"custom": custom}, Body: &body})
		if err != nil {
			t.Fatal(err)
		}
		want := strings.Replace(strings.Replace(raw, "'old.md'", "'new,path.md'", 1), "Original", "Revised", 1)
		if string(out) != want {
			t.Fatalf("got:\n%s\nwant:\n%s", out, want)
		}
		if !bytes.Equal(d.Bytes(), []byte(raw)) || custom["path"] != "new,path.md" {
			t.Fatal("edit mutated input")
		}
		if parse(t, string(out)).Metadata()["verified"] == nil {
			t.Fatal("format editor applied KB verification policy")
		}
	}
	for literal, want := range map[string]string{"+.1234567890123456789": "0.1234567890123456789", "1_234.50_01e+0_3": "1234.5001e+03", "-001.2300": "-1.2300", "12.": "12.0", "!!float 1e400": "1e400", "0x10": "16"} {
		d := parse(t, "---\ntype: Reference\nx: "+literal+"\n---\n")
		if d.Metadata()["x"] != json.Number(want) {
			t.Fatal(literal, d.Metadata())
		}
	}
}

func TestStructuralEditsRetainCommentsAnchorsAndUnknownChildren(t *testing.T) {
	raw := "---\ntype: Reference\ntemplate: &template\n  label: 'keep' # Nested\n  decimal: 0.1234567890123456789\ncustom: *template # Alias\n# Footer\n---\nBody\n"
	d := parse(t, raw)
	custom := d.Metadata()["custom"].(map[string]any)
	custom["added"] = true
	out, err := d.Edit(okf.Edit{Set: map[string]any{"custom": custom}})
	if err != nil {
		t.Fatal(err)
	}
	after := parse(t, string(out)).Metadata()
	if _, exists := after["template"].(map[string]any)["added"]; exists {
		t.Fatal("alias edit changed template")
	}
	if after["custom"].(map[string]any)["added"] != true {
		t.Fatal(after)
	}
	for _, keep := range []string{"&template", "label: 'keep' # Nested", "decimal: 0.1234567890123456789", "# Alias", "# Footer"} {
		if !strings.Contains(string(out), keep) {
			t.Fatalf("lost %s: %s", keep, out)
		}
	}
	_, err = d.Edit(okf.Edit{Remove: []string{"template"}})
	code(t, err, "INVALID_YAML")
	flow := parse(t, "---\n# Heading\n\n{type: Reference, x: [1, 2]}\n\n# Footer\n---\nBody")
	out, err = flow.Edit(okf.Edit{Set: map[string]any{"x": []any{json.Number("1"), json.Number("2"), json.Number("3")}, "new": "value"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range []string{"# Heading", "# Footer"} {
		if strings.Count(string(out), comment) != 1 {
			t.Fatal(string(out))
		}
	}
	if len(parse(t, string(out)).Metadata()["x"].([]any)) != 3 {
		t.Fatal(string(out))
	}
}

func TestNullPresenceNoOpRemovalAndOwnership(t *testing.T) {
	input := []byte("---\ntype: Reference\nexisting: null\n---\nBody")
	d, err := okf.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	input[0] = 'x'
	view := d.Metadata()
	view["type"] = "changed"
	returned := d.Bytes()
	returned[0] = 'x'
	if d.Metadata()["type"] != "Reference" || d.Bytes()[0] != '-' {
		t.Fatal("caller mutation reached document")
	}
	set := map[string]any{"added": nil, "existing": nil}
	out, err := d.Edit(okf.Edit{Set: set})
	if err != nil {
		t.Fatal(err)
	}
	if value, exists := parse(t, string(out)).Metadata()["added"]; !exists || value != nil {
		t.Fatal("absent null key was omitted")
	}
	if len(set) != 2 {
		t.Fatal("caller map changed")
	}
	noOp, err := d.Edit(okf.Edit{Set: map[string]any{"existing": nil}, Remove: []string{"missing"}})
	if err != nil || !bytes.Equal(noOp, d.Bytes()) {
		t.Fatal(err, string(noOp))
	}
	out, err = d.Edit(okf.Edit{Set: map[string]any{"existing": "ignored"}, Remove: []string{"existing"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := parse(t, string(out)).Metadata()["existing"]; exists {
		t.Fatal("remove did not win")
	}
}

func TestMarshalAndEditBoundCallerTrees(t *testing.T) {
	cycle := map[string]any{}
	cycle["self"] = cycle
	slice := make([]any, 1)
	slice[0] = slice
	deep := map[string]any{}
	cursor := deep
	for range 40 {
		next := map[string]any{}
		cursor["child"] = next
		cursor = next
	}
	d := parse(t, "---\ntype: Reference\n---\n")
	for _, value := range []any{cycle, slice, deep, make([]any, 50001)} {
		metadata := map[string]any{"custom": value}
		_, err := okf.Marshal(metadata, "")
		code(t, err, "YAML_LIMIT")
		_, err = d.Edit(okf.Edit{Set: metadata})
		code(t, err, "YAML_LIMIT")
	}
	for _, literal := range []string{"", "null", "true", "false", "[]", "{}", `"12"`, "1x", "01", "NaN", "1 ", "1\n", "1\t", " 1"} {
		metadata := map[string]any{"number": json.Number(literal)}
		_, err := okf.Marshal(metadata, "")
		code(t, err, "INVALID_INPUT")
		_, err = d.Edit(okf.Edit{Set: metadata})
		code(t, err, "INVALID_INPUT")
		if metadata["number"] != json.Number(literal) {
			t.Fatal("caller metadata changed")
		}
	}
	for _, value := range []any{func() {}, &deep} {
		_, err := okf.Marshal(map[string]any{"x": value}, "")
		code(t, err, "INVALID_INPUT")
	}
	_, err := okf.Marshal(map[string]any{"x": "\xff"}, "")
	code(t, err, "INVALID_UTF8")
	_, err = okf.Marshal(map[string]any{"x": strings.Repeat("x", okf.MaxDocumentBytes+1)}, "")
	code(t, err, "LIMIT_EXCEEDED")
	_, err = okf.Marshal(nil, strings.Repeat("x", okf.MaxDocumentBytes+1))
	code(t, err, "LIMIT_EXCEEDED")
	_, err = okf.Marshal(map[string]any{"x": strings.Repeat("x", okf.MaxMetadataBytes+1)}, "")
	code(t, err, "LIMIT_EXCEEDED")
	_, err = d.Edit(okf.Edit{Set: map[string]any{"x": strings.Repeat("x", okf.MaxMetadataBytes+1)}})
	code(t, err, "LIMIT_EXCEEDED")
	var zero okf.Document
	_, err = zero.Edit(okf.Edit{})
	code(t, err, "INVALID_DOCUMENT")
	if ds := zero.Validate(); len(ds) != 1 || ds[0].Code != "INVALID_DOCUMENT" {
		t.Fatal(ds)
	}
	metadata := map[string]any{"type": "Reference", "number": json.Number("1234567890.123456789e+100"), "nested": []any{nil, true, json.Number("900719925474099312345")}}
	out, err := okf.Marshal(metadata, "Body\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := parse(t, string(out)).Metadata(); !reflect.DeepEqual(got, metadata) {
		t.Fatal(got)
	}
}

func FuzzParse(f *testing.F) {
	for _, raw := range []string{"---\ntype: Reference\n---\nBody", "plain", "---\na: &a [*a]\n---\n", "---\ntype: X\ncustom: {42: value}\n---\n"} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > okf.MaxDocumentBytes+1 {
			return
		}
		d, err := okf.Parse([]byte(raw))
		if err != nil {
			return
		}
		_ = d.Validate()
		out, err := d.Edit(okf.Edit{})
		if err != nil || string(out) != raw {
			t.Fatalf("no-op changed document: %v", err)
		}
	})
}

func FuzzMetadataEdit(f *testing.F) {
	f.Add("old.md", "new,path.md")
	f.Add("café 🧪", "quoted\nvalue")
	f.Fuzz(func(t *testing.T, old, next string) {
		if len(old)+len(next) > 2048 {
			return
		}
		raw, err := okf.Marshal(map[string]any{"type": "Reference", "custom": map[string]any{"path": old, "number": json.Number("0.1234567890123456789")}}, "Body")
		if err != nil {
			return
		}
		d, err := okf.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		custom := d.Metadata()["custom"].(map[string]any)
		custom["path"] = next
		out, err := d.Edit(okf.Edit{Set: map[string]any{"custom": custom}})
		if err != nil {
			return
		}
		fresh, err := okf.Parse(out)
		if err != nil {
			t.Fatal(err)
		}
		got := fresh.Metadata()["custom"].(map[string]any)
		if got["path"] != next || got["number"] != json.Number("0.1234567890123456789") {
			t.Fatal(got)
		}
	})
}
