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

func FuzzReserved(f *testing.F) {
	f.Add("---\nokf_version: \"0.2\"\n---\n# Index\n")
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
