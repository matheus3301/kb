// Package setup plans and publishes opt-in agent integrations independently of
// collection storage. Agent adapters render intentions; this package owns files.
package setup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/matheus3301/kb/internal/api"
)

const SchemaVersion = 1
const maxFileBytes = 8 << 20

// Artifact owns only the indicated entry or block, not its containing user file.
type Artifact struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Key     string `json:"key,omitempty"`
	Content []byte `json:"content"`
	Shared  bool   `json:"shared,omitempty"`
}

// Binding selects one effective interface for an agent at one activation scope.
type Binding struct {
	Agent    string `json:"agent"`
	Scope    string `json:"scope"`
	Project  string `json:"project,omitempty"`
	Root     string `json:"root"`
	Mode     string `json:"mode"`
	Consult  string `json:"consult"`
	Learning string `json:"learning"`
	Hooks    string `json:"hooks"`
	Portable bool   `json:"portable,omitempty"`
	SkillDir string `json:"skill_dir,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

type Installation struct {
	ID                string     `json:"id"`
	Version           string     `json:"version"`
	Binding           Binding    `json:"binding"`
	Artifacts         []Artifact `json:"artifacts"`
	Diagnostics       []string   `json:"diagnostics,omitempty"`
	ActivationPending bool       `json:"activation_pending"`
}

// Options makes host paths explicit; tests must supply disposable homes.
type Options struct {
	Home       string
	ConfigHome string
	Version    string
	// Failpoint runs after durable journal preparation and each publication.
	// A returned error exercises rollback; a process crash leaves recoverable state.
	Failpoint func(string) error
}

type Engine struct {
	options  Options
	stateDir string
}

func New(options Options) (*Engine, error) {
	if options.Home == "" {
		return nil, invalid("an explicit home directory is required")
	}
	home, err := Canonical(options.Home)
	if err != nil {
		return nil, err
	}
	options.Home = home
	if options.ConfigHome == "" {
		options.ConfigHome = filepath.Join(home, ".config")
	}
	options.ConfigHome, err = Canonical(options.ConfigHome)
	if err != nil {
		return nil, err
	}
	return &Engine{options: options, stateDir: filepath.Join(options.ConfigHome, "kb", "setup-state")}, nil
}

func invalid(message string) error { return &api.DomainError{Code: "INVALID_INPUT", Message: message} }
func conflict(message string) error {
	return &api.DomainError{Code: "REVISION_CONFLICT", Message: message}
}
func recoveryRequired() error {
	return &api.DomainError{Code: "RECOVERY_REQUIRED", Message: "setup publication is pending; run kb setup recover", Retryable: true}
}
func digest(raw []byte) string { h := sha256.Sum256(raw); return "sha256:" + hex.EncodeToString(h[:]) }
func identity(b Binding) string {
	return digest([]byte(b.Scope + "\x00" + b.Project + "\x00" + b.Agent))[7:31]
}
func assetKey(a Artifact) string { return a.Path + "\x00" + a.Kind + "\x00" + a.Key }
func encode(v any) ([]byte, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	return append(raw, '\n'), err
}

// Snapshot includes existence: an absent file differs from an empty file.
type Snapshot struct {
	Exists  bool   `json:"exists"`
	Content []byte `json:"content,omitempty"`
	Mode    uint32 `json:"mode"`
}

func (s Snapshot) equal(other Snapshot) bool {
	return s.Exists == other.Exists && s.Mode == other.Mode && digest(s.Content) == digest(other.Content)
}

type Change struct {
	Path   string   `json:"path"`
	Before Snapshot `json:"before"`
	After  Snapshot `json:"after"`
}

type asset struct {
	Artifact Artifact `json:"artifact"`
	Borrowed bool     `json:"borrowed"`
	Managed  []byte   `json:"managed,omitempty"`
	Owners   []string `json:"owners"`
}
type fileState struct {
	Before   Snapshot `json:"before"`
	Expected Snapshot `json:"expected"`
	Pristine bool     `json:"pristine"`
}
type catalog struct {
	Schema        int                     `json:"schema_version"`
	Installations map[string]Installation `json:"installations"`
	Assets        map[string]asset        `json:"assets"`
	Files         map[string]fileState    `json:"files"`
}

func emptyCatalog() catalog {
	return catalog{Schema: SchemaVersion, Installations: map[string]Installation{}, Assets: map[string]asset{}, Files: map[string]fileState{}}
}

type scopeManifest struct {
	Schema   int                `json:"schema_version"`
	Bindings map[string]Binding `json:"bindings"`
}

// Plan captures every precondition, including unchanged borrowed/shared assets.
// Build it afresh after a conflict. Dry-run consists only of building a Plan.
type Plan struct {
	Schema        int                 `json:"schema_version"`
	Changes       []Change            `json:"changes"`
	Preconditions map[string]Snapshot `json:"-"`
	Installations []Installation      `json:"installations"`
	OperationID   string              `json:"operation_id,omitempty"`
	stateDir      string
}

// ChangeSummary deliberately excludes before/after bytes: shared host settings
// may contain credentials unrelated to this installation.
type ChangeSummary struct {
	Path       string `json:"path"`
	Action     string `json:"action"`
	BeforeHash string `json:"before_hash,omitempty"`
	AfterHash  string `json:"after_hash,omitempty"`
	Mode       uint32 `json:"mode"`
}
type Result struct {
	Schema      int             `json:"setup_schema_version"`
	Status      string          `json:"status"`
	DryRun      bool            `json:"dry_run"`
	Changes     []ChangeSummary `json:"changes,omitempty"`
	Bindings    []Binding       `json:"bindings,omitempty"`
	Diagnostics []string        `json:"diagnostics,omitempty"`
}

func (p *Plan) Result(dry bool) Result {
	r := Result{Schema: SchemaVersion, Status: "configured", DryRun: dry}
	if len(p.Changes) == 0 {
		r.Status = "already_configured"
	}
	for _, c := range p.Changes {
		a := "update"
		if !c.Before.Exists {
			a = "create"
		}
		if !c.After.Exists {
			a = "remove"
		}
		r.Changes = append(r.Changes, ChangeSummary{Path: c.Path, Action: a, BeforeHash: digest(c.Before.Content), AfterHash: digest(c.After.Content), Mode: c.After.Mode})
	}
	for _, in := range p.Installations {
		r.Bindings = append(r.Bindings, in.Binding)
		r.Diagnostics = append(r.Diagnostics, in.Diagnostics...)
		if in.ActivationPending {
			r.Status = "activation_pending"
		}
	}
	return r
}

func (e *Engine) statePath() string   { return filepath.Join(e.stateDir, "manifest.json") }
func (e *Engine) pendingPath() string { return filepath.Join(e.stateDir, "pending.json") }

func (e *Engine) readCatalog() (catalog, Snapshot, error) {
	if pending, err := readSnapshot(e.pendingPath()); err != nil {
		return catalog{}, Snapshot{}, err
	} else if pending.Exists {
		return catalog{}, Snapshot{}, recoveryRequired()
	}
	snap, err := readSnapshot(e.statePath())
	if err != nil {
		return catalog{}, snap, err
	}
	c := emptyCatalog()
	if !snap.Exists {
		return c, snap, nil
	}
	if err = decodeManifest(snap.Content, &c); err != nil {
		return c, snap, err
	}
	if c.Schema != SchemaVersion {
		return c, snap, conflict(fmt.Sprintf("unsupported setup manifest version %d", c.Schema))
	}
	if c.Installations == nil || c.Assets == nil || c.Files == nil {
		return c, snap, conflict("incomplete setup manifest")
	}
	return c, snap, nil
}

func validateBinding(b *Binding) error {
	if b.Agent == "" {
		return invalid("select at least one --agent")
	}
	if b.Scope != "global" && b.Scope != "project" {
		return invalid("select --project PATH or --global")
	}
	if b.Scope == "project" {
		p, err := Canonical(b.Project)
		if err != nil {
			return err
		}
		b.Project = p
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return invalid("project must be a directory")
		}
	}
	if b.Scope == "global" && b.Project != "" {
		return invalid("global and project scopes conflict")
	}
	if b.Mode != "cli" && b.Mode != "mcp" {
		return invalid("mode must be cli or mcp")
	}
	if b.Consult != "always" && b.Consult != "relevant" {
		return invalid("consult must be always or relevant")
	}
	if b.Learning != "on" && b.Learning != "off" {
		return invalid("select --learning on or off")
	}
	if b.Hooks != "auto" && b.Hooks != "off" && b.Hooks != "required" {
		return invalid("hooks must be auto, off or required")
	}
	if b.Root == "" {
		return invalid("select a Knowledge Base with --root")
	}
	p, err := Canonical(b.Root)
	if err != nil {
		return err
	}
	b.Root = p
	return nil
}

// PlanInstall reconciles all requested agents as one preflight/publication unit.
// Switching mode replaces obsolete assets rather than retaining an alias mode.
func (e *Engine) PlanInstall(ctx context.Context, installs []Installation) (*Plan, error) {
	if len(installs) == 0 {
		return nil, invalid("select at least one --agent")
	}
	c, snap, err := e.readCatalog()
	if err != nil {
		return nil, err
	}
	next := cloneCatalog(c)
	for _, in := range installs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := validateBinding(&in.Binding); err != nil {
			return nil, err
		}
		in.ID = identity(in.Binding)
		in.Version = e.options.Version
		if in.Binding.Portable {
			for _, a := range in.Artifacts {
				if bytes.Contains(a.Content, []byte(e.options.Home)) || bytes.Contains(a.Content, []byte(in.Binding.Root)) {
					return nil, invalid("portable artifact contains an absolute personal path: " + a.Path)
				}
			}
		}
		portable, err := PortableArtifacts(in.Binding, e.options.Home)
		if err != nil {
			return nil, err
		}
		in.Artifacts = append(in.Artifacts, portable...)
		private, notes, err := PrivacyArtifacts(ctx, in.Binding, in.Artifacts, e.options.Home)
		if err != nil {
			return nil, err
		}
		in.Artifacts = append(in.Artifacts, private...)
		in.Diagnostics = append(in.Diagnostics, notes...)
		for i := range in.Artifacts {
			a := &in.Artifacts[i]
			p, err := Canonical(a.Path)
			if err != nil {
				return nil, err
			}
			if p != a.Path {
				return nil, invalid("artifact paths must be canonical: " + a.Path)
			}
			if a.Kind != "file" && a.Kind != "block" && a.Kind != "gitignore" && a.Kind != "json" && a.Kind != "json-array" && a.Kind != "jsonc" && a.Kind != "jsonc-array" && a.Kind != "toml" {
				return nil, invalid("unsupported artifact kind: " + a.Kind)
			}
			if a.Kind != "file" && a.Key == "" {
				return nil, invalid("managed entries require a key")
			}
			if a.Kind == "json" || a.Kind == "json-array" || a.Kind == "jsonc" || a.Kind == "jsonc-array" {
				a.Content = bytes.TrimSpace(a.Content)
			}
		}
		next.Installations[in.ID] = in
	}
	return e.build(ctx, c, next, snap)
}

type Selection struct {
	Scope   string
	Project string
	Agents  []string
	Mode    string
}

func (e *Engine) PlanUninstall(ctx context.Context, s Selection) (*Plan, error) {
	if len(s.Agents) == 0 {
		return nil, invalid("select at least one --agent; uninstall never means all agents implicitly")
	}
	if s.Scope != "project" && s.Scope != "global" {
		return nil, invalid("select --project PATH or --global")
	}
	if s.Mode != "" && s.Mode != "cli" && s.Mode != "mcp" {
		return nil, invalid("mode must be cli or mcp")
	}
	if s.Scope == "project" {
		var err error
		s.Project, err = Canonical(s.Project)
		if err != nil {
			return nil, err
		}
	}
	c, snap, err := e.readCatalog()
	if err != nil {
		return nil, err
	}
	next := cloneCatalog(c)
	for id, in := range next.Installations {
		b := in.Binding
		if b.Scope != s.Scope || b.Project != s.Project || (s.Mode != "" && b.Mode != s.Mode) {
			continue
		}
		for _, agent := range s.Agents {
			if b.Agent == agent {
				delete(next.Installations, id)
			}
		}
	}
	return e.build(ctx, c, next, snap)
}
func cloneCatalog(c catalog) catalog {
	n := emptyCatalog()
	for k, v := range c.Installations {
		n.Installations[k] = v
	}
	for k, v := range c.Assets {
		n.Assets[k] = v
	}
	for k, v := range c.Files {
		n.Files[k] = v
	}
	return n
}
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
