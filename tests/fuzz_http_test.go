package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// Fuzz targets that go through the running application: the sync push, the MCP
// JSON-RPC endpoint and the OAuth request parsers (epic 24 task 8). Each builds
// its app once, as the testing.F of the target, and reuses it for every input.
// Every request comes from a new address so no rate limit turns the run into a
// series of 429s that would prove nothing.

var fuzzIP atomic.Uint32 //nolint:gochecknoglobals // a counter of test addresses

func nextIP() string {
	n := fuzzIP.Add(1)

	return fmt.Sprintf("10.%d.%d.%d", n>>16&255, n>>8&255, n&255)
}

// fuzzWorld is a saas app with two workspaces, each with a Pro plan, a row of
// every kind, and the canary text of the second.
type fuzzWorld struct {
	app            *vukemtest.App
	svc            *syncproto.Service
	ana, bob       models.User
	anaWS, bobWS   int64
	anaAcc, bobAcc models.Account
}

func newFuzzWorld(tb testing.TB) *fuzzWorld {
	tb.Helper()

	app := vukemtest.New(tb, bootstrap.New, vukemtest.Options{Env: map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"}})

	ws, err := workspaces.New(app.App)
	if err != nil {
		tb.Fatal(err)
	}

	fin, err := finance.New(app.App, ws)
	if err != nil {
		tb.Fatal(err)
	}

	db, err := app.DB()
	if err != nil {
		tb.Fatal(err)
	}

	ctx := context.Background()
	w := &fuzzWorld{app: app, svc: &syncproto.Service{DB: ws.DB, WS: ws, Fin: fin, Now: app.Now}}

	for i, name := range []string{"Ana", "Bob"} {
		u := models.User{Name: name, Email: strings.ToLower(name) + "@example.com", Password: "not used"}
		u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = time.Now().UTC(), true

		if err := orm.Create(ctx, db, &u); err != nil {
			tb.Fatal(err)
		}

		space, err := ws.Create(ctx, u, name+" household", "USD")
		if err != nil {
			tb.Fatal(err)
		}

		sub := models.WorkspaceSubscription{WorkspaceID: space.ID, StripeSubscriptionID: "sub_fuzz_" + name, Status: "active", PriceID: "p", UnitAmount: 900}
		if err := orm.Create(ctx, db, &sub); err != nil {
			tb.Fatal(err)
		}

		acc, err := workspaces.Scoped[models.Account](workspaces.WithTenant(ctx, space.ID), db).First(ctx)
		if err != nil {
			tb.Fatal(err)
		}

		if i == 0 {
			w.ana, w.anaWS, w.anaAcc = u, space.ID, acc
		} else {
			w.bob, w.bobWS, w.bobAcc = u, space.ID, acc
		}
	}

	if _, err := db.SQL().ExecContext(ctx, "UPDATE accounts SET name = 'BOB-CANARY' WHERE workspace_id = ?", w.bobWS); err != nil {
		tb.Fatal(err)
	}

	return w
}

// state is what must not change in the other workspace: the rows and their versions.
func (w *fuzzWorld) state(tb testing.TB, workspace int64) string {
	tb.Helper()

	db, err := w.app.DB()
	if err != nil {
		tb.Fatal(err)
	}

	var out strings.Builder

	for _, table := range []string{"accounts", "categories", "transactions", "budgets", "memberships"} {
		var n, v int64

		query := "SELECT COUNT(*), COALESCE(SUM(id), 0) FROM " + table + " WHERE workspace_id = ?"
		if table != "memberships" {
			query = "SELECT COUNT(*), COALESCE(SUM(version), 0) FROM " + table + " WHERE workspace_id = ?"
		}

		if err := db.SQL().QueryRowContext(context.Background(), query, workspace).Scan(&n, &v); err != nil {
			tb.Fatal(err)
		}

		fmt.Fprintf(&out, "%s=%d/%d;", table, n, v)
	}

	return out.String()
}

func FuzzSyncPush(f *testing.F) {
	w := newFuzzWorld(f)
	ctx := workspaces.WithTenant(context.Background(), w.anaWS)
	actor := syncproto.Actor{WorkspaceID: w.anaWS, UserID: w.ana.ID, TokenID: "token-fuzz", Role: models.RoleOwner}

	dev, err := w.svc.RegisterDevice(ctx, actor, "fuzz", "ios")
	if err != nil {
		f.Fatal(err)
	}

	seed := func(muts ...map[string]any) {
		b, _ := json.Marshal(muts)
		f.Add(b)
	}

	op := func(entity, uid, kind string, base int64, fields map[string]any) map[string]any {
		return map[string]any{"op_id": newULID(), "entity": entity, "uid": uid, "op": kind, "base_seq": base, "fields": fields}
	}

	seed(op("account", newULID(), "upsert", 0, map[string]any{"name": "Wallet", "kind": "cash"}))
	seed(op("account", w.bobAcc.UID, "upsert", 0, map[string]any{"name": "Hijacked"}))
	seed(op("account", w.bobAcc.UID, "delete", 0, nil))
	seed(op("transaction", newULID(), "upsert", 0, map[string]any{
		"account_uid": w.bobAcc.UID, "kind": "expense", "amount_cents": 100, "occurred_on": "2026-11-10", "description": "x",
	}))
	seed(op("transaction", newULID(), "upsert", 0, map[string]any{
		"account_uid": w.anaAcc.UID, "kind": "expense", "amount_cents": 9007199254740993, "occurred_on": "2026-02-30", "description": "‮",
	}))
	seed(op("category", newULID(), "upsert", 0, map[string]any{"name": strings.Repeat("c", 500), "kind": "bogus"}))
	seed(op("nothing", "x", "explode", -1, map[string]any{"__proto__": 1}))
	seed(op("budget", newULID(), "restore", 1<<62, nil))
	f.Add([]byte(`[{"op_id":"","entity":"","uid":"","op":""}]`))
	f.Add([]byte(`null`))

	before := w.state(f, w.bobWS)

	f.Fuzz(func(t *testing.T, body []byte) {
		var muts []syncproto.MutationIn
		if json.Unmarshal(body, &muts) != nil || len(muts) > syncproto.MaxMutations {
			return
		}

		results, err := w.svc.Push(ctx, actor, dev, muts)
		if err != nil {
			return
		}

		if len(results) != len(muts) {
			t.Fatalf("%d results for %d mutations", len(results), len(muts))
		}

		for _, r := range results {
			switch r.Status {
			case syncproto.StatusApplied, syncproto.StatusMerged, syncproto.StatusDuplicate, syncproto.StatusRejected:
			default:
				t.Fatalf("unknown status %q", r.Status)
			}
		}

		if after := w.state(t, w.bobWS); after != before {
			t.Fatalf("a push of workspace A changed workspace B: %s, was %s", after, before)
		}

		var name string
		if err := func() error {
			db, err := w.app.DB()
			if err != nil {
				return err
			}

			return db.SQL().QueryRowContext(context.Background(), "SELECT name FROM accounts WHERE workspace_id = ? AND uid = ?", w.bobWS, w.bobAcc.UID).Scan(&name)
		}(); err != nil || name != "BOB-CANARY" {
			t.Fatalf("the account of workspace B is %q (%v)", name, err)
		}

		// Money in what was written stays inside the bound.
		db, _ := w.app.DB()

		var over int64
		_ = db.SQL().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM transactions WHERE ABS(amount_cents) > ?", models.MaxCents).Scan(&over)

		if over != 0 {
			t.Fatalf("%d transactions hold an amount beyond MaxCents", over)
		}
	})
}

// fuzzMCP is the remote MCP stack with a person in each workspace.
func FuzzMCPJSONRPC(f *testing.F) {
	s := oauthtest.New(f, nil)
	ana := s.SignUp(f, "Ana", "ana@example.com")
	bob := s.SignUp(f, "Bob", "bob@example.com")

	db, err := s.App.DB()
	if err != nil {
		f.Fatal(err)
	}

	if _, err := db.SQL().ExecContext(context.Background(), "UPDATE accounts SET name = 'BOB-CANARY' WHERE workspace_id = ?", bob.Workspace.ID); err != nil {
		f.Fatal(err)
	}

	tok := s.Authorize(f, ana, ana.Workspace.ID, "accounts:read", "accounts:write", "transactions:read", "transactions:write", "categories:read")

	meta := `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}`

	f.Add("tools/list", "", "2026-07-28", []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{`+meta+`}}`))
	f.Add("tools/call", "accounts_list", "2026-07-28", []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"accounts_list","arguments":{},`+meta+`}}`))
	f.Add("tools/call", "accounts_update", "2026-07-28", []byte(`{"jsonrpc":"2.0","id":"a","method":"tools/call","params":{"name":"accounts_update","arguments":{"uid":"01JD5Z8K3M4N5P6Q7R8S9T0V1W","name":"x"},`+meta+`}}`))
	f.Add("resources/read", "finances://accounts", "2026-07-28", []byte(`{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"finances://accounts",`+meta+`}}`))
	f.Add("prompts/get", "monthly-review", "2026-07-28", []byte(`{"jsonrpc":"2.0","id":4,"method":"prompts/get","params":{"name":"monthly-review","arguments":{"month":"2026-11"},`+meta+`}}`))
	f.Add("completion/complete", "", "2026-07-28", []byte(`{"jsonrpc":"2.0","id":5,"method":"completion/complete","params":{"ref":{"type":"ref/prompt","name":"monthly-review"},"argument":{"name":"month","value":"20"},`+meta+`}}`))
	f.Add("server/discover", "", "2026-07-28", []byte(`{"jsonrpc":"2.0","id":6,"method":"server/discover","params":{`+meta+`}}`))
	f.Add("ping", "", "2026-07-28", []byte(`{"jsonrpc":"2.0","id":null,"method":"ping"}`))
	f.Add("tools/list", "", "1999-01-01", []byte(`[{"jsonrpc":"2.0","id":1,"method":"tools/list"}]`))
	f.Add("x", "", "", []byte(`{"jsonrpc":"1.0","id":{"a":1},"method":7}`))
	f.Add("tools/call", "", "2026-07-28", []byte(`{"jsonrpc":"2.0","id":99999999999999999999,"method":"tools/call","params":{"name":null,"arguments":[1,2,3]}}`))
	f.Add("", "", "", []byte(``))

	f.Fuzz(func(t *testing.T, method, name, version string, body []byte) {
		// A header value with a control character is not a request a client can send.
		for _, v := range []string{method, name, version} {
			if strings.ContainsAny(v, "\r\n\x00") || strings.TrimSpace(v) != v {
				return
			}
		}

		c := s.App.Client(t).WithIP(nextIP()).WithToken(tok.Access).WithHeader("Accept", "application/json")
		for k, v := range map[string]string{"MCP-Protocol-Version": version, "Mcp-Method": method, "Mcp-Name": name} {
			if v != "" {
				c = c.WithHeader(k, v)
			}
		}

		res := c.Do(http.MethodPost, "/mcp", strings.NewReader(string(body)), "application/json")

		if res.Status() >= 500 {
			t.Fatalf("answered %d: %.200s", res.Status(), res.Body())
		}

		if res.Header("Mcp-Session-Id") != "" {
			t.Fatal("a session id was issued")
		}

		if res.Status() == http.StatusOK && !json.Valid([]byte(res.Body())) {
			t.Fatalf("a 200 that is not JSON: %.200s", res.Body())
		}

		for _, leak := range []string{"BOB-CANARY", tok.Access, tok.Refresh} {
			if strings.Contains(res.Body(), leak) {
				t.Fatalf("the answer holds %q", leak)
			}
		}
	})
}

