package core

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func TestFreshInventoryDistinguishesAddedAndRemovedEmptyFiles(t *testing.T) {
	s := newStore(t, false)
	put(t, s, "empty-old.md", "")
	first, err := s.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := root.Remove("empty-old.md"); err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("empty-new.md", nil, 0600); err != nil {
		t.Fatal(err)
	}
	second, err := s.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if second.Generation == first.Generation || second.Invalid != 1 {
		t.Fatal(first, second)
	}
	if _, err := s.Read(context.Background(), api.ReadRequest{Path: "empty-old.md"}); err == nil {
		t.Fatal("removed empty note remained visible")
	}
	if _, err := s.Read(context.Background(), api.ReadRequest{Path: "empty-new.md"}); err != nil {
		t.Fatal(err)
	}
}

func TestConfinedWalkerRejectsDirectoryReplacementBeforeAndAfterTraversal(t *testing.T) {
	for _, stage := range []string{"before", "after"} {
		t.Run(stage, func(t *testing.T) {
			s := newStore(t, false)
			put(t, s, "nested/source.md", noteText("Original", "Keep"))
			root, err := os.OpenRoot(s.Root())
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := root.Close(); err != nil {
					t.Error(err)
				}
			}()
			changed := false
			err = walkInventory(context.Background(), root, func(_ *os.Root, name string, _ fs.DirEntry) error {
				if !changed && (stage == "before" && name == "nested" || stage == "after" && name == "nested/source.md") {
					changed = true
					if err := root.Rename("nested", "retained"); err != nil {
						return err
					}
					return root.Symlink("retained", "nested")
				}
				return nil
			}, func() {})
			requireCode(t, err, "CONFLICT")
			if !changed {
				t.Fatal("replacement seam not reached")
			}
		})
	}
}

func TestParallelInventoryReturnsFirstTraversalFailureWithScopedPath(t *testing.T) {
	s := newStore(t, false)
	root, err := os.OpenRoot(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := root.Mkdir("a", 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a/oversized.md", "b.md"} {
		f, err := root.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := errors.Join(f.Truncate(api.MaxPlanBytes+1), f.Close()); err != nil {
			t.Fatal(err)
		}
	}
	_, err = s.Status(context.Background())
	var de *api.DomainError
	if !errors.As(err, &de) || de.Code != "LIMIT_EXCEEDED" || de.Details.(map[string]any)["path"] != "a/oversized.md" {
		t.Fatalf("%#v", err)
	}
}

func TestNestedExternalEditWithRestoredTimeInvalidatesFreshBytesAndGraph(t *testing.T) {
	s := newStore(t, false)
	put(t, s, "nested/a.md", noteText("Stable", "[first](first.md)"))
	put(t, s, "nested/first.md", noteText("First", "Source"))
	put(t, s, "nested/other.md", noteText("Other", "Source"))
	initial, err := s.Search(context.Background(), api.SearchRequest{Query: "first"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Links(context.Background(), "nested/a.md", "out"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(s.Root(), "nested/a.md"))
	if err != nil {
		t.Fatal(err)
	}
	put(t, s, "nested/a.md", noteText("Stable", "[other](other.md)"))
	if err := os.Chtimes(filepath.Join(s.Root(), "nested/a.md"), info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	current, err := s.Search(context.Background(), api.SearchRequest{Query: "other"})
	if err != nil {
		t.Fatal(err)
	}
	if current.Generation == initial.Generation {
		t.Fatal("same-size same-mtime edit was missed")
	}
	links, err := s.Links(context.Background(), "nested/a.md", "out")
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].To != "nested/other.md" || !links[0].Resolved {
		t.Fatal(links)
	}
}
