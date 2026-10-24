// Package write applies bounded plans using a durable, recoverable file journal.
package write

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/matheus3301/kb/internal/api"
	"github.com/matheus3301/kb/internal/filelock"
)

type Service struct {
	provider api.Provider
	now      func() time.Time
	// fault is a test seam at durable publication boundaries, including recovery.
	fault func(stage, name string) error
}

func New(provider api.Provider) *Service { return &Service{provider: provider, now: time.Now} }

func (s *Service) boundary(stage, name string) error {
	if s.fault != nil {
		return s.fault(stage, name)
	}
	return nil
}

func (s *Service) locked(ctx context.Context, fn func(*os.Root) (api.Receipt, error)) (receipt api.Receipt, err error) {
	if !s.provider.Config().Write.Enabled {
		return receipt, problem("READ_ONLY", "writes are disabled by configuration")
	}
	root, err := os.OpenRoot(s.provider.Root())
	if err != nil {
		return receipt, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	if err := inspect(root, ".kb/state/lock"); err != nil {
		return receipt, err
	}
	lock, err := filelock.Acquire(ctx, s.provider.Root(), true)
	if err != nil {
		return receipt, err
	}
	defer func() { err = errors.Join(err, lock.Close()) }()
	if err := ctx.Err(); err != nil {
		return receipt, err
	}
	return fn(root)
}

type mutation struct {
	Path   string `json:"path"`
	Before image  `json:"before"`
	After  image  `json:"after"`
}

type journal struct {
	Version int         `json:"version"`
	State   string      `json:"state"`
	Key     string      `json:"key"`
	Digest  string      `json:"digest"`
	Files   []mutation  `json:"files"`
	Receipt api.Receipt `json:"receipt"`
}

type savedReceipt struct {
	Digest  string      `json:"digest"`
	Receipt api.Receipt `json:"receipt"`
}

func receiptPath(key string) string {
	sum := sha256.Sum256([]byte(key))
	return ".kb/state/receipts/" + hex.EncodeToString(sum[:]) + ".json"
}

func decode(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return problem("INVALID_STATE", "expected one JSON document")
	}
	return nil
}

func putJSON(root *os.Root, name string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > maxJournalBytes {
		return problem("INPUT_TOO_LARGE", "transaction state exceeds its byte limit")
	}
	return atomicImage(root, name, image{Exists: true, Data: data, Mode: 0600})
}

func digestPlan(plan api.Plan) (string, error) {
	plan.DryRun = false
	data, err := json.Marshal(plan)
	if err != nil {
		return "", problem("INVALID_PLAN", "plan metadata must be JSON encodable")
	}
	if len(data) > api.MaxPlanBytes {
		return "", problem("INPUT_TOO_LARGE", "plan exceeds 4 MiB")
	}
	return api.Revision(data), nil
}

func (s *Service) Apply(ctx context.Context, plan api.Plan) (api.Receipt, error) {
	if len(plan.Operations) < 1 || len(plan.Operations) > api.MaxOperations || strings.TrimSpace(plan.IdempotencyKey) == "" || strings.TrimSpace(plan.Actor) == "" || strings.TrimSpace(plan.Summary) == "" {
		return api.Receipt{}, problem("INVALID_PLAN", "one to fifty operations, actor, summary and idempotency_key are required")
	}
	if len(plan.Operations) == 1 && plan.Operations[0].OperationID == "" {
		plan.Operations = append([]api.Operation(nil), plan.Operations...)
		plan.Operations[0].OperationID = "op-1"
	}
	digest, err := digestPlan(plan)
	if err != nil {
		return api.Receipt{}, err
	}
	return s.locked(ctx, func(root *os.Root) (api.Receipt, error) {
		if !plan.DryRun {
			if _, err := s.recover(ctx, root, false); err != nil {
				return api.Receipt{}, err
			}
		} else if value, err := readImage(root, pendingPath, maxJournalBytes); err != nil {
			return api.Receipt{}, err
		} else if value.Exists {
			return api.Receipt{}, problem("RECOVERY_REQUIRED", "pending transaction requires recovery")
		}
		stored, err := readImage(root, receiptPath(plan.IdempotencyKey), maxJournalBytes)
		if err != nil {
			return api.Receipt{}, err
		}
		if stored.Exists {
			var saved savedReceipt
			if err := decode(stored.Data, &saved); err != nil {
				return api.Receipt{}, problem("INVALID_STATE", "invalid durable receipt")
			}
			if saved.Digest != digest || saved.Receipt.IdempotencyKey != plan.IdempotencyKey {
				return api.Receipt{}, problem("IDEMPOTENCY_CONFLICT", "idempotency key belongs to a different request")
			}
			saved.Receipt.Replayed = true
			return saved.Receipt, nil
		}
		snapshot, err := s.provider.SnapshotUnlocked(ctx)
		if err != nil {
			return api.Receipt{}, err
		}
		before, after, warnings, err := s.prepare(root, snapshot, plan)
		if err != nil {
			return api.Receipt{}, err
		}
		return s.publish(ctx, root, before, after, plan.IdempotencyKey, digest, plan.DryRun, warnings)
	})
}