func FuzzOAuthAuthorize(f *testing.F) {
	s := oauthtest.New(f, nil)
	ana := s.SignUp(f, "Ana", "ana@example.com")

	_, challenge := oauthtest.Verifier()

	f.Add(oauthtest.ClientID, oauthtest.RedirectURI, "code", "S256", challenge, "accounts:read", oauthtest.Resource, "state-1")
	f.Add(oauthtest.ClientID, oauthtest.RedirectURI+"/x", "code", "S256", challenge, "accounts:read", oauthtest.Resource, "")
	f.Add(oauthtest.ClientID, oauthtest.RedirectURI, "token", "plain", challenge, "*", "https://other.example/mcp", "\r\nX: y")
	f.Add("https://127.0.0.1/c.json", oauthtest.RedirectURI, "code", "S256", challenge, "", oauthtest.Resource, "s")
	f.Add("", "", "", "", "", "", "", "")
	f.Add(oauthtest.ClientID, "javascript:alert(1)", "code", "S256", "", "accounts:read accounts:read", oauthtest.Resource+"#f", "s")

	f.Fuzz(func(t *testing.T, clientID, redirect, responseType, method, challenge, scope, resource, state string) {
		q := url.Values{
			"client_id": {clientID}, "redirect_uri": {redirect}, "response_type": {responseType}, "code_challenge_method": {method},
			"code_challenge": {challenge}, "scope": {scope}, "resource": {resource}, "state": {state},
		}

		res := s.Browser(t, ana, ana.Workspace.ID).WithIP(nextIP()).Get("/authorize?" + q.Encode())

		if res.Status() >= 500 {
			t.Fatalf("answered %d for %v", res.Status(), q)
		}

		loc := res.Header("Location")
		if strings.Contains(loc, "code=") {
			t.Fatalf("an authorization request got a code with no consent: %q", loc)
		}

		// A redirect goes to a page of this app or to the redirect URI the client registered, never anywhere else.
		if loc != "" && !(strings.HasPrefix(loc, "/") && !strings.HasPrefix(loc, "//")) && !strings.HasPrefix(loc, oauthtest.RedirectURI) {
			t.Fatalf("redirected to %q for %v", loc, q)
		}

		// The consent screen is shown only for a request that follows every rule.
		if res.Status() == http.StatusOK {
			ok := clientID == oauthtest.ClientID && redirect == oauthtest.RedirectURI && responseType == "code" &&
				method == "S256" && challenge != "" && resource == oauthtest.Resource
			if !ok {
				t.Fatalf("a consent screen for a request that breaks a rule: %v", q)
			}
		}
	})
}

