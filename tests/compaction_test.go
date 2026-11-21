package tests

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/schedule"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
	"github.com/AlexGalhardo/vukem-finances/routes"
)

// TestCompactionJob deletes an account through the service, lets 91 days
// pass, and runs the nightly job: the tombstone is purged, a younger one is
// kept, and a client that is behind is told to resync.
func TestCompactionJob(t *testing.T) {
	t.Parallel()

	clock := vukemtest.NewFakeClock(time.Now().UTC())
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:   map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"},
		Clock: clock.Now,
	})
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

	old, err := fin.CreateAccount(ctx, "Old", "cash", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := fin.DeleteAccount(ctx, old.ID); err != nil {
		t.Fatal(err)
	}

	clock.Advance(91 * 24 * time.Hour)

	recent, err := fin.CreateAccount(ctx, "Recent", "cash", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := fin.DeleteAccount(ctx, recent.ID); err != nil {
		t.Fatal(err)
	}

	if err := jobs.HandleChangesCompact(app.App)(t.Context(), jobs.ChangesCompact{}); err != nil {
		t.Fatal(err)
	}

	if n := trashedCount[models.Account](t, app, ana); n != 1 {
		t.Fatalf("tombstones left = %d, want only the recent one", n)
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := changes.Since(t.Context(), db, ana.Workspace.ID, 0, 10); err == nil {
		t.Fatal("a client at cursor 0 must be told to resync after a compaction")
	}
}

func TestCompactionIsScheduledDaily(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	s := schedule.New(schedule.Options{})
	routes.Schedule(s, app.App)

	for _, task := range s.Tasks() {
		if task.Name() == "job:changes_compact" {
			if task.Expression() != "30 3 * * *" {
				t.Fatalf("changes_compact runs at %q, want daily at 03:30", task.Expression())
			}

			return
		}
	}

	t.Fatal("changes_compact is not scheduled")
}

func TestCompactionRetentionIsValidatedAtBoot(t *testing.T) {
	t.Parallel()

	for _, v := range []string{"7", "abc", "-1", "29"} {
		if _, err := vukemtest.Build(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"TOMBSTONE_RETENTION_DAYS": v}}); err == nil {
			t.Errorf("TOMBSTONE_RETENTION_DAYS=%s booted, want an error", v)
		}
	}

	if _, err := vukemtest.Build(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"TOMBSTONE_RETENTION_DAYS": "30"}}); err != nil {
		t.Errorf("TOMBSTONE_RETENTION_DAYS=30 = %v, want it accepted", err)
	}
}
