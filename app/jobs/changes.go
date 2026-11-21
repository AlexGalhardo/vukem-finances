package jobs

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// ChangesCompact purges tombstones older than TOMBSTONE_RETENTION_DAYS and
// compacts the change log (docs/specs/sync-data-model.md). It is not tied to
// a workspace: one run covers every workspace.
type ChangesCompact struct{}

// Kind names the job's handler; keep it stable across deploys.
func (ChangesCompact) Kind() string { return "changes_compact" }

// HandleChangesCompact runs the compaction with the configured retention.
func HandleChangesCompact(app *vukem.App) func(ctx context.Context, j ChangesCompact) error {
	return func(ctx context.Context, _ ChangesCompact) error {
		retention, err := config.TombstoneRetention(app.Source)
		if err != nil {
			return err
		}

		db, err := app.DB()
		if err != nil {
			return err
		}

		stats, err := changes.Compact(ctx, db, app.Now(), retention)
		if err != nil {
			return err
		}

		app.Log.InfoContext(ctx, "changes compacted", "purged", stats.Purged, "changes", stats.Changes, "conflicts", stats.Conflicts)

		return nil
	}
}
