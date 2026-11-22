//go:build windows

package main

import (
	"errors"

	"golang.org/x/sys/windows"
)

// lockFile opens the file with no sharing: a second open fails with a sharing
// violation for as long as this handle lives.
func lockFile(path string) (func() error, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	h, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		if errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, ErrAlreadyRunning
		}

		return nil, err
	}

	return func() error { return windows.CloseHandle(h) }, nil
}
