//go:build linux || darwin

// Package filelock coordinates readers and writers of one local knowledge base.
package filelock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// Lock owns an advisory lock and its descriptor. Close is safe to repeat.
type Lock struct {
	mu   sync.Mutex
	file *os.File
}

// Acquire waits for the shared base lock, or the exclusive lock for a writer.
// Operational directories are created only beneath a confined directory handle.
func Acquire(ctx context.Context, root string, exclusive bool) (*Lock, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	for p := abs; ; p = filepath.Dir(p) {
		info, statErr := os.Lstat(p)
		if statErr != nil {
			return nil, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink in base path")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	r, err := os.OpenRoot(abs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	for _, name := range []string{".kb", ".kb/state"} {
		if err := r.Mkdir(name, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		info, err := r.Lstat(name)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("unsafe operational directory")
		}
	}
	f, err := r.OpenFile(".kb/state/lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = f.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("lock must be a regular file")
	}
	mode := syscall.LOCK_SH
	if exclusive {
		mode = syscall.LOCK_EX
	}
	timer := time.NewTicker(10 * time.Millisecond)
	defer timer.Stop()
	for {
		if err := ctx.Err(); err != nil {
			_ = f.Close()
			return nil, err
		}
		err := syscall.Flock(int(f.Fd()), mode|syscall.LOCK_NB)
		if err == nil {
			return &Lock{file: f}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) && !errors.Is(err, syscall.EINTR) {
			_ = f.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// Close releases the lock even when explicitly unlocking reports an error.
func (l *Lock) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	f := l.file
	l.file = nil
	return errors.Join(syscall.Flock(int(f.Fd()), syscall.LOCK_UN), f.Close())
}
