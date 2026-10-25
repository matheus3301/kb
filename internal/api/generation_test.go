package api

import (
	"fmt"
	"strings"
	"testing"
)

func TestGenerationPreservesCanonicalDigests(t *testing.T) {
	for _, fixture := range []struct {
		name  string
		files map[string][]byte
		want  string
	}{
		{"empty", nil, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"ordered paths", map[string][]byte{"a.md": []byte("one"), "b.md": []byte("two")}, "94cd3ea4ddf11dbde8620b2650bd80266f10217eb65a9286291e4cfbe47eb5bf"},
		{"exact bytes and delimiters", map[string][]byte{"nested/ação.md": []byte("---\r\ntype: Reference\r\n---\r\n\x00\xff"), "empty.md": {}, "line\nbreak.md": []byte("evidence\n")}, "18de2d3b303fb65fb6e276c2399d868d3d8ca50f4e0b7b8979bbeeb91688f7ca"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			if got := Generation(fixture.files); got != "sha256:"+fixture.want {
				t.Fatalf("canonical inventory digest changed: %s", got)
			}
		})
	}
}

func BenchmarkGenerationLargeCorpus(b *testing.B) {
	for _, count := range []int{10000, 50000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			files := make(map[string][]byte, count)
			for i := range count {
				files[fmt.Sprintf("group-%03d/note-%06d.md", i/500, i)] = []byte(strings.Repeat("x", 2048))
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				Generation(files)
			}
		})
	}
}
