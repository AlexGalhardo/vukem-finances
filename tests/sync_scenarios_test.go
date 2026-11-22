package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// scenarioDir holds the conformance scenarios: language-neutral files that
// the mobile client engine (epic 20) runs against its own implementation.
// docs/specs/sync-protocol.md describes the format.
const scenarioDir = "../testdata/sync"

type scenarioStep struct {
	Device string         `json:"device"`
	Action string         `json:"action"`
	Note   string         `json:"note"`
	Input  map[string]any `json:"input"`
	Expect map[string]any `json:"expect"`
}

type scenario struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Workspace   struct {
		Plan    string            `json:"plan"`
		Devices map[string]string `json:"devices"`
	} `json:"workspace"`
	Steps []scenarioStep `json:"steps"`
	Final struct {
		DevicesEqual []string                             `json:"devices_equal"`
		Server       map[string]map[string]map[string]any `json:"server"`
		Absent       []string                             `json:"absent"`
	} `json:"final"`
}

// phone is a simulated client: its replica, its cursor and the newest seq it
// has seen for each row (the base_seq of its next edit).
type phone struct {
	syncDeviceClient
	replica replica
	cursor  int64
	seen    map[string]int64
}

// subset reports why got does not contain want ("" when it does). Maps match
// by the keys of want, arrays by length and position, scalars by value; JSON
// null in want matches null or a missing key.
func subset(got, want any, path string) string {
	switch w := want.(type) {
	case nil:
		if got != nil {
			return fmt.Sprintf("%s = %v, want null", path, got)
		}
	case map[string]any:
		g, _ := got.(map[string]any)
		if got != nil && g == nil {
			return fmt.Sprintf("%s = %v, want an object", path, got)
		}

		for k, wv := range w {
			if why := subset(g[k], wv, path+"."+k); why != "" {
				return why
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return fmt.Sprintf("%s = %v, want %d items", path, got, len(w))
		}

		for i := range w {
			if why := subset(g[i], w[i], fmt.Sprintf("%s[%d]", path, i)); why != "" {
				return why
			}
		}
	default:
		if !reflect.DeepEqual(got, want) {
			return fmt.Sprintf("%s = %v, want %v", path, got, want)
		}
	}

	return ""
}

func decode(t *testing.T, res *vukemtest.Response) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal([]byte(res.Body()), &out); err != nil {
		t.Fatalf("answer is not JSON: %v: %s", err, res.Body())
	}

	return out
}

// checkHTTP compares the status and the problem code of an answer.
func checkHTTP(t *testing.T, step scenarioStep, res *vukemtest.Response) bool {
	t.Helper()

	want := 200
	if v, ok := step.Expect["status"].(float64); ok {
		want = int(v)
	}

	if res.Status() != want {
		t.Errorf("%s %s: status %d, want %d: %s", step.Device, step.Action, res.Status(), want, res.Body())
		return false
	}

	if code, ok := step.Expect["code"].(string); ok {
		if got := decode(t, res)["code"]; got != code {
			t.Errorf("%s %s: code %v, want %s", step.Device, step.Action, got, code)
			return false
		}
	}

	return want == 200
}

func (p *phone) resolveBase(m map[string]any) {
	if m["base_seq"] == "$seen" {
		m["base_seq"] = p.seen[m["uid"].(string)]
	}
}

func (p *phone) learn(entity, uid string, seq int64, row map[string]any, deleted bool) {
	p.seen[uid] = max(p.seen[uid], seq)

	if deleted {
		delete(p.replica[entity], uid)
	} else if row != nil {
		p.replica.put(entity, uid, row)
	}
}

func (p *phone) push(t *testing.T, step scenarioStep) {
	t.Helper()

	muts, _ := step.Input["mutations"].([]any)
	for _, m := range muts {
		p.resolveBase(m.(map[string]any))
	}

	res := p.client().PostJSON("/api/v1/sync/push", map[string]any{"mutations": muts})
	if !checkHTTP(t, step, res) {
		return
	}

	body := decode(t, res)
	if why := subset(body["results"], step.Expect["results"], "results"); step.Expect["results"] != nil && why != "" {
		t.Errorf("%s push: %s\nanswer: %s", step.Device, why, res.Body())
	}

	results, _ := body["results"].([]any)
	for i, r := range results {
		r := r.(map[string]any)

		entity, _ := r["entity"].(string)
		uid, _ := r["uid"].(string)
		seq, _ := r["seq"].(float64)
		row, _ := r["row"].(map[string]any)

		if status := r["status"]; status == "applied" || status == "merged" || (status == "duplicate" && r["replaced_by_uid"] == nil) {
			p.learn(entity, uid, int64(seq), row, row == nil && muts[i].(map[string]any)["op"] == "delete")
		}

		// A duplicate of another row: the client adopts that uid.
		if adopted, ok := r["replaced_by_uid"].(string); ok {
			p.seen[adopted] = max(p.seen[adopted], int64(seq))
		}
	}
}

