package cli

import (
	"context"
	"os/exec"

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
	result.Diagnostics = append(result.Diagnostics, "ready checks local artifacts/root/executable only; native version, trust and session activation remain unobserved")
}
