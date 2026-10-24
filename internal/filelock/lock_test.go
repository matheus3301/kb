package filelock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSharedExclusiveCancellationAndRelease(t *testing.T) {
	root := tempRoot(t)
	a, err := Acquire(context.Background(), root, false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Acquire(context.Background(), root, false)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, err := Acquire(ctx, root, true); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked writer: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	c, err := Acquire(context.Background(), root, true)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			if err := c.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	d, err := Acquire(context.Background(), root, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRejectSymlinkOperationalPaths(t *testing.T) {
	for _, component := range []string{"root", ".kb", "state", "lock"} {
		t.Run(component, func(t *testing.T) {
			root, outside := tempRoot(t), tempRoot(t)
			var target string
			switch component {
			case "root":
				target = filepath.Join(root, "alias")
				root = target
			case ".kb":
				target = filepath.Join(root, ".kb")
			case "state":
				if err := os.Mkdir(filepath.Join(root, ".kb"), 0700); err != nil {
					t.Fatal(err)
				}
				target = filepath.Join(root, ".kb/state")
			case "lock":
				if err := os.MkdirAll(filepath.Join(root, ".kb/state"), 0700); err != nil {
					t.Fatal(err)
				}
				target = filepath.Join(root, ".kb/state/lock")
			}
			if err := os.Symlink(outside, target); err != nil {
				t.Fatal(err)
			}
			if l, err := Acquire(context.Background(), root, true); err == nil {
				_ = l.Close()
				t.Fatal("accepted symlink")
			}
		})
	}
}

func TestAlreadyCancelledAndInvalidRoot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Acquire(ctx, tempRoot(t), false); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := Acquire(context.Background(), filepath.Join(tempRoot(t), "missing"), false); err == nil {
		t.Fatal("missing root")
	}
	root := tempRoot(t)
	if err := os.WriteFile(filepath.Join(root, ".kb"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Acquire(context.Background(), root, false); err == nil {
		t.Fatal("regular operational directory")
	}
}

func tempRoot(t *testing.T) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
