package changes_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/ulid"
)

// TestConcurrentWrites runs 8 writers on one workspace: each inserts its own
// accounts and edits them, and all of them edit one shared account. The log
// must hold no duplicate seq, and the row_versions of every row must run
// 1, 2, 3 ... in seq order with the stored version at the last one.
func TestConcurrentWrites(t *testing.T) {
	t.Parallel()

	const writers, own, edits = 8, 5, 4

	db := openDB(t)
	gen := ulid.NewGenerator()
	shared := gen.New()

	exec(t, db, insertAccount, shared)

	var wg sync.WaitGroup

	errs := make(chan error, writers)

	for w := range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			run := func(query string, args ...any) error {
				_, err := db.SQL().ExecContext(t.Context(), query, args...)

				return err
			}

			for i := range own {
				uid := gen.New()

				if err := run("INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (1, ?, ?, 'cash', "+now+", "+now+")", uid, fmt.Sprintf("w%d-%d", w, i)); err != nil {
					errs <- err

					return
				}

				for e := range edits {
					if err := run("UPDATE accounts SET name = ? WHERE uid = ?", fmt.Sprintf("w%d-%d-edit%d", w, i, e), uid); err != nil {
						errs <- err

						return
					}

					if err := run("UPDATE accounts SET opening_cents = ? WHERE uid = ?", w*1000+e, shared); err != nil {
						errs <- err

						return
					}
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatal(err)
	}

	all := logOf(t, db, "")

	seen := map[int64]bool{}
	next := map[string]int64{}

	for _, r := range all {
		if seen[r.Seq] {
			t.Fatalf("duplicate seq %d", r.Seq)
		}

		seen[r.Seq] = true

		next[r.UID]++
		if r.RowVersion != next[r.UID] {
			t.Fatalf("row %s: change at seq %d has version %d, want %d", r.UID, r.Seq, r.RowVersion, next[r.UID])
		}
	}

	if got, want := len(next), writers*own+1; got != want {
		t.Fatalf("%d rows in the log, want %d", got, want)
	}

	// Each own account: 1 insert + 4 edits. The shared one: 1 insert + every
	// edit that changed a value (the writers' values are all different, but a
	// write of the value already stored is correctly not logged).
	for uid, n := range next {
		var stored int64
		if err := db.SQL().QueryRowContext(t.Context(), "SELECT version FROM accounts WHERE uid = ?", uid).Scan(&stored); err != nil {
			t.Fatal(err)
		}

		if stored != n {
			t.Errorf("row %s: stored version %d, log holds %d changes", uid, stored, n)
		}

		if uid != shared && n != 1+edits {
			t.Errorf("row %s: %d changes, want %d", uid, n, 1+edits)
		}
	}
}
