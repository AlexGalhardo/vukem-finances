package main

import (
	"fmt"
	"os"
	"sync"
)

// rotatingFile is the log sink of the desktop app: it writes to path and, when
// the file passes maxSize, moves it to path.1 (path.1 to path.2, and so on up
// to keep files) so the logs of a long-lived install stay bounded.
type rotatingFile struct {
	mu      sync.Mutex
	path    string
	maxSize int64
	keep    int
	f       *os.File
	size    int64
}

func newRotatingFile(path string, maxSize int64, keep int) (*rotatingFile, error) {
	r := &rotatingFile{path: path, maxSize: maxSize, keep: keep}
	if err := r.open(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *rotatingFile) open() error {
	f, err := os.OpenFile(r.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}

	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}

	r.f, r.size = f, st.Size()

	return nil
}

func (r *rotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.f == nil {
		return 0, os.ErrClosed
	}

	if r.size > 0 && r.size+int64(len(p)) > r.maxSize {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := r.f.Write(p)
	r.size += int64(n)

	return n, err
}

func (r *rotatingFile) rotate() error {
	if err := r.f.Close(); err != nil {
		return err
	}

	r.f = nil

	for i := r.keep; i >= 1; i-- {
		from := r.path
		if i > 1 {
			from = fmt.Sprintf("%s.%d", r.path, i-1)
		}

		to := fmt.Sprintf("%s.%d", r.path, i)

		if i == r.keep {
			_ = os.Remove(to)
		}

		if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return r.open()
}

// Close closes the current file.
func (r *rotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.f == nil {
		return nil
	}

	err := r.f.Close()
	r.f = nil

	return err
}
