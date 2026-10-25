package write

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/matheus3301/kb/internal/api"
)

func cloneFiles(files map[string][]byte) map[string][]byte {
	copy := make(map[string][]byte, len(files))
	for name, data := range files {
		copy[name] = bytes.Clone(data)
	}
	return copy
}

func reserved(name string) bool {
	base := path.Base(name)
	return base == "index.md" || base == "log.md"
}

func (s *Service) conform(files map[string][]byte) error {
	if err := validatePathSet(files); err != nil {
		return err
	}
	for name, raw := range files {
		if len(raw) > api.MaxNoteBytes || !utf8.Valid(raw) {
			return problem("INPUT_TOO_LARGE", "note is oversized or invalid UTF-8")
		}
		var diagnostics []api.Diagnostic
		if reserved(name) {
			diagnostics = s.provider.Engine().ValidateReserved(name, raw)
		} else {
			note, findings := s.provider.Engine().Parse(name, raw)
			diagnostics = findings
			if strings.TrimSpace(note.Type) == "" {
				return problem("INVALID_FRONTMATTER", "every concept requires a nonempty textual type")
			}
		}
		for _, finding := range diagnostics {
			if finding.Layer == "conformance_errors" || finding.Layer == "policy_findings" || finding.Layer == "conformance" || finding.Layer == "" {
				return &api.DomainError{Code: "INVALID_FRONTMATTER", Message: "base does not conform", Details: diagnostics}
			}
		}
	}
	return nil
}

