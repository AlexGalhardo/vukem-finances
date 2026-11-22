package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/AlexGalhardo/vukem/database"
)

// storedResult returns the result kept for op_id, if a push already ran it.
func storedResult(ctx context.Context, tx *database.Tx, ws int64, opID string) (Result, bool, error) {
	var raw string

	err := tx.QueryRowContext(ctx, "SELECT result FROM sync_ops WHERE workspace_id = ? AND op_id = ?", ws, opID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, false, nil
	}

	if err != nil {
		return Result{}, false, err
	}

	var res Result
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return Result{}, false, fmt.Errorf("sync: stored result of %s: %w", opID, err)
	}

	return res, true, nil
}

// storeResult keeps the result of a mutation so a retry returns it unchanged.
// The unique (workspace_id, op_id) index makes two concurrent pushes of one
// op_id impossible: the second fails its transaction and the client retries.
func storeResult(ctx context.Context, tx *database.Tx, ws int64, device string, res Result, now time.Time) error {
	raw, err := json.Marshal(res)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO sync_ops (workspace_id, op_id, device_id, result, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		ws, res.OpID, device, string(raw), now, now)

	return err
}

// storeConflicts records the values that lost a merge for 30 days.
func storeConflicts(ctx context.Context, tx *database.Tx, ws int64, entity, uid, device string, conflicts []ConflictOut, now time.Time) error {
	for _, c := range conflicts {
		lost, err := json.Marshal(c.LostValue)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO sync_conflicts (workspace_id, entity, uid, field, kind, lost_value, lost_by_device, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ws, entity, uid, c.Field, c.Kind, string(lost), c.LostBy, now, now)
		if err != nil {
			return err
		}
	}

	return nil
}

// PruneOps deletes the stored results older than OpRetention and returns how
// many it removed.
func (s *Service) PruneOps(ctx context.Context, now time.Time) (int64, error) {
	res, err := s.DB.SQL().ExecContext(ctx, "DELETE FROM sync_ops WHERE created_at < ?", now.UTC().Add(-OpRetention))
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}
