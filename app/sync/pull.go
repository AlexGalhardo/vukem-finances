package sync

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/AlexGalhardo/vukem/database"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// PullItem is the current state of one row that changed after the cursor. A
// delete (a tombstone, or a row compaction purged) has a null row.
type PullItem struct {
	Seq     int64          `json:"seq"`
	Entity  string         `json:"entity"`
	UID     string         `json:"uid"`
	Op      string         `json:"op"`
	Version int64          `json:"version"`
	Row     map[string]any `json:"row"`
}

// PullResult is a page of changes.
type PullResult struct {
	Changes    []PullItem `json:"changes"`
	NextCursor int64      `json:"next_cursor"`
	HasMore    bool       `json:"has_more"`
	ServerTime time.Time  `json:"server_time"`
}

// Pull returns the changes of the workspace after cursor, grouped by row and
// ordered parents first.
//
// The log and the rows are read in one read-only transaction, a single
// snapshot of the database. A change that commits after the snapshot has a
// higher seq than everything in it (SQLite hands out seq inside the one
// write transaction, so seq order is commit order), and next_cursor is the
// seq of the last change read, never a clock or a count: a writer that
// commits late is simply on the next page. A transfer leg whose partner is
// still to come in a later page is sent now with the partner, so legs are
// never split.
func (s *Service) Pull(ctx context.Context, a Actor, dev models.SyncDevice, cursor int64, limit int) (PullResult, error) {
	if cursor < 0 {
		return PullResult{}, fail(http.StatusBadRequest, CodeBadRequest, "The cursor must be zero or a seq from a previous answer.")
	}

	switch {
	case limit <= 0:
		limit = DefaultPullLimit
	case limit > MaxPullLimit:
		limit = MaxPullLimit
	}

	var res PullResult

	err := s.DB.TxWith(ctx, database.TxOptions{ReadOnly: true}, func(tx *database.Tx) error {
		page, err := changes.Since(ctx, tx, a.WorkspaceID, cursor, limit)
		if errors.Is(err, changes.ErrResyncRequired) {
			return fail(http.StatusGone, CodeResyncRequired, "The cursor is older than the change log: download a snapshot and pull from its snapshot_cursor.")
		}

		if err != nil {
			return err
		}

		items, err := s.items(ctx, tx, a.WorkspaceID, page, cursor)
		if err != nil {
			return err
		}

		res = PullResult{Changes: items, NextCursor: page.Next, HasMore: page.HasMore, ServerTime: s.now()}

		return nil
	})
	if err != nil {
		return PullResult{}, err
	}

	s.notePull(ctx, a.WorkspaceID, dev, res.NextCursor)

	return res, nil
}

// notePull records how far a device pulled. It is bookkeeping for the devices
// page, so it never makes a pull wait for the single writer (a long write
// transaction would hold every reader behind it) and a failure is only logged.
func (s *Service) notePull(ctx context.Context, ws int64, dev models.SyncDevice, next int64) {
	if next <= dev.LastPullSeq {
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()

	if _, err := s.DB.SQL().ExecContext(ctx,
		"UPDATE sync_devices SET last_pull_seq = ? WHERE id = ? AND workspace_id = ? AND last_pull_seq < ?",
		next, dev.ID, ws, next); err != nil && s.Log != nil {
		s.Log.WarnContext(ctx, "sync: could not record the pull of a device", "device", dev.DeviceID, "error", err)
	}
}

type rowKey struct{ entity, uid string }

func (s *Service) items(ctx context.Context, ex database.Executor, ws int64, page changes.Page, cursor int64) ([]PullItem, error) {
	last := map[rowKey]changes.Entry{}
	uidsOf := map[string][]string{}

	for _, e := range page.Entries {
		k := rowKey{e.Entity, e.UID}
		if _, seen := last[k]; !seen {
			uidsOf[e.Entity] = append(uidsOf[e.Entity], e.UID)
		}

		last[k] = e
	}

	rows := map[rowKey]StoredRow{}

	for entity, uids := range uidsOf {
		ent, ok := changes.Lookup(entity)
		if !ok {
			continue
		}

		found, err := rowsByUID(ctx, ex, ent, ws, uids)
		if err != nil {
			return nil, err
		}

		for uid, r := range found {
			rows[rowKey{entity, uid}] = r
		}
	}

	items := make([]PullItem, 0, len(last))
	for k, e := range last {
		items = append(items, itemOf(e.Seq, k, e.RowVersion, rows))
	}

	siblings, err := s.legs(ctx, ex, ws, items, rows, page.Next)
	if err != nil {
		return nil, err
	}

	items = append(items, siblings...)

	rank := map[string]int{}
	for i, e := range Order() {
		rank[e.Name] = i
	}

	slices.SortFunc(items, func(a, b PullItem) int {
		if rank[a.Entity] != rank[b.Entity] {
			return rank[a.Entity] - rank[b.Entity]
		}

		return int(a.Seq - b.Seq)
	})

	return items, nil
}

func itemOf(seq int64, k rowKey, version int64, rows map[rowKey]StoredRow) PullItem {
	r, ok := rows[k]

	switch {
	case !ok:
		// Purged by compaction: the client deletes it.
		return PullItem{Seq: seq, Entity: k.entity, UID: k.uid, Op: models.OpDelete, Version: version}
	case r.Deleted:
		return PullItem{Seq: seq, Entity: k.entity, UID: k.uid, Op: models.OpDelete, Version: r.Version}
	default:
		return PullItem{Seq: seq, Entity: k.entity, UID: k.uid, Op: models.OpUpsert, Version: r.Version, Row: r.Wire()}
	}
}

// legs finds the partner of every transfer leg in items that is not in the
// page and whose newest change is after next: a later page would split the
// pair. The partner is sent now (and again later, which a client applies
// twice at worst).
func (s *Service) legs(ctx context.Context, ex database.Executor, ws int64, items []PullItem, rows map[rowKey]StoredRow, next int64) ([]PullItem, error) {
	ent, ok := changes.Lookup("transaction")
	if !ok {
		return nil, nil
	}

	have := map[string]bool{}
	for _, it := range items {
		if it.Entity == "transaction" {
			have[it.UID] = true
		}
	}

	var out []PullItem

	for k, r := range rows {
		if k.entity != "transaction" {
			continue
		}

		transfer, _ := r.Fields["transfer_uid"].(string)
		if transfer == "" {
			continue
		}

		partners, err := selectRows(ctx, ex, ent, ws, "t.transfer_uid = ? AND t.uid <> ?", []any{transfer, k.uid}, "t.id", 0)
		if err != nil {
			return nil, err
		}

		for _, p := range partners {
			if have[p.UID] {
				continue
			}

			var seq int64

			err := ex.QueryRowContext(ctx,
				"SELECT COALESCE(MAX(seq), 0) FROM changes WHERE workspace_id = ? AND entity = 'transaction' AND uid = ?",
				ws, p.UID).Scan(&seq)
			if err != nil {
				return nil, err
			}

			if seq <= next {
				continue
			}

			have[p.UID] = true

			out = append(out, itemOf(seq, rowKey{"transaction", p.UID}, p.Version, map[rowKey]StoredRow{{"transaction", p.UID}: p}))
		}
	}

	return out, nil
}
