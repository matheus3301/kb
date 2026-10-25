package core

import (
	"strings"

	"github.com/matheus3301/kb/internal/api"
)

// ruleHelp gives deterministic guidance, not inferred content or verification.
var ruleHelp = map[string]string{
	"MISSING_FRONTMATTER": "Add a YAML frontmatter mapping with an evidence-appropriate nonempty type; preserve the Markdown body.",
	"MISSING_TYPE":        "Set type to a nonempty string describing this note; preserve unknown metadata.",
	"INVALID_YAML":        "Repair the YAML mapping and delimiters while preserving values, comments and the Markdown body.",
	"DUPLICATE_KEY":       "Reconcile duplicate YAML keys explicitly; do not silently discard conflicting values.",
	"YAML_LIMIT":          "Reduce YAML nesting, aliases or node count without discarding source evidence.",
	"INVALID_UTF8":        "Restore valid UTF-8 from the original source before editing.",
	"LIMIT_EXCEEDED":      "Reduce the document to the supported size without discarding evidence; oversized files require operator preparation.",
	"METADATA_KEY_POLICY": "Use textual YAML mapping keys for JSON interoperability while preserving their meaning.",
	"INVALID_STATUS":      "Use draft, stable or deprecated for status; preserve unrelated workflow status under an extension key.",
	"INVALID_METADATA":    "Use a string for the named field, or explicitly preserve incompatible data under an extension key.",
	"INVALID_TAGS":        "Represent tags as a list of strings without inventing classifications.",
	"INVALID_TIME":        "Use a timezone-bearing ISO 8601 instant backed by evidence; do not invent historical dates.",
	"INVALID_ACTOR":       "Use a producer event mapping with nonempty by and any required at; never fabricate human verification.",
	"INVALID_SOURCES":     "Represent sources as a list of mappings with resource referring to evidence actually consulted.",
	"INVALID_SOURCE":      "Supply a source mapping with a nonempty resource identifying actual evidence.",
	"INVALID_COMPUTATION": "Correct the attested computation fields from evidence; never execute declared computations automatically.",
	"INVALID_RESERVED":    "Repair the reserved index/log structure explicitly; preserve manual content and chronological history.",
	"BROKEN_LINK":         "Read the source and intended target, then correct the link or restore the intended target; do not remove evidence just to clear a warning.",
	"ORPHAN":              "Add a meaningful incoming reference or index entry when appropriate; do not invent a relationship.",
	"UNINDEXED":           "Add the note to the appropriate index; managed index maintenance can help after format errors are resolved.",
	"BASIS_CHANGED":       "Read the current dependencies and review the synthesis; update recorded revisions only after reconciling its claims.",
	"STALE":               "Recheck the underlying evidence; do not renew dates or verification merely to clear this finding.",
}

func enrichDiagnostic(d api.Diagnostic) api.Diagnostic {
	d.Rule = d.Code
	if d.Code == "INVALID_NOTE" {
		rule, _, ok := strings.Cut(d.Message, ":")
		if ok {
			d.Rule = rule
		}
	}
	d.Severity = "error"
	if d.Layer == "advisories" {
		d.Severity = "warning"
	}
	d.Hint = ruleHelp[d.Rule]
	if d.Hint == "" {
		d.Hint = "Inspect this finding and its source before preparing an explicit correction."
	}
	return d
}

func validationReport(profile, generation string, diagnostics []api.Diagnostic, files map[string][]byte) api.ValidationReport {
	report := api.ValidationReport{Profile: profile, Generation: generation, Diagnostics: make([]api.Diagnostic, 0, len(diagnostics))}
	for _, finding := range diagnostics {
		d := enrichDiagnostic(finding)
		if raw, ok := files[d.Path]; ok {
			d.Revision = api.Revision(raw)
		}
		report.Diagnostics = append(report.Diagnostics, d)
		if d.Severity == "error" {
			report.Summary.Errors++
		} else {
			report.Summary.Warnings++
		}
	}
	report.Conformant = report.Summary.Errors == 0
	report.Clean = report.Conformant && report.Summary.Warnings == 0
	return report
}
