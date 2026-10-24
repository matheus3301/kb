package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
)

// BenchmarkReadCore10K measures a synthetic corpus; it is not a target assertion.
// Run a compiled test binary with -test.run '^$' -test.bench ReadCore10K -test.benchtime 20x.
func BenchmarkReadCore10K(b *testing.B) {
	root, err := filepath.EvalSymlinks(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	s, err := Init(context.Background(), root, false)
	if err != nil {
		b.Fatal(err)
	}
	for i := range 10000 {
		raw := fmt.Sprintf("---\ntype: Reference\ntitle: Synthetic note %05d\ntags: [synthetic, benchmark]\n---\n# Evidence\nSynthetic s3 k8s ação recovery content %05d.\n", i, i)
		raw += strings.Repeat("Synthetic evidence for local retrieval. ", 2048/39)
		raw = raw[:2048]
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("note-%05d.md", i)), []byte(raw), 0600); err != nil {
			b.Fatal(err)
		}
	}
	cold := time.Now()
	if _, err := s.Search(context.Background(), api.SearchRequest{Query: "s3 recovery"}); err != nil {
		b.Fatal(err)
	}
	coldDuration := time.Since(cold)
	durations := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for range b.N {
		start := time.Now()
		result, err := s.Search(context.Background(), api.SearchRequest{Query: "s3 recovery"})
		if err != nil || result.Total != 10000 {
			b.Fatal(err, result.Total)
		}
		durations = append(durations, time.Since(start))
	}
	b.StopTimer()
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	b.ReportMetric(float64(coldDuration.Microseconds())/1000, "cold-ms")
	if len(durations) > 0 {
		b.ReportMetric(float64(durations[(len(durations)-1)*95/100].Microseconds())/1000, "p95-ms")
	}
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		b.Fatal(err)
	}
	rss := float64(usage.Maxrss)
	if runtime.GOOS == "linux" {
		rss *= 1024
	}
	b.ReportMetric(rss/(1<<20), "peak-RSS-MiB")
	b.Logf("hardware: %s/%s CPUs=%d runtime=%s; corpus=10000 synthetic notes of 2048 bytes", runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.Version())
}