func (s *Service) publish(ctx context.Context, root *os.Root, before, after map[string][]byte, key, digest string, dryRun bool, warnings []api.Diagnostic) (api.Receipt, error) {
	receipt := api.Receipt{IdempotencyKey: key, Generation: api.Generation(after), DryRun: dryRun, Changes: []api.Change{}, Warnings: warnings}
	paths := map[string]bool{}
	for name := range before {
		paths[name] = true
	}
	for name := range after {
		paths[name] = true
	}
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	sort.Strings(names)
	j := journal{Version: 1, State: "prepared", Key: key, Digest: digest}
	diffBudget := api.MaxReadBytes
	for _, name := range names {
		old, oldExists := before[name]
		fresh, freshExists := after[name]
		if oldExists == freshExists && bytes.Equal(old, fresh) {
			continue
		}
		if err := inspect(root, name); err != nil {
			return api.Receipt{}, err
		}
		current, err := readImage(root, name, api.MaxNoteBytes)
		if err != nil {
			return api.Receipt{}, err
		}
		if !sameImage(current, image{Exists: oldExists, Data: old}) {
			return api.Receipt{}, problem("REVISION_CONFLICT", "a file changed during planning")
		}
		mode := current.Mode
		if mode == 0 {
			mode = 0600
		}
		j.Files = append(j.Files, mutation{Path: name, Before: current, After: image{Exists: freshExists, Data: fresh, Mode: mode}})
		change := api.Change{Path: name, Kind: "patch"}
		if oldExists {
			change.BeforeRevision = api.Revision(old)
		} else {
			change.Kind = "create"
		}
		if freshExists {
			change.AfterRevision = api.Revision(fresh)
		} else {
			change.Kind = "delete"
		}
		diff := fmt.Sprintf("--- %s\n+++ %s\n-%s\n+%s\n", name, name, old, fresh)
		if len(diff) > diffBudget {
			diff = diff[:diffBudget]
			for !utf8.ValidString(diff) && len(diff) > 0 {
				diff = diff[:len(diff)-1]
			}
		}
		change.Diff = diff
		diffBudget -= len(diff)
		receipt.Changes = append(receipt.Changes, change)
	}
	if dryRun {
		return receipt, nil
	}
	j.Receipt = receipt
	if err := ctx.Err(); err != nil {
		return api.Receipt{}, err
	}
	// Recheck the complete affected set immediately before preparing the journal.
	if err := checkImages(root, j.Files, false); err != nil {
		return api.Receipt{}, err
	}
	if err := s.checkInventory(ctx, before); err != nil {
		return api.Receipt{}, err
	}
	if err := putJSON(root, pendingPath, j); err != nil {
		return api.Receipt{}, err
	}
	if err := s.boundary("prepared", pendingPath); err != nil {
		return api.Receipt{}, err
	}
	for _, file := range j.Files {
		if err := ctx.Err(); err != nil {
			return api.Receipt{}, err
		}
		current, err := readImage(root, file.Path, api.MaxNoteBytes)
		if err != nil {
			return api.Receipt{}, err
		}
		if !sameImage(current, file.Before) {
			return api.Receipt{}, problem("REVISION_CONFLICT", "a file changed at publication")
		}
		if err := replaceImage(root, file.Path, file.After, &file.Before); err != nil {
			return api.Receipt{}, err
		}
		if err := s.boundary("published", file.Path); err != nil {
			return api.Receipt{}, err
		}
	}
	if err := checkImages(root, j.Files, true); err != nil {
		return api.Receipt{}, err
	}
	if err := s.checkInventory(ctx, after); err != nil {
		return api.Receipt{}, err
	}
	j.State = "committed"
	if err := putJSON(root, pendingPath, j); err != nil {
		return api.Receipt{}, err
	}
	if err := s.boundary("committed", pendingPath); err != nil {
		return api.Receipt{}, err
	}
	if err := s.finish(root, j); err != nil {
		return api.Receipt{}, err
	}
	return receipt, nil
}

func (s *Service) checkInventory(ctx context.Context, expected map[string][]byte) error {
	current, err := s.provider.SnapshotUnlocked(ctx)
	if err != nil {
		return err
	}
	if api.Generation(current.Files) != api.Generation(expected) {
		return problem("REVISION_CONFLICT", "base inventory changed during the transaction")
	}
	return nil
}