// pullInto reads pages until the end and applies them; it returns the items
// of every page and false when the server refused.
func (p *phone) pullInto(t *testing.T, step scenarioStep, limit int, from int64) ([]map[string]any, bool) {
	t.Helper()

	var all []map[string]any

	for range 1000 {
		res := p.client().Get(fmt.Sprintf("/api/v1/sync/pull?cursor=%d&limit=%d", from, limit))
		if !checkHTTP(t, step, res) {
			return nil, false
		}

		body := decode(t, res)
		items, _ := body["changes"].([]any)

		for _, raw := range items {
			it := raw.(map[string]any)
			all = append(all, it)

			entity, uid := it["entity"].(string), it["uid"].(string)
			seq, _ := it["seq"].(float64)
			row, _ := it["row"].(map[string]any)
			p.learn(entity, uid, int64(seq), row, it["op"] == "delete")
		}

		next, _ := body["next_cursor"].(float64)
		p.cursor, from = int64(next), int64(next)

		if more, _ := body["has_more"].(bool); !more {
			return all, true
		}

		if _, paged := step.Expect["changes"]; paged {
			return all, true
		}
	}

	t.Fatal("pull never ended")

	return nil, false
}

func (p *phone) pull(t *testing.T, step scenarioStep) {
	t.Helper()

	limit := 100
	if v, ok := step.Input["limit"].(float64); ok {
		limit = int(v)
	}

	from := p.cursor
	if step.Input["from"] == "start" {
		from = 0
	}

	items, ok := p.pullInto(t, step, limit, from)
	if !ok {
		return
	}

	if want, ok := step.Expect["changes"]; ok {
		got := make([]any, len(items))
		for i := range items {
			got[i] = items[i]
		}

		if why := subset(got, want, "changes"); why != "" {
			t.Errorf("%s pull: %s", step.Device, why)
		}
	}

	if want, ok := step.Expect["contains"].([]any); ok {
		for _, w := range want {
			w := w.(map[string]any)

			idx := slices.IndexFunc(items, func(it map[string]any) bool { return it["uid"] == w["uid"] })
			if idx < 0 {
				t.Errorf("%s pull: no item for %v", step.Device, w["uid"])
				continue
			}

			if why := subset(items[idx], w, "item "+fmt.Sprint(w["uid"])); why != "" {
				t.Errorf("%s pull: %s", step.Device, why)
			}
		}
	}

	if want, ok := step.Expect["absent_uids"].([]any); ok {
		for _, w := range want {
			if slices.ContainsFunc(items, func(it map[string]any) bool { return it["uid"] == w }) {
				t.Errorf("%s pull: %v must not be there", step.Device, w)
			}
		}
	}

	if step.Expect["parents_first"] == true {
		rank := map[string]int{}
		for i, e := range []string{"account", "category", "tag", "goal"} {
			rank[e] = i
		}

		last := -1

		for _, it := range items {
			r, parent := rank[it["entity"].(string)]
			if parent && r < last {
				t.Errorf("%s pull: %v comes after a row that is not a root", step.Device, it["uid"])
			}

			if !parent {
				last = len(rank)
			}
		}
	}

	if want, ok := step.Expect["has_more"].(bool); ok && want {
		t.Logf("has_more is checked by TestSyncPull; a scenario pull reads every page")
	}
}

// snapshot rebuilds the replica from scratch, the way a device that was told
// to resync does, and returns it with the cursor the device ends at.
func (p *phone) snapshot(t *testing.T) (replica, int64) {
	t.Helper()

	return rebuildFrom(t, p.syncDeviceClient, 100, nil)
}

func (p *phone) rebuildFromSnapshot(t *testing.T, step scenarioStep) {
	t.Helper()

	p.replica, p.cursor = p.snapshot(t)

	if want, ok := step.Expect["replica_contains"].(map[string]any); ok {
		for entity, uids := range want {
			for _, uid := range uids.([]any) {
				if _, found := p.replica[entity][uid.(string)]; !found {
					t.Errorf("%s snapshot: no %s %v in the rebuilt replica", step.Device, entity, uid)
				}
			}
		}
	}
}

type scenarioRun struct {
	t       *testing.T
	app     *vukemtest.App
	owner   who
	phones  map[string]*phone
	auditor *phone
}