func (s *Service) prepare(root *os.Root, snapshot api.Snapshot, plan api.Plan) (map[string][]byte, map[string][]byte, []api.Diagnostic, error) {
	before := cloneFiles(snapshot.Files)
	for name := range before {
		if !inScope(name, s.provider.Config()) {
			return nil, nil, nil, problem("INVALID_STATE", "provider inventory contains an out-of-scope path")
		}
		if err := inspect(root, name); err != nil {
			return nil, nil, nil, err
		}
	}
	if !plan.Repair {
		if snapshot.Degraded {
			return nil, nil, nil, problem("INVALID_FRONTMATTER", "normal writes require a conformant base")
		}
		if err := s.conform(before); err != nil {
			return nil, nil, nil, err
		}
	}
	after := cloneFiles(before)
	seenPaths, seenIDs := map[string]bool{}, map[string]bool{}
	moves := map[string]string{}
	for _, op := range plan.Operations {
		if op.OperationID == "" || seenIDs[op.OperationID] {
			return nil, nil, nil, problem("INVALID_PLAN", "operations require unique operation_id values")
		}
		seenIDs[op.OperationID] = true
		if op.Kind != "create" && op.Kind != "patch" && op.Kind != "move" {
			return nil, nil, nil, problem("INVALID_PLAN", "unsupported operation kind")
		}
		if plan.Incremental && op.Kind != "patch" {
			return nil, nil, nil, problem("INVALID_PLAN", "incremental repair accepts patches to existing files only")
		}
		paths := []string{op.Path}
		if op.Kind == "move" {
			paths = append(paths, op.To)
		} else if op.To != "" {
			return nil, nil, nil, problem("INVALID_PLAN", "to is only valid for move")
		}
		for _, name := range paths {
			if err := cleanPath(name); err != nil {
				return nil, nil, nil, err
			}
			if !inScope(name, s.provider.Config()) {
				return nil, nil, nil, problem("PATH_OUTSIDE_ROOT", "file is outside the configured Markdown scope")
			}
			if seenPaths[caseKey(name)] {
				return nil, nil, nil, problem("INVALID_PLAN", "operations overlap or collide in letter case")
			}
			for existing := range seenPaths {
				folded := caseKey(name)
				if strings.HasPrefix(folded, existing+"/") || strings.HasPrefix(existing, folded+"/") {
					return nil, nil, nil, problem("INVALID_PLAN", "operation paths overlap as files and directories")
				}
			}
			seenPaths[caseKey(name)] = true
			if err := inspect(root, name); err != nil {
				return nil, nil, nil, err
			}
		}
		if op.Document != nil && !plan.Repair {
			return nil, nil, nil, problem("INVALID_PLAN", "document replacement requires repair")
		}
		if op.Body != nil && len(op.BodyEdits) > 0 {
			return nil, nil, nil, problem("INVALID_PLAN", "body and body_edits are mutually exclusive")
		}
		if op.Document != nil && (op.Body != nil || len(op.BodyEdits) > 0 || len(op.MetadataPatch) > 0 || len(op.RemoveMetadataKeys) > 0 || len(op.Metadata) > 0 || op.Type != "" || op.Title != "") {
			return nil, nil, nil, problem("INVALID_PLAN", "document replacement cannot be combined with partial edits")
		}
		if _, exists := op.Metadata["verified"]; exists {
			return nil, nil, nil, problem("INVALID_PLAN", "writes cannot assert human verification")
		}
		if value, exists := op.MetadataPatch["verified"]; exists && value != nil {
			return nil, nil, nil, problem("INVALID_PLAN", "writes cannot assert human verification")
		}
		old, exists := before[op.Path]
		if op.Kind == "create" {
			if exists {
				return nil, nil, nil, problem("PATH_COLLISION", "create requires a new target")
			}
			if op.ExpectedRevision != "" || len(op.MetadataPatch) > 0 || len(op.RemoveMetadataKeys) > 0 || len(op.BodyEdits) > 0 {
				return nil, nil, nil, problem("INVALID_PLAN", "create uses type, title, metadata and body")
			}
			typeName := op.Type
			if typeName == "" {
				typeName, _ = op.Metadata["type"].(string)
			}
			if strings.TrimSpace(typeName) == "" && op.Document == nil {
				return nil, nil, nil, problem("INVALID_FRONTMATTER", "create requires a textual type")
			}
		} else {
			if !exists {
				return nil, nil, nil, problem("NOT_FOUND", "operation target does not exist in scope")
			}
			if op.ExpectedRevision == "" || op.ExpectedRevision != api.Revision(old) {
				return nil, nil, nil, problem("REVISION_CONFLICT", "expected_revision does not match current bytes")
			}
			if len(op.Metadata) > 0 || op.Type != "" || op.Title != "" {
				return nil, nil, nil, problem("INVALID_PLAN", "patch and move use metadata_patch")
			}
			if plan.Repair && !reserved(op.Path) {
				note, _ := s.provider.Engine().Parse(op.Path, old)
				if note.Type == "" && op.Document == nil {
					value, ok := op.MetadataPatch["type"].(string)
					if !ok || strings.TrimSpace(value) == "" {
						return nil, nil, nil, problem("INVALID_PLAN", "repair requires explicit document or metadata correction")
					}
				}
			}
		}
		if op.Kind == "move" {
			if reserved(op.Path) || reserved(op.To) {
				return nil, nil, nil, problem("INVALID_PLAN", "reserved navigation files cannot be moved")
			}
			if _, exists := before[op.To]; exists {
				return nil, nil, nil, problem("PATH_COLLISION", "move destination already exists")
			}
			moves[op.Path] = op.To
		}
		if op.Kind == "create" || op.Kind == "move" {
			target := op.Path
			if op.Kind == "move" {
				target = op.To
			}
			value, err := readImage(root, target, api.MaxNoteBytes)
			if err != nil {
				return nil, nil, nil, err
			}
			if value.Exists {
				return nil, nil, nil, problem("PATH_COLLISION", "destination exists on disk")
			}
		}
	}
	var warnings []api.Diagnostic
	authored := map[string]string{}
	for _, op := range plan.Operations {
		old := before[op.Path]
		var fresh []byte
		var err error
		if op.Document != nil {
			fresh = []byte(*op.Document)
		} else if op.Kind == "move" && op.Body == nil && len(op.BodyEdits) == 0 && len(op.MetadataPatch) == 0 && len(op.RemoveMetadataKeys) == 0 {
			fresh = bytes.Clone(old)
		} else {
			fresh, err = s.provider.Engine().Patch(op.Path, old, op, plan.Actor, s.now().UTC())
		}
		if err != nil {
			return nil, nil, nil, err
		}
		if !bytes.Equal(old, fresh) && len(old) > 0 && !reserved(op.Path) {
			oldNote, _ := s.provider.Engine().Parse(op.Path, old)
			if _, exists := oldNote.Metadata["verified"]; exists {
				fresh, err = s.provider.Engine().Patch(op.Path, fresh, api.Operation{Kind: "patch", RemoveMetadataKeys: []string{"verified"}}, plan.Actor, s.now().UTC())
				if err != nil {
					return nil, nil, nil, err
				}
				warnings = append(warnings, api.Diagnostic{Path: op.Path, Code: "VERIFICATION_CLEARED", Layer: "advisories", Message: "content changed; prior human verification was cleared"})
			}
		}
		if op.Document != nil {
			note, _ := s.provider.Engine().Parse(op.Path, fresh)
			if _, exists := note.Metadata["verified"]; exists {
				return nil, nil, nil, problem("INVALID_PLAN", "document replacement cannot assert human verification")
			}
		}
		target := op.Path
		if op.Kind == "move" {
			delete(after, op.Path)
			target = op.To
		}
		after[target] = fresh
		if !bytes.Equal(old, fresh) && (op.Kind != "move" || op.Document != nil || op.Body != nil || len(op.BodyEdits) > 0 || len(op.MetadataPatch) > 0 || len(op.RemoveMetadataKeys) > 0) {
			authored[target] = op.Path
		}
	}
	if len(moves) > 0 {
		var err error
		warnings, err = s.relocate(snapshot, before, after, moves, plan.Actor, warnings)
		if err != nil {
			return nil, nil, nil, err
		}
		warnings = append(warnings, s.basisWarnings(before, after)...)
	}
	conform := func() error { return s.conform(after) }
	if plan.Incremental {
		conform = func() error { return s.conformIncremental(before, after) }
		if prior := s.errorCount(before); prior > 0 && s.errorCount(after) >= prior {
			return nil, nil, nil, problem("INVALID_PLAN", "incremental repair must reduce existing format/policy errors")
		}
	}
	if err := conform(); err != nil {
		return nil, nil, nil, err
	}
	if changed(before, after) {
		if s.provider.Config().Write.MaintainIndexes {
			if plan.Incremental && !s.canMaintain(after, "index.md") {
				warnings = append(warnings, deferredNavigation("index.md"))
			} else if err := s.indexes(after); err != nil {
				return nil, nil, nil, err
			}
		}
		if s.provider.Config().Write.MaintainLog {
			if plan.Incremental && !s.canMaintain(after, "log.md") {
				warnings = append(warnings, deferredNavigation("log.md"))
			} else if err := s.log(after, plan.Summary, plan.Actor, warnings, before); err != nil {
				return nil, nil, nil, err
			}
		}
	}
	if err := conform(); err != nil {
		return nil, nil, nil, err
	}
	if err := s.dependencies(before, after, moves, authored); err != nil {
		return nil, nil, nil, err
	}
	return before, after, warnings, nil
}