func checkImages(root *os.Root, files []mutation, after bool) error {
	for _, file := range files {
		current, err := readImage(root, file.Path, api.MaxNoteBytes)
		if err != nil {
			return err
		}
		expected := file.Before
		if after {
			expected = file.After
		}
		if !sameImage(current, expected) {
			return problem("REVISION_CONFLICT", "a transaction file changed externally")
		}
	}
	return nil
}

func (s *Service) finish(root *os.Root, j journal) error {
	if err := putJSON(root, receiptPath(j.Key), savedReceipt{Digest: j.Digest, Receipt: j.Receipt}); err != nil {
		return err
	}
	if err := s.boundary("receipt", receiptPath(j.Key)); err != nil {
		return err
	}
	if err := atomicImage(root, pendingPath, image{}); err != nil {
		return err
	}
	return s.boundary("removed", pendingPath)
}

func (s *Service) Recover(ctx context.Context, dryRun bool) (api.Receipt, error) {
	return s.locked(ctx, func(root *os.Root) (api.Receipt, error) { return s.recover(ctx, root, dryRun) })
}

func (s *Service) recover(ctx context.Context, root *os.Root, dryRun bool) (api.Receipt, error) {
	value, err := readImage(root, pendingPath, maxJournalBytes)
	if err != nil {
		return api.Receipt{}, err
	}
	if !value.Exists {
		return api.Receipt{DryRun: dryRun, Changes: []api.Change{}, Warnings: []api.Diagnostic{}}, nil
	}
	var j journal
	if err := decode(value.Data, &j); err != nil {
		return api.Receipt{}, problem("INVALID_STATE", "invalid pending journal")
	}
	if err := s.validateJournal(root, j); err != nil {
		return api.Receipt{}, err
	}
	receipt := j.Receipt
	receipt.DryRun = dryRun
	if j.State == "prepared" {
		receipt.Warnings = append(receipt.Warnings, api.Diagnostic{Code: "ROLLBACK", Layer: "advisories", Message: "prepared transaction will be rolled back"})
	} else {
		receipt.Warnings = append(receipt.Warnings, api.Diagnostic{Code: "ROLLFORWARD", Layer: "advisories", Message: "committed transaction will be completed"})
	}
	// Refuse to overwrite a third-party edit, including during a retry of recovery.
	for _, file := range j.Files {
		current, err := readImage(root, file.Path, api.MaxNoteBytes)
		if err != nil {
			return api.Receipt{}, err
		}
		if !sameImage(current, file.Before) && !sameImage(current, file.After) {
			return api.Receipt{}, problem("REVISION_CONFLICT", "recovery would overwrite an external edit")
		}
	}
	if dryRun {
		return receipt, nil
	}
	for _, file := range j.Files {
		if err := ctx.Err(); err != nil {
			return api.Receipt{}, err
		}
		target := file.Before
		if j.State == "committed" {
			target = file.After
		}
		current, err := readImage(root, file.Path, api.MaxNoteBytes)
		if err != nil {
			return api.Receipt{}, err
		}
		if !sameImage(current, file.Before) && !sameImage(current, file.After) {
			return api.Receipt{}, problem("REVISION_CONFLICT", "a file changed during recovery")
		}
		if err := replaceImage(root, file.Path, target, &current); err != nil {
			return api.Receipt{}, err
		}
		if err := s.boundary("recovered", file.Path); err != nil {
			return api.Receipt{}, err
		}
	}
	if err := checkImages(root, j.Files, j.State == "committed"); err != nil {
		return api.Receipt{}, err
	}
	if j.State == "committed" {
		err = s.finish(root, j)
	} else {
		err = atomicImage(root, pendingPath, image{})
	}
	if err != nil {
		return api.Receipt{}, err
	}
	return receipt, nil
}

func (s *Service) validateJournal(root *os.Root, j journal) error {
	if j.Version != 1 || (j.State != "prepared" && j.State != "committed") || j.Key == "" || !validRevision(j.Digest) || j.Receipt.IdempotencyKey != j.Key || j.Receipt.DryRun || len(j.Files) > 10000 {
		return problem("INVALID_STATE", "invalid journal header")
	}
	seen := map[string]bool{}
	paths := map[string][]byte{}
	for _, file := range j.Files {
		if !inScope(file.Path, s.provider.Config()) || seen[caseKey(file.Path)] || len(file.Before.Data) > api.MaxNoteBytes || len(file.After.Data) > api.MaxNoteBytes || (!file.Before.Exists && len(file.Before.Data) > 0) || (!file.After.Exists && len(file.After.Data) > 0) || file.Before.Mode > 0777 || file.After.Mode > 0777 {
			return problem("INVALID_STATE", "invalid journal file")
		}
		seen[caseKey(file.Path)] = true
		paths[file.Path] = nil
		if err := inspect(root, file.Path); err != nil {
			return err
		}
	}
	return validatePathSet(paths)
}

func validRevision(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}
