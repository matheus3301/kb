package write

import (
	"context"
	"errors"
	"os"
	"path"
	"sync"
	"syscall"
)

const restorePendingLimit = 4

// Preparation and ownership bookkeeping stay on the caller. Pending work holds
// descriptors, never payload buffers; every flush finishes before caller hooks.
type restoreBatch struct {
	ctx         context.Context
	dest        *os.Root
	directories *[]string
	files       *[]restoredFile
	boundary    func(string, string) error
	pending     [restorePendingLimit]pendingRestoreFile
	count       int
	workers     sync.WaitGroup
}

type pendingRestoreFile struct {
	entry    backupEntry
	parent   *os.Root
	file     *os.File
	identity os.FileInfo
	writeErr error
	record   bool
	err      error
}

func (b *restoreBatch) add(entry backupEntry, data []byte) error {
	if err := b.ctx.Err(); err != nil {
		return err
	}
	prepared, err := prepareRestoreFile(b.dest, entry, data, b.directories)
	if err != nil {
		return err
	}
	b.pending[b.count] = prepared
	b.count++
	if err := b.ctx.Err(); err != nil {
		return err
	}
	if b.count == len(b.pending) {
		return b.flush(true)
	}
	return nil
}

func (b *restoreBatch) flush(deliver bool) error {
	if b.count == 0 {
		return nil
	}
	n := b.count
	b.count = 0
	defer clear(b.pending[:n])
	for i := range n {
		if n == 1 || b.ctx.Err() != nil {
			b.pending[i].flush(b.dest)
			continue
		}
		b.workers.Add(1)
		go func(p *pendingRestoreFile) {
			defer b.workers.Done()
			p.flush(b.dest)
		}(&b.pending[i])
	}
	b.workers.Wait()
	var err error
	// Register every completed leaf before any error or hook can start cleanup.
	for i := range n {
		p := &b.pending[i]
		if p.record {
			*b.files = append(*b.files, restoredFile{entry: p.entry, identity: p.identity})
		}
	}
	for i := range n {
		err = errors.Join(err, b.pending[i].finish())
	}
	if err != nil {
		return errors.Join(err, b.ctx.Err())
	}
	if deliver {
		for i := range n {
			if err := b.ctx.Err(); err != nil {
				return err
			}
			if err := b.boundary("restore-file", b.pending[i].entry.Path); err != nil {
				return err
			}
		}
	}
	return b.ctx.Err()
}

func prepareRestoreFile(dest *os.Root, entry backupEntry, data []byte, directories *[]string) (pending pendingRestoreFile, err error) {
	parent, err := backupParent(dest, entry.Path, directories)
	if err != nil {
		return pending, err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, parent.Close())
		}
	}()
	name := path.Base(entry.Path)
	f, err := parent.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return pending, err
	}
	created, err := f.Stat()
	if err != nil {
		return pending, errors.Join(err, f.Close(), parent.Remove(name))
	}
	_, writeErr := f.Write(data)
	return pendingRestoreFile{entry: entry, parent: parent, file: f, identity: created, writeErr: writeErr}, nil
}

func (p *pendingRestoreFile) flush(dest *os.Root) {
	// Failed-leaf cleanup and parent closure belong to the caller after joining.
	if err := errors.Join(p.writeErr, p.file.Sync(), p.file.Close()); err != nil {
		p.err = err
		return
	}
	p.record = true
	p.err = errors.Join(syncDirectory(p.parent, "."), backupParentIdentity(dest, p.entry.Path, p.parent))
}

func (p *pendingRestoreFile) finish() error {
	var cleanupErr error
	if !p.record {
		cleanupErr = removeOwnedRestoreLeaf(p.parent, path.Base(p.entry.Path), p.identity)
	}
	return errors.Join(p.err, cleanupErr, p.parent.Close())
}
