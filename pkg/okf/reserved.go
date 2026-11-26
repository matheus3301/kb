package okf

import (
	"strings"
	"time"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// ReservedRole identifies the logical role of a reserved Markdown file, without
// opening a path or choosing a storage layout.
type ReservedRole string

const (
	// RootIndex is the bundle-root index.md, permitting bundle metadata.
	RootIndex ReservedRole = "root-index"
	// Index is any non-root index.md, which must not have frontmatter.
	Index ReservedRole = "index"
	// Log is a log.md at any level, which must not have frontmatter.
	Log ReservedRole = "log"
)

// ValidateReserved checks the supported reserved-file structure: UTF-8/size
// bounds, frontmatter placement, root-index metadata and newest-first YYYY-MM-DD
// log headings. Exactly okf_version "0.3" permits optional string title,
// description and license plus opaque extension keys. Null and empty lists for
// those optional fields mean absent. Other/absent versions retain the v0.2
// okf_version-only contract; unknown nonempty version strings remain accepted.
// It does not certify versions, validate listings/links/license expressions,
// walk a bundle or recognize KB managed markers. Findings describe structure,
// not optional concept guidance.
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
			// Inspect only known values; do not expand/copy opaque extension trees.
			var version string
			for i := 0; i < len(d.front.node.Content); i += 2 {
				if d.front.node.Content[i].Value == "okf_version" {
					version, _ = reservedScalar(d.front.node.Content[i+1]).(string)
				}
			}
			// Walk syntax order, not map iteration, for deterministic findings.
			for i := 0; i < len(d.front.node.Content); i += 2 {
				key := d.front.node.Content[i].Value
				if key == "okf_version" {
					if version == "" {
						add("okf_version must be a string")
					}
				} else if version != "0.3" {
					add("root index metadata permits only okf_version")
				} else if key == "title" || key == "description" || key == "license" {
					value := d.front.node.Content[i+1]
					for value.Alias != nil {
						value = value.Alias
					}
					_, isString := reservedScalar(value).(string)
					emptyList := value.Kind == yaml.SequenceNode && len(value.Content) == 0
					if !isString && value.Tag != "!!null" && !emptyList {
						add(key + " must be a string when present")
					}
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

// Parse has already bounded aliases and rejected cycles. Containers are not
// decoded: an invalid known field must not allocate an expanded metadata tree.
func reservedScalar(value *yaml.Node) any {
	for value.Alias != nil {
		value = value.Alias
	}
	if value.Kind != yaml.ScalarNode {
		return nil
	}
	return nodeValue(value)
}
