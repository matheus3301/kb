package write

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/matheus3301/kb/internal/api"
)

func waitRestoreDescriptorClosed(t *testing.T, file *os.File) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := file.Stat(); errors.Is(err, os.ErrClosed) {
			return
		} else if err != nil {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("restore worker did not reach descriptor closure")
		}
		runtime.Gosched()
	}
}

func TestRestoreBatchJoinsBeforeFailedLeafCleanup(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := root.Mkdir("destination", 0700); err != nil {
		t.Fatal(err)
	}
	created, err := root.Lstat("destination")
	if err != nil {
		t.Fatal(err)
	}
	dest, err := root.OpenRoot("destination")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dest.Close() }()
	var files []restoredFile
	var directories []string
	var hooks atomic.Int32
	batch := restoreBatch{ctx: context.Background(), dest: dest, files: &files, directories: &directories,
		boundary: func(string, string) error { hooks.Add(1); return nil }}
	for i := range restorePendingLimit {
		data := []byte(fmt.Sprintf("synthetic payload %d\n", i))
		entry := backupEntry{Path: fmt.Sprintf("payload-%d.bin", i), Size: int64(len(data)), Revision: api.Revision(data)}
		pending, err := prepareRestoreFile(dest, entry, data, &directories)
		if err != nil {
			t.Fatal(err)
		}
		batch.pending[i] = pending
		batch.count++
	}
	partial := []byte("synthetic")
	if err := batch.pending[0].file.Truncate(int64(len(partial))); err != nil {
		t.Fatal(err)
	}
	batch.pending[0].writeErr = io.ErrShortWrite

	// A pipe produces a real Sync error. A separate RawConn reference holds the
	// second pipe's Close blocked; no production worker invokes a test callback.
	var releaseOnce sync.Once
	release := make(chan struct{})
	defer releaseOnce.Do(func() { close(release) })
	controlReady := make(chan struct{})
	controlDone := make(chan error, 1)
	var blocked *os.File
	for i := range 2 {
		if err := batch.pending[i].file.Close(); err != nil {
			t.Fatal(err)
		}
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = writer.Close() }()
		batch.pending[i].file = reader
		if i == 1 {
			blocked = reader
			raw, err := reader.SyscallConn()
			if err != nil {
				t.Fatal(err)
			}
			go func() {
				controlDone <- raw.Control(func(uintptr) { close(controlReady); <-release })
			}()
		}
	}
	<-controlReady
	failed := batch.pending[0].file
	completed := batch.pending[2].file
	failedParent := batch.pending[0].parent
	finished := make(chan error, 1)
	go func() { finished <- batch.flush(true) }()
	joined, controlJoined := false, false
	defer func() {
		releaseOnce.Do(func() { close(release) })
		if !controlJoined {
			if err := <-controlDone; err != nil {
				t.Errorf("blocked descriptor control failed during teardown: %v", err)
			}
		}
		batch.workers.Wait()
		if !joined {
			if err := <-finished; err == nil {
				t.Error("failed restore batch unexpectedly succeeded during teardown")
			}
		}
	}()
	waitRestoreDescriptorClosed(t, failed)
	waitRestoreDescriptorClosed(t, blocked)
	waitRestoreDescriptorClosed(t, completed)
	select {
	case err := <-finished:
		joined = true
		t.Fatalf("batch returned before its blocked worker joined: %v", err)
	default:
	}
	if _, err := failedParent.Stat("."); err != nil {
		t.Fatalf("caller closed a pending parent before joining: %v", err)
	}
	for i := range 2 {
		name := fmt.Sprintf("payload-%d.bin", i)
		want := fmt.Sprintf("synthetic payload %d\n", i)
		if i == 0 {
			want = string(partial)
		}
		if raw, err := dest.ReadFile(name); err != nil || string(raw) != want {
			t.Fatalf("failed-leaf cleanup ran before every worker joined: %s %v", name, err)
		}
	}
	if hooks.Load() != 0 {
		t.Fatal("batch delivered a boundary while a worker remained blocked")
	}
	releaseOnce.Do(func() { close(release) })
	controlErr := <-controlDone
	controlJoined = true
	if controlErr != nil {
		t.Fatal(controlErr)
	}
	batchErr := <-finished
	joined = true
	if !errors.Is(batchErr, io.ErrShortWrite) {
		t.Fatalf("partial-write failure was lost: %v", batchErr)
	}
	var syncErr *os.PathError
	if !errors.As(batchErr, &syncErr) || syncErr.Op != "sync" {
		t.Fatalf("actual worker Sync error was lost: %v", batchErr)
	}
	if hooks.Load() != 0 {
		t.Fatal("failed batch delivered a successful file boundary")
	}
	if _, err := failedParent.Stat("."); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("joined batch retained its parent descriptor: %v", err)
	}
	for i := range 2 {
		if _, err := dest.Lstat(fmt.Sprintf("payload-%d.bin", i)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("joined caller did not remove its unchanged failed leaf: %v", err)
		}
	}
	if err := cleanupRestore(root, dest, "destination", created, files, directories); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Lstat("destination"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("completed future files escaped owned cleanup: %v", err)
	}
}

