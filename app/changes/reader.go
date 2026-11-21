package changes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// ErrResyncRequired is returned by Since when the cursor is older than what
// compaction kept: the client must pull a full Snapshot.
var ErrResyncRequired = errors.New("changes: cursor is behind the compacted log; pull a snapshot")

// Limits of a page of Since.
const (
	DefaultLimit = 200
	MaxLimit     = 1000
)

// Entry is one change of the log as a client reads it.
type Entry struct {
	Seq        int64     `json:"seq"`
	Entity     string    `json:"entity"`
	UID        string    `json:"uid"`
	Op         string    `json:"op"`
	RowVersion int64     `json:"row_version"`
	Fields     []string  `json:"fields"`
	At         time.Time `json:"at"`
}

// Page is a slice of the log after a cursor. Next is the cursor to send back;
// it equals the cursor given when there was nothing new.
type Page struct {
	Entries []Entry `json:"changes"`
	Next    int64   `json:"next"`
	HasMore bool    `json:"has_more"`
}

// Since returns the changes of one workspace with seq greater than cursor,
// oldest first, at most limit of them (DefaultLimit when limit is zero or
// negative, MaxLimit at most). The log of another workspace is never read:
// the workspace id is part of every query. A cursor below the workspace's
// compacted_through_seq gets ErrResyncRequired.
func Since(ctx context.Context, ex database.Executor, workspaceID, cursor int64, limit int) (Page, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}

	limit = min(limit, MaxLimit)

	compacted, err := compactedThrough(ctx, ex, workspaceID)
	if err != nil {
		return Page{}, err
	}

	if cursor < compacted {
		return Page{Next: cursor}, ErrResyncRequired
	}

	rows, err := orm.QueryOf[models.Change](ex).
		Where("workspace_id", workspaceID).Where("seq", ">", cursor).
		OrderBy("seq", orm.Asc).Limit(limit + 1).Get(ctx)
	if err != nil {
		return Page{}, err
	}

	page := Page{Entries: make([]Entry, 0, len(rows)), Next: cursor}

	if len(rows) > limit {
		page.HasMore = true
		rows = rows[:limit]
	}

	for _, r := range rows {
		var fields []string
		if err := json.Unmarshal([]byte(r.Fields), &fields); err != nil {
			return Page{}, fmt.Errorf("changes: change %d has bad fields: %w", r.Seq, err)
		}

		page.Entries = append(page.Entries, Entry{
			Seq: r.Seq, Entity: r.Entity, UID: r.UID, Op: r.Op, RowVersion: r.RowVersion, Fields: fields, At: r.CreatedAt,
		})
		page.Next = r.Seq
	}

	return page, nil
}

// Cursor is the position of a client that has seen everything: the newest
// change of the workspace, or the compacted seq when compaction removed the
// newest ones. Below that, Since would send a fresh snapshot back to resync
// forever.
func Cursor(ctx context.Context, ex database.Executor, workspaceID int64) (int64, error) {
	compacted, err := compactedThrough(ctx, ex, workspaceID)
	if err != nil {
		return 0, err
	}

	seq, err := orm.QueryOf[models.Change](ex).Where("workspace_id", workspaceID).Max[int64](ctx, "seq")
	if errors.Is(err, orm.ErrNotFound) {
		return compacted, nil
	}

	if err != nil {
		return 0, err
	}

	return max(seq, compacted), nil
}

func compactedThrough(ctx context.Context, ex database.Executor, workspaceID int64) (int64, error) {
	st, err := orm.QueryOf[models.SyncState](ex).Where("workspace_id", workspaceID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return 0, nil
	}

	return st.CompactedThroughSeq, err
}

// SnapshotResult is the live state of a workspace plus the cursor to continue from.
// Rows are keyed by entity name; each row has "uid", "version", the writable
// fields and "<entity>_uid" for every parent (never the integer ids).
type SnapshotResult struct {
	Cursor int64                       `json:"cursor"`
	Rows   map[string][]map[string]any `json:"rows"`
}

// Snapshot reads the workspace's live rows of every registered entity, with
// the cursor taken in the same transaction, so Since(cursor) replays exactly
// what happened after the rows were read. Deleted rows are left out: a client
// that starts from a snapshot has nothing to delete.
func Snapshot(ctx context.Context, db *database.DB, workspaceID int64) (SnapshotResult, error) {
	snap := SnapshotResult{Rows: map[string][]map[string]any{}}

	err := db.Tx(ctx, func(tx *database.Tx) error {
		cursor, err := Cursor(ctx, tx, workspaceID)
		if err != nil {
			return err
		}

		snap.Cursor = cursor

		for _, e := range Entities() {
			rows, err := entityRows(ctx, tx, e, workspaceID)
			if err != nil {
				return err
			}

			snap.Rows[e.Name] = rows
		}

		return nil
	})

	return snap, err
}

func entityRows(ctx context.Context, tx *database.Tx, e Entity, workspaceID int64) ([]map[string]any, error) {
	cols := []string{"t.uid AS uid", "t.version AS version"}
	joins := ""
	names := []string{"uid", "version"}

	refs := map[string]Ref{}
	for _, r := range e.Refs {
		refs[r.Column] = r
	}

	for i, f := range e.Fields {
		if r, ok := refs[f]; ok {
			parent, ok := Lookup(r.Entity)
			if !ok {
				return nil, fmt.Errorf("changes: %s refs unknown entity %s", e.Name, r.Entity)
			}

			alias := fmt.Sprintf("p%d", i)
			cols = append(cols, alias+".uid AS "+r.Entity+"_uid")
			joins += fmt.Sprintf(" LEFT JOIN %s %s ON %s.id = t.%s", parent.Table, alias, alias, f)

			names = append(names, r.Entity+"_uid")

			continue
		}

		cols = append(cols, "t."+f+" AS "+f)
		names = append(names, f)
	}

	query := "SELECT " + strings.Join(cols, ", ") + " FROM " + e.Table + " t" + joins +
		" WHERE t.workspace_id = ? AND t.deleted_at IS NULL ORDER BY t.id"

	rs, err := tx.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rs.Close() }()

	out := []map[string]any{}

	for rs.Next() {
		vals := make([]any, len(names))
		ptrs := make([]any, len(names))

		for i := range vals {
			ptrs[i] = &vals[i]
		}

		if err := rs.Scan(ptrs...); err != nil {
			return nil, err
		}

		row := make(map[string]any, len(names))

		for i, n := range names {
			if b, ok := vals[i].([]byte); ok {
				vals[i] = string(b)
			}

			if text, ok := vals[i].(string); ok && slices.Contains(e.JSON, n) {
				var doc any
				if err := json.Unmarshal([]byte(text), &doc); err != nil {
					return nil, fmt.Errorf("changes: %s.%s of %s holds bad JSON: %w", e.Table, n, e.Name, err)
				}

				vals[i] = doc
			}

			row[n] = vals[i]
		}

		out = append(out, row)
	}

	return out, rs.Err()
}
