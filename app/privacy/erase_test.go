package privacy_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/privacy"
)

func TestDatabaseFiles(t *testing.T) {
	t.Parallel()

	got, err := privacy.DatabaseFiles("file:storage/db.sqlite?_pragma=x")
	want := []string{"storage/db.sqlite", "storage/db.sqlite-wal", "storage/db.sqlite-shm", "storage/db.sqlite-journal"}

	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("DatabaseFiles = %v, %v", got, err)
	}

	if got, _ := privacy.DatabaseFiles(""); got[0] != "storage/database.sqlite" {
		t.Errorf("the default is %v", got)
	}

	for _, url := range []string{"file:x?mode=memory&cache=shared", ":memory:", "file::memory:", "postgres://u@h/db", "mysql://u@h/db"} {
		if _, err := privacy.DatabaseFiles(url); !errors.Is(err, privacy.ErrNoDatabaseFile) {
			t.Errorf("%q: err = %v, want ErrNoDatabaseFile", url, err)
		}
	}
}

// TestPersonalErase: the database closes and no file of the old data stays.
func TestPersonalErase(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "app.sqlite")

	app := newApp(t, map[string]string{"APP_EDITION": "personal", "DATABASE_URL": "file:" + filepath.ToSlash(path)})
	newWorld(t, app, "ana@example.com")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	// The write-ahead log is live while the database is open.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the database file is not at %s: %v", path, err)
	}

	if _, err := os.Stat(path + "-wal"); err != nil {
		t.Fatalf("the database is not in WAL mode here: %v", err)
	}

	if err := privacy.EraseLocalData(db, "file:"+filepath.ToSlash(path)); err != nil {
		t.Fatal(err)
	}

	left, err := filepath.Glob(path + "*")
	if err != nil || len(left) != 0 {
		t.Fatalf("files of the old data remain: %v %v", left, err)
	}

	if err := db.Ping(t.Context()); err == nil {
		t.Error("the database is still open")
	}

	// Erasing again, or an unknown file, is not an error: there is nothing left.
	if err := privacy.EraseLocalData(db, "file:"+filepath.ToSlash(path)); err != nil {
		t.Errorf("erasing twice: %v", err)
	}
}

func TestEraseIsPersonalOnly(t *testing.T) {
	t.Parallel()

	app := newApp(t) // saas
	svc := newService(t, app)

	if got := statusOf(svc.ScheduleErase(privacy.EraseWord)); got != 404 {
		t.Fatalf("saas ScheduleErase status = %d, want 404", got)
	}
}

func TestEraseNeedsTheWord(t *testing.T) {
	t.Parallel()

	app := newApp(t, map[string]string{"APP_EDITION": "personal"})
	svc := newService(t, app)
	svc.EraseDelay = 1<<62 - 1 // never fires in this test

	for _, word := range []string{"", "erase", "ERASE ", "yes"} {
		if got := statusOf(svc.ScheduleErase(word)); got != 422 {
			t.Errorf("%q: status %d, want 422", word, got)
		}
	}
}
