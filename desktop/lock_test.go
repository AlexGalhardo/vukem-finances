package main

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSingleInstanceSecondLockFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")

	first, err := AcquireLock(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := AcquireLock(path); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second lock = %v, want ErrAlreadyRunning", err)
	}

	if err := first.Release(); err != nil {
		t.Fatal(err)
	}

	// Released: the next launch can start, and releasing twice is harmless.
	again, err := AcquireLock(path)
	if err != nil {
		t.Fatalf("lock after release = %v", err)
	}

	if err := again.Release(); err != nil || again.Release() != nil {
		t.Errorf("release = %v", err)
	}
}
