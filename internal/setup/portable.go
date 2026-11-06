package setup

import (
	"bytes"
	"path/filepath"
)

// PortableArtifacts returns an explicit shareable relative binding. It contains
// neither local registry identities nor absolute home/root/executable paths.
func PortableArtifacts(b Binding, home string) ([]Artifact, error) {
	if !b.Portable {
		return nil, nil
	}
	if b.Scope != "project" {
		return nil, invalid("portable bindings require project scope")
	}
	root, err := filepath.Rel(b.Project, b.Root)
	if err != nil {
		return nil, err
	}
	public := b
	public.Root = filepath.ToSlash(root)
	public.Project = ""
	if b.SkillDir != "" {
		public.SkillDir, err = filepath.Rel(b.Project, b.SkillDir)
		if err != nil {
			return nil, err
		}
		public.SkillDir = filepath.ToSlash(public.SkillDir)
	}
	raw, err := encode(public)
	if err != nil {
		return nil, err
	}
	if bytes.Contains(raw, []byte(home)) || bytes.Contains(raw, []byte(b.Root)) {
		return nil, invalid("portable binding contains a personal absolute path")
	}
	path := filepath.Join(b.Project, ".kb-agent", "setup.json")
	return []Artifact{{Path: path, Kind: "json", Key: "schema_version", Content: []byte("1"), Shared: true}, {Path: path, Kind: "json", Key: "bindings." + b.Agent, Content: raw, Shared: true}}, nil
}
