package privacy_test

import (
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/privacy"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func TestCancelIsRefusedOnceTheGracePeriodEnded(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	past := time.Now().UTC().Add(-time.Hour)

	if _, err := db.ExecContext(t.Context(),
		"UPDATE workspaces SET state = ?, purge_at = ?, deletion_requested_at = ?, deletion_requested_by = ? WHERE id = ?",
		models.StatePendingDeletion, past, past, w.owner.ID, w.space.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(t.Context(), "UPDATE users SET purge_at = ?, deletion_requested_at = ? WHERE id = ?",
		past, past, w.owner.ID); err != nil {
		t.Fatal(err)
	}

	ctx := workspaces.WithTenant(t.Context(), w.space.ID)
	if got := statusOf(w.svc.CancelWorkspaceDeletion(ctx, w.owner)); got != http.StatusConflict {
		t.Fatalf("cancel of a due workspace = %d, want 409", got)
	}

	if got := statusOf(w.svc.CancelAccountDeletion(t.Context(), w.owner)); got != http.StatusConflict {
		t.Fatalf("cancel of a due account = %d, want 409", got)
	}

	var state string
	if err := db.QueryRowContext(t.Context(), "SELECT state FROM workspaces WHERE id = ?", w.space.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}

	if state != models.StatePendingDeletion {
		t.Fatalf("state = %q, want it still pending", state)
	}
}

func TestFailedEraseEndsTheProcessOnce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "app.sqlite")

	app := newApp(t, map[string]string{"APP_EDITION": "personal", "DATABASE_URL": "file:" + filepath.ToSlash(path)})
	w := newWorld(t, app, "ana@example.com")

	// A directory with a file in it cannot be removed, so the erase fails after closing the database.
	blocked := filepath.Join(dir, "blocked")
	if err := os.MkdirAll(filepath.Join(blocked, "child"), 0o750); err != nil {
		t.Fatal(err)
	}

	var restarts atomic.Int32

	done := make(chan struct{}, 2)
	w.svc.DatabaseURL = "file:" + filepath.ToSlash(blocked)
	w.svc.EraseDelay = 10 * time.Millisecond
	w.svc.Restart = func() {
		restarts.Add(1)
		done <- struct{}{}
	}

	for range 2 {
		if err := w.svc.ScheduleErase(privacy.EraseWord); err != nil {
			t.Fatal(err)
		}
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a failed erase left the process running with its database closed")
	}

	time.Sleep(100 * time.Millisecond)

	if got := restarts.Load(); got != 1 {
		t.Fatalf("restarts = %d, want 1", got)
	}
}
