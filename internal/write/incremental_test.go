package write

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

type incrementalProvider struct {
	*fakeProvider
	scanError bool
}

func (p incrementalProvider) SnapshotUnlocked(ctx context.Context) (api.Snapshot, error) {
	snap, err := p.fakeProvider.SnapshotUnlocked(ctx)
	snap.Diagnostics = nil
	for name, raw := range snap.Files {
		ds := p.Engine().ValidateReserved(name, raw)
		if !reserved(name) {
			_, ds = p.Engine().Parse(name, raw)
		}
		for _, d := range ds {
			d.Path = name
			snap.Diagnostics = append(snap.Diagnostics, d)
		}
	}
	if p.scanError {
		snap.Diagnostics = append(snap.Diagnostics, api.Diagnostic{Path: "unreadable.md", Code: "SCAN_FAILED", Layer: "conformance_errors"})
	}
	return snap, err
}

func incrementalFixture(t *testing.T) (*Service, *fakeProvider, api.Plan) {
	t.Helper()
	s, p := setup(t)
	s.provider = incrementalProvider{fakeProvider: p}
	put(t, p, "a.md", []byte("Original A"))
	put(t, p, "b.md", []byte("Original B"))
	request := plan(api.Operation{Kind: "patch", Path: "a.md", ExpectedRevision: api.Revision(get(t, p, "a.md")), Document: body(string(document("Reference", "Original A")))})
	request.Repair = true
	request.Incremental = true
	// Production init creates the shared lock before concurrent requests begin.
	if _, err := s.Recover(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	return s, p, request
}

func TestIncrementalRepairRecoversAtEveryPublicationBoundary(t *testing.T) {
	for _, stage := range []string{"prepared", "published", "committed", "receipt", "removed"} {
		t.Run(stage, func(t *testing.T) {
			s, p, request := incrementalFixture(t)
			before := get(t, p, "a.md")
			s.fault = func(at, _ string) error {
				if at == stage {
					return errors.New("synthetic interruption")
				}
				return nil
			}
			if _, err := s.Apply(context.Background(), request); err == nil {
				t.Fatal("missing injected failure")
			}
			s.fault = nil
			if _, err := s.Recover(context.Background(), false); err != nil {
				t.Fatal(err)
			}
			committed := stage == "committed" || stage == "receipt" || stage == "removed"
			if committed == (string(get(t, p, "a.md")) == string(before)) {
				t.Fatal("recovery chose wrong outcome")
			}
			receipt, err := s.Apply(context.Background(), request)
			if err != nil || receipt.RepairProgress == nil || receipt.RepairProgress.ErrorsRemaining != 1 {
				t.Fatal(receipt, err)
			}
			if string(get(t, p, "b.md")) != "Original B" {
				t.Fatal("untouched invalid note changed")
			}
		})
	}
}

func TestConcurrentIncrementalRepairsSerializeAndRetainCAS(t *testing.T) {
	s, p, request := incrementalFixture(t)
	var wg sync.WaitGroup
	codes := make(chan string, 2)
	for i := 0; i < 2; i++ {
		wg.Go(func() {
			copy := request
			// Different keys must reach revision checks instead of replaying one write.
			copy.IdempotencyKey = request.IdempotencyKey + strconv.Itoa(i)
			_, err := s.Apply(context.Background(), copy)
			if err == nil {
				codes <- "success"
				return
			}
			var de *api.DomainError
			if errors.As(err, &de) {
				codes <- de.Code
				return
			}
			codes <- err.Error()
		})
	}
	wg.Wait()
	close(codes)
	counts := map[string]int{}
	for code := range codes {
		counts[code]++
	}
	if counts["success"] != 1 || counts["REVISION_CONFLICT"] != 1 {
		t.Fatal(counts)
	}
	if string(get(t, p, "b.md")) != "Original B" {
		t.Fatal("invalid note changed")
	}
}

func TestIncrementalRepairDefersInvalidReservedNavigation(t *testing.T) {
	for _, name := range []string{"index.md", "log.md"} {
		t.Run(name, func(t *testing.T) {
			s, p, request := incrementalFixture(t)
			put(t, p, name, []byte("INVALID existing manual content"))
			receipt, err := s.Apply(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if string(get(t, p, name)) != "INVALID existing manual content" {
				t.Fatal("invalid navigation overwritten")
			}
			if receipt.RepairProgress.ErrorsRemaining != 2 {
				t.Fatal(receipt)
			}
			found := false
			for _, d := range receipt.Warnings {
				if d.Code == "NAVIGATION_DEFERRED" {
					found = true
				}
			}
			if !found {
				t.Fatal("deferred navigation not reported")
			}
		})
	}
}

func TestIncrementalRepairRejectsIncompleteInventoryAndInvalidChangedFiles(t *testing.T) {
	s, p, request := incrementalFixture(t)
	s.provider = incrementalProvider{fakeProvider: p, scanError: true}
	_, err := s.Apply(context.Background(), request)
	errorCode(t, err, "INVALID_FRONTMATTER")
	if _, err := os.Stat(p.root + "/" + pendingPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("scan failure created journal")
	}
	s.provider = incrementalProvider{fakeProvider: p}
	// Fix one file but break another valid file: total count alone is insufficient.
	put(t, p, "valid.md", document("Reference", "Preserve valid"))
	request.Operations[0].OperationID = "fix"
	request.Operations = append(request.Operations, api.Operation{OperationID: "fix-b", Kind: "patch", Path: "b.md", ExpectedRevision: api.Revision(get(t, p, "b.md")), Document: body(string(document("Reference", "Original B")))}, api.Operation{OperationID: "break", Kind: "patch", Path: "valid.md", ExpectedRevision: api.Revision(get(t, p, "valid.md")), Document: body("invalid")})
	_, err = s.Apply(context.Background(), request)
	errorCode(t, err, "INVALID_FRONTMATTER")
	if string(get(t, p, "a.md")) != "Original A" {
		t.Fatal("rejected plan changed content")
	}
}
