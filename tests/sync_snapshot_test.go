package tests

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// replica is what a phone holds: the live rows by entity and uid.
type replica map[string]map[string]map[string]any

func (r replica) put(entity, uid string, row map[string]any) {
	if r[entity] == nil {
		r[entity] = map[string]map[string]any{}
	}

	r[entity][uid] = row
}

func (r replica) apply(items []pullItem) {
	for _, it := range items {
		if it.Op == "delete" {
			delete(r[it.Entity], it.UID)
			continue
		}

		r.put(it.Entity, it.UID, it.Row)
	}
}

// snapshotPage is one page of GET /snapshot.
type snapshotPage struct {
	SnapshotID     string           `json:"snapshot_id"`
	SnapshotCursor int64            `json:"snapshot_cursor"`
	Rows           []map[string]any `json:"rows"`
	NextAfter      string           `json:"next_after"`
	HasMore        bool             `json:"has_more"`
}

func (d syncDeviceClient) snapshotPage(entity, after, id string, limit int) snapshotPage {
	d.t.Helper()

	res := d.client().Get(fmt.Sprintf("/api/v1/sync/snapshot?entity=%s&after=%s&limit=%d&snapshot_id=%s", entity, after, limit, id)).AssertOK()

	var page snapshotPage
	if err := json.Unmarshal([]byte(res.Body()), &page); err != nil {
		d.t.Fatalf("snapshot answer: %v: %s", err, res.Body())
	}

	return page
}

// rebuild downloads every entity page by page, calling between(entity, page)
// after each page, then pulls from the snapshot cursor. It returns the replica.
func rebuild(t *testing.T, d syncDeviceClient, limit int, between func()) replica {
	t.Helper()

	r, _ := rebuildFrom(t, d, limit, between)

	return r
}

// rebuildFrom is rebuild that also returns the cursor the device ends at.
func rebuildFrom(t *testing.T, d syncDeviceClient, limit int, between func()) (replica, int64) {
	t.Helper()

	out := replica{}
	id, cursor := "", int64(0)

	var entities []struct {
		Name string `json:"name"`
	}

	info := d.client().Get("/api/v1/sync/info").AssertOK()
	raw, _ := info.JSONPath("entities")

	b, _ := json.Marshal(raw)
	if err := json.Unmarshal(b, &entities); err != nil {
		t.Fatal(err)
	}

	for _, e := range entities {
		for after := ""; ; {
			page := d.snapshotPage(e.Name, after, id, limit)
			if id == "" {
				id, cursor = page.SnapshotID, page.SnapshotCursor
			} else if page.SnapshotCursor != cursor {
				t.Fatalf("snapshot_cursor moved from %d to %d", cursor, page.SnapshotCursor)
			}

			for _, row := range page.Rows {
				out.put(e.Name, row["uid"].(string), row)
			}

			if between != nil {
				between()
			}

			if !page.HasMore {
				break
			}

			after = page.NextAfter
		}
	}

	items, end := d.pullAll(cursor)
	out.apply(items)

	return out, end
}

// serverState is the live rows the server holds, read by a fresh full pull.
func serverState(d syncDeviceClient) replica {
	items, _ := d.pullAll(0)
	out := replica{}
	out.apply(items)

	return out
}

func sameReplica(t *testing.T, got, want replica) {
	t.Helper()

	for entity, rows := range want {
		if len(rows) != len(got[entity]) {
			t.Errorf("%s: %d rows, want %d", entity, len(got[entity]), len(rows))
		}

		for uid, row := range rows {
			if !reflect.DeepEqual(got[entity][uid], row) {
				t.Errorf("%s %s = %v, want %v", entity, uid, got[entity][uid], row)
			}
		}
	}

	for entity, rows := range got {
		if len(want[entity]) != len(rows) {
			t.Errorf("%s: %d rows, the server has %d", entity, len(rows), len(want[entity]))
		}
	}
}

func TestSyncSnapshot(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")
	account, category, txn := seedLedger(t, d)
	d.one(mut("transaction", txn, "delete", 0, nil))

	// Pages walk the rows in uid order without repeating or missing one.
	var seen []string

	id := ""

	for after := ""; ; {
		page := d.snapshotPage("category", after, id, 2)
		id = page.SnapshotID

		for _, row := range page.Rows {
			seen = append(seen, row["uid"].(string))
		}

		if !page.HasMore {
			break
		}

		after = page.NextAfter
	}

	if len(seen) != 8 {
		t.Errorf("snapshot of categories = %d rows, want the 7 defaults and the one pushed", len(seen))
	}

	for i := 1; i < len(seen); i++ {
		if seen[i-1] >= seen[i] {
			t.Errorf("rows not in uid order at %d: %s, %s", i, seen[i-1], seen[i])
		}
	}

	// A snapshot holds live rows only: the deleted transaction is not in it.
	if page := d.snapshotPage("transaction", "", "", 100); len(page.Rows) != 0 {
		t.Errorf("snapshot of transactions = %v, want none (deleted)", page.Rows)
	}

	if page := d.snapshotPage("account", "", "", 100); !containsUID(page.Rows, account) {
		t.Errorf("snapshot of accounts misses %s", account)
	}

	_ = category

	assertProblem(t, d.client().Get("/api/v1/sync/snapshot?entity=widget"), 400, "unknown_entity")
	assertProblem(t, d.client().Get("/api/v1/sync/snapshot?entity=account&snapshot_id=junk"), 400, "invalid_snapshot")
	assertProblem(t, d.client().Get("/api/v1/sync/snapshot?entity=account&snapshot_id=s1.999999.5"), 400, "invalid_snapshot")
}

