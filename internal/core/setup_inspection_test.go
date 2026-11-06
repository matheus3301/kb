package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupInspectionDoesNotCreateOperationalFiles(t *testing.T) {
	s := newStore(t, false)
	state := filepath.Join(s.Root(), ".kb", "state")
	if err := os.RemoveAll(state); err != nil {
		t.Fatal(err)
	}
	put(t, s, "note.md", noteText("Fixture", "Observed knowledge."))
	report, err := s.InspectSetup(context.Background())
	if err != nil || !report.Conformant {
		t.Fatal(report, err)
	}
	if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("inspection created operational state: %v", err)
	}
	put(t, s, "invalid.md", "Missing required frontmatter.\n")
	report, err = s.InspectSetup(context.Background())
	if err != nil || report.Conformant || report.Summary.Errors == 0 {
		t.Fatal(report, err)
	}
	if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid inspection created operational state: %v", err)
	}
}

func TestSetupInspectionRejectsPendingRecoveryAndCancellation(t *testing.T) {
	s := newStore(t, false)
	state := filepath.Join(s.Root(), ".kb", "state")
	if err := os.MkdirAll(state, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "pending.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := s.InspectSetup(context.Background())
	requireCode(t, err, "RECOVERY_REQUIRED")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.InspectSetup(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestSetupScopeMatchesConfiguredInventory(t *testing.T) {
	s := newStore(t, false)
	for _, name := range []string{".agents/skills/knowledge-base/SKILL.md", ".kb/state/pending.json", "../escape.md", "/absolute.md"} {
		if s.IncludesPath(name) {
			t.Errorf("excluded/invalid path included: %s", name)
		}
	}
	if !s.IncludesPath("AGENTS.md") || !s.IncludesPath("nested/note.md") {
		t.Fatal("ordinary scoped notes were hidden by integration inspection")
	}
}
