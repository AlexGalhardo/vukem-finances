package main

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsTargetsAndTheirPackage(t *testing.T) {
	root := t.TempDir()

	write(t, filepath.Join(root, "a"), "x_test.go", "package a\n\nfunc FuzzOne(f *testing.F) {}\nfunc TestNotFuzz(t *testing.T) {}\nfunc fuzzHelper(f *testing.F) {}\n")
	write(t, filepath.Join(root, "b", "c"), "y_test.go", "package c\n\nfunc FuzzTwo(f *testing.F) {\n}\n")
	write(t, filepath.Join(root, "node_modules", "z"), "z_test.go", "package z\n\nfunc FuzzSkipped(f *testing.F) {}\n")
	write(t, filepath.Join(root, "a"), "helper.go", "package a\n\nfunc FuzzNotATestFile(f *testing.F) {}\n")

	got, err := scan(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got["FuzzOne"] != "./a" || got["FuzzTwo"] != "./b/c" {
		t.Fatalf("scan = %v, want FuzzOne in a and FuzzTwo in b/c only", got)
	}
}

func TestScanRefusesAnAmbiguousName(t *testing.T) {
	root := t.TempDir()

	write(t, filepath.Join(root, "a"), "x_test.go", "package a\n\nfunc FuzzSame(f *testing.F) {}\n")
	write(t, filepath.Join(root, "b"), "y_test.go", "package b\n\nfunc FuzzSame(f *testing.F) {}\n")

	if _, err := scan(root); err == nil {
		t.Fatal("two targets with one name were accepted")
	}
}
