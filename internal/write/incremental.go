package write

import (
	"bytes"
	"fmt"
	"path"

	"github.com/matheus3301/kb/internal/api"
)

func (s *Service) fileDiagnostics(name string, raw []byte) []api.Diagnostic {
	if reserved(name) {
		return s.provider.Engine().ValidateReserved(name, raw)
	}
	_, ds := s.provider.Engine().Parse(name, raw)
	return ds
}

func blocking(d api.Diagnostic) bool {
	return d.Layer == "conformance_errors" || d.Layer == "policy_findings" || d.Layer == "conformance" || d.Layer == ""
}

func (s *Service) errorCount(files map[string][]byte) int {
	count := 0
	for name, raw := range files {
		for _, d := range s.fileDiagnostics(name, raw) {
			if blocking(d) {
				count++
			}
		}
	}
	return count
}

// Incremental repair never carries an invalid changed file into publication.
// Unchanged invalid files keep exactly their original bytes and diagnostics.
func (s *Service) conformIncremental(before, after map[string][]byte) error {
	if err := validatePathSet(after); err != nil {
		return err
	}
	for name, raw := range after {
		if bytes.Equal(before[name], raw) {
			continue
		}
		if err := s.conform(map[string][]byte{name: raw}); err != nil {
			return err
		}
	}
	for name := range before {
		if _, ok := after[name]; !ok {
			return problem("INVALID_PLAN", "incremental repair cannot delete or move files")
		}
	}
	return nil
}

func (s *Service) canMaintain(files map[string][]byte, base string) bool {
	for name, raw := range files {
		if path.Base(name) != base {
			continue
		}
		for _, d := range s.fileDiagnostics(name, raw) {
			if blocking(d) {
				return false
			}
		}
	}
	return true
}

func deferredNavigation(base string) api.Diagnostic {
	return api.Diagnostic{Code: "NAVIGATION_DEFERRED", Layer: "advisories", Severity: "warning", Rule: "NAVIGATION_DEFERRED", Message: fmt.Sprintf("%s maintenance deferred until its existing format errors are repaired", base), Hint: "Repair the invalid reserved files, then run structural maintenance after the base conforms."}
}
