package setup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/matheus3301/kb/internal/api"
)

type Resolution struct {
	Schema int `json:"setup_schema_version"`
	Binding
	Source      string   `json:"source"`
	Status      string   `json:"status"`
	Diagnostics []string `json:"diagnostics,omitempty"`
}

// Resolve searches ancestors only through the nearest Git/worktree boundary.
// A saved binding wins over ambient KB_ROOT; data commands do not call Resolve.
func (e *Engine) Resolve(ctx context.Context, agent, cwd, environmentRoot string) (Resolution, error) {
	if agent == "" {
		return Resolution{}, invalid("select --agent")
	}
	cwd, err := Canonical(cwd)
	if err != nil {
		return Resolution{}, err
	}
	c, _, err := e.readCatalog()
	if err != nil {
		return Resolution{}, err
	}
	var global *Binding
	for _, in := range c.Installations {
		b := in.Binding
		if b.Agent == agent && b.Scope == "global" {
			copy := b
			global = &copy
		}
	}
	finish := func(b Binding, source string) (Resolution, error) {
		r := Resolution{Schema: SchemaVersion, Binding: b, Source: source, Status: "configured"}
		if b.Disabled {
			r.Status = "disabled"
		}
		if environmentRoot != "" {
			env, err := Canonical(environmentRoot)
			if err != nil || env != b.Root {
				r.Diagnostics = append(r.Diagnostics, "ambient KB_ROOT differs from the selected integration; explicit binding wins")
			}
		}
		return r, nil
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if err := ctx.Err(); err != nil {
			return Resolution{}, err
		}
		for _, in := range c.Installations {
			b := in.Binding
			if b.Agent == agent && b.Scope == "project" && b.Project == dir {
				return finish(b, e.statePath())
			}
		}
		path := filepath.Join(dir, ".kb-agent", "setup.json")
		snap, err := readSnapshot(path)
		if err != nil {
			return Resolution{}, err
		}
		if snap.Exists {
			var m scopeManifest
			if err := decodeManifest(snap.Content, &m); err != nil {
				return Resolution{}, err
			}
			if m.Schema != SchemaVersion {
				return Resolution{}, conflict("unsupported portable binding version")
			}
			if b, ok := m.Bindings[agent]; ok {
				if !b.Portable || b.Scope != "project" || b.Agent != agent || filepath.IsAbs(b.Root) || b.Project != "" {
					return Resolution{}, conflict("invalid portable binding; root must be relative to its project")
				}
				b.Project = dir
				b.Root, err = Canonical(filepath.Join(dir, b.Root))
				if err != nil {
					return Resolution{}, err
				}
				if b.SkillDir != "" && !filepath.IsAbs(b.SkillDir) {
					b.SkillDir = filepath.Join(dir, b.SkillDir)
				}
				if err := validateBinding(&b); err != nil {
					return Resolution{}, err
				}
				return finish(b, path)
			}
		}
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return Resolution{}, err
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	if global != nil {
		return finish(*global, filepath.Join(e.options.ConfigHome, "kb", "agent-setup.json"))
	}
	return Resolution{}, &api.DomainError{Code: "NOT_FOUND", Message: "no integration binding for " + agent + "; run kb setup install with an explicit scope and root"}
}

type ArtifactStatus struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Status   string `json:"status"`
	Borrowed bool   `json:"borrowed"`
}
type CheckResult struct {
	Resolution
	Artifacts          []ArtifactStatus `json:"artifacts"`
	Configured         bool             `json:"configured"`
	Ready              bool             `json:"ready"`
	ActivationObserved bool             `json:"activation_observed"`
}

func (e *Engine) Check(ctx context.Context, agent, cwd, environmentRoot string) (CheckResult, error) {
	resolved, err := e.Resolve(ctx, agent, cwd, environmentRoot)
	if err != nil {
		return CheckResult{}, err
	}
	result := CheckResult{Resolution: resolved, Configured: true, Artifacts: []ArtifactStatus{}}
	if resolved.Disabled {
		return result, nil
	}
	c, _, err := e.readCatalog()
	if err != nil {
		return result, err
	}
	id := identity(resolved.Binding)
	if in, ok := c.Installations[id]; ok {
		if in.ActivationPending {
			result.Status = "activation_pending"
		}
		result.Diagnostics = append(result.Diagnostics, in.Diagnostics...)
	} else {
		result.Diagnostics = append(result.Diagnostics, "portable binding is not locally installed; artifact ownership cannot be verified")
		result.Configured = false
	}
	for _, key := range sortedKeys(c.Assets) {
		a := c.Assets[key]
		owned := false
		for _, owner := range a.Owners {
			owned = owned || owner == id
		}
		if !owned {
			continue
		}
		status := ArtifactStatus{Path: a.Artifact.Path, Kind: a.Artifact.Kind, Status: "configured", Borrowed: a.Borrowed}
		snap, err := readSnapshot(a.Artifact.Path)
		if err != nil {
			status.Status = "conflict"
		} else if !snap.Exists {
			status.Status = "missing"
		} else {
			_, _, _, err = editArtifact(snap.Content, a.Artifact, &a, false)
			if err != nil {
				status.Status = "conflict"
			}
		}
		if status.Status != "configured" {
			result.Configured = false
			result.Status = "conflict"
			result.Diagnostics = append(result.Diagnostics, status.Status+": "+status.Path)
		}
		result.Artifacts = append(result.Artifacts, status)
	}
	if result.Configured {
		result.Ready = true
	}
	return result, nil
}

// ProtectCollection refuses newly introduced raw Markdown in the selected
// collection unless that path is already excluded by the collection profile.
// Existing valid scoped notes can retain their metadata and receive a block.
func ProtectCollection(root string, artifacts []Artifact, included func(string) bool) error {
	for _, a := range artifacts {
		if !within(root, a.Path) || !strings.EqualFold(filepath.Ext(a.Path), ".md") {
			continue
		}
		rel, err := filepath.Rel(root, a.Path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if included != nil && !included(rel) {
			continue
		}
		if a.Kind == "block" {
			snap, err := readSnapshot(a.Path)
			if err != nil {
				return err
			}
			content := strings.TrimPrefix(string(snap.Content), "\ufeff")
			if snap.Exists && (strings.HasPrefix(content, "---\n") || strings.HasPrefix(content, "---\r\n")) {
				continue
			}
		}
		return conflict("agent artifact would introduce non-OKF Markdown into the selected collection: " + rel + "; select a separate project or explicitly exclude the agent directory in the collection configuration before setup")
	}
	return nil
}
