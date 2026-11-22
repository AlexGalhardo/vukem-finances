package tests

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem/ulid"
	"github.com/AlexGalhardo/vukem/vukemtest"
)

var syncUIDs = ulid.NewGenerator() //nolint:gochecknoglobals // test id source

func newULID() string { return syncUIDs.New() }

// syncDeviceClient is one phone: a token, a registered device and the headers
// of the protocol.
type syncDeviceClient struct {
	t      *testing.T
	app    *vukemtest.App
	token  string
	Device string
}

// newSyncDevice registers a device for token.
func newSyncDevice(t *testing.T, app *vukemtest.App, token, name string) syncDeviceClient {
	t.Helper()

	c := apiClient(t, app, token).WithHeader("X-Sync-Protocol", "1")
	res := c.PostJSON("/api/v1/sync/devices", map[string]any{"name": name, "platform": "ios"}).AssertCreated()

	return syncDeviceClient{t: t, app: app, token: token, Device: uidOf(t, res, "device_id")}
}

func (d syncDeviceClient) client() *vukemtest.Client {
	return apiClient(d.t, d.app, d.token).WithHeader("X-Sync-Protocol", "1").WithHeader("X-Device-Id", d.Device)
}

// mut builds a mutation.
func mut(entity, uid, op string, base int64, fields map[string]any) map[string]any {
	return map[string]any{"op_id": newULID(), "entity": entity, "uid": uid, "op": op, "base_seq": base, "fields": fields}
}

func (d syncDeviceClient) push(muts ...map[string]any) *vukemtest.Response {
	d.t.Helper()

	return d.client().PostJSON("/api/v1/sync/push", map[string]any{"mutations": muts})
}

// pushOK pushes and returns the decoded results.
func (d syncDeviceClient) pushOK(muts ...map[string]any) []pushResult {
	d.t.Helper()

	res := d.push(muts...).AssertOK()

	var out struct {
		Results []pushResult `json:"results"`
	}

	if err := json.Unmarshal([]byte(res.Body()), &out); err != nil {
		d.t.Fatalf("push answer: %v: %s", err, res.Body())
	}

	return out.Results
}

type pushResult struct {
	OpID          string         `json:"op_id"`
	Status        string         `json:"status"`
	Entity        string         `json:"entity"`
	UID           string         `json:"uid"`
	Row           map[string]any `json:"row"`
	Version       int64          `json:"version"`
	Seq           int64          `json:"seq"`
	ReplacedByUID string         `json:"replaced_by_uid"`
	Conflicts     []struct {
		Field        string `json:"field"`
		Kind         string `json:"kind"`
		LostValue    any    `json:"lost_value"`
		AppliedValue any    `json:"applied_value"`
	} `json:"conflicts"`
	Effects []struct {
		Entity string `json:"entity"`
		UID    string `json:"uid"`
		Op     string `json:"op"`
		Seq    int64  `json:"seq"`
	} `json:"effects"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Field   string `json:"field"`
	} `json:"error"`
}

func (r pushResult) code() string {
	if r.Error == nil {
		return ""
	}

	return r.Error.Code
}

type pullItem struct {
	Seq     int64          `json:"seq"`
	Entity  string         `json:"entity"`
	UID     string         `json:"uid"`
	Op      string         `json:"op"`
	Version int64          `json:"version"`
	Row     map[string]any `json:"row"`
}

type pullPage struct {
	Changes    []pullItem `json:"changes"`
	NextCursor int64      `json:"next_cursor"`
	HasMore    bool       `json:"has_more"`
}

// pull reads one page after cursor.
func (d syncDeviceClient) pull(cursor int64, limit int) pullPage {
	d.t.Helper()

	res := d.client().Get("/api/v1/sync/pull?cursor=" + itoa(cursor) + "&limit=" + itoa(int64(limit))).AssertOK()

	var page pullPage
	if err := json.Unmarshal([]byte(res.Body()), &page); err != nil {
		d.t.Fatalf("pull answer: %v: %s", err, res.Body())
	}

	return page
}

// pullAll pulls every page after cursor and returns the items and the final cursor.
func (d syncDeviceClient) pullAll(cursor int64) ([]pullItem, int64) {
	d.t.Helper()

	var items []pullItem

	for range 1000 {
		page := d.pull(cursor, 100)
		items, cursor = append(items, page.Changes...), page.NextCursor

		if !page.HasMore {
			return items, cursor
		}
	}

	d.t.Fatal("pull never ended")

	return nil, 0
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// syncFixture is an owner with a token that holds the sync scope.
type syncFixture struct {
	app   *vukemtest.App
	who   who
	token string
}

func newSyncFixture(t *testing.T) syncFixture {
	t.Helper()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	return syncFixture{app: app, who: ana, token: rawToken(t, app, ana.User, ana.Workspace.ID, "sync")}
}

func (f syncFixture) device(t *testing.T, name string) syncDeviceClient {
	t.Helper()

	return newSyncDevice(t, f.app, f.token, name)
}
