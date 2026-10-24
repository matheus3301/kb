package core

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/matheus3301/kb/internal/api"
)

func health(snap api.Snapshot) []api.Diagnostic {
	var ds []api.Diagnostic
	incoming := map[string]bool{}
	indexed := map[string]bool{}
	for _, link := range snap.Links {
		if link.External {
			continue
		}
		if !link.Resolved {
			ds = append(ds, diagnostic(link.From, "BROKEN_LINK", "unresolved reference: "+link.To, "advisories"))
		} else {
			incoming[link.To] = true
			if path.Base(link.From) == "index.md" {
				indexed[link.To] = true
			}
		}
	}
	for _, note := range snap.Notes {
		if !incoming[note.Path] {
			ds = append(ds, diagnostic(note.Path, "ORPHAN", "no incoming scoped reference", "advisories"))
		}
		if !indexed[note.Path] {
			ds = append(ds, diagnostic(note.Path, "UNINDEXED", "concept is absent from scoped indexes", "advisories"))
		}
		if note.BasisOutdated {
			ds = append(ds, diagnostic(note.Path, "BASIS_CHANGED", "a recorded dependency revision changed or is unavailable; review required", "advisories"))
		}
		if note.Stale {
			ds = append(ds, diagnostic(note.Path, "STALE", "stale_after instant has been reached", "advisories"))
		}
	}
	return ds
}

// Validate keeps format/policy findings separate from static health advisories.
func (s *Store) Validate(ctx context.Context, profile string) (api.ValidationReport, error) {
	if profile == "" {
		profile = "okf"
	}
	if profile != "okf" && profile != "health" {
		return api.ValidationReport{}, fault("INVALID_INPUT", "validation profile must be okf or health")
	}
	snap, err := s.readSnapshot(ctx)
	if err != nil {
		return api.ValidationReport{}, err
	}
	ds := append([]api.Diagnostic(nil), snap.Diagnostics...)
	if profile == "health" {
		ds = append(ds, health(snap)...)
	}
	return api.ValidationReport{Profile: profile, Generation: snap.Generation, Conformant: !hasErrors(snap.Diagnostics), Diagnostics: ds}, nil
}

// Rebuild updates only disposable cache data under the shared base lock.
func (s *Store) Rebuild(ctx context.Context) (api.Status, error) {
	var result api.Status
	err := s.locked(ctx, func(r *os.Root) error {
		snap, err := s.snapshot(ctx, r)
		if err != nil {
			return err
		}
		if err := safePath(r, ".kb/cache", true); err != nil {
			return err
		}
		if err := r.MkdirAll(".kb/cache", 0700); err != nil {
			return err
		}
		if err := safePath(r, ".kb/cache/generation", true); err != nil {
			return err
		}
		// A rebuild is harmless if interrupted; replace the cache marker atomically.
		tmp, err := r.OpenFile(".kb/cache/generation.tmp", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if errors.Is(err, os.ErrExist) {
			return fault("CONFLICT", "another cache rebuild is in progress")
		}
		if err != nil {
			return err
		}
		_, writeErr := tmp.WriteString(snap.Generation)
		if err := errorsJoin(writeErr, tmp.Close()); err != nil {
			return err
		}
		if err := r.Rename(".kb/cache/generation.tmp", ".kb/cache/generation"); err != nil {
			return err
		}
		result = statusOf(s.root, snap)
		result.IndexStale = false
		return nil
	})
	return result, err
}

// Export writes a new validated knowledge bundle and an explicit manifest.
func (s *Store) Export(ctx context.Context, destination string) (api.ExportResult, error) {
	var result api.ExportResult
	err := s.locked(ctx, func(r *os.Root) error {
		snap, err := s.snapshot(ctx, r)
		if err != nil {
			return err
		}
		abs, err := filepath.Abs(destination)
		if err != nil {
			return err
		}
		if abs == s.root || strings.HasPrefix(abs, s.root+string(filepath.Separator)) {
			return fault("INVALID_PATH", "export destination must be outside the base")
		}
		if _, err := os.Lstat(abs); err == nil {
			return fault("ALREADY_EXISTS", "export destination already exists")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil {
			return err
		}
		abs = filepath.Join(parent, filepath.Base(abs))
		if abs == s.root || strings.HasPrefix(abs, s.root+string(filepath.Separator)) {
			return fault("INVALID_PATH", "export destination resolves inside the base")
		}
		parentRoot, err := os.OpenRoot(parent)
		if err != nil {
			return err
		}
		defer func() { _ = parentRoot.Close() }()
		if err := parentRoot.Mkdir(filepath.Base(abs), 0700); err != nil {
			return err
		}
		dest, err := parentRoot.OpenRoot(filepath.Base(abs))
		if err != nil {
			return err
		}
		defer func() { _ = dest.Close() }()
		result = api.ExportResult{Destination: abs, Generation: snap.Generation, Conformant: !hasErrors(snap.Diagnostics), Files: []string{}, Excluded: []string{}}
		invalid := map[string]bool{}
		for _, d := range snap.Diagnostics {
			if d.Layer == "conformance_errors" || d.Layer == "policy_findings" {
				invalid[d.Path] = true
			}
		}
		paths := make([]string, 0, len(snap.Files))
		for p := range snap.Files {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		for _, p := range paths {
			if err := ctx.Err(); err != nil {
				return err
			}
			if invalid[p] {
				result.Excluded = append(result.Excluded, p)
				continue
			}
			if err := dest.MkdirAll(path.Dir(p), 0700); err != nil {
				return err
			}
			if err := createMissing(dest, p, snap.Files[p]); err != nil {
				return err
			}
			result.Files = append(result.Files, p)
		}
		err = fs.WalkDir(r.FS(), ".", func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if p == "." {
				return nil
			}
			if d.IsDir() {
				if alwaysExcluded[d.Name()] {
					result.Excluded = append(result.Excluded, p+"/")
					return fs.SkipDir
				}
				return nil
			}
			if _, ok := snap.Files[p]; !ok {
				result.Excluded = append(result.Excluded, p)
			}
			return nil
		})
		if err != nil {
			return err
		}
		sort.Strings(result.Excluded)
		manifest := struct {
			api.ExportResult
			Report api.ValidationReport
		}{result, api.ValidationReport{Profile: "okf", Generation: snap.Generation, Conformant: result.Conformant, Diagnostics: snap.Diagnostics}}
		raw, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return err
		}
		return createMissing(dest, "manifest.json", append(raw, '\n'))
	})
	return result, err
}
