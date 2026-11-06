package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/internal/setup"
)

// ready means local artifacts, selected collection and host executable exist.
// It does not establish native version compatibility, trust, rule loading or
// retrieval in a fresh host session; activation_observed stays false.
func (r *runner) setupReadiness(ctx context.Context, result *setup.CheckResult) {
	if result.Disabled {
		return
	}
	reader, _, err := r.deps.Open(result.Root)
	if err != nil {
		result.Ready = false
		result.Status = "degraded"
		result.Diagnostics = append(result.Diagnostics, "selected collection is unavailable")
	} else if inspector, ok := reader.(interface {
		InspectSetup(context.Context) (api.ValidationReport, error)
	}); ok {
		report, inspectErr := inspector.InspectSetup(ctx)
		if inspectErr != nil || !report.Conformant {
			result.Ready = false
			result.Status = "degraded"
			result.Diagnostics = append(result.Diagnostics, "selected collection requires attention")
		}
	}
	binary := result.Agent
	if binary == "copilot" {
		binary = "code"
	}
	if _, err := exec.LookPath(binary); err != nil {
		result.Ready = false
		result.Status = "degraded"
		result.Diagnostics = append(result.Diagnostics, "native agent executable is unavailable on PATH: "+binary)
	}
	command := result.Executable
	if command == "kb" {
		if _, err := exec.LookPath(command); err != nil {
			result.Ready = false
			result.Status = "degraded"
			result.Diagnostics = append(result.Diagnostics, "portable kb command is unavailable on PATH")
		}
	} else {
		//nolint:gosec // Only stats the explicitly recorded integration executable; never executes it or reads arbitrary collection content.
		info, err := os.Stat(command)
		if command == "" || err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
			result.Ready = false
			result.Status = "degraded"
			result.Diagnostics = append(result.Diagnostics, "configured kb executable is unavailable or not executable")
		}
	}
	host, err := r.setupHost()
	if err != nil {
		result.Ready = false
		result.Status = "degraded"
		result.Diagnostics = append(result.Diagnostics, "current native profile could not be inspected")
	} else {
		f := &setupFlags{mode: result.Mode, consult: result.Consult, learning: result.Learning, hooks: result.Hooks, agents: []string{result.Agent}, portable: result.Portable}
		// Rendering uses this runner's root only as an explicit input; restore it
		// after inspection so a check never changes a later command's data root.
		savedRoot := r.root
		r.root = result.Root
		current, renderErr := r.renderInstall(host, f, result.Scope, result.Project)
		r.root = savedRoot
		if renderErr != nil {
			result.Ready = false
			result.Status = "degraded"
			result.Diagnostics = append(result.Diagnostics, "current native profile/configuration is unsupported")
		} else {
			installed := map[string]bool{}
			for _, a := range result.Artifacts {
				installed[filepath.Clean(a.Path)] = true
			}
			for _, a := range allArtifacts(current) {
				if !installed[filepath.Clean(a.Path)] {
					result.Ready = false
					result.Status = "degraded"
					result.Diagnostics = append(result.Diagnostics, "native instruction/profile precedence changed; reconcile setup: "+a.Path)
					break
				}
			}
		}
	}
	result.Diagnostics = append(result.Diagnostics, "ready checks local artifacts/root/executable only; native version, trust and session activation remain unobserved")
}
