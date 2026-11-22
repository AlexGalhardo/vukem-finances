package tests

import (
	"testing"
)

func TestSyncDevices(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	c := apiClient(t, f.app, f.token).WithHeader("X-Sync-Protocol", "1")

	res := c.PostJSON("/api/v1/sync/devices", map[string]any{"name": "Ana's phone", "platform": "android"}).AssertCreated()
	id := uidOf(t, res, "device_id")

	if len(id) != 26 {
		t.Fatalf("device_id = %q, want a ULID", id)
	}

	c.Get("/api/v1/sync/devices").AssertOK().AssertJSONCount("data", 1).AssertJSONPath("data.0.device_id", id).
		AssertJSONPath("data.0.platform", "android")

	assertProblem(t, c.PostJSON("/api/v1/sync/devices", map[string]any{"name": "  ", "platform": "ios"}), 422, "validation_failed")
	assertProblem(t, c.PostJSON("/api/v1/sync/devices", map[string]any{"name": "x", "platform": "palm"}), 422, "validation_failed")

	// A device belongs to the token that made it: another token of the same
	// person cannot use it.
	other := rawToken(t, f.app, f.who.User, f.who.Workspace.ID, "sync")
	stolen := apiClient(t, f.app, other).WithHeader("X-Sync-Protocol", "1").WithHeader("X-Device-Id", id)
	assertProblem(t, stolen.Get("/api/v1/sync/pull"), 403, "device_revoked")
	assertProblem(t, c.WithHeader("X-Device-Id", "01JD5Z8K3M4N5P6Q7R8S9T0V1W").Get("/api/v1/sync/pull"), 403, "device_revoked")
}

// The Devices page lists the person's own devices and revokes them.
func TestSyncDevicesPage(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	d := f.device(t, "Ana's phone")

	bob := signUp(t, f.app, "Bob", "bob@example.com")
	newSyncDevice(t, f.app, rawToken(t, f.app, bob.User, bob.Workspace.ID, "sync"), "Bob's phone")

	page := f.who.Client.Inertia("").Get("/devices").AssertOK()
	page.AssertJSONCount("props.devices", 1).AssertJSONPath("props.devices.0.name", "Ana's phone").
		AssertJSONPath("props.devices.0.status", "live")

	// Nobody revokes a device of another person.
	bob.Client.Delete("/devices/" + d.Device).AssertNotFound()
	d.client().Get("/api/v1/sync/pull").AssertOK()

	f.who.Client.Delete("/devices/" + d.Device).AssertRedirect("/devices")
	assertProblem(t, d.client().Get("/api/v1/sync/pull"), 403, "device_revoked")

	f.who.Client.Inertia("").Get("/devices").AssertOK().AssertJSONPath("props.devices.0.status", "revoked")
}
