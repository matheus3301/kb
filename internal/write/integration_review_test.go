package write

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func TestBlockingPolicyDiagnosticsApplyBeforeAndAfterWithHealthNonblocking(t *testing.T) {
	for _, layer := range []string{"conformance_errors", "policy_findings", "advisories"} {
		t.Run(layer, func(t *testing.T) {
			s, p := setup(t)
			request := create("a.md")
			request.Operations[0].Metadata = map[string]any{"test_diagnostic_layer": layer}
			_, err := s.Apply(context.Background(), request)
			if layer == "advisories" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				errorCode(t, err, "INVALID_FRONTMATTER")
				if len(inventory(t, p)) != 0 {
					t.Fatal("invalid post-state was published")
				}
			}
			put(t, p, "prior.md", []byte("---\n{\"type\":\"Concept\",\"test_diagnostic_layer\":\""+layer+"\"}\n---\nPrior"))
			request = create("b.md")
			request.IdempotencyKey = "second"
			_, err = s.Apply(context.Background(), request)
			if layer == "advisories" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				errorCode(t, err, "INVALID_FRONTMATTER")
			}
		})
	}
}

func TestMovePreservesLargeNumbersInRewrittenUnknownMetadata(t *testing.T) {
	s, p := setup(t)
	put(t, p, "basis.md", document("Concept", "Basis"))
	revision := api.Revision(get(t, p, "basis.md"))
	large := "900719925474099312345678901234567890"
	put(t, p, "ref.md", []byte("---\n{\"type\":\"Synthesis\",\"sources\":[{\"resource\":\"basis.md\",\"unknown\":{\"sequence\":"+large+",\"values\":[1.25,"+large+"]}}],\"basis_revisions\":[{\"path\":\"basis.md\",\"revision\":\""+revision+"\",\"unknown\":"+large+"}]}\n---\n[Evidence](basis.md)"))
	request := plan(api.Operation{Kind: "move", Path: "basis.md", To: "nested/basis.md", ExpectedRevision: revision})
	if _, err := s.Apply(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	note, _ := p.Engine().Parse("ref.md", get(t, p, "ref.md"))
	source := note.Metadata["sources"].([]any)[0].(map[string]any)
	unknown := source["unknown"].(map[string]any)
	if unknown["sequence"] != json.Number(large) || unknown["values"].([]any)[1] != json.Number(large) {
		t.Fatalf("source metadata lost precision: %+v", unknown)
	}
	basis := note.Metadata["basis_revisions"].([]any)[0].(map[string]any)
	if basis["unknown"] != json.Number(large) || basis["revision"] != revision || basis["path"] != "nested/basis.md" || source["resource"] != "nested/basis.md" {
		t.Fatalf("basis/provenance changed unexpectedly: %+v %+v", basis, source)
	}
}

func TestMovingStaleSynthesisPreservesEvidenceAndRequestsReview(t *testing.T) {
	s, p := setup(t)
	put(t, p, "basis.md", document("Concept", "Current basis"))
	stale := api.Revision([]byte("older basis bytes"))
	put(t, p, "synthesis.md", []byte("---\n{\"type\":\"Synthesis\",\"basis_revisions\":[{\"path\":\"basis.md\",\"revision\":\""+stale+"\"}]}\n---\n[Evidence](basis.md#anchor)"))
	request := plan(api.Operation{Kind: "move", Path: "synthesis.md", To: "nested/synthesis.md", ExpectedRevision: api.Revision(get(t, p, "synthesis.md"))})
	receipt, err := s.Apply(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	fresh := get(t, p, "nested/synthesis.md")
	if !strings.Contains(string(fresh), stale) || !strings.Contains(string(fresh), "../basis.md#anchor") {
		t.Fatal("move refreshed stale evidence or failed to relocate reference")
	}
	assertReviewWarning(t, receipt, "nested/synthesis.md")
	request = plan(api.Operation{Kind: "patch", Path: "nested/synthesis.md", ExpectedRevision: api.Revision(fresh), Body: body("Authored new synthesis")})
	request.IdempotencyKey = "authored-patch"
	_, err = s.Apply(context.Background(), request)
	errorCode(t, err, "REVISION_CONFLICT")
}

func TestMovingDependencyKeepsStaleAndNewlyOutdatedIncomingSynthesisEvidence(t *testing.T) {
	s, p := setup(t)
	put(t, p, "other.md", document("Concept", "Other"))
	put(t, p, "basis.md", document("Concept", "[Other](other.md#anchor)"))
	beforeRevision := api.Revision(get(t, p, "basis.md"))
	staleRevision := api.Revision([]byte("older version"))
	for name, revision := range map[string]string{"fresh.md": beforeRevision, "stale.md": staleRevision} {
		put(t, p, name, []byte("---\n{\"type\":\"Synthesis\",\"basis_revisions\":[{\"path\":\"basis.md\",\"revision\":\""+revision+"\"}]}\n---\n[Evidence](basis.md#evidence)"))
	}
	request := plan(api.Operation{Kind: "move", Path: "basis.md", To: "nested/basis.md", ExpectedRevision: beforeRevision})
	receipt, err := s.Apply(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if api.Revision(get(t, p, "nested/basis.md")) == beforeRevision {
		t.Fatal("fixture did not change moved dependency bytes")
	}
	for name, revision := range map[string]string{"fresh.md": beforeRevision, "stale.md": staleRevision} {
		note, _ := p.Engine().Parse(name, get(t, p, name))
		entry := note.Metadata["basis_revisions"].([]any)[0].(map[string]any)
		if entry["path"] != "nested/basis.md" || entry["revision"] != revision {
			t.Fatalf("recorded evidence refreshed: %+v", entry)
		}
		if !strings.Contains(note.Body, "nested/basis.md#evidence") {
			t.Fatal("incoming reference was not updated")
		}
		assertReviewWarning(t, receipt, name)
	}
}

func assertReviewWarning(t *testing.T, receipt api.Receipt, name string) {
	t.Helper()
	for _, warning := range receipt.Warnings {
		if warning.Path == name && warning.Code == "BASIS_CHANGED" && warning.Layer == "advisories" {
			return
		}
	}
	t.Fatalf("missing review warning for %s: %+v", name, receipt.Warnings)
}

func TestAuthoredSynthesisRequiresFinalDependencyBytesAfterMove(t *testing.T) {
	s, p := setup(t)
	put(t, p, "other.md", document("Concept", "Other"))
	put(t, p, "basis.md", document("Concept", "[Other](other.md#anchor)"))
	prior := api.Revision(get(t, p, "basis.md"))
	request := api.Plan{Actor: "synthetic-agent", Summary: "Move evidence and author synthesis", IdempotencyKey: "final-dependency", Operations: []api.Operation{
		{OperationID: "move", Kind: "move", Path: "basis.md", To: "nested/basis.md", ExpectedRevision: prior},
		{OperationID: "synthesis", Kind: "create", Path: "new.md", Type: "Synthesis", Body: body("New authored synthesis"), Metadata: map[string]any{"basis_revisions": []any{map[string]any{"path": "basis.md", "revision": prior}}}},
	}}
	before := inventory(t, p)
	_, err := s.Apply(context.Background(), request)
	errorCode(t, err, "REVISION_CONFLICT")
	equalInventory(t, before, inventory(t, p))
	if _, err := os.Stat(filepath.Join(p.root, pendingPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("stale authored evidence created a journal")
	}
	final := api.Revision(document("Concept", "[Other](../other.md#anchor)"))
	request.Operations[1].Metadata["basis_revisions"] = []any{map[string]any{"path": "nested/basis.md", "revision": final}}
	if _, err := s.Apply(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if api.Revision(get(t, p, "nested/basis.md")) != final {
		t.Fatal("dependency final bytes differed from its checked revision")
	}
}
