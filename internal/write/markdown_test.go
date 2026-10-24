package write

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/matheus3301/kb/internal/api"
)

func TestIndexMarkersInCodeExamplesPreserveManualText(t *testing.T) {
	for _, example := range []string{
		"```html\n" + indexStart + "\nExample content\n" + indexEnd + "\n```\n",
		"~~~html\n" + indexStart + "\nExample content\n" + indexEnd + "\n~~~\n",
		"````html\n```\n" + indexStart + "\nExample content\n" + indexEnd + "\n````\n",
		"```html\n```go\n" + indexStart + "\nExample content\n" + indexEnd + "\n``` \t\n",
		"`" + indexStart + "` and `" + indexEnd + "` are examples.\n",
		"Example `multiline\n" + indexStart + "\nExample content\n" + indexEnd + "\ncode` text.\n",
		"    " + indexStart + "\n    Example content\n    " + indexEnd + "\n",
		"\t" + indexStart + "\n\tExample content\n\t" + indexEnd + "\n",
	} {
		for _, managed := range []bool{false, true} {
			t.Run(example+strconv.FormatBool(managed), func(t *testing.T) {
				s, p := setup(t)
				manual := "# Index\n\n" + example + "\nKeep manual prose.\n"
				initial := manual
				if managed {
					initial += "\n" + indexStart + "\nOld generated entry\n" + indexEnd + "\n\nManual suffix.\n"
				}
				put(t, p, "index.md", []byte(initial))
				put(t, p, "a.md", document("Concept", "Body"))
				if _, err := s.Maintain(context.Background(), false); err != nil {
					t.Fatal(err)
				}
				actual := string(get(t, p, "index.md"))
				if !strings.HasPrefix(actual, manual) {
					t.Fatalf("manual code example changed:\n%s", actual)
				}
				if !strings.Contains(actual, "- [a](a.md)") {
					t.Fatal("missing actual generated navigation")
				}
				if managed && (!strings.HasSuffix(actual, "\n\nManual suffix.\n") || strings.Contains(actual, "Old generated entry")) {
					t.Fatal("managed region or manual suffix changed incorrectly")
				}
				before := inventory(t, p)
				receipt, err := s.Maintain(context.Background(), false)
				if err != nil {
					t.Fatal(err)
				}
				if len(receipt.Changes) != 0 {
					t.Fatal("repeated maintenance changed code examples or index")
				}
				equalInventory(t, before, inventory(t, p))
			})
		}
	}
}

func TestIndexRegionIgnoresFrontmatterAndRejectsUnsafeAppend(t *testing.T) {
	frontmatter := "---\nokf_version: |\n  " + indexStart + "\n  " + indexEnd + "\n---\n"
	text := frontmatter + indexStart + "\nNavigation\n" + indexEnd + "\n"
	start, end, safe, err := indexRegion(text)
	if err != nil || !safe || start != len(frontmatter) || end != len(frontmatter)+len(indexStart)+len("\nNavigation\n") {
		t.Fatalf("frontmatter markers activated: %d %d %v %v", start, end, safe, err)
	}
	for _, manual := range []string{"# Index\n\n```html\n" + indexStart + "\n" + indexEnd + "\n", "# Index\n\n<!-- Unclosed comment\n" + indexStart + "\n" + indexEnd + "\n"} {
		s, p := setup(t)
		put(t, p, "index.md", []byte(manual))
		put(t, p, "a.md", document("Concept", "Body"))
		before := inventory(t, p)
		_, err := s.Maintain(context.Background(), false)
		errorCode(t, err, "INVALID_INDEX")
		equalInventory(t, before, inventory(t, p))
	}
}

func TestFenceWithInfoStringDoesNotCloseCodeOrExposeLinks(t *testing.T) {
	for _, delimiter := range []string{"```", "~~~"} {
		text := delimiter + "html\n[example](ignored.md)\n" + delimiter + "go\n[still code](ignored.md)\n" + delimiter + " \t\n[actual](actual.md)\n"
		spans := markdownDestinations(text)
		if len(spans) != 1 || spans[0].value != "actual.md" {
			t.Fatalf("invalid closing fence exposed code links: %+v", spans)
		}
	}
}

func TestMalformedStandaloneIndexMarkersRemainRejected(t *testing.T) {
	for _, text := range []string{indexStart + "\n", indexEnd + "\n", indexEnd + "\n" + indexStart + "\n", indexStart + "\n" + indexStart + "\n" + indexEnd + "\n", indexStart + "\n" + indexEnd + "\n" + indexEnd + "\n"} {
		_, _, _, err := indexRegion(text)
		errorCode(t, err, "INVALID_INDEX")
	}
}

func FuzzIndexRegion(f *testing.F) {
	for _, seed := range []string{indexStart + "\n" + indexEnd + "\n", "```html\n" + indexStart + "\n" + indexEnd + "\n```\n", "`" + indexStart + "`\n", "```\n```go\n" + indexStart + "\n```\n"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		start, end, _, err := indexRegion(text)
		if err != nil {
			return
		}
		if start >= 0 && (end <= start || end+len(indexEnd) > len(text) || text[start:start+len(indexStart)] != indexStart || text[end:end+len(indexEnd)] != indexEnd) {
			t.Fatalf("invalid managed region offsets: %d %d", start, end)
		}
	})
}

func TestNestedListLinksRelocateWhileIndentedCodeRemainsUnchanged(t *testing.T) {
	for _, container := range []string{"- Parent\n    - ", "1. Parent\n    - ", "-\tParent\n\t- ", "> - Parent\n>     - "} {
		t.Run(container, func(t *testing.T) {
			s, p := setup(t)
			put(t, p, "a.md", document("Concept", "# Anchor\nEvidence"))
			content := container + "[Evidence](a.md#anchor)\n\nOutside paragraph.\n\n    [Code example](a.md#anchor)\n\n- Parent\n\n      [Nested code](a.md#anchor)\n"
			put(t, p, "ref.md", document("Concept", content))
			// The Provider's parser has already confirmed the nested list reference.
			p.extraLinks = []api.Link{{From: "ref.md", To: "a.md", Kind: "markdown", Resolved: true, Line: 6}}
			request := plan(api.Operation{Kind: "move", Path: "a.md", To: "nested/a.md", ExpectedRevision: api.Revision(get(t, p, "a.md"))})
			if _, err := s.Apply(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			actual := string(get(t, p, "ref.md"))
			if !strings.Contains(actual, container+"[Evidence](nested/a.md#anchor)") {
				t.Fatalf("nested reference not relocated: %s", actual)
			}
			for _, code := range []string{"    [Code example](a.md#anchor)", "      [Nested code](a.md#anchor)"} {
				if !strings.Contains(actual, code) {
					t.Fatalf("indented code was rewritten: %s", actual)
				}
			}
		})
	}
}

func TestListContainerFencesRemainCode(t *testing.T) {
	text := "- Parent\n\n  ```html\n  - [Example](a.md)\n  ```go\n  - [Still code](a.md)\n  ```\n\n  - [Actual](a.md)\n"
	spans := markdownDestinations(text)
	if len(spans) != 1 || text[spans[0].start:spans[0].end] != "a.md" || spans[0].start < strings.Index(text, "[Actual]") {
		t.Fatalf("list fences exposed examples: %+v", spans)
	}
}
