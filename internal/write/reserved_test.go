package write

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func TestNestedReservedLogDoesNotBlockWritesOrReceiveGlobalEvents(t *testing.T) {
	s, p := setup(t)
	manual := []byte("# Project log\n\nKeep this manual introduction.\n\n## 2026-10-23\n\n- Prior event with [local evidence](../existing.md#anchor).\n")
	put(t, p, "project/log.md", manual)
	put(t, p, "existing.md", document("Concept", "# Anchor\nOriginal"))
	if _, err := s.Apply(context.Background(), create("new.md")); err != nil {
		t.Fatal(err)
	}
	request := plan(api.Operation{Kind: "patch", Path: "existing.md", ExpectedRevision: api.Revision(get(t, p, "existing.md")), Body: body("# Anchor\nUpdated")})
	request.IdempotencyKey = "patch-with-nested-log"
	if _, err := s.Apply(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Maintain(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(get(t, p, "project/log.md"), manual) {
		t.Fatal("automatic maintenance overwrote the project-local log")
	}
	if strings.Count(string(get(t, p, "log.md")), "Record synthetic learning") != 2 {
		t.Fatal("logical write events were not recorded only in the root log")
	}
	move := plan(api.Operation{Kind: "move", Path: "existing.md", To: "archive/existing.md", ExpectedRevision: api.Revision(get(t, p, "existing.md"))})
	move.IdempotencyKey = "move-with-nested-log"
	if _, err := s.Apply(context.Background(), move); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(get(t, p, "project/log.md"), manual) {
		t.Fatal("automatic reference maintenance overwrote the project-local log")
	}
}

func TestReservedNamesApplyAtEveryDepth(t *testing.T) {
	for _, name := range []string{"index.md", "log.md", "project/index.md", "project/log.md", "project/archive/log.md"} {
		if !reserved(name) {
			t.Errorf("reserved name treated as concept: %s", name)
		}
	}
	for _, name := range []string{"catalog.md", "project/logbook.md", "project/concept.md"} {
		if reserved(name) {
			t.Errorf("concept treated as reserved: %s", name)
		}
	}
}
