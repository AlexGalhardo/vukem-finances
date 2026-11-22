package jobs

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"

	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
)

// SyncOpsPrune deletes the stored results of pushed operations older than 30
// days (docs/specs/sync-protocol.md): a client that retries an op_id that late
// has lost the answer for good. It is not tied to a workspace.
type SyncOpsPrune struct{}

// Kind names the job's handler; keep it stable across deploys.
func (SyncOpsPrune) Kind() string { return "sync_ops_prune" }

// HandleSyncOpsPrune runs the pruning.
func HandleSyncOpsPrune(app *vukem.App) func(ctx context.Context, j SyncOpsPrune) error {
	return func(ctx context.Context, _ SyncOpsPrune) error {
		db, err := app.DB()
		if err != nil {
			return err
		}

		n, err := (&syncproto.Service{DB: db}).PruneOps(ctx, app.Now())
		if err != nil {
			return err
		}

		app.Log.InfoContext(ctx, "sync ops pruned", "removed", n)

		return nil
	}
}
