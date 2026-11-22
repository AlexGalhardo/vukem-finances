package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingFileKeepsBoundedFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")

	r, err := newRotatingFile(path, 100, 2)
	if err != nil {
		t.Fatal(err)
	}

	for i := range 20 {
		if _, err := r.Write([]byte(strings.Repeat(string(rune('a'+i)), 60) + "\n")); err != nil {
			t.Fatal(err)
		}
	}

	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 3 {
		t.Fatalf("got %d files, want app.log plus 2 rotated", len(entries))
	}

	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), strings.Repeat("t", 60)) {
		t.Errorf("app.log = %q, want the newest line", got)
	}
}
