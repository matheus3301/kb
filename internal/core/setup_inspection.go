package core

import (
	"context"
	"errors"
	"os"

	"github.com/matheus3301/kb/internal/api"
)

// InspectSetup validates the configured collection without creating operational
// directories, a lock file, or a cache. This is an observational preflight, not
// a transaction snapshot or a reservation against concurrent edits. Mutation
// callers must continue using the normal locked APIs. Pending recovery before
// or after inspection prevents this result from claiming a settled collection.
func (s *Store) InspectSetup(ctx context.Context) (api.ValidationReport, error) {
	if err := ctx.Err(); err != nil {
		return api.ValidationReport{}, err
	}
	r, err := os.OpenRoot(s.root)
	if err != nil {
		return api.ValidationReport{}, err
	}
	defer func() { _ = r.Close() }()
	if err := setupPending(r); err != nil {
		return api.ValidationReport{}, err
	}
	snap, err := s.snapshot(ctx, r)
	if err != nil {
		return api.ValidationReport{}, err
	}
	if err := setupPending(r); err != nil {
		return api.ValidationReport{}, err
	}
	return validationReport("okf", snap.Generation, snap.Diagnostics, snap.Files), nil
}

func setupPending(r *os.Root) error {
	const pending = ".kb/state/pending.json"
	if err := safePath(r, pending, true); err != nil {
		return err
	}
	if _, err := r.Lstat(pending); err == nil {
		return fault("RECOVERY_REQUIRED", "pending transaction requires recovery")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
