package write

import (
	"errors"
	"github.com/matheus3301/kb/internal/api"
	"os"
	"path/filepath"
	"testing"
)

func TestBulkInspectionPreservesPerPathConfinementAndCaseChecks(t *testing.T) {
	for _, scenario := range []string{"valid", "missing", "file symlink", "directory symlink", "case collision", "parent case collision", "file parent"} {
		t.Run(scenario, func(t *testing.T) {
			_, p := setup(t)
			name := "nested/source.md"
			put(t, p, name, document("Reference", "source"))
			switch scenario {
			case "missing":
				name = "missing/new.md"
			case "file symlink":
				if err := os.Symlink(filepath.Join(p.root, name), filepath.Join(p.root, "alias.md")); err != nil {
					t.Fatal(err)
				}
				name = "alias.md"
			case "directory symlink":
				if err := os.Symlink(filepath.Join(p.root, "nested"), filepath.Join(p.root, "alias")); err != nil {
					t.Fatal(err)
				}
				name = "alias/source.md"
			case "case collision":
				name = "nested/Source.md"
			case "parent case collision":
				name = "Nested/source.md"
			case "file parent":
				name = "nested/source.md/child.md"
			}
			root, err := os.OpenRoot(p.root)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := root.Close(); err != nil {
					t.Error(err)
				}
			}()
			expected := inspect(root, name)
			actual := inspectInventory(root, map[string][]byte{name: nil})
			if (expected == nil) != (actual == nil) || expected != nil && errors.Is(expected, os.ErrNotExist) != errors.Is(actual, os.ErrNotExist) {
				t.Fatalf("%v != %v", expected, actual)
			}
			var expectedDomain, actualDomain *api.DomainError
			if errors.As(expected, &expectedDomain) && (!errors.As(actual, &actualDomain) || expectedDomain.Code != actualDomain.Code) {
				t.Fatalf("%v != %v", expected, actual)
			}
		})
	}
}

func TestValidationMemoRechecksChangedBytesAndPathWithinPlan(t *testing.T) {
	s, _ := setup(t)
	s.validationCache = map[validationKey]validationRecord{}
	valid := document("Reference", "source")
	if s.validation("a.md", valid).missingType {
		t.Fatal("valid note rejected")
	}
	invalid := []byte("missing frontmatter")
	if !s.validation("a.md", invalid).missingType {
		t.Fatal("stale valid result accepted changed bytes")
	}
	if s.validation("index.md", []byte("# Index\n")).missingType {
		t.Fatal("reserved file treated as concept")
	}
	if !s.validation("note.md", []byte("# Index\n")).missingType {
		t.Fatal("cached reserved result accepted as concept")
	}
}