func containsUID(rows []map[string]any, uid string) bool {
	for _, r := range rows {
		if r["uid"] == uid {
			return true
		}
	}

	return false
}

// A client that snapshots while another device writes ends equal to the
// server after one pull.
func TestSnapshotWithConcurrentWrites(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	phone, writer := f.device(t, "phone"), f.device(t, "writer")
	account, _, txn := seedLedger(t, writer)

	step := 0

	between := func() {
		step++

		switch step {
		case 1:
			writer.one(mut("account", newULID(), "upsert", 0, map[string]any{"name": "Made during", "kind": "cash"}))
		case 2:
			writer.one(mut("account", account, "upsert", 0, map[string]any{"name": "Renamed during"}))
		case 3:
			writer.one(mut("transaction", txn, "delete", 0, nil))
		case 4:
			writer.one(mut("category", newULID(), "upsert", 0, map[string]any{"name": "Also during", "kind": "income"}))
		}
	}

	got := rebuild(t, phone, 2, between)

	if step < 4 {
		t.Fatalf("only %d pages, the writes did not all happen", step)
	}

	sameReplica(t, got, serverState(writer))
}

// A cursor behind compaction is told to resync, and snapshot plus pull
// rebuilds the device exactly.
func TestSyncPullResync(t *testing.T) {
	t.Parallel()

	clock := vukemtest.NewFakeClock(time.Now().UTC())
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:   map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"},
		Clock: clock.Now,
	})
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	old := newSyncDevice(t, app, rawToken(t, app, ana.User, ana.Workspace.ID, "sync"), "old phone")
	account, _, txn := seedLedger(t, old)
	_, behind := old.pullAll(0)

	old.one(mut("transaction", txn, "delete", 0, nil))

	clock.Advance(91 * 24 * time.Hour)

	if err := jobs.HandleChangesCompact(app.App)(t.Context(), jobs.ChangesCompact{}); err != nil {
		t.Fatal(err)
	}

	// The token of the old phone expired meanwhile: a new one, a new device.
	fresh := newSyncDevice(t, app, rawToken(t, app, ana.User, ana.Workspace.ID, "sync"), "new phone")
	fresh.one(mut("account", newULID(), "upsert", 0, map[string]any{"name": "After compaction", "kind": "cash"}))

	for _, cursor := range []int64{0, behind} {
		res := fresh.client().Get("/api/v1/sync/pull?cursor=" + itoa(cursor))
		assertProblem(t, res, 410, "resync_required")
	}

	got := rebuild(t, fresh, 3, nil)

	// Another device rebuilding in one big page is the reference: a full pull
	// from 0 is refused after a compaction.
	reference := newSyncDevice(t, app, rawToken(t, app, ana.User, ana.Workspace.ID, "sync"), "reference")
	sameReplica(t, got, rebuild(t, reference, 1000, nil))

	if _, ok := got["account"][account]; !ok {
		t.Error("the rebuilt replica lost an account")
	}

	if _, ok := got["transaction"][txn]; ok {
		t.Error("the rebuilt replica holds a purged transaction")
	}

	// From the snapshot cursor on, pull works again.
	_, cursor := fresh.pullAll(mustCursor(t, fresh))
	fresh.pull(cursor, 1)

	db, _ := app.DB()
	if page, err := changes.Since(t.Context(), db, ana.Workspace.ID, cursor, 1); err != nil || len(page.Entries) != 0 {
		t.Errorf("after the rebuild: %v, %v", page, err)
	}
}

// mustCursor is the newest seq of the workspace as a snapshot reports it.
func mustCursor(t *testing.T, d syncDeviceClient) int64 {
	t.Helper()

	return d.snapshotPage("tag", "", "", 1).SnapshotCursor
}

