package core

import (
	"github.com/matheus3301/kb/internal/api"
	"reflect"
	"testing"
	"time"
)

func FuzzWordNormalizationPreservesLexicalContract(f *testing.F) {
	for _, seed := range []string{"Ação memória s3 k8s", "İIıi Straße Σσς", "a\u0301 ação\u0345", "東京 हिन्दी العربية", "\u0301\u0345", "ﬀ ŉ K Å ᾈ", "a/b.md#link foo_bar 123", "ASCII UPPER lowerCase A1 Z9", "lowercase numbers123 punctuation_/.-", "ASCII\xffUPPER \x80lowerCase", ""} {
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

func TestLexicalASCIIWordsPreserveFoldingAndBoundaries(t *testing.T) {
	for _, test := range []struct {
		text    string
		accents bool
		want    []string
	}{
		{"ASCII UPPER lowerCase A1 Z9", false, []string{"ascii", "upper", "lowercase", "a1", "z9"}},
		{"lower_case a1/123.z9#UPPER", true, []string{"lower", "case", "a1", "123", "z9", "upper"}},
		{"A\u0301 AÇÃO Straße İ ﬀ", true, []string{"a", "acao", "strasse", "i", "ff"}},
		{"A\u0301 AÇÃO Straße İ ﬀ", false, []string{"á", "ação", "strasse", "i\u0307", "ff"}},
		{"ASCII\xffUPPER \x80lowerCase", true, []string{"ascii", "upper", "lowercase"}},
	} {
		var actual []string
		forEachToken(test.text, test.accents, func(token string) { actual = append(actual, token) })
		if !reflect.DeepEqual(actual, test.want) {
			t.Fatalf("word folding changed for %q (accents=%v): got %q, want %q", test.text, test.accents, actual, test.want)
		}
	}
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
