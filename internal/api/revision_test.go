package api

import "testing"

func TestRevisionUsesCompleteFileBytes(t *testing.T) {
	if got := Revision([]byte("hello")); got != "sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("unexpected canonical revision: %s", got)
	}
	if Revision([]byte("hello")) == Revision([]byte("hello\n")) {
		t.Fatal("line-ending changes must invalidate a revision")
	}
}

func TestGenerationIsStableAcrossMapOrderAndTracksPaths(t *testing.T) {
	a := map[string][]byte{"a.md": []byte("one"), "b.md": []byte("two")}
	b := map[string][]byte{"b.md": []byte("two"), "a.md": []byte("one")}
	if Generation(a) != Generation(b) {
		t.Fatal("map traversal order must not affect generation")
	}
	b["b.md"] = []byte("changed")
	if Generation(a) == Generation(b) {
		t.Fatal("content changes must invalidate generation")
	}
	delete(b, "b.md")
	b["moved.md"] = []byte("two")
	if Generation(a) == Generation(b) {
		t.Fatal("moves must invalidate generation")
	}
	if Generation(nil) != Revision(nil) {
		t.Fatal("empty inventory must have the empty byte revision")
	}
}