// TestSyncPullNeverSkipsALateCommit is the watermark proof. A pull cursor is
// the seq of the last change read; a writer that commits after another one
// that started later still holds a higher seq, because SQLite hands out seq
// inside the single write transaction. So nothing is committed "behind" a
// cursor, and a pull taken while a writer is mid-transaction neither sees its
// rows nor moves past them.
func TestSyncPullNeverSkipsALateCommit(t *testing.T) {
	t.Parallel()

	path := filepath.ToSlash(filepath.Join(t.TempDir(), "database.sqlite"))
	app := newSaaSApp(t, map[string]string{"DATABASE_URL": "file:" + path})
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	d := newSyncDevice(t, app, rawToken(t, app, ana.User, ana.Workspace.ID, "sync"), "phone")
	other := newSyncDevice(t, app, rawToken(t, app, ana.User, ana.Workspace.ID, "sync"), "other")
	_, cursor := d.pullAll(0)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	slow := newULID()
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)

	go func() {
		done <- db.Tx(t.Context(), func(tx *database.Tx) error {
			if _, err := tx.ExecContext(t.Context(),
				"INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (?, ?, 'Slow', 'cash', datetime('now'), datetime('now'))",
				ana.Workspace.ID, slow); err != nil {
				return err
			}

			close(started)
			<-release

			return nil
		})
	}()

	<-started

	// A second writer queues behind the slow one: it will commit after it.
	fast := newULID()
	fastDone := make(chan pushResult, 1)

	go func() {
		fastDone <- other.one(mut("account", fast, "upsert", 0, map[string]any{"name": "Fast", "kind": "cash"}))
	}()

	// While the slow writer is open, a pull sees neither row and keeps its cursor.
	mid := d.pull(cursor, 100)
	if len(mid.Changes) != 0 || mid.NextCursor != cursor {
		t.Fatalf("a pull during an open write transaction = %+v, want nothing and the same cursor", mid)
	}

	close(release)

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	if r := <-fastDone; r.Status != "applied" {
		t.Fatalf("the queued push = %+v", r)
	}

	// Both rows are delivered, in commit order, from the old cursor.
	after := d.pull(cursor, 100)
	got := only(after.Changes, slow, fast)

	if len(got) != 2 || got[0].UID != slow || got[1].UID != fast || got[0].Seq >= got[1].Seq {
		t.Fatalf("after the commits = %+v, want the slow row then the fast one", after.Changes)
	}
}

// Writers that commit while a client pulls in small pages never leave a row
// behind: when the writers stop, what the puller saw is exactly what exists.
func TestSyncPullUnderConcurrentWriters(t *testing.T) {
	t.Parallel()

	path := filepath.ToSlash(filepath.Join(t.TempDir(), "database.sqlite"))
	app := newSaaSApp(t, map[string]string{"DATABASE_URL": "file:" + path})
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	d := newSyncDevice(t, app, rawToken(t, app, ana.User, ana.Workspace.ID, "sync"), "phone")
	ws := service(t, app)

	fin, err := finance.New(app.App, ws)
	if err != nil {
		t.Fatal(err)
	}

	svc := &syncproto.Service{DB: ws.DB, WS: ws, Fin: fin, Now: app.Now}
	actor := syncproto.Actor{WorkspaceID: ana.Workspace.ID, UserID: ana.User.ID, Role: "owner"}

	dev, err := svc.Device(workspaces.WithTenant(t.Context(), ana.Workspace.ID), syncproto.Actor{
		WorkspaceID: ana.Workspace.ID, UserID: ana.User.ID, TokenID: tokenID(d.token), Role: "owner",
	}, d.Device)
	if err != nil {
		t.Fatal(err)
	}

	db, _ := app.DB()

	const writers, each = 6, 15

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		written = map[string]bool{}
		stop    = make(chan struct{})
	)

	for w := range writers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for i := range each {
				uid := newULID()

				err := db.Tx(t.Context(), func(tx *database.Tx) error {
					_, err := tx.ExecContext(t.Context(),
						"INSERT INTO accounts (workspace_id, uid, name, kind, created_at, updated_at) VALUES (?, ?, ?, 'cash', datetime('now'), datetime('now'))",
						ana.Workspace.ID, uid, fmt.Sprintf("w%d-%d", w, i))

					return err
				})
				if err != nil {
					t.Error(err)
					return
				}

				mu.Lock()
				written[uid] = true
				mu.Unlock()
			}
		}()
	}

	go func() { wg.Wait(); close(stop) }()

	seen := map[string]bool{}
	cursor := int64(0)
	for {
		// A pull that starts after every writer finished and has no more pages
		// is the last one.
		var finished bool

		select {
		case <-stop:
			finished = true
		default:
		}

		res, err := svc.Pull(t.Context(), actor, dev, cursor, 7)
		if err != nil {
			t.Fatal(err)
		}

		for _, it := range res.Changes {
			seen[it.UID] = true
		}

		cursor = res.NextCursor

		if finished && !res.HasMore {
			break
		}
	}

	mu.Lock()
	defer mu.Unlock()

	for uid := range written {
		if !seen[uid] {
			t.Errorf("a committed row (%s) was never pulled", uid)
		}
	}

}