func TestRestoreBatchSuccessfulArchiveOrder(t *testing.T) {
	s, p, _ := backupReceiptHistory(t, 12)
	root, err := os.OpenRoot(p.root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	manifest, err := backupInventory(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var expected []string
	for _, entry := range manifest.Files {
		if entry.Path != ".kb/config.toml" {
			expected = append(expected, entry.Path)
		}
	}
	archive := api.BackupInput{Archive: ".kb/backups/ordered.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	destination := ".kb/restores/ordered"
	var observed []string
	s.fault = func(stage, name string) error {
		if stage != "restore-file" {
			return nil
		}
		observed = append(observed, name)
		if !bytes.Equal(get(t, p, destination+"/"+name), get(t, p, name)) {
			t.Fatal("ordered boundary did not observe the exact payload")
		}
		if _, err := root.Lstat(destination + "/.kb/config.toml"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("successful payload boundary exposed configuration: %v", err)
		}
		return nil
	}
	if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: destination}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(observed, expected) {
		t.Fatalf("multi-batch boundaries differ from archive order: got %v, want %v", observed, expected)
	}
	if !bytes.Equal(get(t, p, destination+"/.kb/config.toml"), get(t, p, ".kb/config.toml")) {
		t.Fatal("ordered restore did not publish exact configuration last")
	}
}

func TestRestoreBatchFailureAndResidualCancellation(t *testing.T) {
	for _, mode := range []string{"fault-with-ahead-files", "cancel-first", "cancel-final-residual"} {
		t.Run(mode, func(t *testing.T) {
			s, p, _ := backupReceiptHistory(t, 6)
			root, err := os.OpenRoot(p.root)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = root.Close() }()
			manifest, err := backupInventory(context.Background(), root)
			if err != nil {
				t.Fatal(err)
			}
			if (len(manifest.Files)-1)%restorePendingLimit == 0 {
				put(t, p, "zz-residual.bin", []byte("synthetic residual payload"))
				manifest, err = backupInventory(context.Background(), root)
				if err != nil {
					t.Fatal(err)
				}
			}
			var payloads []string
			for _, entry := range manifest.Files {
				if entry.Path != ".kb/config.toml" {
					payloads = append(payloads, entry.Path)
				}
			}
			archive := api.BackupInput{Archive: ".kb/backups/batch.tar"}
			if _, err := s.Backup(context.Background(), archive); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fault := errors.New("stop a joined restore batch")
			target := payloads[0]
			if mode == "cancel-final-residual" {
				target = payloads[len(payloads)-1]
			}
			reached := false
			s.fault = func(stage, name string) error {
				if stage != "restore-file" {
					return nil
				}
				if reached {
					t.Fatal("restore delivered another payload callback after cancellation or fault")
				}
				if name != target {
					return nil
				}
				reached = true
				if _, err := root.Lstat(".kb/restores/failed/.kb/config.toml"); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("file boundary exposed configuration: %v", err)
				}
				if strings.HasPrefix(mode, "cancel-") {
					cancel()
					return nil
				}
				return fault
			}
			_, err = s.Restore(ctx, api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/failed"})
			wantErr := fault
			if strings.HasPrefix(mode, "cancel-") {
				wantErr = context.Canceled
			}
			if !reached || !errors.Is(err, wantErr) {
				t.Fatalf("joined boundary error was lost: reached=%v err=%v", reached, err)
			}
			if _, err := root.Lstat(".kb/restores/failed"); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed batch retained ahead/residual files: %v", err)
			}
			for _, entry := range manifest.Files {
				if api.Revision(get(t, p, entry.Path)) != entry.Revision {
					t.Fatal("failed batch changed source authority")
				}
			}
		})
	}
}

