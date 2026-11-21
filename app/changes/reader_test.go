package changes_test

import (
	"errors"
	"testing"

	"github.com/AlexGalhardo/vukem/ulid"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

func TestSince(t *testing.T) {
	t.Parallel()

	db := openDB(t)
	gen := ulid.NewGenerator()

	var mine []string

	for _, ws := range []int64{1, 2, 1, 2, 1} {
		uid := gen.New()
		exec(t, db, "INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (?, ?, 'A', 'cash', "+now+", "+now+")", ws, uid)

		if ws == 1 {
			mine = append(mine, uid)
		}
	}

	t.Run("ordered and isolated", func(t *testing.T) {
		page, err := changes.Since(t.Context(), db, 1, 0, 10)
		if err != nil {
			t.Fatal(err)
		}

		if len(page.Entries) != 3 || page.HasMore {
			t.Fatalf("page = %+v, want the 3 changes of workspace 1", page)
		}

		for i, e := range page.Entries {
			if e.UID != mine[i] || e.Entity != "account" || e.Op != "upsert" || e.RowVersion != 1 {
				t.Errorf("entry %d = %+v, want account %s", i, e, mine[i])
			}

			if i > 0 && e.Seq <= page.Entries[i-1].Seq {
				t.Errorf("seq %d does not grow after %d", e.Seq, page.Entries[i-1].Seq)
			}
		}

		if page.Next != page.Entries[2].Seq {
			t.Errorf("next = %d, want %d", page.Next, page.Entries[2].Seq)
		}

		other, err := changes.Since(t.Context(), db, 2, 0, 10)
		if err != nil || len(other.Entries) != 2 {
			t.Fatalf("workspace 2 page = %+v, %v; want its 2 changes", other, err)
		}

		for _, e := range other.Entries {
			if e.UID == mine[0] || e.UID == mine[1] || e.UID == mine[2] {
				t.Fatalf("workspace 2 can read the change of %s", e.UID)
			}
		}

		none, err := changes.Since(t.Context(), db, 3, 0, 10)
		if err != nil || len(none.Entries) != 0 || none.Next != 0 {
			t.Fatalf("an unknown workspace read %+v, %v", none, err)
		}
	})

	t.Run("limit and hasMore", func(t *testing.T) {
		first, err := changes.Since(t.Context(), db, 1, 0, 2)
		if err != nil || len(first.Entries) != 2 || !first.HasMore {
			t.Fatalf("first page = %+v, %v; want 2 entries and hasMore", first, err)
		}

		second, err := changes.Since(t.Context(), db, 1, first.Next, 2)
		if err != nil || len(second.Entries) != 1 || second.HasMore || second.Entries[0].UID != mine[2] {
			t.Fatalf("second page = %+v, %v; want the last entry and no more", second, err)
		}

		empty, err := changes.Since(t.Context(), db, 1, second.Next, 2)
		if err != nil || len(empty.Entries) != 0 || empty.Next != second.Next {
			t.Fatalf("after the end = %+v, %v; want nothing and the same cursor", empty, err)
		}

		huge, err := changes.Since(t.Context(), db, 1, 0, 1_000_000)
		if err != nil || len(huge.Entries) != 3 {
			t.Fatalf("a huge limit = %+v, %v", huge, err)
		}
	})

	t.Run("cursor", func(t *testing.T) {
		got, err := changes.Cursor(t.Context(), db, 1)
		if err != nil {
			t.Fatal(err)
		}

		page, _ := changes.Since(t.Context(), db, 1, 0, 10)
		if got != page.Next {
			t.Errorf("cursor = %d, want %d", got, page.Next)
		}

		if got, err := changes.Cursor(t.Context(), db, 3); err != nil || got != 0 {
			t.Errorf("cursor of a workspace without changes = %d, %v; want 0", got, err)
		}
	})

	t.Run("a cursor behind the compacted log needs a snapshot", func(t *testing.T) {
		exec(t, db, "INSERT INTO sync_state (workspace_id, compacted_through_seq, created_at, updated_at) VALUES (1, 3, "+now+", "+now+")")

		if _, err := changes.Since(t.Context(), db, 1, 2, 10); !errors.Is(err, changes.ErrResyncRequired) {
			t.Fatalf("cursor 2 = %v, want ErrResyncRequired", err)
		}

		if _, err := changes.Since(t.Context(), db, 1, 3, 10); err != nil {
			t.Fatalf("cursor at the compacted seq = %v, want it served", err)
		}

		if _, err := changes.Since(t.Context(), db, 2, 0, 10); err != nil {
			t.Fatalf("another workspace is not affected: %v", err)
		}
	})
}

func TestSnapshot(t *testing.T) {
	t.Parallel()

	db := openDB(t)
	gen := ulid.NewGenerator()
	acc, cat, gone := gen.New(), gen.New(), gen.New()

	exec(t, db, "INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, ?, 'Main', 'cash', "+now+", "+now+")", acc)
	exec(t, db, "INSERT INTO categories (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, ?, 'Food', 'expense', "+now+", "+now+")", cat)
	exec(t, db, "INSERT INTO transactions (workspace_id, uid, account_id, category_id, kind, amount_cents, description, occurred_on, created_at, updated_at) VALUES (1, ?, 1, 1, 'expense', 500, 'x', '2026-01-01', "+now+", "+now+")", gone)
	exec(t, db, "UPDATE transactions SET deleted_at = "+now+" WHERE uid = ?", gone)
	exec(t, db, "INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (2, ?, 'Other', 'cash', "+now+", "+now+")", gen.New())

	snap, err := changes.Snapshot(t.Context(), db, 1)
	if err != nil {
		t.Fatal(err)
	}

	if len(snap.Rows["account"]) != 1 || snap.Rows["account"][0]["uid"] != acc || snap.Rows["account"][0]["name"] != "Main" {
		t.Errorf("accounts = %+v, want only the workspace's Main", snap.Rows["account"])
	}

	if len(snap.Rows["transaction"]) != 0 {
		t.Errorf("transactions = %+v, want the deleted one left out", snap.Rows["transaction"])
	}

	if len(snap.Rows["category"]) != 1 {
		t.Errorf("categories = %+v, want 1", snap.Rows["category"])
	}

	want, _ := changes.Cursor(t.Context(), db, 1)
	if snap.Cursor != want || want == 0 {
		t.Errorf("snapshot cursor = %d, want %d", snap.Cursor, want)
	}

	exec(t, db, "UPDATE transactions SET deleted_at = NULL WHERE uid = ?", gone)

	snap, err = changes.Snapshot(t.Context(), db, 1)
	if err != nil || len(snap.Rows["transaction"]) != 1 {
		t.Fatalf("after the restore: %+v, %v", snap.Rows["transaction"], err)
	}

	row := snap.Rows["transaction"][0]
	if row["account_uid"] != acc || row["category_uid"] != cat {
		t.Errorf("transaction refs = %v / %v, want the parents' uids %s / %s", row["account_uid"], row["category_uid"], acc, cat)
	}

	if _, leaked := row["account_id"]; leaked {
		t.Error("the snapshot leaks an integer id")
	}
}
