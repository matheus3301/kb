package setup

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/matheus3301/kb/internal/api"
)

func preflightWritable(change Change) error {
	if change.Before.Exists && os.FileMode(change.Before.Mode).Perm()&0222 == 0 {
		return &api.DomainError{Code: "ACCESS_DENIED", Message: "setup target is read-only: " + change.Path}
	}
	for path := filepath.Dir(change.Path); ; path = filepath.Dir(path) {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			if filepath.Dir(path) == path {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode().Perm()&0222 == 0 {
			return &api.DomainError{Code: "ACCESS_DENIED", Message: "setup destination directory is not writable: " + path}
		}
		return nil
	}
}
