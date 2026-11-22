package tests

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/jobs"
)

// makeDue ends the grace period of the workspace of w, the way time would.
func makeDue(t *testing.T, app *vukemtest.App, w who) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	past := time.Now().UTC().Add(-time.Hour)
	if _, err := db.ExecContext(t.Context(),
		"UPDATE workspaces SET state = 'pending_deletion', purge_at = ?, deletion_requested_at = ?, deletion_requested_by = ? WHERE id = ?",
		past, past.Add(-720*time.Hour), w.User.ID, w.Workspace.ID); err != nil {
		t.Fatal(err)
	}
}

// TestSyncDeletedWorkspace: a workspace that waits to be purged, and one that
// was purged, both answer 410 workspace_deleted on sync, so the phone tells the
// person and erases it locally. An unknown token is still a plain 401.
func TestSyncDeletedWorkspace(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	code := pairingCode(t, f.who)
	phoneToken := rawToken(t, f.app, f.who.User, f.who.Workspace.ID, "sync")
	phone := newSyncDevice(t, f.app, phoneToken, "phone")

	// Before: the device syncs.
	phone.client().Get("/api/v1/sync/pull?cursor=0&limit=10").AssertOK()

	makeDue(t, f.app, f.who)

	t.Run("while it waits to be purged", func(t *testing.T) {
		for _, path := range []string{"/api/v1/sync/info", "/api/v1/sync/devices", "/api/v1/sync/pull?cursor=0&limit=10", "/api/v1/sync/snapshot?entity=account"} {
			assertProblem(t, phone.client().Get(path), http.StatusGone, "workspace_deleted")
		}

		assertProblem(t, phone.push(mut("account", newULID(), "upsert", 0, map[string]any{"name": "Late", "kind": "cash"})),
			http.StatusGone, "workspace_deleted")

		// A pairing code made before the request cannot enrol a new device now.
		assertProblem(t, pairClient(t, f.app).PostJSON("/api/v1/sync/pair", pairBody(code, "second phone")), http.StatusGone, "workspace_deleted")
	})

	if err := jobs.HandlePrivacyPurge(f.app.App)(t.Context(), jobs.PrivacyPurge{}); err != nil {
		t.Fatal(err)
	}

	t.Run("after the purge", func(t *testing.T) {
		// The token no longer exists, but its public id is remembered.
		assertProblem(t, phone.client().Get("/api/v1/sync/pull?cursor=0&limit=10"), http.StatusGone, "workspace_deleted")
		assertProblem(t, phone.push(mut("account", newULID(), "upsert", 0, map[string]any{"name": "Late", "kind": "cash"})),
			http.StatusGone, "workspace_deleted")

		// Nothing else is told: a token nobody issued is still unknown.
		forged := "vf_" + strings.Repeat("A", 26) + "." + strings.Repeat("B", 26)
		assertProblem(t, apiClient(t, f.app, forged).WithHeader("X-Sync-Protocol", "1").Get("/api/v1/sync/info"), http.StatusUnauthorized, "unauthorized")
	})
}
