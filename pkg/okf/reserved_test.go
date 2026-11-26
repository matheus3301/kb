package okf_test

import (
	"strings"
	"testing"

	"github.com/matheus3301/kb/pkg/okf"
)

func TestReservedFormatRolesAndSupportedBoundaries(t *testing.T) {
	for _, test := range []struct {
		role         okf.ReservedRole
		raw, message string
	}{
		{okf.RootIndex, "---\nokf_version: \"0.2\"\n---\n# Topics\n", ""},
		{okf.RootIndex, "---\nokf_version: \"99.0\"\n---\n", ""},
		{okf.RootIndex, "---\nokf_version: 0.2\n---\n", "okf_version must be a string"},
		{okf.RootIndex, "---\ntype: Reference\n---\n", "root index metadata permits only okf_version"},
		{okf.Index, "# Topics\n[missing](missing.md)\n<!-- kb:index:end -->\n", ""},
		{okf.Index, "---\nokf_version: \"0.2\"\n---\n", "only the root index permits frontmatter"},
		{okf.Log, "---\ntype: Reference\n---\n", "only the root index permits frontmatter"},
		{okf.Log, "# Log\n## 2026-01-02\n- Entry\n## 2026-01-01\n- Earlier\n", ""},
		{okf.Log, "## 2026-01-01\n## 2026-01-02\n", "log dates must be newest first"},
		{okf.Log, "## 2026-02-30\n", "log date headings must use YYYY-MM-DD"},
		{okf.Log, "# No enforced entries\n", ""},
		{okf.Index, "\xff", "reserved file exceeds the size limit or is not UTF-8"},
		{okf.RootIndex, "---\nx: [\n---\n", "INVALID_YAML: frontmatter must be a YAML mapping"},
	} {
		ds := okf.ValidateReserved(test.role, []byte(test.raw))
		if test.message == "" {
			if len(ds) != 0 {
				t.Fatal(ds)
			}
			continue
		}
		if len(ds) != 1 || ds[0].Message != test.message || ds[0].Code != "INVALID_RESERVED" || ds[0].Kind != okf.Requirement {
			t.Fatal(ds)
		}
	}
	if ds := okf.ValidateReserved("unknown", nil); len(ds) != 1 || ds[0].Code != "INVALID_INPUT" {
		t.Fatal(ds)
	}
}

func TestDeclaredV03RootMetadata(t *testing.T) {
	for _, header := range []string{
		`okf_version: "0.3"`,
		"title: Synthetic bundle\ndescription: A synthetic scope.\nlicense: CC-BY-4.0\nokf_version: \"0.3\"",
		"okf_version: '0.3'\nx-producer: {nested: [one, {precision: 1234567890.123456789}]}\nowner: synthetic",
		"okf_version: '0.3'\ntitle: null\ndescription: []\nlicense: ''",
		"okf_version: &version '0.3'\ntitle: *version\nlicense: /LICENSE.txt",
		"okf_version: '0.3'\nlicense: ../../outside.txt\nverified: {by: 'human:synthetic'}",
	} {
		raw := []byte("---\n" + header + "\n---\n# Topics\n")
		if ds := okf.ValidateReserved(okf.RootIndex, raw); len(ds) != 0 {
			t.Fatalf("%s: %+v", header, ds)
		}
		for _, role := range []okf.ReservedRole{okf.Index, okf.Log} {
			if ds := okf.ValidateReserved(role, raw); len(ds) != 1 || ds[0].Message != "only the root index permits frontmatter" {
				t.Fatalf("%s: %+v", role, ds)
			}
		}
	}
	for _, field := range []string{"title", "description", "license"} {
		for _, value := range []string{"42", "true", "[one]", "{nested: value}"} {
			raw := []byte("---\nokf_version: '0.3'\n" + field + ": " + value + "\n---\n")
			ds := okf.ValidateReserved(okf.RootIndex, raw)
			if len(ds) != 1 || ds[0].Code != "INVALID_RESERVED" || ds[0].Kind != okf.Requirement || ds[0].Message != field+" must be a string when present" {
				t.Fatalf("%s %s: %+v", field, value, ds)
			}
		}
	}
	// Version recognition is exact, never numeric coercion or version ordering.
	for _, declaration := range []string{"", "okf_version: '0.2'\n", "okf_version: '99.0'\n", "okf_version: '0.30'\n"} {
		raw := []byte("---\n" + declaration + "title: Synthetic\n---\n")
		ds := okf.ValidateReserved(okf.RootIndex, raw)
		if len(ds) != 1 || ds[0].Message != "root index metadata permits only okf_version" {
			t.Fatalf("%s: %+v", declaration, ds)
		}
	}
	for _, header := range []string{
		"okf_version: 0.3",
		"okf_version: '0.3'\nx-extra: {duplicate: one, duplicate: two}",
		"okf_version: '0.3'\nx-extra: &cycle [*cycle]",
	} {
		if ds := okf.ValidateReserved(okf.RootIndex, []byte("---\n"+header+"\n---\n")); len(ds) == 0 {
			t.Fatalf("invalid metadata accepted: %s", header)
		}
	}
	raw := []byte("---\nokf_version: '0.3'\nlicense: false\ntitle: 42\ndescription: {}\n---\n")
	ds := okf.ValidateReserved(okf.RootIndex, raw)
	if len(ds) != 3 || ds[0].Message != "license must be a string when present" || ds[1].Message != "title must be a string when present" || ds[2].Message != "description must be a string when present" {
		t.Fatalf("diagnostic syntax order: %+v", ds)
	}
}

func FuzzReserved(f *testing.F) {
	f.Add("---\nokf_version: \"0.2\"\n---\n# Index\n")
	f.Add("---\nokf_version: \"0.3\"\ntitle: Synthetic\nlicense: null\nx-fixture: {nested: [1, true]}\n---\n# Index\n")
	f.Add("## 2026-01-02\n## 2026-01-01\n")
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > okf.MaxDocumentBytes+1 {
			return
		}
		for _, role := range []okf.ReservedRole{okf.RootIndex, okf.Index, okf.Log} {
			for _, finding := range okf.ValidateReserved(role, []byte(raw)) {
				if finding.Code == "" || strings.TrimSpace(finding.Message) == "" {
					t.Fatal(finding)
				}
			}
		}
	})
}
