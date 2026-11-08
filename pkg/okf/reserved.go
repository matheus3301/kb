package okf

import (
	"strings"
	"time"
	"unicode/utf8"
)

// ReservedRole identifies the logical role of a reserved Markdown file, without
// opening a path or choosing a storage layout.
type ReservedRole string

const (
	// RootIndex is the bundle-root index.md, permitting okf_version metadata.
	RootIndex ReservedRole = "root-index"
	// Index is any non-root index.md, which must not have frontmatter.
	Index ReservedRole = "index"
	// Log is a log.md at any level, which must not have frontmatter.
	Log ReservedRole = "log"
)

// ValidateReserved checks the implemented subset of upstream sections 8 and 9:
// UTF-8/size bounds, frontmatter placement, root-index metadata restricted to a
// nonempty string okf_version when present, and YYYY-MM-DD log headings ordered
// newest first. It does not require headings/entries, validate index listings or
// links, walk a bundle, interpret versions, or recognize KB managed markers.
// Findings describe required reserved structure, not optional-field guidance.
// Size/encoding failures use the same bounded implementation limits as Parse.
func ValidateReserved(role ReservedRole, raw []byte) []Diagnostic {
	var ds []Diagnostic
	add := func(message string) {
		ds = append(ds, Diagnostic{Code: "INVALID_RESERVED", Message: message, Kind: Requirement})
	}
	if role != RootIndex && role != Index && role != Log {
		return []Diagnostic{{Code: "INVALID_INPUT", Message: "unknown reserved file role", Kind: Requirement}}
	}
	if len(raw) > MaxDocumentBytes || !utf8.Valid(raw) {
		add("reserved file exceeds the size limit or is not UTF-8")
		return ds
	}
	body := string(raw)
	if strings.HasPrefix(body, "---\n") || strings.HasPrefix(body, "---\r\n") {
		d, err := Parse(raw)
		if err != nil {
			add(err.Error())
			return ds
		}
		if role != RootIndex {
			add("only the root index permits frontmatter")
		} else {
			// Walk syntax order, not map iteration, for deterministic findings.
			m := d.Metadata()
			for i := 0; i < len(d.front.node.Content); i += 2 {
				key := d.front.node.Content[i].Value
				if key != "okf_version" {
					add("root index metadata permits only okf_version")
				} else if value, ok := m[key].(string); !ok || value == "" {
					add("okf_version must be a string")
				}
			}
		}
		body = d.Body()
	}
	if role == Log {
		previous := "9999-99-99"
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "## ") {
				date := strings.TrimSpace(strings.TrimPrefix(line, "## "))
				if _, err := time.Parse("2006-01-02", date); err != nil {
					add("log date headings must use YYYY-MM-DD")
				} else if date > previous {
					add("log dates must be newest first")
				}
				previous = date
			}
		}
	}
	return ds
}
