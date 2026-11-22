package tests

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// pairingCode presses "Add device" as w and returns the code the page shows.
func pairingCode(t *testing.T, w who) string {
	t.Helper()

	res := w.Client.Inertia("").PostForm("/devices/pairing", nil).AssertOK()

	return uidOf(t, res, "props.pairing.code")
}

func pairBody(code, name string) map[string]any {
	return map[string]any{"code": code, "device_name": name, "platform": "android"}
}

func pairClient(t *testing.T, app *vukemtest.App) *vukemtest.Client {
	t.Helper()

	return app.Client(t).WithHeader("X-Sync-Protocol", "1")
}

func TestPairing(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	code := pairingCode(t, f.who)

	if len(code) != 9 || code[4] != '-' {
		t.Fatalf("code = %q, want XXXX-XXXX", code)
	}

	// The server address the page shows is the one the request came to.
	f.who.Client.Inertia("").PostForm("/devices/pairing", nil).AssertOK().AssertJSONPath("props.pairing.server", "http://example.com")

	// Typed in lower case and without the dash, the code still pairs.
	typed := strings.ToLower(strings.ReplaceAll(code, "-", " "))
	res := pairClient(t, f.app).PostJSON("/api/v1/sync/pair", pairBody(typed, "Ana's phone")).AssertCreated()

	token := uidOf(t, res, "token")
	device := uidOf(t, res, "device_id")

	if len(device) != 26 {
		t.Fatalf("device_id = %q", device)
	}

	res.AssertJSONPath("workspace.name", f.who.Workspace.Name).AssertJSONPath("workspace.currency", f.who.Workspace.Currency)

	if _, ok := res.JSONPath("server_time"); !ok {
		t.Error("no server_time")
	}

	// The token works for sync with the device it came with, and for nothing else.
	phone := apiClient(t, f.app, token).WithHeader("X-Sync-Protocol", "1").WithHeader("X-Device-Id", device)
	phone.Get("/api/v1/sync/pull").AssertOK()
	assertProblem(t, phone.Get("/api/v1/accounts"), 403, "missing_scope")

	// The Devices page lists it, and the Tokens page names the token after it.
	f.who.Client.Inertia("").Get("/devices").AssertOK().AssertJSONCount("props.devices", 1).
		AssertJSONPath("props.devices.0.name", "Ana's phone").AssertJSONPath("props.pairing", nil)
	f.who.Client.Inertia("").Get("/tokens").AssertOK().AssertJSONPath("props.tokens.0.name", "Phone Ana's phone")

	// Revoking the device stops the token's sync at once.
	f.who.Client.Delete("/devices/" + device).AssertRedirect("/devices")
	assertProblem(t, phone.Get("/api/v1/sync/pull"), 403, "device_revoked")
}

// The token is limited by the role of the person who made the code.
func TestPairing_TokenHasTheRoleCeiling(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	vera := viewerOf(t, f.app, f.who)
	code := pairingCode(t, vera)

	res := pairClient(t, f.app).PostJSON("/api/v1/sync/pair", pairBody(code, "Vera's phone")).AssertCreated()
	phone := syncDeviceClient{t: t, app: f.app, token: uidOf(t, res, "token"), Device: uidOf(t, res, "device_id")}

	phone.pull(0, 10)

	got := phone.pushOK(mut("account", newULID(), "upsert", 0, map[string]any{"name": "Nope", "kind": "checking"}))[0]
	if got.Status != "rejected" || got.code() != "forbidden" {
		t.Errorf("a viewer's phone wrote: %+v %+v", got, got.Error)
	}
}

func TestPairing_ExpiredCodeFails(t *testing.T) {
	t.Parallel()

	app, clock := clockApp(t, "2026-11-24")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	code := pairingCode(t, ana)

	clock.Advance(syncproto.PairingTTL + time.Second)
	assertProblem(t, pairClient(t, app).PostJSON("/api/v1/sync/pair", pairBody(code, "late")), 422, "invalid_pairing_code")

	// A code that is still inside its 10 minutes pairs.
	fresh := pairingCode(t, ana)

	clock.Advance(syncproto.PairingTTL - time.Minute)
	pairClient(t, app).PostJSON("/api/v1/sync/pair", pairBody(fresh, "on time")).AssertCreated()
}

