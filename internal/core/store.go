package core

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/internal/filelock"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

const indexStart = "<!-- kb:index:start -->"
const indexEnd = "<!-- kb:index:end -->"

var alwaysExcluded = map[string]bool{".kb": true, ".git": true, "node_modules": true, "vendor": true, ".claude": true, ".codex": true, ".agents": true, ".obsidian": true, ".venv": true, "venv": true, "__pycache__": true, ".idea": true}

// Store binds all operations to one initialized local base.
// Configure SetAsOf before sharing the store with read workers.
type Store struct {
	root   string
	config api.Config
	mu     sync.Mutex
	asOf   time.Time
	cached *api.Snapshot
}

func (s *Store) Root() string { return s.root }
func (s *Store) Config() api.Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.config
	c.Content.Include = append([]string(nil), c.Content.Include...)
	c.Content.Exclude = append([]string(nil), c.Content.Exclude...)
	return c
}
func (s *Store) Engine() api.Codec { return codec{} }

// SetAsOf pins temporal diagnostics; it never supplies generated timestamps.
func (s *Store) SetAsOf(at time.Time) { s.mu.Lock(); defer s.mu.Unlock(); s.asOf = at; s.cached = nil }
func (s *Store) now() time.Time {
	if s.asOf.IsZero() {
		return time.Now()
	}
	return s.asOf
}

func validPath(p string) error {
	if p == "" || strings.ContainsAny(p, "\\\x00") || strings.HasPrefix(p, "/") || !fs.ValidPath(p) || p == "." {
		return fault("INVALID_PATH", "file path must be a clean root-relative path")
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." || part == "." || strings.Contains(part, ":") {
			return fault("INVALID_PATH", "unsafe path component")
		}
	}
	return nil
}
func canonicalRoot(root string, create bool) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	// Canonicalize existing ancestors, while refusing a symlink at the requested root.
	// The selected root is intentionally caller-controlled; descendants use os.Root.
	//nolint:gosec // This establishes the explicit filesystem boundary, rather than accessing a note.
	if info, err := os.Lstat(abs); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fault("INVALID_PATH", "base root must not be a symlink")
		}
		if !info.IsDir() {
			return "", fault("INVALID_PATH", "base root must be a directory")
		}
		return filepath.EvalSymlinks(abs)
	} else if !errors.Is(err, os.ErrNotExist) || !create {
		return "", err
	}
	parent, err := canonicalRoot(filepath.Dir(abs), true)
	if err != nil {
		return "", err
	}
	name := filepath.Join(parent, filepath.Base(abs))
	//nolint:gosec // Only creates the requested base beneath its canonicalized parent.
	if err := os.Mkdir(name, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	return name, nil
}

