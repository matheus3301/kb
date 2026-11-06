package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
)

func aggregate(installs map[string]Installation) (map[string]asset, error) {
	desired := map[string]asset{}
	for _, id := range sortedKeys(installs) {
		in := installs[id]
		seen := map[string]bool{}
		for _, a := range in.Artifacts {
			key := assetKey(a)
			if seen[key] {
				return nil, conflict("duplicate artifact intention: " + a.Path)
			}
			seen[key] = true
			if prior, ok := desired[key]; ok {
				if !a.Shared || !prior.Artifact.Shared || !bytes.Equal(a.Content, prior.Artifact.Content) {
					return nil, conflict("incompatible shared artifact: " + a.Path)
				}
				prior.Owners = append(prior.Owners, id)
				desired[key] = prior
			} else {
				desired[key] = asset{Artifact: a, Owners: []string{id}}
			}
		}
	}
	return desired, nil
}

func (e *Engine) build(ctx context.Context, old, next catalog, stateSnapshot Snapshot) (*Plan, error) {
	desired, err := aggregate(next.Installations)
	if err != nil {
		return nil, err
	}
	p := &Plan{Schema: SchemaVersion, Preconditions: map[string]Snapshot{e.statePath(): stateSnapshot}, stateDir: e.stateDir}
	paths := map[string]bool{}
	keys := map[string]bool{}
	for key, a := range old.Assets {
		paths[a.Artifact.Path] = true
		keys[key] = true
	}
	for key, a := range desired {
		paths[a.Artifact.Path] = true
		keys[key] = true
	}
	current := map[string]Snapshot{}
	working := map[string]Snapshot{}
	for _, path := range sortedKeys(paths) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		snap, err := readSnapshot(path)
		if err != nil {
			return nil, pathError(path, err)
		}
		current[path] = snap
		working[path] = snap
		p.Preconditions[path] = snap
		f, exists := next.Files[path]
		if !exists {
			f = fileState{Before: snap, Expected: snap, Pristine: true}
		} else if !snap.equal(f.Expected) {
			f.Pristine = false
		}
		next.Files[path] = f
	}
	// A full file cannot coexist with managed fragments at the same path.
	for _, path := range sortedKeys(paths) {
		count := 0
		whole := false
		for _, a := range desired {
			if a.Artifact.Path == path {
				count++
				whole = whole || a.Artifact.Kind == "file"
			}
		}
		if whole && count > 1 {
			return nil, conflict("whole-file and entry ownership overlap: " + path)
		}
	}
	next.Assets = map[string]asset{}
	for _, key := range sortedKeys(keys) {
		previous, had := old.Assets[key]
		wanted, keep := desired[key]
		a := previous.Artifact
		if keep {
			a = wanted.Artifact
		}
		var prior *asset
		if had {
			copy := previous
			prior = &copy
		}
		snap := working[a.Path]
		var raw []byte
		if snap.Exists {
			raw = snap.Content
			if raw == nil {
				raw = []byte{}
			}
		}
		out, managed, borrowed, err := editArtifact(raw, a, prior, !keep)
		if err != nil {
			return nil, pathError(a.Path, err)
		}
		if keep {
			wanted.Managed = append([]byte(nil), managed...)
			wanted.Borrowed = borrowed
			next.Assets[key] = wanted
		}
		exists := snap.Exists || keep
		if a.Kind == "file" && !keep && !previous.Borrowed {
			exists = false
		}
		mode := snap.Mode
		if mode == 0 {
			mode = 0600
		}
		working[a.Path] = Snapshot{Exists: exists, Content: out, Mode: mode}
		if !exists {
			working[a.Path] = Snapshot{}
		}
	}
	for _, path := range sortedKeys(paths) {
		has := false
		for _, a := range next.Assets {
			if a.Artifact.Path == path {
				has = true
				break
			}
		}
		f := next.Files[path]
		after := working[path]
		if !has {
			if f.Pristine {
				after = f.Before
			}
			delete(next.Files, path)
		} else {
			f.Expected = after
			next.Files[path] = f
		}
		if !current[path].equal(after) {
			p.Changes = append(p.Changes, Change{Path: path, Before: current[path], After: after})
		}
	}
	// All private bindings live outside repositories. Canonical project paths are
	// exact registry keys, so copying a worktree never copies personal activation.
	bindings := scopeManifest{Schema: SchemaVersion, Bindings: map[string]Binding{}}
	for _, id := range sortedKeys(next.Installations) {
		in := next.Installations[id]
		bindings.Bindings[id] = in.Binding
	}
	bindingPath := filepath.Join(e.options.ConfigHome, "kb", "agent-setup.json")
	before, err := readSnapshot(bindingPath)
	if err != nil {
		return nil, err
	}
	p.Preconditions[bindingPath] = before
	if before.Exists {
		var parsed scopeManifest
		if err := decodeManifest(before.Content, &parsed); err != nil {
			return nil, err
		}
		if parsed.Schema != SchemaVersion {
			return nil, conflict("unsupported binding registry version")
		}
	}
	if len(bindings.Bindings) > 0 || before.Exists {
		raw, err := encode(bindings)
		if err != nil {
			return nil, err
		}
		after := Snapshot{Exists: true, Content: raw, Mode: 0600}
		if len(bindings.Bindings) == 0 {
			after = Snapshot{}
		}
		if !before.equal(after) {
			p.Changes = append(p.Changes, Change{Path: bindingPath, Before: before, After: after})
		}
	}
	// The private ownership manifest is the final publication, after external
	// configurations and routing. No-op plans leave its bytes and mtime unchanged.
	stateRaw, err := encode(next)
	if err != nil {
		return nil, err
	}
	stateAfter := Snapshot{Exists: true, Content: stateRaw, Mode: 0600}
	if len(next.Installations) == 0 && len(next.Assets) == 0 {
		stateAfter = Snapshot{}
	}
	if !stateSnapshot.equal(stateAfter) {
		p.Changes = append(p.Changes, Change{Path: e.statePath(), Before: stateSnapshot, After: stateAfter})
	}
	// Keep state last; unregistration entries precede helper removal.
	sort.SliceStable(p.Changes, func(i, j int) bool {
		rank := func(c Change) int {
			if c.Path == e.statePath() {
				return 3
			}
			if c.Path == bindingPath {
				return 2
			}
			if filepath.Ext(c.Path) == ".json" || filepath.Ext(c.Path) == ".toml" || filepath.Ext(c.Path) == ".jsonc" {
				return 0
			}
			return 1
		}
		return rank(p.Changes[i]) < rank(p.Changes[j])
	})
	for _, change := range p.Changes {
		if err := preflightWritable(change); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func decodeManifest(raw []byte, dst any) error {
	if _, err := parseJSON(raw); err != nil {
		return conflict("invalid setup metadata")
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return conflict(fmt.Sprintf("invalid setup metadata: %v", err))
	}
	return nil
}
