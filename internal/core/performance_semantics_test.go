package core

import (
	"github.com/matheus3301/kb/internal/api"
	"reflect"
	"testing"
	"time"
)

func FuzzWordNormalizationPreservesLexicalContract(f *testing.F) {
	for _, seed := range []string{"Ação memória s3 k8s", "İIıi Straße Σσς", "a\u0301 ação\u0345", "東京 हिन्दी العربية", "\u0301\u0345", "ﬀ ŉ K Å ᾈ", "a/b.md#link foo_bar 123", ""} {
		f.Add(seed, true)
		f.Add(seed, false)
	}
	f.Fuzz(func(t *testing.T, text string, accents bool) {
		if len(text) > 8192 {
			t.Skip()
		}
		expected := tokens(text, accents)
		actual := []string{}
		forEachToken(text, accents, func(token string) { actual = append(actual, token) })
		if len(expected) == 0 && len(actual) == 0 {
			return
		}
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("token semantics changed for %q (%v): %q != %q", text, accents, expected, actual)
		}
	})
}

func FuzzPatchDoesNotMutateSourceImages(f *testing.F) {
	f.Add("Original evidence", "Corrected evidence")
	f.Add("# Heading\nUnicode ação", "")
	f.Fuzz(func(t *testing.T, before, after string) {
		if len(before)+len(after) > 8192 {
			t.Skip()
		}
		raw := []byte("---\ntype: Reference\ndescription: Original\ngenerated: {by: process:fixture, at: \"2026-10-25T00:00:00Z\"}\nverified: {by: human:synthetic, at: \"2026-10-25T00:00:00Z\"}\n# Preserve metadata\n---\n" + before)
		original := string(raw)
		_, _ = (codec{}).Patch("note.md", raw, api.Operation{Kind: "patch", Body: &after, MetadataPatch: map[string]any{"description": after}}, "process:fuzz", time.Unix(0, 0))
		if string(raw) != original {
			t.Fatal("codec mutated input bytes shared by plan images")
		}
	})
}
