package okf

import (
	"strings"
	"time"
)

// Validate checks concept requirements and the implemented optional-field
// guidance, not bundle conformance. Findings are capped at MaxDiagnostics.
func (d *Document) Validate() []Diagnostic {
	if d.front.node == nil {
		return []Diagnostic{{Code: "INVALID_DOCUMENT", Message: "document is not parsed", Kind: Requirement}}
	}
	f := d.front
	var ds []Diagnostic
	add := func(code, msg string, kind Kind) {
		if len(ds) < MaxDiagnostics {
			d := Diagnostic{Code: code, Message: msg, Kind: kind}
			field, _, _ := strings.Cut(msg, " ")
			field = strings.TrimPrefix(field, "source.")
			if strings.HasPrefix(msg, "source.") {
				field = "sources"
			}
			for i := 0; i < len(f.node.Content); i += 2 {
				key := f.node.Content[i]
				if key.Value == strings.Split(field, ".")[0] {
					d.Field = field
					d.Line = key.Line + 1
					break
				}
			}
			if code == "MISSING_TYPE" {
				d.Field = "type"
			}
			ds = append(ds, d)
		}
	}
	m := d.Metadata()
	conceptType, _ := m["type"].(string)
	if strings.TrimSpace(conceptType) == "" {
		add("MISSING_TYPE", "type must be a nonempty string", Requirement)
	}
	for _, key := range []string{"title", "description", "resource", "status", "stale_after", "runtime", "computation"} {
		if v, ok := m[key]; ok {
			if _, ok := v.(string); !ok {
				add("INVALID_METADATA", key+" must be a string", Guidance)
			}
		}
	}
	if v, ok := m["status"].(string); ok {
		if v != "draft" && v != "stable" && v != "deprecated" {
			add("INVALID_STATUS", "status must be draft, stable, or deprecated", Guidance)
		}
	}
	if tags, ok := m["tags"]; ok {
		a, ok := tags.([]any)
		if !ok {
			add("INVALID_TAGS", "tags must be a list of strings", Guidance)
		} else {
			for _, v := range a {
				if _, ok := v.(string); !ok {
					add("INVALID_TAGS", "tags must be strings", Guidance)
				}
			}
		}
	}
	validateTime := func(key string, v any) {
		s, ok := v.(string)
		if !ok {
			add("INVALID_TIME", key+" must be a timezone-bearing ISO 8601 instant", Guidance)
			return
		}
		if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
			add("INVALID_TIME", key+" must be a timezone-bearing ISO 8601 instant", Guidance)
		}
	}
	validateActor := func(key string, v any, requireAt bool) {
		event, ok := v.(map[string]any)
		if !ok {
			add("INVALID_ACTOR", key+" must be a mapping", Guidance)
			return
		}
		by, ok := event["by"].(string)
		if !ok || strings.TrimSpace(by) == "" {
			add("INVALID_ACTOR", key+".by must be nonempty", Guidance)
		}
		if at, ok := event["at"]; ok {
			validateTime(key+".at", at)
		} else if requireAt {
			add("INVALID_TIME", key+".at is required", Guidance)
		}
	}
	if v, ok := m["generated"]; ok {
		validateActor("generated", v, false)
	}
	if v, ok := m["verified"]; ok {
		events, ok := v.([]any)
		if !ok {
			events = []any{v}
		}
		for _, event := range events {
			validateActor("verified", event, true)
		}
	}
	if v, ok := m["stale_after"]; ok {
		validateTime("stale_after", v)
	}
	if v, ok := m["sources"]; ok {
		sources, ok := v.([]any)
		if !ok {
			add("INVALID_SOURCES", "sources must be a list of mappings with resource", Guidance)
		} else {
			for _, source := range sources {
				obj, ok := source.(map[string]any)
				if !ok {
					add("INVALID_SOURCE", "source must be a mapping", Guidance)
					continue
				}
				resource, ok := obj["resource"].(string)
				if !ok || strings.TrimSpace(resource) == "" {
					add("INVALID_SOURCE", "source.resource is required", Guidance)
				}
				if v, ok := obj["last_modified"]; ok {
					validateTime("source.last_modified", v)
				}
			}
		}
	}
	if conceptType == "Attested Computation" {
		if v, ok := m["runtime"].(string); !ok || strings.TrimSpace(v) == "" {
			add("INVALID_COMPUTATION", "Attested Computation requires runtime", Guidance)
		}
		for _, key := range []string{"executor", "attester"} {
			if v, ok := m[key]; ok {
				obj, ok := v.(map[string]any)
				if !ok {
					add("INVALID_COMPUTATION", key+" must be a mapping", Guidance)
				} else if s, ok := obj["resource"].(string); !ok || s == "" {
					add("INVALID_COMPUTATION", key+".resource is required", Guidance)
				}
			}
		}
	}
	return ds
}
