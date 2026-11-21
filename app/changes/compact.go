package changes

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
)

// ConflictRetention is how long a lost value stays in sync_conflicts.
const ConflictRetention = 30 * 24 * time.Hour

// isoSeconds is the layout compared with substr(column, 1, 19): the ORM and
// the triggers both write "2006-01-02T15:04:05.000Z", so the first 19
// characters of two timestamps compare like the instants.
const isoSeconds = "2006-01-02T15:04:05"

// Stats says what a compaction removed.
type Stats struct {
	// Purged counts physically deleted tombstones by entity name.
	Purged map[string]int64
	// Changes is the number of change-log rows removed.
	Changes int64
	// Conflicts is the number of expired sync_conflicts removed.
	Conflicts int64
}

// Compact is the daily clean-up of the sync data. With cutoff = now -
// retention it:
//
//  1. purges physically the soft-deleted rows deleted before the cutoff, when
//     no change newer than the cutoff exists for them and no other row still
//     points at them (children go first, so a deleted account leaves with its
//     transactions);
//  2. removes every change-log row of a purged row, and the older rows of a
//     live one that a newer change supersedes, so the newest change per
//     (workspace, entity, uid) always stays;
//  3. raises sync_state.compacted_through_seq of each workspace to the newest
//     seq it removed, which makes Since answer ErrResyncRequired to a client
//     whose cursor is behind it;
//  4. drops sync_conflicts older than ConflictRetention.
//
// It runs in one transaction, so a failure removes nothing.
func Compact(ctx context.Context, db *database.DB, now time.Time, retention time.Duration) (Stats, error) {
	stats := Stats{Purged: map[string]int64{}}
	cut := now.UTC().Add(-retention).Format(isoSeconds)
	cutConflicts := now.UTC().Add(-ConflictRetention).Format(isoSeconds)

	err := db.Tx(ctx, func(tx *database.Tx) error {
		through := map[int64]int64{}

		for _, e := range purgeOrder() {
			if err := purge(ctx, tx, e, cut, through, &stats); err != nil {
				return err
			}
		}

		if err := supersede(ctx, tx, cut, through, &stats); err != nil {
			return err
		}

		for ws, seq := range through {
			_, err := tx.ExecContext(ctx, `INSERT INTO sync_state (workspace_id, compacted_through_seq, created_at, updated_at)
VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
ON CONFLICT (workspace_id) DO UPDATE SET
	compacted_through_seq = MAX(compacted_through_seq, excluded.compacted_through_seq),
	updated_at = excluded.updated_at`, ws, seq)
			if err != nil {
				return err
			}
		}

		res, err := tx.ExecContext(ctx, "DELETE FROM sync_conflicts WHERE substr(created_at, 1, 19) < ?", cutConflicts)
		if err != nil {
			return err
		}

		stats.Conflicts, _ = res.RowsAffected()

		return nil
	})
	if err != nil {
		return Stats{}, fmt.Errorf("changes: compact: %w", err)
	}

	return stats, nil
}

// purgeOrder lists the entities children first: an entity comes before every
// entity it refers to.
func purgeOrder() []Entity {
	all := Entities()
	depth := map[string]int{}

	var depthOf func(name string, seen map[string]bool) int

	depthOf = func(name string, seen map[string]bool) int {
		e, ok := Lookup(name)
		if !ok || seen[name] {
			return 0
		}

		seen[name] = true
		d := 0

		for _, r := range e.Refs {
			d = max(d, depthOf(r.Entity, seen)+1)
		}

		return d
	}

	for _, e := range all {
		depth[e.Name] = depthOf(e.Name, map[string]bool{})
	}

	slices.SortStableFunc(all, func(a, b Entity) int { return depth[b.Name] - depth[a.Name] })

	return all
}

// eligible is the WHERE clause (on alias t) of a tombstone that may be purged;
// it takes the cutoff twice.
func eligible(e Entity) string {
	var guards []string

	for _, child := range Entities() {
		for _, r := range child.Refs {
			if r.Entity == e.Name {
				guards = append(guards, fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s c WHERE c.%s = t.id)", child.Table, r.Column))
			}
		}
	}

	clause := `t.deleted_at IS NOT NULL AND substr(t.deleted_at, 1, 19) < ?
AND NOT EXISTS (SELECT 1 FROM changes n WHERE n.workspace_id = t.workspace_id AND n.entity = '` + e.Name + `'
	AND n.uid = t.uid AND substr(n.created_at, 1, 19) >= ?)`

	if len(guards) > 0 {
		clause += " AND " + strings.Join(guards, " AND ")
	}

	return clause
}

func purge(ctx context.Context, tx *database.Tx, e Entity, cut string, through map[int64]int64, stats *Stats) error {
	where := eligible(e)

	rows, err := tx.QueryContext(ctx, `SELECT c.workspace_id, MAX(c.seq) FROM changes c
JOIN `+e.Table+` t ON t.workspace_id = c.workspace_id AND t.uid = c.uid
WHERE c.entity = '`+e.Name+`' AND `+where+` GROUP BY c.workspace_id`, cut, cut)
	if err != nil {
		return err
	}

	if err := noteMax(rows, through); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM changes WHERE entity = '`+e.Name+`' AND EXISTS (
SELECT 1 FROM `+e.Table+` t WHERE t.workspace_id = changes.workspace_id AND t.uid = changes.uid AND `+where+`)`, cut, cut)
	if err != nil {
		return err
	}

	n, _ := res.RowsAffected()
	stats.Changes += n

	res, err = tx.ExecContext(ctx, "DELETE FROM "+e.Table+" AS t WHERE "+where, cut, cut)
	if err != nil {
		return err
	}

	if n, _ := res.RowsAffected(); n > 0 {
		stats.Purged[e.Name] += n
	}

	return nil
}

func supersede(ctx context.Context, tx *database.Tx, cut string, through map[int64]int64, stats *Stats) error {
	const old = `substr(created_at, 1, 19) < ? AND seq < (SELECT MAX(m.seq) FROM changes m
WHERE m.workspace_id = changes.workspace_id AND m.entity = changes.entity AND m.uid = changes.uid)`

	rows, err := tx.QueryContext(ctx, "SELECT workspace_id, MAX(seq) FROM changes WHERE "+old+" GROUP BY workspace_id", cut)
	if err != nil {
		return err
	}

	if err := noteMax(rows, through); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, "DELETE FROM changes WHERE "+old, cut)
	if err != nil {
		return err
	}

	n, _ := res.RowsAffected()
	stats.Changes += n

	return nil
}

// noteMax folds (workspace_id, max seq) rows into through and closes rows.
func noteMax(rows *sql.Rows, through map[int64]int64) error {
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var ws, seq int64
		if err := rows.Scan(&ws, &seq); err != nil {
			return err
		}

		through[ws] = max(through[ws], seq)
	}

	return rows.Err()
}
