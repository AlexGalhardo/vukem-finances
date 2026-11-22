package tests

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/schedule"

	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/routes"
)

func TestSyncOpsPruned(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")

	fresh := d.one(mut("tag", newULID(), "upsert", 0, map[string]any{"name": "fresh"}))
	stale := d.one(mut("tag", newULID(), "upsert", 0, map[string]any{"name": "stale"}))

	db, _ := f.app.DB()
	old := time.Now().UTC().Add(-31 * 24 * time.Hour)

	if _, err := db.SQL().ExecContext(t.Context(), "UPDATE sync_ops SET created_at = ? WHERE op_id = ?", old, stale.OpID); err != nil {
		t.Fatal(err)
	}

	if err := jobs.HandleSyncOpsPrune(f.app.App)(t.Context(), jobs.SyncOpsPrune{}); err != nil {
		t.Fatal(err)
	}

	count := func(op string) int {
		var n int
		if err := db.SQL().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM sync_ops WHERE op_id = ?", op).Scan(&n); err != nil {
			t.Fatal(err)
		}

		return n
	}

	if count(stale.OpID) != 0 || count(fresh.OpID) != 1 {
		t.Errorf("after the prune: stale %d, fresh %d; want 0 and 1", count(stale.OpID), count(fresh.OpID))
	}

	s := schedule.New(schedule.Options{})
	routes.Schedule(s, f.app.App)

	for _, task := range s.Tasks() {
		if task.Name() == "job:sync_ops_prune" {
			return
		}
	}

	t.Error("sync_ops_prune is not scheduled")
}
