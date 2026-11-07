package cli

import (
	"fmt"
	"io"

	"github.com/matheus3301/kb/internal/agentsetup"
	"github.com/matheus3301/kb/internal/setup"
)

// setupHuman formats only explicit setup commands. Existing data commands keep
// their default JSON envelope. Shared configuration bytes never reach output.
func (r *runner) setupHuman(data any) (bool, error) {
	write := func(format string, args ...any) error { _, err := fmt.Fprintf(r.out, format, args...); return err }
	var result *setup.Result
	switch value := data.(type) {
	case setup.Result:
		result = &value
	case *setup.Result:
		result = value
	case setup.Resolution:
		if err := write("%s: %s %s; root=%s mode=%s consult=%s learning=%s\n", value.Status, value.Agent, value.Scope, value.Root, value.Mode, value.Consult, value.Learning); err != nil {
			return true, err
		}
		for _, note := range value.Diagnostics {
			if err := write("  %s\n", note); err != nil {
				return true, err
			}
		}
		return true, nil
	case setup.CheckResult:
		if err := write("%s: %s %s; root=%s mode=%s configured=%t ready=%t activation_observed=%t\n", value.Status, value.Agent, value.Scope, value.Root, value.Mode, value.Configured, value.Ready, value.ActivationObserved); err != nil {
			return true, err
		}
		for _, a := range value.Artifacts {
			if err := write("  %s %s\n", a.Status, a.Path); err != nil {
				return true, err
			}
		}
		for _, note := range value.Diagnostics {
			if err := write("  %s\n", note); err != nil {
				return true, err
			}
		}
		return true, nil
	case []agentsetup.Descriptor:
		for _, a := range value {
			if err := write("%s: project=%t global=%t mcp=%t hooks=%t\n", a.ID, a.Project, a.Global, a.MCP, a.Hooks); err != nil {
				return true, err
			}
		}
		return true, nil
	case map[string]any:
		if status, ok := value["status"].(string); ok {
			_, err := io.WriteString(r.out, status+"\n")
			return true, err
		}
	}
	if result == nil {
		return false, nil
	}
	preview := ""
	if result.DryRun {
		preview = " (dry-run; no changes made)"
	}
	if err := write("%s%s\n", result.Status, preview); err != nil {
		return true, err
	}
	for _, b := range result.Bindings {
		if err := write("  %s %s: root=%s mode=%s consult=%s learning=%s\n", b.Agent, b.Scope, b.Root, b.Mode, b.Consult, b.Learning); err != nil {
			return true, err
		}
	}
	for _, c := range result.Changes {
		if err := write("  %s %s\n", c.Action, c.Path); err != nil {
			return true, err
		}
	}
	for _, note := range result.Diagnostics {
		if err := write("  %s\n", note); err != nil {
			return true, err
		}
	}
	return true, nil
}