func changed(before, after map[string][]byte) bool {
	if len(before) != len(after) {
		return true
	}
	for name, data := range before {
		if !bytes.Equal(data, after[name]) {
			return true
		}
	}
	return false
}

func metadataJSON(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	err = decoder.Decode(&result)
	return result, err
}

func (s *Service) dependencies(before, after map[string][]byte, moves map[string]string, authored map[string]string) error {
	for name, original := range authored {
		raw := after[name]
		if reserved(name) || bytes.Equal(before[original], raw) {
			continue
		}
		note, _ := s.provider.Engine().Parse(name, raw)
		value, exists := note.Metadata["basis_revisions"]
		if !exists {
			continue
		}
		value, err := metadataJSON(value)
		if err != nil {
			return problem("INVALID_PLAN", "basis_revisions must be JSON-compatible")
		}
		items, ok := value.([]any)
		if !ok {
			return problem("INVALID_PLAN", "basis_revisions must be a list")
		}
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				return problem("INVALID_PLAN", "basis_revisions entries require path and revision")
			}
			dependency, _ := entry["path"].(string)
			revision, _ := entry["revision"].(string)
			if !inScope(dependency, s.provider.Config()) || !validRevision(revision) {
				return problem("INVALID_PLAN", "invalid synthesis dependency")
			}
			if moved, ok := moves[dependency]; ok {
				dependency = moved
			}
			data, exists := after[dependency]
			if !exists || api.Revision(data) != revision {
				return problem("REVISION_CONFLICT", "synthesis dependency revision changed")
			}
		}
	}
	return nil
}

// Automatic reference rewrites retain recorded revisions, even when outdated.
func (s *Service) basisWarnings(before, after map[string][]byte) []api.Diagnostic {
	var warnings []api.Diagnostic
	for name, raw := range after {
		if reserved(name) || bytes.Equal(before[name], raw) {
			continue
		}
		note, _ := s.provider.Engine().Parse(name, raw)
		value, err := metadataJSON(note.Metadata["basis_revisions"])
		if err != nil {
			continue
		}
		items, _ := value.([]any)
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			dependency, _ := entry["path"].(string)
			revision, _ := entry["revision"].(string)
			data, exists := after[dependency]
			if !exists || api.Revision(data) != revision {
				warnings = append(warnings, api.Diagnostic{Path: name, Code: "BASIS_CHANGED", Layer: "advisories", Message: "a recorded dependency revision changed or is unavailable; review required"})
				break
			}
		}
	}
	return warnings
}
