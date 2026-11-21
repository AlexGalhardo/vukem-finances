package tests

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/AlexGalhardo/vukem/database"
)

// tableCounts returns the row count of every user table of db.
func tableCounts(ctx context.Context, t *testing.T, db *sql.DB) map[string]int {
	t.Helper()

	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		t.Fatal(err)
	}

	var names []string

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}

		names = append(names, name)
	}

	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}

	counts := make(map[string]int, len(names))

	for _, name := range names {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "`+name+`"`).Scan(&n); err != nil {
			t.Fatal(err)
		}

		counts[name] = n
	}

	return counts
}

// TestDBBackupRestoreRoundTrip drives the framework db:backup command the way
// the deploy guide documents it: a snapshot with -to, then -restore -from -to
// into another directory, and the restored file holds the same rows.
func TestDBBackupRestoreRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	live := filepath.ToSlash(filepath.Join(dir, "storage", "database.sqlite"))

	if err := os.MkdirAll(filepath.Dir(live), 0o750); err != nil {
		t.Fatal(err)
	}

	app := newSaaSApp(t, map[string]string{"DATABASE_URL": "file:" + live})
	signUp(t, app, "Ana", "ana@example.com")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	want := tableCounts(t.Context(), t, db.SQL())
	if want["users"] == 0 || want["categories"] == 0 {
		t.Fatalf("the fixture is empty: %v", want)
	}

	snapshot := filepath.Join(dir, "storage", "backups", "snapshot.sqlite")
	restored := filepath.Join(dir, "storage", "restore", "database.sqlite")

	for _, d := range []string{filepath.Dir(snapshot), filepath.Dir(restored)} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	if err := app.Call(t.Context(), "db:backup", []string{"-to", snapshot}); err != nil {
		t.Fatal(err)
	}

	if err := app.Call(t.Context(), "db:backup", []string{"-to", snapshot}); err == nil {
		t.Fatal("a second snapshot over the same path must be refused")
	}

	if err := app.Call(t.Context(), "db:backup", []string{"-restore", "-from", snapshot, "-to", restored}); err != nil {
		t.Fatal(err)
	}

	back, err := database.Open(t.Context(), "file:"+filepath.ToSlash(restored))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = back.Close() })

	got := tableCounts(t.Context(), t, back.SQL())

	if len(got) != len(want) {
		t.Fatalf("restored tables = %v, want %v", got, want)
	}

	for name, n := range want {
		if got[name] != n {
			t.Errorf("table %s: restored %d rows, want %d", name, got[name], n)
		}
	}
}
