package core

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/matheus3301/kb/internal/api"
)

func TestCodecReadsExactDecimalMetadata(t *testing.T) {
	for literal, want := range map[string]string{
		"1234567890.123456789":     "1234567890.123456789",
		"0.1234567890123456789":    "0.1234567890123456789",
		"1.234567890123456789e+23": "1.234567890123456789e+23",
		"+.1234567890123456789":    "0.1234567890123456789",
		"-001.2300":                "-1.2300", "1_234.50_01e+0_3": "1234.5001e+03",
		"12.": "12.0", "-.0": "-0.0", "!!float 1e400": "1e400",
	} {
		t.Run(literal, func(t *testing.T) {
			raw := []byte("---\ntype: Reference\ncustom: {value: " + literal + "}\n---\nBody\n")
			note, ds := (codec{}).Parse("number.md", raw)
			if hasErrors(ds) {
				t.Fatal(ds)
			}
			custom := note.Metadata["custom"].(map[string]any)
			if custom["value"] != json.Number(want) {
				t.Fatalf("decimal changed: %#v, want %s", custom["value"], want)
			}
			encoded, err := json.Marshal(custom)
			if err != nil || string(encoded) != `{"value":`+want+`}` {
				t.Fatalf("JSON metadata rounded: %s %v", encoded, err)
			}
		})
	}
}

func FuzzMetadataScalarPatch(f *testing.F) {
	f.Add("old.md", "new.md")
	f.Add("old\"file.md", "new,path.md")
	f.Add("café 🧪", "multi\nline # quoted")
	f.Fuzz(func(t *testing.T, old, next string) {
		if len(old) > 1024 || len(next) > 1024 || !utf8.ValidString(old) || !utf8.ValidString(next) {
			return
		}
		literal, err := json.Marshal(old)
		if err != nil {
			t.Fatal(err)
		}
		raw := []byte("---\ntype: Reference\ncustom: {label: 'café 🧪', path: " + string(literal) + ", decimal: 1234567890.123456789} # Keep\n---\nBody\n")
		note, ds := (codec{}).Parse("note.md", raw)
		if hasErrors(ds) {
			return
		}
		custom := note.Metadata["custom"].(map[string]any)
		custom["path"] = next
		out, err := (codec{}).Patch("note.md", raw, api.Operation{MetadataPatch: map[string]any{"custom": custom}}, "process:fuzz", time.Unix(0, 0).UTC())
		if err != nil {
			return
		}
		fresh, ds := (codec{}).Parse("note.md", out)
		if hasErrors(ds) {
			t.Fatal(ds)
		}
		values := fresh.Metadata["custom"].(map[string]any)
		if values["path"] != next || values["label"] != "café 🧪" || values["decimal"] != json.Number("1234567890.123456789") || strings.Count(string(out), "# Keep") != 1 {
			t.Fatalf("successful scalar patch changed unrelated metadata: %s", out)
		}
	})
}

func TestCodecStructuralMetadataEditsPreserveUnknownChildren(t *testing.T) {
	raw := []byte("---\ntype: Reference\n# Collection annotation\ncustom:\n  first: 'keep' # Inline annotation\n  decimal: 1234567890.123456789\n  list: [0.1234567890123456789, 'quoted'] # List annotation\n---\nBody\n")
	change := map[string]any{"first": "keep", "decimal": json.Number("1234567890.123456789"), "list": []any{json.Number("0.1234567890123456789"), "quoted", "added"}, "added": true}
	out, err := (codec{}).Patch("note.md", raw, api.Operation{MetadataPatch: map[string]any{"custom": change}}, "process:test", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, annotation := range []string{"# Collection annotation", "first: 'keep' # Inline annotation", "decimal: 1234567890.123456789", "0.1234567890123456789", "'quoted'", "# List annotation"} {
		if strings.Count(string(out), annotation) != 1 {
			t.Errorf("annotation or value lost/duplicated: %q\n%s", annotation, out)
		}
	}
}

func TestCodecNestedScalarEditsPreserveUnicodeAndQuoteSpans(t *testing.T) {
	for _, raw := range []string{
		"---\ntype: Reference\ncustom: {label: 'café 🧪', path: 'old''file.md', extra: 1.234567890123456789} # Keep\n---\nBody\n",
		"---\ntype: Reference\ncustom: {label: 'café 🧪', path: \"old\\\"file.md\", extra: 1.234567890123456789} # Keep\n---\nBody\n",
	} {
		note, _ := (codec{}).Parse("note.md", []byte(raw))
		custom := note.Metadata["custom"].(map[string]any)
		custom["path"] = "new,path.md"
		out, err := (codec{}).Patch("note.md", []byte(raw), api.Operation{MetadataPatch: map[string]any{"custom": custom}}, "process:test", time.Now())
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"label: 'café 🧪'", "extra: 1.234567890123456789} # Keep"} {
			if !strings.Contains(string(out), want) {
				t.Fatalf("unrelated span changed: %s", out)
			}
		}
		parsed, _ := (codec{}).Parse("note.md", out)
		if parsed.Metadata["custom"].(map[string]any)["path"] != "new,path.md" {
			t.Fatal("new scalar did not round-trip in flow context")
		}
	}
}