func FuzzOAuthToken(f *testing.F) {
	s := oauthtest.New(f, nil)

	f.Add("authorization_code", "code", oauthtest.RedirectURI, oauthtest.ClientID, "verifier", oauthtest.Resource, "", "accounts:read")
	f.Add("refresh_token", "", "", oauthtest.ClientID, "", oauthtest.Resource, "not-a-token", "accounts:write")
	f.Add("client_credentials", "", "", "", "", "", "", "")
	f.Add("password", "", "", "x", "", "", "", "*")
	f.Add("", "", "", "", "", "", "", "")

	f.Fuzz(func(t *testing.T, grant, code, redirect, clientID, verifier, resource, refresh, scope string) {
		form := url.Values{
			"grant_type": {grant}, "code": {code}, "redirect_uri": {redirect}, "client_id": {clientID}, "code_verifier": {verifier},
			"resource": {resource}, "refresh_token": {refresh}, "scope": {scope},
		}

		res := s.App.Client(t).WithIP(nextIP()).PostForm("/token", form)

		if res.Status() >= 500 {
			t.Fatalf("answered %d for %v", res.Status(), form)
		}

		if res.Status() == http.StatusOK {
			t.Fatalf("a token was issued for made-up credentials: %v", form)
		}

		if strings.Contains(res.Body(), "access_token") {
			t.Fatalf("a token in the answer: %.200s", res.Body())
		}
	})
}
