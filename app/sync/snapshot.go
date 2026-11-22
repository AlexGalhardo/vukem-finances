package sync

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem/database"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

// SnapshotPage is a page of the live rows of one entity.
type SnapshotPage struct {
	SnapshotID     string           `json:"snapshot_id"`
	SnapshotCursor int64            `json:"snapshot_cursor"`
	Entity         string           `json:"entity"`
	Rows           []map[string]any `json:"rows"`
	NextAfter      string           `json:"next_after"`
	HasMore        bool             `json:"has_more"`
}

// makeSnapshotID is stateless: it carries the workspace and the cursor fixed by
// the first call. A forged id can only make a client pull more or less of its
// own workspace, never read another one.
func makeSnapshotID(ws, cursor int64) string { return fmt.Sprintf("s1.%d.%d", ws, cursor) }

func parseSnapshotID(id string, ws int64) (int64, bool) {
	parts := strings.Split(id, ".")
	if len(parts) != 3 || parts[0] != "s1" {
		return 0, false
	}

	w, err1 := strconv.ParseInt(parts[1], 10, 64)
	c, err2 := strconv.ParseInt(parts[2], 10, 64)

	return c, err1 == nil && err2 == nil && w == ws && c >= 0
}

// Snapshot returns live rows of one entity in uid order after a uid. The first
// call (no snapshotID) fixes snapshot_cursor, the newest seq at that moment,
// read in the same transaction as the rows; the client sends the id back on
// every later call and, once every entity is downloaded, pulls from
// snapshot_cursor. A change made while the snapshot runs is then applied twice
// at worst, because a pull is idempotent.
func (s *Service) Snapshot(ctx context.Context, a Actor, entity, after, snapshotID string, limit int) (SnapshotPage, error) {
	ent, ok := changes.Lookup(entity)
	if !ok {
		return SnapshotPage{}, fail(http.StatusBadRequest, "unknown_entity", "That entity is not synced: see GET /sync/info.")
	}

	switch {
	case limit <= 0:
		limit = DefaultSnapshotRow
	case limit > MaxSnapshotRows:
		limit = MaxSnapshotRows
	}

	var (
		cursor int64
		have   bool
	)

	if snapshotID != "" {
		if cursor, have = parseSnapshotID(snapshotID, a.WorkspaceID); !have {
			return SnapshotPage{}, fail(http.StatusBadRequest, CodeInvalidSnapshot, "That snapshot_id is not one this server made for this workspace.")
		}
	}

	page := SnapshotPage{Entity: ent.Name, Rows: []map[string]any{}}

	err := s.DB.TxWith(ctx, database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		if !have {
			c, err := changes.Cursor(ctx, tx, a.WorkspaceID)
			if err != nil {
				return err
			}

			cursor = c
		}

		where, args := "t.deleted_at IS NULL", []any(nil)
		if after != "" {
			where, args = where+" AND t.uid > ?", []any{after}
		}

		rows, err := selectRows(ctx, tx, ent, a.WorkspaceID, where, args, "t.uid", limit+1)
		if err != nil {
			return err
		}

		if len(rows) > limit {
			page.HasMore, rows = true, rows[:limit]
		}

		for _, r := range rows {
			page.Rows = append(page.Rows, r.Wire())
			page.NextAfter = r.UID
		}

		return nil
	})
	if err != nil {
		return SnapshotPage{}, err
	}

	page.SnapshotCursor, page.SnapshotID = cursor, snapshotID
	if !have {
		page.SnapshotID = makeSnapshotID(a.WorkspaceID, cursor)
	}

	return page, nil
}