// ResolveRoot selects an explicit root, KB_ROOT, then an ancestor config.
func ResolveRoot(explicit string) (string, error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		return abs, err
	}
	if root := os.Getenv("KB_ROOT"); root != "" {
		return filepath.Abs(root)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for p := cwd; ; p = filepath.Dir(p) {
		if info, err := os.Lstat(filepath.Join(p, ".kb/config.toml")); err == nil && !info.IsDir() {
			return canonicalRoot(p, false)
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return "", fault("NOT_INITIALIZED", "no ancestor knowledge-base config found")
}
func defaults() api.Config {
	var c api.Config
	c.Version = 1
	c.Content.Include = []string{"**"}
	c.Content.Exclude = []string{}
	c.PARA.Inbox = "00_Inbox"
	c.PARA.Projects = "01_Projects"
	c.PARA.Areas = "02_Areas"
	c.PARA.Resources = "03_Resources"
	c.PARA.Archive = "04_Archive"
	c.Write.Enabled = true
	c.Write.MaintainIndexes = true
	c.Write.MaintainLog = true
	c.Search.AccentFolding = true
	return c
}

// Open validates config and operational paths without changing notes.
func Open(root string) (*Store, error) {
	abs, err := canonicalRoot(root, false)
	if err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(abs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	if err := safePath(r, ".kb/config.toml", false); err != nil {
		return nil, err
	}
	raw, err := readLimited(r, ".kb/config.toml", api.MaxNoteBytes)
	if err != nil {
		return nil, err
	}
	c := defaults()
	if err := toml.Unmarshal(raw, &c); err != nil {
		return nil, fault("INVALID_CONFIG", "config is not valid TOML")
	}
	if err := validateConfig(c); err != nil {
		return nil, err
	}
	return &Store{root: abs, config: c}, nil
}
func validateConfig(c api.Config) error {
	if c.Version != 1 {
		return fault("INVALID_CONFIG", "unsupported config version")
	}
	for _, pattern := range append(append([]string(nil), c.Content.Include...), c.Content.Exclude...) {
		if pattern == "" || strings.HasPrefix(pattern, "/") || strings.Contains(pattern, "\\") {
			return fault("INVALID_CONFIG", "scope patterns must be root-relative")
		}
		for _, part := range strings.Split(pattern, "/") {
			if part == ".." {
				return fault("INVALID_CONFIG", "scope pattern traversal")
			}
		}
		if _, err := path.Match(pattern, ""); err != nil {
			return fault("INVALID_CONFIG", "invalid scope pattern")
		}
	}
	seen := map[string]bool{}
	for _, folder := range []string{c.PARA.Inbox, c.PARA.Projects, c.PARA.Areas, c.PARA.Resources, c.PARA.Archive} {
		if err := validPath(folder); err != nil {
			return fault("INVALID_CONFIG", "PARA folder must be relative")
		}
		fold := caseKey(folder)
		if seen[fold] {
			return fault("INVALID_CONFIG", "PARA folders collide")
		}
		seen[fold] = true
	}
	return nil
}

// Init creates only missing local files, and is idempotent.
func Init(ctx context.Context, root string, para bool) (*Store, error) {
	if root == "" {
		root = os.Getenv("KB_ROOT")
		if root == "" {
			var err error
			root, err = os.Getwd()
			if err != nil {
				return nil, err
			}
		}
	}
	abs, err := canonicalRoot(root, true)
	if err != nil {
		return nil, err
	}
	lock, err := filelock.Acquire(ctx, abs, true)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lock.Close() }()
	r, err := os.OpenRoot(abs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	if _, err := r.Lstat(".kb/state/pending.json"); err == nil {
		return nil, fault("RECOVERY_REQUIRED", "pending transaction requires recovery")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	c := defaults()
	raw, err := toml.Marshal(c)
	if err != nil {
		return nil, err
	}
	if err := createMissing(r, ".kb/config.toml", raw); err != nil {
		return nil, err
	}
	folders := []string{""}
	if para {
		folders = append(folders, c.PARA.Inbox, c.PARA.Projects, c.PARA.Areas, c.PARA.Resources, c.PARA.Archive)
	}
	for _, folder := range folders {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if folder != "" {
			if err := safePath(r, folder, true); err != nil {
				return nil, err
			}
			if err := r.MkdirAll(folder, 0700); err != nil {
				return nil, err
			}
		}
		name := path.Join(folder, "index.md")
		heading := "Knowledge index"
		if folder != "" {
			heading = folder
		}
		text := "# " + heading + "\n\n" + indexStart + "\n" + indexEnd + "\n"
		if folder == "" {
			text = "---\nokf_version: \"0.2\"\n---\n" + text
		}
		if err := createMissing(r, name, []byte(text)); err != nil {
			return nil, err
		}
	}
	if err := createMissing(r, "log.md", []byte("# Knowledge update log\n")); err != nil {
		return nil, err
	}
	return Open(abs)
}
func createMissing(r *os.Root, p string, raw []byte) error {
	if err := safePath(r, p, true); err != nil {
		return err
	}
	f, err := r.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0600)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, writeErr := f.Write(raw)
	return errorsJoin(writeErr, f.Close())
}
func caseKey(p string) string { return cases.Fold().String(norm.NFC.String(p)) }
func safePath(r *os.Root, p string, missing bool) error {
	if err := validPath(p); err != nil {
		return err
	}
	parts := strings.Split(p, "/")
	parent := "."
	for i, part := range parts {
		entries, err := fs.ReadDir(r.FS(), parent)
		if err != nil {
			if missing && errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		for _, entry := range entries {
			if entry.Name() != part && caseKey(entry.Name()) == caseKey(part) {
				return fault("PATH_COLLISION", "case-folded path collision")
			}
		}
		name := strings.Join(parts[:i+1], "/")
		info, err := r.Lstat(name)
		if err != nil {
			if missing && errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fault("INVALID_PATH", "symlinks are not allowed")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fault("INVALID_PATH", "path component is not a directory")
		}
		if i == len(parts)-1 && !info.Mode().IsRegular() && !info.IsDir() {
			return fault("INVALID_PATH", "special files are not allowed")
		}
		parent = name
	}
	return nil
}
func readLimited(r *os.Root, p string, limit int) ([]byte, error) {
	f, err := r.OpenFile(p, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	raw, readErr := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	err = errorsJoin(readErr, f.Close())
	if err != nil {
		return nil, err
	}
	if len(raw) > limit {
		return nil, fault("LIMIT_EXCEEDED", "file exceeds size ceiling")
	}
	return raw, nil
}
func globMatch(pattern, p string) bool {
	if pattern == "**" {
		return true
	}
	parts, values := strings.Split(pattern, "/"), strings.Split(p, "/")
	var match func(int, int) bool
	match = func(i, j int) bool {
		if i == len(parts) {
			return j == len(values)
		}
		if parts[i] == "**" {
			if match(i+1, j) {
				return true
			}
			return j < len(values) && match(i, j+1)
		}
		if j == len(values) {
			return false
		}
		ok, _ := path.Match(parts[i], values[j])
		return ok && match(i+1, j+1)
	}
	return match(0, 0)
}
func (s *Store) included(p string) bool {
	for _, part := range strings.Split(p, "/") {
		if alwaysExcluded[part] {
			return false
		}
	}
	for _, pattern := range s.config.Content.Exclude {
		if globMatch(pattern, p) {
			return false
		}
	}
	if p == "index.md" || p == "log.md" {
		return true
	}
	for _, pattern := range s.config.Content.Include {
		if globMatch(pattern, p) {
			return true
		}
	}
	return false
}
func (s *Store) locked(ctx context.Context, fn func(*os.Root) error) error {
	lock, err := filelock.Acquire(ctx, s.root, false)
	if err != nil {
		return err
	}
	defer func() { _ = lock.Close() }()
	r, err := os.OpenRoot(s.root)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	if err := safePath(r, ".kb/state/pending.json", true); err != nil {
		return err
	}
	if _, err := r.Lstat(".kb/state/pending.json"); err == nil {
		return fault("RECOVERY_REQUIRED", "pending transaction requires recovery")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return fn(r)
}
func (s *Store) Snapshot(ctx context.Context) (api.Snapshot, error) {
	var out api.Snapshot
	err := s.locked(ctx, func(r *os.Root) error { var err error; out, err = s.snapshot(ctx, r); return err })
	return out, err
}

// SnapshotUnlocked is reserved for callers holding the exclusive writer lock.
func (s *Store) SnapshotUnlocked(ctx context.Context) (api.Snapshot, error) {
	r, err := os.OpenRoot(s.root)
	if err != nil {
		return api.Snapshot{}, err
	}
	defer func() { _ = r.Close() }()
	return s.snapshot(ctx, r)
}
func (s *Store) snapshot(ctx context.Context, r *os.Root) (api.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files := map[string][]byte{}
	var ds []api.Diagnostic
	seen := map[string]string{}
	err := fs.WalkDir(r.FS(), ".", func(p string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if p == "." {
			return nil
		}
		if !s.included(p) && entry.IsDir() {
			for _, part := range strings.Split(p, "/") {
				if alwaysExcluded[part] {
					return fs.SkipDir
				}
			}
			for _, pattern := range s.config.Content.Exclude {
				if globMatch(pattern, p) || globMatch(pattern, p+"/") {
					return fs.SkipDir
				}
			}
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if s.included(p) {
				ds = append(ds, diagnostic(p, "INVALID_PATH", "symlink excluded from scope", "conformance_errors"))
			}
			return nil
		}
		key := caseKey(p)
		if old, ok := seen[key]; ok && old != p {
			ds = append(ds, diagnostic(p, "PATH_COLLISION", "case-folded path collision", "conformance_errors"))
		}
		seen[key] = p
		if entry.IsDir() || !strings.HasSuffix(p, ".md") || !s.included(p) {
			return nil
		}
		if !entry.Type().IsRegular() {
			ds = append(ds, diagnostic(p, "INVALID_PATH", "special file excluded", "conformance_errors"))
			return nil
		}
		raw, err := readLimited(r, p, api.MaxNoteBytes)
		if err != nil {
			var domain *api.DomainError
			if errors.As(err, &domain) && domain.Code == "LIMIT_EXCEEDED" {
				ds = append(ds, diagnostic(p, domain.Code, domain.Message, "conformance_errors"))
				return nil
			}
			return err
		}
		files[p] = raw
		return nil
	})
	if err != nil {
		return api.Snapshot{}, err
	}
	generation := api.Generation(files)
	if s.cached != nil && s.cached.Generation == generation && reflectDiagnostics(ds, s.cached.Diagnostics) {
		return cloneSnapshot(*s.cached), nil
	}
	out := api.Snapshot{Generation: generation, Files: files, Diagnostics: ds, Notes: []api.Note{}, Links: []api.Link{}}
	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		note, nds := (codec{}).Parse(p, files[p])
		out.Diagnostics = append(out.Diagnostics, nds...)
		if !hasErrors(nds) && !reserved(p) {
			note.PARA = s.category(p)
			if stamp, ok := note.Metadata["stale_after"].(string); ok {
				at, err := time.Parse(time.RFC3339Nano, stamp)
				note.Stale = err == nil && !s.now().Before(at)
			}
			out.Notes = append(out.Notes, note)
		}
	}
	out.Links = buildLinks(files, out.Notes)
	for i := range out.Notes {
		out.Notes[i].BasisOutdated = basisChanged(out.Notes[i], files)
	}
	out.Degraded = hasErrors(out.Diagnostics)
	s.cached = &out
	return cloneSnapshot(out), nil
}
func reflectDiagnostics(scan, all []api.Diagnostic) bool {
	var only []api.Diagnostic
	for _, d := range all {
		if d.Code == "INVALID_PATH" || d.Code == "PATH_COLLISION" || d.Code == "LIMIT_EXCEEDED" {
			only = append(only, d)
		}
	}
	return sameValue(scan, only)
}
func cloneSnapshot(in api.Snapshot) api.Snapshot {
	out := in
	out.Files = make(map[string][]byte, len(in.Files))
	for p, b := range in.Files {
		out.Files[p] = bytesClone(b)
	}
	out.Notes = append([]api.Note(nil), in.Notes...)
	for i := range out.Notes {
		out.Notes[i].Raw = out.Files[out.Notes[i].Path]
		out.Notes[i].Tags = append([]string(nil), out.Notes[i].Tags...)
		out.Notes[i].Metadata = cloneMap(out.Notes[i].Metadata)
	}
	out.Diagnostics = append([]api.Diagnostic(nil), in.Diagnostics...)
	out.Links = append([]api.Link(nil), in.Links...)
	return out
}
func bytesClone(b []byte) []byte { return append([]byte(nil), b...) }
func cloneMap(m map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = cloneValue(v)
	}
	return out
}
func cloneValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return cloneMap(x)
	case []any:
		a := make([]any, len(x))
		for i, v := range x {
			a[i] = cloneValue(v)
		}
		return a
	default:
		return v
	}
}
func (s *Store) category(p string) string {
	for _, v := range []struct{ folder, name string }{{s.config.PARA.Inbox, "inbox"}, {s.config.PARA.Projects, "projects"}, {s.config.PARA.Areas, "areas"}, {s.config.PARA.Resources, "resources"}, {s.config.PARA.Archive, "archive"}} {
		if p == v.folder || strings.HasPrefix(p, v.folder+"/") {
			return v.name
		}
	}
	return ""
}
func statusOf(root string, snap api.Snapshot) api.Status {
	invalid := map[string]bool{}
	for _, d := range snap.Diagnostics {
		if d.Layer == "conformance_errors" || d.Layer == "policy_findings" {
			invalid[d.Path] = true
		}
	}
	excluded := []string{"operational directories", "dependency directories", "agent directories", "out-of-scope files", "non-Markdown source artifacts"}
	return api.Status{Root: root, Generation: snap.Generation, Notes: len(snap.Notes), Invalid: len(invalid), Degraded: snap.Degraded, IndexStale: true, Diagnostics: snap.Diagnostics, Excluded: excluded}
}
func (s *Store) Status(ctx context.Context) (api.Status, error) {
	var out api.Status
	err := s.locked(ctx, func(r *os.Root) error {
		snap, err := s.snapshot(ctx, r)
		if err != nil {
			return err
		}
		out = statusOf(s.root, snap)
		if err := safePath(r, ".kb/cache/generation", true); err != nil {
			return err
		}
		raw, err := r.ReadFile(".kb/cache/generation")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		out.IndexStale = string(raw) != snap.Generation
		return nil
	})
	return out, err
}

var _ api.Provider = (*Store)(nil)
