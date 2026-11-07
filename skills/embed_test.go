package skills

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestKnowledgeBaseReferenceDocumentsResolve(t *testing.T) {
	raw, err := KnowledgeBase.ReadFile("knowledge-base/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	links := regexp.MustCompile(`\]\((references/[^)]+\.md)\)`).FindAllSubmatch(raw, -1)
	expected := map[string]bool{"retrieval.md": false, "writing.md": false, "repair.md": false, "maintenance.md": false, "results.md": false}
	for _, link := range links {
		path := string(link[1])
		document, err := KnowledgeBase.ReadFile("knowledge-base/" + path)
		if err != nil {
			t.Fatalf("installed skill link %s is broken: %v", path, err)
		}
		if !strings.Contains(string(document), "# ") {
			t.Fatalf("installed reference %s lacks workflow guidance", path)
		}
		expected[strings.TrimPrefix(path, "references/")] = true
	}
	for path, found := range expected {
		if !found {
			t.Errorf("required workflow reference %s not linked by the installed skill", path)
		}
	}
	if _, err := fs.Stat(KnowledgeBase, "knowledge-base/references/results.md"); err != nil {
		t.Fatal(err)
	}
}
