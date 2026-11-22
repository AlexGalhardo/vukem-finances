package tests

import (
	"strings"
	"testing"
	"time"
)

func TestSyncLimits(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d, other := f.device(t, "phone"), f.device(t, "tablet")

	// 60 pulls a minute per device; the 61st is refused with Retry-After.
	for i := range 60 {
		if res := d.client().Get("/api/v1/sync/pull?cursor=0&limit=1"); res.Status() != 200 {
			t.Fatalf("pull %d = %d", i+1, res.Status())
		}
	}

	res := d.client().Get("/api/v1/sync/pull?cursor=0&limit=1")
	assertProblem(t, res, 429, "rate_limited")

	if res.Header("Retry-After") == "" {
		t.Error("a limited pull sends no Retry-After")
	}

	// The limit is per device: another device of the same token still pulls.
	other.client().Get("/api/v1/sync/pull?cursor=0&limit=1").AssertOK()

	// 30 pushes a minute per device.
	for i := range 30 {
		if res := d.client().PostJSON("/api/v1/sync/push", map[string]any{"mutations": []any{}}); res.Status() != 200 {
			t.Fatalf("push %d = %d", i+1, res.Status())
		}
	}

	assertProblem(t, d.client().PostJSON("/api/v1/sync/push", map[string]any{"mutations": []any{}}), 429, "rate_limited")
}

func TestSyncPushSizeLimits(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "phone")

	big := strings.Repeat("x", 1<<20+1)
	assertProblem(t, d.push(mut("account", newULID(), "upsert", 0, map[string]any{"name": big, "kind": "cash"})), 413, "bad_request")

	// A body that is not JSON, or has the wrong shape, is a 4xx and writes nothing.
	for _, body := range []string{`{`, `[]`, `{"mutations": "x"}`, `{"mutations": [1]}`} {
		res := d.client().Do("POST", "/api/v1/sync/push", strings.NewReader(body), "application/json")
		if res.Status() < 400 || res.Status() >= 500 {
			t.Errorf("body %q = %d, want a 4xx", body, res.Status())
		}
	}

	// A device may not push more than 5000 mutations an hour.
	db, _ := f.app.DB()
	now := time.Now().UTC()

	for i := range 4999 {
		if _, err := db.SQL().ExecContext(t.Context(),
			"INSERT INTO sync_ops (workspace_id, op_id, device_id, result, created_at, updated_at) VALUES (?, ?, ?, '{}', ?, ?)",
			f.who.Workspace.ID, newULID()+"", d.Device, now, now); err != nil {
			t.Fatal(i, err)
		}
	}

	d.pushOK(mut("tag", newULID(), "upsert", 0, map[string]any{"name": "last one"}))
	assertProblem(t, d.push(mut("tag", newULID(), "upsert", 0, map[string]any{"name": "one too many"})), 429, "rate_limited")
}

func TestSyncRevokedDevice(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d, keep := f.device(t, "phone"), f.device(t, "tablet")

	d.pull(0, 1)

	db, _ := f.app.DB()
	if _, err := db.SQL().ExecContext(t.Context(), "UPDATE sync_devices SET revoked_at = ? WHERE device_id = ?", time.Now().UTC(), d.Device); err != nil {
		t.Fatal(err)
	}

	assertProblem(t, d.client().Get("/api/v1/sync/pull"), 403, "device_revoked")
	assertProblem(t, d.client().Get("/api/v1/sync/snapshot?entity=account"), 403, "device_revoked")
	assertProblem(t, d.push(mut("tag", newULID(), "upsert", 0, map[string]any{"name": "x"})), 403, "device_revoked")

	// Another device of the same token is untouched.
	keep.pull(0, 1)
}

func TestSyncTokenRevoked(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	secret := makeToken(t, f.app, f.who, 30, "sync")
	d := newSyncDevice(t, f.app, secret, "phone")

	d.pull(0, 1)
	f.who.Client.Delete("/tokens/" + tokenID(secret)).AssertRedirect("/tokens")

	assertProblem(t, d.client().Get("/api/v1/sync/pull"), 401, "unauthorized")

	// Its devices are revoked in the database too, not just unreachable.
	db, _ := f.app.DB()

	var n int
	if err := db.SQL().QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM sync_devices WHERE device_id = ? AND revoked_at IS NOT NULL", d.Device).Scan(&n); err != nil || n != 1 {
		t.Errorf("revoked devices of the token = %d (%v), want 1", n, err)
	}
}
