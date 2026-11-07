package cli

import (
	"path/filepath"
	"strings"

	"github.com/matheus3301/kb/internal/mcpserver"
)

// Expand only the selected user's home prefix; never evaluate shell syntax or
// another account's home. This also applies to pasted wizard create: answers.
func expandSetupPath(path, home string) (string, error) {
	if path == "~" {
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	if strings.HasPrefix(path, "~") {
		return "", mcpserver.Invalid("unsupported home alias; use ~/PATH or an absolute path")
	}
	return path, nil
}
