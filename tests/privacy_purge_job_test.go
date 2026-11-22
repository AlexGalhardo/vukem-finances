package tests

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/schedule"

	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/routes"
)

func TestPrivacyPurgeJobErasesADueWorkspaceAndSparesTheRest(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	past := time.Now().UTC().Add(-time.Hour)
	if _, err := db.ExecContext(t.Context(),
		"UPDATE workspaces SET state = 'pending_deletion', purge_at = ?, deletion_requested_at = ? WHERE id = ?",
		past, past.Add(-720*time.Hour), ana.Workspace.ID); err != nil {
		t.Fatal(err)
	}

	for range 2 { // the job is idempotent
		if err := jobs.HandlePrivacyPurge(app.App)(t.Context(), jobs.PrivacyPurge{}); err != nil {
			t.Fatal(err)
		}
	}

	var gone, kept int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM accounts WHERE workspace_id = ?", ana.Workspace.ID).Scan(&gone); err != nil {
		t.Fatal(err)
	}

	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM accounts WHERE workspace_id = ?", bob.Workspace.ID).Scan(&kept); err != nil {
		t.Fatal(err)
	}

	if gone != 0 || kept == 0 {
		t.Fatalf("accounts of the purged workspace = %d, of the other = %d", gone, kept)
	}
}

func TestPrivacyPurgeIsScheduledDaily(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	s := schedule.New(schedule.Options{})

	routes.Schedule(s, app.App)

	for _, task := range s.Tasks() {
		if task.Name() == "job:privacy_purge" {
			if task.Expression() != "40 4 * * *" {
				t.Fatalf("privacy_purge runs at %q, want daily at 04:40", task.Expression())
			}

			return
		}
	}

	t.Fatal("privacy_purge is not scheduled")
}
