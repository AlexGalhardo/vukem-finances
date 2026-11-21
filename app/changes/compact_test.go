package changes_test

import (
	"errors"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/ulid"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

const retention = 90 * 24 * time.Hour

func ago(days int) string {
	return time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format("2006-01-02T15:04:05.000Z")
}

func countOf(t *testing.T, db *database.DB, query string, args ...any) int {
	t.Helper()

	var n int
	if err := db.SQL().QueryRowContext(t.Context(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

// age moves every change of uid, and the deleted_at of a tombstone, days into the past.
func age(t *testing.T, db *database.DB, table, uid string, days int) {
	t.Helper()

	exec(t, db, "UPDATE changes SET created_at = ? WHERE uid = ?", ago(days), uid)

	if table != "" {
		exec(t, db, "UPDATE "+table+" SET deleted_at = ? WHERE uid = ? AND deleted_at IS NOT NULL", ago(days), uid)
		exec(t, db, "UPDATE changes SET created_at = ? WHERE uid = ?", ago(days), uid)
	}
}

func TestCompaction(t *testing.T) {
	t.Parallel()

	db := openDB(t)
	gen := ulid.NewGenerator()
	ts := now + ", " + now

	account := func(ws int64, name string) string {
		uid := gen.New()
		exec(t, db, "INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (?, ?, ?, 'cash', "+ts+")", ws, uid, name)

		return uid
	}

	oldGone := account(1, "old tombstone")
	exec(t, db, "INSERT INTO transactions (workspace_id, uid, account_id, kind, amount_cents, description, occurred_on, created_at, updated_at) VALUES (1, ?, (SELECT id FROM accounts WHERE uid = ?), 'expense', 100, 'x', '2026-01-01', "+ts+")", gen.New(), oldGone)
	exec(t, db, "UPDATE transactions SET deleted_at = "+now+" WHERE account_id = (SELECT id FROM accounts WHERE uid = ?)", oldGone)
	exec(t, db, "UPDATE accounts SET deleted_at = "+now+" WHERE uid = ?", oldGone)
	age(t, db, "accounts", oldGone, 100)
	exec(t, db, "UPDATE transactions SET deleted_at = ?", ago(100))
	exec(t, db, "UPDATE changes SET created_at = ? WHERE entity = 'transaction'", ago(100))

	young := account(1, "young tombstone")
	exec(t, db, "UPDATE accounts SET deleted_at = "+now+" WHERE uid = ?", young)
	age(t, db, "accounts", young, 10)

	edited := account(1, "edited a lot")
	for _, name := range []string{"b", "c", "d"} {
		exec(t, db, "UPDATE accounts SET name = ? WHERE uid = ?", name, edited)
	}

	age(t, db, "", edited, 100)

	single := account(1, "single old change")
	age(t, db, "", single, 200)

	other := account(2, "other workspace")
	exec(t, db, "UPDATE accounts SET deleted_at = "+now+" WHERE uid = ?", other)
	age(t, db, "accounts", other, 100)

	// A deleted category still pointed at by a younger deleted transaction waits.
	cat := gen.New()
	exec(t, db, "INSERT INTO categories (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, ?, 'held', 'expense', "+ts+")", cat)
	exec(t, db, "INSERT INTO transactions (workspace_id, uid, account_id, category_id, kind, amount_cents, description, occurred_on, created_at, updated_at) VALUES (1, ?, (SELECT id FROM accounts WHERE uid = ?), (SELECT id FROM categories WHERE uid = ?), 'expense', 1, 'y', '2026-01-01', "+ts+")", gen.New(), young, cat)
	exec(t, db, "UPDATE transactions SET deleted_at = "+now+" WHERE category_id = (SELECT id FROM categories WHERE uid = ?)", cat)
	exec(t, db, "UPDATE transactions SET deleted_at = ? WHERE category_id = (SELECT id FROM categories WHERE uid = ?)", ago(10), cat)
	exec(t, db, "UPDATE categories SET deleted_at = ? WHERE uid = ?", ago(100), cat)
	exec(t, db, "UPDATE changes SET created_at = ? WHERE uid = ?", ago(100), cat)

	exec(t, db, "INSERT INTO sync_conflicts (workspace_id, entity, uid, field, created_at, updated_at) VALUES (1, 'account', ?, 'name', ?, ?)", edited, ago(40), ago(40))
	exec(t, db, "INSERT INTO sync_conflicts (workspace_id, entity, uid, field, created_at, updated_at) VALUES (1, 'account', ?, 'name', ?, ?)", edited, ago(5), ago(5))

	before, _ := changes.Cursor(t.Context(), db, 1)

	stats, err := changes.Compact(t.Context(), db, time.Now(), retention)
	if err != nil {
		t.Fatal(err)
	}

	has := func(table, uid string) bool {
		return countOf(t, db, "SELECT COUNT(*) FROM "+table+" WHERE uid = ?", uid) == 1
	}

	t.Run("old tombstones are purged with their log", func(t *testing.T) {
		if has("accounts", oldGone) || has("accounts", other) {
			t.Fatal("an old tombstone survived")
		}

		if countOf(t, db, "SELECT COUNT(*) FROM transactions WHERE account_id NOT IN (SELECT id FROM accounts)") != 0 {
			t.Fatal("a transaction is left pointing at a purged account")
		}

		if n := countOf(t, db, "SELECT COUNT(*) FROM changes WHERE uid IN (?, ?)", oldGone, other); n != 0 {
			t.Fatalf("%d change rows of purged rows remain", n)
		}

		if stats.Purged["account"] != 2 || stats.Purged["transaction"] != 1 {
			t.Fatalf("purged = %v, want 2 accounts and 1 transaction", stats.Purged)
		}
	})

	t.Run("a younger tombstone is kept with its log", func(t *testing.T) {
		if !has("accounts", young) || countOf(t, db, "SELECT COUNT(*) FROM changes WHERE uid = ?", young) == 0 {
			t.Fatal("a tombstone inside the window was purged")
		}
	})

	t.Run("the newest change per row is kept and older ones go", func(t *testing.T) {
		if n := countOf(t, db, "SELECT COUNT(*) FROM changes WHERE uid = ?", edited); n != 1 {
			t.Fatalf("a row edited 3 times keeps %d changes, want its newest only", n)
		}

		var version int64
		if err := db.SQL().QueryRowContext(t.Context(), "SELECT row_version FROM changes WHERE uid = ?", edited).Scan(&version); err != nil || version != 4 {
			t.Fatalf("kept change has version %d (%v), want the newest, 4", version, err)
		}

		if n := countOf(t, db, "SELECT COUNT(*) FROM changes WHERE uid = ?", single); n != 1 {
			t.Fatalf("the only change of a row was removed (%d left)", n)
		}
	})

	t.Run("a deleted category waits for the younger rows that point at it", func(t *testing.T) {
		if !has("categories", cat) {
			t.Fatal("a category was purged while a transaction still points at it")
		}
	})

	t.Run("compacted_through_seq is recorded per workspace", func(t *testing.T) {
		var through int64
		if err := db.SQL().QueryRowContext(t.Context(), "SELECT compacted_through_seq FROM sync_state WHERE workspace_id = 1").Scan(&through); err != nil {
			t.Fatal(err)
		}

		if through == 0 || through > before {
			t.Fatalf("compacted_through_seq = %d, want a seq between 1 and %d", through, before)
		}

		if _, err := changes.Since(t.Context(), db, 1, 0, 10); !errors.Is(err, changes.ErrResyncRequired) {
			t.Fatalf("Since(0) after compaction = %v, want ErrResyncRequired", err)
		}

		if _, err := changes.Since(t.Context(), db, 1, through, 10); err != nil {
			t.Fatalf("Since(compacted_through_seq) = %v, want it served", err)
		}

		if countOf(t, db, "SELECT COUNT(*) FROM sync_state WHERE workspace_id = 2") != 1 {
			t.Fatal("workspace 2 purged a tombstone and must have its own state")
		}
	})

	t.Run("expired conflicts go, recent ones stay", func(t *testing.T) {
		if n := countOf(t, db, "SELECT COUNT(*) FROM sync_conflicts"); n != 1 || stats.Conflicts != 1 {
			t.Fatalf("conflicts left = %d, removed = %d; want 1 and 1", n, stats.Conflicts)
		}
	})

	t.Run("a second run has nothing to do and the category goes once it is free", func(t *testing.T) {
		again, err := changes.Compact(t.Context(), db, time.Now(), retention)
		if err != nil || len(again.Purged) != 0 || again.Changes != 0 {
			t.Fatalf("second run = %+v, %v; want no work", again, err)
		}

		later, err := changes.Compact(t.Context(), db, time.Now().Add(200*24*time.Hour), retention)
		if err != nil {
			t.Fatal(err)
		}

		if has("categories", cat) || has("accounts", young) || later.Purged["category"] != 1 {
			t.Fatalf("after the window the held rows must go: %+v", later)
		}
	})
}

func TestCompactionKeepsALiveRow(t *testing.T) {
	t.Parallel()

	db := openDB(t)
	uid := ulid.NewGenerator().New()

	exec(t, db, insertAccount, uid)
	age(t, db, "", uid, 400)

	if _, err := changes.Compact(t.Context(), db, time.Now(), retention); err != nil {
		t.Fatal(err)
	}

	if countOf(t, db, "SELECT COUNT(*) FROM accounts WHERE uid = ?", uid) != 1 {
		t.Fatal("compaction purged a live row")
	}
}
