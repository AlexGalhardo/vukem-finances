package main

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupZip(t *testing.T) {
	paths, err := ResolvePaths(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}

	s := startTestServer(t, paths, startOptions{})

	live, err := s.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := live.SQL().ExecContext(t.Context(), "INSERT INTO users (name, email, password, created_at, updated_at) VALUES ('Ana', 'ana@example.com', 'x', '2026-01-01', '2026-01-01')"); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := WriteBackupZip(t.Context(), live.SQL(), paths, &buf); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	files := map[string][]byte{}

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}

		files[f.Name], _ = io.ReadAll(rc)
		_ = rc.Close()
	}

	if len(files) != 2 || files["database.sqlite"] == nil || files["app.key"] == nil {
		t.Fatalf("zip entries = %d, want database.sqlite and app.key", len(files))
	}

	wantKey, _ := os.ReadFile(paths.Key)
	if !bytes.Equal(files["app.key"], wantKey) {
		t.Error("app.key in the zip differs from the key on disk")
	}

	// The database in the zip opens and holds what the live one held.
	out := filepath.Join(t.TempDir(), "restored.sqlite")
	if err := os.WriteFile(out, files["database.sqlite"], 0o600); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", out)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var email string
	if err := db.QueryRowContext(t.Context(), "SELECT email FROM users").Scan(&email); err != nil || email != "ana@example.com" {
		t.Fatalf("restored database: email = %q, err = %v", email, err)
	}

	// No scratch directory is left in the data folder.
	if m, _ := filepath.Glob(filepath.Join(paths.Data, "backup-*")); len(m) != 0 {
		t.Errorf("left behind: %v", m)
	}
}
