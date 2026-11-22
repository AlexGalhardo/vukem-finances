package tests

import (
	"context"
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// simDevice is a phone that talks to the sync service directly (no HTTP, so
// no rate limits): a replica, a queue of mutations made offline and the seq it
// last saw of each row.
type simDevice struct {
	name    string
	svc     *syncproto.Service
	ctx     context.Context
	actor   syncproto.Actor
	dev     models.SyncDevice
	replica replica
	deleted map[string]string // uid -> entity of rows this phone deleted and may restore
	pending []syncproto.MutationIn
	seen    map[string]int64
	cursor  int64
	stale   bool
	stats   map[string]int
}

func (d *simDevice) live(entity string) []string {
	var out []string
	for uid := range d.replica[entity] {
		out = append(out, uid)
	}

	slices.Sort(out)

	return out
}

func (d *simDevice) queue(entity, uid, op string, fields map[string]any) {
	d.pending = append(d.pending, syncproto.MutationIn{
		OpID: newULID(), Entity: entity, UID: uid, Op: op, BaseSeq: d.seen[uid], Fields: fields,
	})
}

// act makes one random local change, applied to the replica at once and queued.
func (d *simDevice) act(rng *rand.Rand, n int) {
	accounts, categories, txns := d.live("account"), d.live("category"), d.live("transaction")

	switch k := rng.IntN(10); {
	case k < 2 || len(accounts) == 0:
		uid := newULID()
		fields := map[string]any{"name": fmt.Sprintf("acc %s %d", d.name, n), "kind": "cash"}
		d.queue("account", uid, "upsert", fields)
		d.replica.put("account", uid, map[string]any{"uid": uid, "name": fields["name"]})
	case k < 3:
		uid := newULID()
		// A small pool of names so two phones collide now and then.
		fields := map[string]any{"name": fmt.Sprintf("cat %d", rng.IntN(4)), "kind": "expense"}
		d.queue("category", uid, "upsert", fields)
		d.replica.put("category", uid, map[string]any{"uid": uid, "name": fields["name"]})
	case k < 6:
		uid := newULID()
		fields := map[string]any{
			"account_uid": accounts[rng.IntN(len(accounts))], "kind": "expense", "amount_cents": 1 + rng.IntN(5000),
			"occurred_on": "2026-11-10", "description": fmt.Sprintf("tx %s %d", d.name, n),
		}

		if len(categories) > 0 && rng.IntN(2) == 0 {
			fields["category_uid"] = categories[rng.IntN(len(categories))]
		}

		d.queue("transaction", uid, "upsert", fields)
		d.replica.put("transaction", uid, map[string]any{"uid": uid})
	case k < 8 && len(txns) > 0:
		uid := txns[rng.IntN(len(txns))]
		field, value := "description", any(fmt.Sprintf("edit %s %d", d.name, n))

		if rng.IntN(2) == 0 {
			field, value = "amount_cents", 1+rng.IntN(5000)
		}

		d.queue("transaction", uid, "upsert", map[string]any{field: value})
	case k < 9 && len(accounts) > 0:
		d.queue("account", accounts[rng.IntN(len(accounts))], "upsert", map[string]any{"name": fmt.Sprintf("renamed %s %d", d.name, n)})
	case len(txns) > 0 && rng.IntN(2) == 0:
		uid := txns[rng.IntN(len(txns))]
		d.queue("transaction", uid, "delete", nil)
		delete(d.replica["transaction"], uid)

		d.deleted[uid] = "transaction"
	case len(d.deleted) > 0:
		for uid, entity := range d.deleted {
			d.queue(entity, uid, "restore", nil)
			delete(d.deleted, uid)

			break
		}
	}
}

func (d *simDevice) learn(entity, uid string, seq int64, row map[string]any, gone bool) {
	d.seen[uid] = max(d.seen[uid], seq)

	if gone {
		delete(d.replica[entity], uid)
	} else if row != nil {
		d.replica.put(entity, uid, row)
	}
}

// sync pushes the queue, learns from the answers, then pulls to the end.
func (d *simDevice) sync(t *testing.T) {
	t.Helper()

	for len(d.pending) > 0 {
		batch := d.pending[:min(len(d.pending), 150)]
		d.pending = d.pending[len(batch):]

		results, err := d.svc.Push(d.ctx, d.actor, d.dev, batch)
		if err != nil {
			t.Fatalf("%s push: %v", d.name, err)
		}

		for i, r := range results {
			d.stats[r.Status]++

			switch r.Status {
			case syncproto.StatusApplied, syncproto.StatusMerged:
				d.learn(r.Entity, r.UID, r.Seq, r.Row, r.Row == nil && batch[i].Op == "delete")
			case syncproto.StatusDuplicate:
				if r.ReplacedByUID != "" {
					delete(d.replica[r.Entity], r.UID)
				} else {
					d.learn(r.Entity, r.UID, r.Seq, r.Row, false)
				}
			case syncproto.StatusRejected:
				// What the phone believes about this row is wrong: resync it.
				d.stale = true
			}
		}
	}

	if d.stale {
		d.resync(t)
		return
	}

	d.pull(t)
}

func (d *simDevice) pull(t *testing.T) {
	t.Helper()

	for {
		page, err := d.svc.Pull(d.ctx, d.actor, d.dev, d.cursor, 40)
		if err != nil {
			t.Fatalf("%s pull: %v", d.name, err)
		}

		for _, it := range page.Changes {
			d.learn(it.Entity, it.UID, it.Seq, it.Row, it.Op == "delete")
		}

		d.cursor = page.NextCursor

		if !page.HasMore {
			return
		}
	}
}

// resync rebuilds the replica from a snapshot, as a device does when a push
// told it that its local state is wrong.
func (d *simDevice) resync(t *testing.T) {
	t.Helper()

	d.replica, d.stale, d.deleted = replica{}, false, map[string]string{}
	id, cursor := "", int64(0)

	for _, e := range syncproto.Order() {
		for after := ""; ; {
			page, err := d.svc.Snapshot(d.ctx, d.actor, e.Name, after, id, 50)
			if err != nil {
				t.Fatalf("%s snapshot: %v", d.name, err)
			}

			id, cursor = page.SnapshotID, page.SnapshotCursor

			for _, row := range page.Rows {
				uid := row["uid"].(string)
				d.replica.put(e.Name, uid, row)
				d.seen[uid] = max(d.seen[uid], 0)
			}

			if !page.HasMore {
				break
			}

			after = page.NextAfter
		}
	}

	d.cursor = cursor
	d.pull(t)
}

// TestSyncConverges runs random interleavings of two phones' offline changes,
// pushes and pulls; after a final sync both phones and the server hold the
// same rows. Every run is seeded and prints its seed.
func TestSyncConverges(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ws := service(t, app)

	fin, err := finance.New(app.App, ws)
	if err != nil {
		t.Fatal(err)
	}

	svc := &syncproto.Service{DB: ws.DB, WS: ws, Fin: fin, Now: app.Now}

	const runs = 200

	stats := map[string]int{}

	for seed := range uint64(runs) {
		convergeRun(t, app, svc, seed, stats)

		if t.Failed() {
			t.Fatalf("seed %d did not converge", seed)
		}
	}

	// The generator must really provoke merges, duplicates and rejections.
	t.Logf("results over %d runs: %v", runs, stats)

	for _, status := range []string{syncproto.StatusApplied, syncproto.StatusMerged, syncproto.StatusRejected} {
		if stats[status] == 0 {
			t.Errorf("no mutation ended %s in %d runs: the property test proves nothing", status, runs)
		}
	}
}

func convergeRun(t *testing.T, app *vukemtest.App, svc *syncproto.Service, seed uint64, stats map[string]int) {
	t.Helper()

	rng := rand.New(rand.NewPCG(seed, 18))
	ana := signUp(t, app, "Ana", fmt.Sprintf("ana%d@example.com", seed))
	makePro(t, app, ana)

	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	mk := func(name string) *simDevice {
		actor := syncproto.Actor{WorkspaceID: ana.Workspace.ID, UserID: ana.User.ID, TokenID: "token-" + name, Role: models.RoleOwner}

		dev, err := svc.RegisterDevice(ctx, actor, name, "ios")
		if err != nil {
			t.Fatal(err)
		}

		d := &simDevice{
			name: name, svc: svc, ctx: ctx, actor: actor, dev: dev, replica: replica{}, deleted: map[string]string{},
			seen: map[string]int64{}, stats: stats,
		}
		d.resync(t)

		return d
	}

	devices := []*simDevice{mk("a"), mk("b")}
	steps := 20 + rng.IntN(40)

	for n := range steps {
		d := devices[rng.IntN(2)]

		if rng.IntN(4) == 0 {
			d.sync(t)
		} else {
			d.act(rng, n)
		}
	}

	// A final sync of each phone, twice: the second sees what the first learned of the other.
	for range 2 {
		for _, d := range devices {
			d.sync(t)
		}
	}

	auditor := mk("auditor")

	for _, d := range devices {
		if len(d.pending) != 0 {
			t.Errorf("seed %d: %s still has %d pending mutations", seed, d.name, len(d.pending))
		}

		compare(t, seed, d, auditor)
	}
}

// compare checks that a phone holds the server's live rows, and only those.
func compare(t *testing.T, seed uint64, d, auditor *simDevice) {
	t.Helper()

	for _, e := range changes.Entities() {
		got, want := d.replica[e.Name], auditor.replica[e.Name]
		if len(got) != len(want) {
			t.Errorf("seed %d: %s holds %d %s rows, the server %d", seed, d.name, len(got), e.Name, len(want))
		}

		for uid, row := range want {
			if !reflect.DeepEqual(got[uid], row) {
				t.Errorf("seed %d: %s %s %s = %v, the server has %v", seed, d.name, e.Name, uid, got[uid], row)
			}
		}
	}
}