func TestPairing_Validation(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	code := pairingCode(t, f.who)
	c := pairClient(t, f.app)

	assertProblem(t, c.PostJSON("/api/v1/sync/pair", pairBody(code, "  ")), 422, "validation_failed")
	assertProblem(t, c.PostJSON("/api/v1/sync/pair", map[string]any{"code": code, "device_name": "x", "platform": "palm"}), 422, "validation_failed")
	assertProblem(t, c.PostJSON("/api/v1/sync/pair", pairBody("SHORT", "x")), 422, "invalid_pairing_code")
	assertProblem(t, c.PostJSON("/api/v1/sync/pair", pairBody("0O1I-0O1I", "x")), 422, "invalid_pairing_code")

	// A refused request for a bad name does not spend the code.
	c.PostJSON("/api/v1/sync/pair", pairBody(code, "ok")).AssertCreated()

	// A person holds at most five live codes.
	for range syncproto.PairingMaxLive {
		pairingCode(t, f.who)
	}

	f.who.Client.PostForm("/devices/pairing", nil).AssertRedirect("/devices")
}

func TestSecurity_PairingCodeSingleUse(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	code := pairingCode(t, f.who)
	c := pairClient(t, f.app)

	c.PostJSON("/api/v1/sync/pair", pairBody(code, "first")).AssertCreated()

	again := c.PostJSON("/api/v1/sync/pair", pairBody(code, "second"))
	assertProblem(t, again, 422, "invalid_pairing_code")

	// A used code and a code that never existed look the same.
	unknown := c.PostJSON("/api/v1/sync/pair", pairBody("ABCD-EFGH", "third"))
	assertProblem(t, unknown, 422, "invalid_pairing_code")

	for _, path := range []string{"code", "detail", "status", "title"} {
		a, _ := again.JSONPath(path)
		u, _ := unknown.JSONPath(path)

		if a != u {
			t.Errorf("%s: a used code answers %v and an unknown one %v", path, a, u)
		}
	}

	if n := rowCount(t, f.app, "sync_devices"); n != 1 {
		t.Errorf("devices = %d, want 1", n)
	}

	// Only the hash of a code is stored.
	if n := rowCountWhere(t, f.app, "pairing_codes", "code_hash = '"+strings.ReplaceAll(code, "-", "")+"'"); n != 0 {
		t.Error("the code is stored in clear")
	}
}

func TestSecurity_PairingConcurrent(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	code := pairingCode(t, f.who)

	acc, err := accounts.New(f.app.App)
	if err != nil {
		t.Fatal(err)
	}

	ws, err := workspaces.New(f.app.App)
	if err != nil {
		t.Fatal(err)
	}

	svc := &syncproto.Service{DB: ws.DB, WS: ws, Tokens: acc.Tokens}

	var (
		wg     sync.WaitGroup
		paired atomic.Int32
		start  = make(chan struct{})
	)

	for i := range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start

			if _, err := svc.Pair(context.Background(), code, "phone "+itoa(int64(i)), "ios"); err == nil {
				paired.Add(1)
			} else if !syncproto.IsInvalidPairing(err) {
				t.Errorf("attempt %d: %v", i, err)
			}
		}()
	}

	close(start)
	wg.Wait()

	if paired.Load() != 1 {
		t.Fatalf("%d attempts paired with one code, want 1", paired.Load())
	}

	if n := rowCount(t, f.app, "sync_devices"); n != 1 {
		t.Errorf("devices = %d, want 1", n)
	}
}

func TestSecurity_PairingRateLimit(t *testing.T) {
	t.Parallel()

	f := newSyncFixture(t)
	code := pairingCode(t, f.who)
	c := pairClient(t, f.app)

	for i := range syncproto.PairAttemptsPerMinute {
		assertProblem(t, c.PostJSON("/api/v1/sync/pair", pairBody("ABCD-EFG"+string(rune('H'+i)), "guess")), 422, "invalid_pairing_code")
	}

	// Past the limit even the right code is refused, with a wait time.
	res := c.PostJSON("/api/v1/sync/pair", pairBody(code, "right"))
	assertProblem(t, res, 429, "rate_limited")

	if res.Header("Retry-After") == "" {
		t.Error("no Retry-After")
	}

	if n := rowCount(t, f.app, "sync_devices"); n != 0 {
		t.Errorf("a refused attempt paired %d devices", n)
	}
}

// The pairing route is in the document without a token requirement.
func TestPairing_RouteIsPublicInTheDocument(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	res := app.Client(t).Get("/api/v1/openapi.json").AssertOK()

	if _, ok := res.JSONPath("paths./sync/pair.post.operationId"); !ok {
		t.Fatalf("no sync.pair in the document: %s", res.Body())
	}

	if v, ok := res.JSONPath("paths./sync/pair.post.security"); ok && v != nil {
		t.Errorf("sync.pair names a security scheme: %v", v)
	}
}
