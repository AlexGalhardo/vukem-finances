//go:build !windows

package main

import "os"

// restrictToUser makes the file readable by its owner only, whatever the umask.
func restrictToUser(path string) error { return os.Chmod(path, 0o600) }
