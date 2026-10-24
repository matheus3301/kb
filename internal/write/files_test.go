package write

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestFIFOJournalReturnsPromptlyAndReleasesExclusiveLock(t *testing.T) {
	s, p := setup(t)
	put(t, p, "a.md", document("Concept", "A"))
	if err := os.MkdirAll(filepath.Join(p.root, ".kb/state"), 0700); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(p.root, filepath.FromSlash(pendingPath))
	if err := syscall.Mkfifo(name, 0600); err != nil {
		t.Fatal(err)
	}
	testRoot, err := os.OpenRoot(p.root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := testRoot.Close(); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := s.Recover(ctx, false); done <- err }()
	select {
	case err := <-done:
		errorCode(t, err, "INVALID_PATH")
	case <-time.After(500 * time.Millisecond):
		// Release a regressed blocking FIFO open before failing the test.
		f, err := testRoot.OpenFile(pendingPath, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			if closeErr := f.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		}
		t.Fatal("special journal blocked recovery beyond its cancellation deadline")
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if _, err := s.Apply(ctx2, create("new.md")); err != nil {
		t.Fatalf("exclusive lock was not released: %v", err)
	}
}

func TestSpecialFileInInspectedParentCannotBlock(t *testing.T) {
	_, p := setup(t)
	name := filepath.Join(p.root, "pipe")
	if err := syscall.Mkfifo(name, 0600); err != nil {
		t.Fatal(err)
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
	done := make(chan error, 1)
	go func() { done <- inspect(root, "pipe/note.md") }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO parent accepted as a directory")
		}
	case <-time.After(500 * time.Millisecond):
		f, err := root.OpenFile("pipe", os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			if closeErr := f.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		}
		t.Fatal("special parent blocked inspection")
	}
}