func (r *scenarioRun) admin(step scenarioStep) {
	r.t.Helper()

	db, err := r.app.DB()
	if err != nil {
		r.t.Fatal(err)
	}

	exec := func(query string, args ...any) {
		r.t.Helper()

		if _, err := db.SQL().ExecContext(r.t.Context(), query, args...); err != nil {
			r.t.Fatalf("%s: %v", query, err)
		}
	}

	switch step.Input["do"] {
	case "revoke_device":
		exec("UPDATE sync_devices SET revoked_at = ? WHERE device_id = ?", time.Now().UTC(), r.phones[step.Input["target"].(string)].Device)
	case "compact":
		exec(`INSERT INTO sync_state (workspace_id, compacted_through_seq, created_at, updated_at)
			SELECT ?, COALESCE(MAX(seq), 0), datetime('now'), datetime('now') FROM changes WHERE workspace_id = ?
			ON CONFLICT (workspace_id) DO UPDATE SET compacted_through_seq = excluded.compacted_through_seq`,
			r.owner.Workspace.ID, r.owner.Workspace.ID)
	case "age_tombstones":
		old := time.Now().UTC().Add(-31 * 24 * time.Hour)

		for _, e := range changes.Entities() {
			exec("UPDATE "+e.Table+" SET deleted_at = ? WHERE deleted_at IS NOT NULL AND workspace_id = ?", old, r.owner.Workspace.ID)
		}
	default:
		r.t.Fatalf("unknown admin step %v", step.Input["do"])
	}
}

// converge syncs a phone to the end, resyncing when the server says so.
func (p *phone) converge(t *testing.T) {
	t.Helper()

	res := p.client().Get(fmt.Sprintf("/api/v1/sync/pull?cursor=%d&limit=100", p.cursor))
	if res.Status() == 410 {
		p.rebuildFromSnapshot(t, scenarioStep{Device: "final"})
		return
	}

	step := scenarioStep{Device: "final", Action: "pull", Expect: map[string]any{}}
	p.pullInto(t, step, 100, p.cursor)
}

func TestSyncScenarios(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob(filepath.Join(scenarioDir, "*.json"))
	if err != nil || len(files) < 12 {
		t.Fatalf("scenarios: %v, %d files, want at least 12", err, len(files))
	}

	for _, file := range files {
		t.Run(strings.TrimSuffix(filepath.Base(file), ".json"), func(t *testing.T) {
			t.Parallel()

			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}

			var sc scenario
			if err := json.Unmarshal(raw, &sc); err != nil {
				t.Fatalf("%s: %v", file, err)
			}

			runScenario(t, sc)
		})
	}
}

func runScenario(t *testing.T, sc scenario) {
	t.Helper()

	app := newApp(t)
	owner := signUp(t, app, "Ana", "ana@example.com")

	if sc.Workspace.Plan == "pro" {
		makePro(t, app, owner)
	}

	run := &scenarioRun{t: t, app: app, owner: owner, phones: map[string]*phone{}}

	newPhone := func(name string, user models.User) *phone {
		token := rawToken(t, app, user, owner.Workspace.ID, "sync")

		return &phone{
			syncDeviceClient: newSyncDevice(t, app, token, name),
			replica:          replica{},
			seen:             map[string]int64{},
		}
	}

	roleUsers := map[string]models.User{"owner": owner.User}

	for name, role := range sc.Workspace.Devices {
		user, ok := roleUsers[role]
		if !ok {
			switch role {
			case "viewer":
				user = viewerOf(t, app, owner).User
			default:
				user = newVerifiedUser(t, app, strings.ToUpper(role), role+"@example.com")
				join(t, app, owner, user, role)
			}

			roleUsers[role] = user
		}

		run.phones[name] = newPhone(name, user)
	}

	run.auditor = newPhone("auditor", owner.User)

	for i, step := range sc.Steps {
		p := run.phones[step.Device]
		if p == nil && step.Action != "admin" {
			t.Fatalf("step %d: no device %q", i+1, step.Device)
		}

		switch step.Action {
		case "push":
			p.push(t, step)
		case "pull":
			p.pull(t, step)
		case "snapshot":
			p.rebuildFromSnapshot(t, step)
		case "admin":
			run.admin(step)
		default:
			t.Fatalf("step %d: unknown action %q", i+1, step.Action)
		}

		if t.Failed() {
			t.Fatalf("scenario %s failed at step %d (%s %s): %s", sc.Name, i+1, step.Device, step.Action, step.Note)
		}
	}

	// The server's live state, read by a device that has never synced.
	server, _ := run.auditor.snapshot(t)

	for entity, rows := range sc.Final.Server {
		for uid, want := range rows {
			got, ok := server[entity][uid]
			if !ok {
				t.Errorf("final: no %s %s on the server", entity, uid)
				continue
			}

			if why := subset(got, want, entity+" "+uid); why != "" {
				t.Errorf("final: %s", why)
			}
		}
	}

	for _, uid := range sc.Final.Absent {
		for entity, rows := range server {
			if _, found := rows[uid]; found {
				t.Errorf("final: %s %s must not exist", entity, uid)
			}
		}
	}

	var first replica

	for _, name := range sc.Final.DevicesEqual {
		p := run.phones[name]
		p.converge(t)

		if first == nil {
			first = p.replica
			sameReplica(t, p.replica, server)

			continue
		}

		sameReplica(t, p.replica, first)
	}
}
