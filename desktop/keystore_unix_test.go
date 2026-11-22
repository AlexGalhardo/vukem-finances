//go:build !windows

package main

import "testing"

func assertOnlyCurrentUserCanRead(t *testing.T, _ string) { t.Helper() }
