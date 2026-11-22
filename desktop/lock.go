package main

import (
	"errors"
	"fmt"
)

// ErrAlreadyRunning means another process holds the lock of the data directory.
var ErrAlreadyRunning = errors.New("vukem-finances is already running with this data directory")

// Lock is the single-instance lock of a data directory: two processes must
// never write one SQLite file. The operating system drops it when the process
// dies, so a crash leaves no stale lock.
type Lock struct{ release func() error }

// AcquireLock takes the lock on path or fails with ErrAlreadyRunning.
func AcquireLock(path string) (*Lock, error) {
	release, err := lockFile(path)
	if errors.Is(err, ErrAlreadyRunning) {
		return nil, err
	}

	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}

	return &Lock{release: release}, nil
}

// Release frees the lock. It is safe to call more than once.
func (l *Lock) Release() error {
	if l == nil || l.release == nil {
		return nil
	}

	release := l.release
	l.release = nil

	return release()
}
