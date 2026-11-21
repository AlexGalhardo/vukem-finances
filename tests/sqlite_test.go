package tests

import (
	"path/filepath"
	"testing"
)

// TestSQLiteIsInWALMode runs the app on a SQLite file, the way the container
// does (an in-memory database has no WAL), and checks the pragmas the
// framework driver sets: WAL for concurrent readers and a busy timeout for
// the single writer.
func TestSQLiteIsInWALMode(t *testing.T) {
	t.Parallel()

	path := filepath.ToSlash(filepath.Join(t.TempDir(), "database.sqlite"))
	app := newPersonalApp(t, map[string]string{"DATABASE_URL": "file:" + path})

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var mode string
	if err := db.SQL().QueryRowContext(t.Context(), "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}

	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}

	var timeout int
	if err := db.SQL().QueryRowContext(t.Context(), "PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatal(err)
	}

	if timeout < 5000 {
		t.Fatalf("busy_timeout = %d, want at least 5000 ms", timeout)
	}
}
