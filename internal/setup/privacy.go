package setup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// PrivacyArtifacts discloses and manages local Git exclusions for generated
// personal configuration. Already tracked files cannot be made private by an
// exclude rule, so they are rejected instead of giving false protection.
func PrivacyArtifacts(ctx context.Context, b Binding, artifacts []Artifact, home string) ([]Artifact, []string, error) {
	if b.Scope != "project" || b.Portable {
		return nil, nil, nil
	}
	var paths []string
	for _, a := range artifacts {
		if !within(b.Project, a.Path) {
			continue
		}
		if !bytes.Contains(a.Content, []byte(b.Root)) && !bytes.Contains(a.Content, []byte(home)) && !bytes.Contains(a.Content, []byte(b.Project)) {
			continue
		}
		rel, err := filepath.Rel(b.Project, a.Path)
		if err != nil {
			return nil, nil, err
		}
		paths = append(paths, filepath.ToSlash(rel))
	}
	if len(paths) == 0 {
		return nil, nil, nil
	}
	sort.Strings(paths)
	paths = compactStrings(paths)
	hasGit, err := gitAncestor(b.Project)
	if err != nil {
		return nil, nil, err
	}
	exclude := filepath.Join(b.Project, ".gitignore")
	var notes []string
	patternRoot := b.Project
	if hasGit {
		for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE"} {
			if os.Getenv(key) != "" {
				return nil, nil, invalid("unset " + key + " before setup privacy preflight")
			}
		}
		git, err := exec.LookPath("git")
		if err != nil {
			return nil, nil, invalid("Git is required to inspect exclusions for this repository")
		}
		command := exec.CommandContext(ctx, git, "-C", b.Project, "rev-parse", "--path-format=absolute", "--git-path", "info/exclude") // #nosec G204 -- fixed executable lookup and argument vector, never a shell.
		raw, err := command.Output()
		if err != nil {
			return nil, nil, conflict("Git could not inspect repository exclusions; fix repository access before setup")
		}
		exclude = strings.TrimSpace(string(raw))
		if !filepath.IsAbs(exclude) {
			return nil, nil, conflict("Git returned a nonabsolute exclude path")
		}
		top := exec.CommandContext(ctx, git, "-C", b.Project, "rev-parse", "--show-toplevel") // #nosec G204 -- PATH-resolved Git with fixed read-only argv; project is explicit and no shell runs.
		topRaw, err := top.Output()
		if err != nil {
			return nil, nil, err
		}
		patternRoot, err = Canonical(strings.TrimSpace(string(topRaw)))
		if err != nil {
			return nil, nil, err
		}
		for _, path := range paths {
			cmd := exec.CommandContext(ctx, git, "-C", b.Project, "ls-files", "--error-unmatch", "--", path) // #nosec G204 -- Same checked Git executable, literal operation and separate canonical project/path arguments.
			if err := cmd.Run(); err == nil {
				return nil, nil, conflict("personal config is already tracked: " + path + "; use a private profile/global scope or remove it from Git explicitly before setup")
			} else {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 {
					return nil, nil, err
				}
			}
		}
	} else {
		notes = append(notes, "project is not a Git repository; adding explicit private-file patterns to .gitignore")
	}
	var lines []string
	for _, path := range paths {
		if patternRoot != b.Project {
			var err error
			path, err = filepath.Rel(patternRoot, filepath.Join(b.Project, filepath.FromSlash(path)))
			if err != nil {
				return nil, nil, err
			}
			path = filepath.ToSlash(path)
		}
		if strings.ContainsAny(path, "\r\n") {
			return nil, nil, invalid("private artifact path contains a newline")
		}
		escaped := strings.NewReplacer("\\", "\\\\", " ", "\\ ", "#", "\\#", "!", "\\!", "[", "\\[", "*", "\\*", "?", "\\?").Replace(path)
		lines = append(lines, "/"+escaped)
	}
	content := []byte(strings.Join(lines, "\n"))
	notes = append(notes, "local private-file exclusion bookkeeping: "+exclude)
	return []Artifact{{Path: exclude, Kind: "gitignore", Key: "kb-private-" + identity(b), Content: content}}, notes, nil
}

func gitAncestor(project string) (bool, error) {
	for dir := project; ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		if filepath.Dir(dir) == dir {
			return false, nil
		}
	}
}
func compactStrings(values []string) []string {
	out := values[:0]
	for _, s := range values {
		if len(out) == 0 || out[len(out)-1] != s {
			out = append(out, s)
		}
	}
	return out
}