func TestRestoreBatchPreparationFailurePreservesForeignLeaf(t *testing.T) {
	s, p, _ := backupReceiptHistory(t, 8)
	root, err := os.OpenRoot(p.root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	manifest, err := backupInventory(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var payloads []string
	for _, entry := range manifest.Files {
		if entry.Path != ".kb/config.toml" {
			payloads = append(payloads, entry.Path)
		}
	}
	archive := api.BackupInput{Archive: ".kb/backups/preparation.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	archiveBytes := get(t, p, archive.Archive)
	destination := ".kb/restores/occupied"
	foreignPath := destination + "/" + payloads[6]
	foreign := []byte("unrelated externally occupied bytes")
	injected := false
	s.fault = func(stage, _ string) error {
		if stage == "restore-file" && !injected {
			injected = true
			return root.WriteFile(foreignPath, foreign, 0600)
		}
		return nil
	}
	_, err = s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: destination})
	if !injected || !errors.Is(err, os.ErrExist) {
		t.Fatalf("future exclusive creation did not report its collision: %v", err)
	}
	if got, err := root.ReadFile(foreignPath); err != nil || !bytes.Equal(got, foreign) {
		t.Fatalf("failed preparation removed external state: %v", err)
	}
	for _, name := range payloads {
		if name == payloads[6] {
			continue
		}
		if _, err := root.Lstat(destination + "/" + name); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("pending or completed owned payload escaped cleanup: %s %v", name, err)
		}
	}
	if _, err := root.Lstat(destination + "/.kb/config.toml"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed preparation published configuration: %v", err)
	}
	for _, entry := range manifest.Files {
		if api.Revision(get(t, p, entry.Path)) != entry.Revision {
			t.Fatal("preparation failure changed source authority")
		}
	}
	if !bytes.Equal(get(t, p, archive.Archive), archiveBytes) {
		t.Fatal("preparation failure changed archive authority")
	}
}

func TestRestoreBatchBoundedDescriptors(t *testing.T) {
	if root := os.Getenv("KB_TEST_RESTORE_FD_ROOT"); root != "" {
		var limit syscall.Rlimit
		if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
			t.Fatal(err)
		}
		if limit.Cur > 64 {
			limit.Cur = 64
			if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
				t.Fatal(err)
			}
		}
		p := &fakeProvider{root: root}
		p.config.Write.Enabled = true
		if _, err := New(p).Restore(context.Background(), api.RestoreInput{Archive: ".kb/backups/bounded.tar", Destination: ".kb/restores/bounded"}); err != nil {
			t.Fatal(err)
		}
		return
	}
	s, p, requests := backupReceiptHistory(t, 128)
	if _, err := s.Backup(context.Background(), api.BackupInput{Archive: ".kb/backups/bounded.tar"}); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command(executable, "-test.run=^TestRestoreBatchBoundedDescriptors$") // #nosec G204 -- Re-executes this test binary with fixed test arguments.
	child.Env = append(os.Environ(), "KB_TEST_RESTORE_FD_ROOT="+p.root)
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("restore exceeded its descriptor bound: %v %s", err, output)
	}
	copied := *p
	copied.root = filepath.Join(p.root, ".kb/restores/bounded")
	for _, request := range requests {
		name := receiptPath(request.IdempotencyKey)
		if !bytes.Equal(get(t, p, name), get(t, &copied, name)) {
			t.Fatal("descriptor-bounded restore changed a confirmed receipt")
		}
	}
	if !bytes.Equal(get(t, p, ".kb/config.toml"), get(t, &copied, ".kb/config.toml")) {
		t.Fatal("descriptor-bounded restore failed to publish exact configuration")
	}
}

func TestRestoreBatchInterruptedReceiptHistory(t *testing.T) {
	s, p, requests := backupReceiptHistory(t, 16)
	archive := api.BackupInput{Archive: ".kb/backups/interrupted.tar"}
	if _, err := s.Backup(context.Background(), archive); err != nil {
		t.Fatal(err)
	}
	observed := killAtOperationalBoundary(t, p.root, "restore-file", 1, false)
	if !strings.HasPrefix(observed, ".kb/state/receipts/") {
		t.Fatal("interruption did not observe a settled receipt batch")
	}
	if _, err := os.Stat(filepath.Join(p.root, ".kb/restores/interrupted/.kb/config.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("interrupted batch published configuration: %v", err)
	}
	_, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: ".kb/restores/interrupted"})
	errorCode(t, err, "DESTINATION_EXISTS")
	destination := ".kb/restores/retry-batch"
	if _, err := s.Restore(context.Background(), api.RestoreInput{Archive: archive.Archive, Destination: destination}); err != nil {
		t.Fatal(err)
	}
	copied := *p
	copied.root = filepath.Join(p.root, destination)
	for _, request := range requests {
		name := receiptPath(request.IdempotencyKey)
		if !bytes.Equal(get(t, p, name), get(t, &copied, name)) {
			t.Fatal("fresh retry after interrupted batch changed a receipt")
		}
	}
	if receipt, err := New(&copied).Apply(context.Background(), requests[0]); err != nil || !receipt.Replayed {
		t.Fatalf("restored interrupted-batch receipt did not replay: %v", err)
	}
}
