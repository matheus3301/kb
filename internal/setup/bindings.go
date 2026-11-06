package setup

import (
	"context"
	"errors"

	"github.com/matheus3301/kb/internal/api"
)

// Bindings lists already configured effective integrations for wizard reruns.
// It does not discover agents, scan homes, or authorize additional installation.
func (e *Engine) Bindings(ctx context.Context, cwd string) ([]Binding, error) {
	c, _, err := e.readCatalog()
	if err != nil {
		return nil, err
	}
	agents := map[string]bool{}
	for _, in := range c.Installations {
		agents[in.Binding.Agent] = true
	}
	var result []Binding
	for _, agent := range sortedKeys(agents) {
		resolved, err := e.Resolve(ctx, agent, cwd, "")
		if err != nil {
			var domain *api.DomainError
			if errors.As(err, &domain) && domain.Code == "NOT_FOUND" {
				continue
			}
			return nil, err
		}
		result = append(result, resolved.Binding)
	}
	return result, nil
}
