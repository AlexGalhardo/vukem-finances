package tests

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
)

// rpc is a decoded answer of /mcp.
type rpc struct {
	Result struct {
		StructuredContent map[string]any `json:"structuredContent"`
		IsError           bool           `json:"isError"`
		Content           []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	raw string
}

// text is everything the answer says, for a refusal check.
func (r rpc) text() string {
	var b strings.Builder

	for _, c := range r.Result.Content {
		b.WriteString(c.Text)
	}

	if r.Error != nil {
		b.WriteString(r.Error.Message)
	}

	return b.String()
}

func (r rpc) refused() bool { return r.Result.IsError || r.Error != nil }

func decodeRPC(t *testing.T, res *vukemtest.Response) rpc {
	t.Helper()

	var out rpc
	if err := json.Unmarshal([]byte(res.Body()), &out); err != nil {
		t.Fatalf("not a JSON-RPC answer (%d): %v: %.200s", res.Status(), err, res.Body())
	}

	out.raw = res.Body()

	return out
}

func callTool(t *testing.T, s *oauthtest.Stack, access, tool string, args map[string]any) rpc {
	t.Helper()

	return decodeRPC(t, s.Call(t, access, "tools/call", map[string]any{"name": tool, "arguments": args}))
}

func readResource(t *testing.T, s *oauthtest.Stack, access, uri string) rpc {
	t.Helper()

	return decodeRPC(t, s.Call(t, access, "resources/read", map[string]any{"uri": uri}))
}

func accountUIDs(t *testing.T, r rpc) []string {
	t.Helper()

	data, _ := r.Result.StructuredContent["data"].([]any)

	var uids []string

	for _, row := range data {
		uids = append(uids, row.(map[string]any)["uid"].(string))
	}

	return uids
}

func stackSQL(t *testing.T, s *oauthtest.Stack, query string, args ...any) {
	t.Helper()

	db, err := s.App.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.SQL().ExecContext(t.Context(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func TestOAuthConsent(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")
	carl := s.SignUp(t, "Carl", "carl@example.com")
	s.AddMember(t, ana.Workspace.ID, carl, models.RoleMember)

	_, challenge := oauthtest.Verifier()

	t.Run("a visitor is sent to sign in", func(t *testing.T) {
		res := s.App.Client(t).Get("/authorize?" + oauthtest.AuthorizeQuery(challenge, "accounts:read").Encode())
		if loc := res.Header("Location"); !strings.Contains(loc, "/login") {
			t.Fatalf("a visitor got %d %q, want a redirect to /login", res.Status(), loc)
		}
	})

	t.Run("the screen names the client, where it returns, the workspace and what the role lacks", func(t *testing.T) {
		q := oauthtest.AuthorizeQuery(challenge, "accounts:read", "accounts:write")
		res := s.Browser(t, carl, ana.Workspace.ID).Inertia("").Get("/authorize?" + q.Encode()).AssertOK()

		var page struct {
			Props struct {
				ClientName   string `json:"clientName"`
				ClientID     string `json:"clientId"`
				RedirectHost string `json:"redirectHost"`
				Workspace    string `json:"workspace"`
				Role         string `json:"role"`
				Scopes       []struct {
					Name    string `json:"name"`
					Allowed bool   `json:"allowed"`
				} `json:"scopes"`
			} `json:"props"`
		}

		if err := json.Unmarshal([]byte(res.Body()), &page); err != nil {
			t.Fatal(err)
		}

		p := page.Props
		if p.ClientName != "Test client" || p.ClientID != oauthtest.ClientID || p.RedirectHost != "client.example.test" ||
			p.Workspace != ana.Workspace.Name || p.Role != models.RoleMember {
			t.Fatalf("props = %+v", p)
		}

		if len(p.Scopes) != 2 || !p.Scopes[0].Allowed || p.Scopes[1].Allowed {
			t.Fatalf("scopes = %+v, want accounts:read allowed and accounts:write not", p.Scopes)
		}
	})

	approve := func(who oauthtest.Person, ws int64, mutate func(url.Values)) (status int, loc string) {
		action, form := s.Consent(t, who, ws, oauthtest.AuthorizeQuery(challenge, "accounts:read", "accounts:write"))
		form.Set("decision", "approve")

		if mutate != nil {
			mutate(form)
		}

		res := s.Browser(t, who, ws).PostForm(action, form)

		return res.Status(), res.Header("Location")
	}

	noCode := func(t *testing.T, status int, loc string) {
		t.Helper()

		if strings.Contains(loc, "code=") {
			t.Fatalf("a code was issued (%d %q)", status, loc)
		}
	}

	t.Run("a member who approves a scope the role lacks gets no code", func(t *testing.T) {
		status, loc := approve(carl, ana.Workspace.ID, nil)
		noCode(t, status, loc)

		if status != 403 {
			t.Fatalf("the decision answered %d, want 403", status)
		}
	})

	t.Run("a forged POST gets no code", func(t *testing.T) {
		for name, mutate := range map[string]func(url.Values){
			"a tampered csrf":       func(f url.Values) { f.Set("csrf", "forged-"+f.Get("csrf")) },
			"no csrf":               func(f url.Values) { f.Del("csrf") },
			"a widened scope":       func(f url.Values) { f.Set("scope", "accounts:read accounts:write budgets:write") },
			"another redirect":      func(f url.Values) { f.Set("redirect_uri", "https://evil.example/callback") },
			"another resource":      func(f url.Values) { f.Set("resource", "https://other.example/mcp") },
			"another client":        func(f url.Values) { f.Set("client_id", "someone-else") },
			"a different challenge": func(f url.Values) { f.Set("code_challenge", "AAAA"+f.Get("code_challenge")) },
		} {
			status, loc := approve(ana, ana.Workspace.ID, mutate)
			if strings.Contains(loc, "code=") || status == 200 {
				t.Errorf("%s: answered %d %q", name, status, loc)
			}
		}
	})

	t.Run("a form of another person is refused", func(t *testing.T) {
		action, form := s.Consent(t, ana, ana.Workspace.ID, oauthtest.AuthorizeQuery(challenge, "accounts:read"))
		form.Set("decision", "approve")

		res := s.Browser(t, carl, ana.Workspace.ID).PostForm(action, form)
		noCode(t, res.Status(), res.Header("Location"))
	})

	t.Run("a session pointing at a workspace the person is not in falls back to their own", func(t *testing.T) {
		action, form := s.Consent(t, ana, carl.Workspace.ID, oauthtest.AuthorizeQuery(challenge, "accounts:read"))
		form.Set("decision", "approve")

		res := s.Browser(t, ana, carl.Workspace.ID).PostForm(action, form)
		if !strings.Contains(res.Header("Location"), "code=") {
			t.Fatalf("the decision answered %d %q", res.Status(), res.Header("Location"))
		}

		var ws int64

		db, err := s.App.DB()
		if err != nil {
			t.Fatal(err)
		}

		if err := db.SQL().QueryRowContext(t.Context(), "SELECT workspace_id FROM mcp_grants ORDER BY id DESC LIMIT 1").Scan(&ws); err != nil || ws != ana.Workspace.ID {
			t.Fatalf("the grant is bound to workspace %d (%v), want the person's own %d", ws, err, ana.Workspace.ID)
		}
	})

	t.Run("picking a workspace comes back to the authorization request and nowhere else", func(t *testing.T) {
		back := "/authorize?" + oauthtest.AuthorizeQuery(challenge, "accounts:read").Encode()

		res := s.Browser(t, ana, ana.Workspace.ID).PostForm("/oauth/workspace",
			url.Values{"workspace": {itoa(ana.Workspace.ID)}, "return": {back}})
		if res.Status() != 303 || res.Header("Location") != back {
			t.Fatalf("picking answered %d %q", res.Status(), res.Header("Location"))
		}

		for _, bad := range []string{"https://evil.example/", "//evil.example/", "/tokens", "/authorize\r\nX: y", ""} {
			res := s.Browser(t, ana, ana.Workspace.ID).PostForm("/oauth/workspace",
				url.Values{"workspace": {itoa(ana.Workspace.ID)}, "return": {bad}})
			if res.Status() != 400 {
				t.Errorf("return %q answered %d, want 400", bad, res.Status())
			}
		}

		// A workspace the person is not in is never the one consented for: the tenant
		// middleware answers with their own.
		res = s.Browser(t, ana, ana.Workspace.ID).PostForm("/oauth/workspace",
			url.Values{"workspace": {itoa(carl.Workspace.ID)}, "return": {back}})
		if res.Status() == 303 {
			page := s.Browser(t, ana, carl.Workspace.ID).Inertia("").Get(back).AssertOK()
			if !strings.Contains(page.Body(), ana.Workspace.Name) || strings.Contains(page.Body(), `"workspace":"`+carl.Workspace.Name) {
				t.Fatalf("consent for a workspace the person is not in: %.300s", page.Body())
			}
		}
	})
}

// The scopes of a grant are fixed at consent: a role downgrade afterwards never
// widens them, and a downgrade cuts what the token may do on the next call.
func TestMCPScopesAreFixedAtConsent(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")
	carl := s.SignUp(t, "Carl", "carl@example.com")
	s.AddMember(t, ana.Workspace.ID, carl, models.RoleAdmin)

	tok := s.Authorize(t, carl, ana.Workspace.ID, "tags:read", "tags:write")

	created := callTool(t, s, tok.Access, "tags_create", map[string]any{"name": "Joint"})
	if created.refused() {
		t.Fatalf("an admin with the scope could not create: %s", created.text())
	}

	var approved string

	db, err := s.App.DB()
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SQL().QueryRowContext(t.Context(), "SELECT scopes FROM mcp_grants").Scan(&approved); err != nil || approved != "tags:read tags:write" {
		t.Fatalf("approved scopes = %q (%v)", approved, err)
	}

	t.Run("a downgrade cuts the write on the next call", func(t *testing.T) {
		stackSQL(t, s, "UPDATE memberships SET role = ? WHERE user_id = ? AND workspace_id = ?", models.RoleViewer, carl.User.ID, ana.Workspace.ID)

		if r := callTool(t, s, tok.Access, "tags_create", map[string]any{"name": "Another"}); !r.refused() {
			t.Fatalf("a viewer created an account: %s", r.raw)
		}

		if r := callTool(t, s, tok.Access, "tags_list", map[string]any{}); r.refused() {
			t.Fatalf("a viewer could not read: %s", r.text())
		}
	})

	t.Run("a grant whose scopes grew beyond the approved set is not found", func(t *testing.T) {
		stackSQL(t, s, "UPDATE oauth_grants SET scopes = scopes || ' budgets:write'")
		s.Call(t, tok.Access, "tools/list", nil).AssertUnauthorized()
	})
}

func TestMCPRemoteIsolation(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")
	bob := s.SignUp(t, "Bob", "bob@example.com")

	scopes := []string{"accounts:read", "accounts:write", "transactions:read", "transactions:write"}
	anaTok := s.Authorize(t, ana, ana.Workspace.ID, scopes...)
	bobTok := s.Authorize(t, bob, bob.Workspace.ID, scopes...)

	anaAccounts := accountUIDs(t, callTool(t, s, anaTok.Access, "accounts_list", map[string]any{}))
	bobAccounts := accountUIDs(t, callTool(t, s, bobTok.Access, "accounts_list", map[string]any{}))

	if len(anaAccounts) != 1 || len(bobAccounts) != 1 || anaAccounts[0] == bobAccounts[0] {
		t.Fatalf("accounts of A %v and of B %v", anaAccounts, bobAccounts)
	}

	t.Run("every id of another workspace is not found", func(t *testing.T) {
		for tool, args := range map[string]map[string]any{
			"accounts_show":     {"uid": bobAccounts[0]},
			"accounts_update":   {"uid": bobAccounts[0], "name": "Hijacked"},
			"transactions_list": {"account": bobAccounts[0]},
			"transactions_create": {
				"account_uid": bobAccounts[0], "kind": "expense", "amount_cents": 100, "date": "2026-11-20",
			},
		} {
			r := callTool(t, s, anaTok.Access, tool, args)
			if tool == "transactions_list" {
				// A filter by a foreign uid finds nothing; it must not find B's rows.
				if r.refused() {
					continue
				}

				if data, _ := r.Result.StructuredContent["data"].([]any); len(data) != 0 {
					t.Errorf("%s returned rows of another workspace: %s", tool, r.raw)
				}

				continue
			}

			if !r.refused() || !strings.Contains(r.text(), "not_found") {
				t.Errorf("%s with an id of another workspace answered %s", tool, r.raw)
			}
		}

		if n := accountUIDs(t, callTool(t, s, bobTok.Access, "accounts_list", map[string]any{})); len(n) != 1 {
			t.Fatalf("B has %d accounts after A's attempts", len(n))
		}
	})

	t.Run("resources read the grant's workspace only", func(t *testing.T) {
		r := readResource(t, s, anaTok.Access, "finances://accounts")
		if r.Error != nil || strings.Contains(r.raw, bobAccounts[0]) {
			t.Fatalf("the accounts resource: %s", r.raw)
		}
	})

	t.Run("a token for one workspace never reaches another one the person is in", func(t *testing.T) {
		s.AddMember(t, bob.Workspace.ID, ana, models.RoleOwner)

		if got := accountUIDs(t, callTool(t, s, anaTok.Access, "accounts_list", map[string]any{})); len(got) != 1 || got[0] != anaAccounts[0] {
			t.Fatalf("the grant of A's workspace saw %v", got)
		}
	})

	t.Run("a scope the grant lacks is refused", func(t *testing.T) {
		readOnly := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")

		if r := callTool(t, s, readOnly.Access, "accounts_create", map[string]any{"name": "Nope", "kind": "cash"}); !r.refused() {
			t.Fatalf("a read-only grant created an account: %s", r.raw)
		}

		list := decodeRPC(t, s.Call(t, readOnly.Access, "tools/list", nil))
		if strings.Contains(list.raw, "accounts_create") || strings.Contains(list.raw, "transactions_list") {
			t.Fatalf("tools outside the scope are listed: %.300s", list.raw)
		}
	})

	t.Run("a removed member fails on the next call", func(t *testing.T) {
		carl := s.SignUp(t, "Carl", "carl@example.com")
		s.AddMember(t, ana.Workspace.ID, carl, models.RoleMember)

		tok := s.Authorize(t, carl, ana.Workspace.ID, "accounts:read")
		s.Call(t, tok.Access, "tools/list", nil).AssertOK()

		stackSQL(t, s, "DELETE FROM memberships WHERE user_id = ? AND workspace_id = ?", carl.User.ID, ana.Workspace.ID)

		s.Call(t, tok.Access, "tools/list", nil).AssertUnauthorized()
		s.Call(t, tok.Access, "tools/call", map[string]any{"name": "accounts_list", "arguments": map[string]any{}}).AssertUnauthorized()
	})

	t.Run("a member cannot use a tool that removes data", func(t *testing.T) {
		dan := s.SignUp(t, "Dan", "dan@example.com")
		s.AddMember(t, ana.Workspace.ID, dan, models.RoleMember)

		tok := s.Authorize(t, dan, ana.Workspace.ID, "transactions:read", "transactions:write")
		if r := callTool(t, s, tok.Access, "transactions_delete", map[string]any{"uid": "01JD5Z8K3M4N5P6Q7R8S9T0V1W"}); !r.refused() ||
			!strings.Contains(r.text(), "owner or an admin") {
			t.Fatalf("a member's delete answered %s", r.raw)
		}
	})
}

// Every way into the account that ends sessions ends the grants too: the
// access token fails on its next call and the refresh token no longer refreshes.
func TestMCPGrantsEndWithTheirPerson(t *testing.T) {
	t.Parallel()

	refresh := func(s *oauthtest.Stack, token string) int {
		return s.Token(t, url.Values{
			"grant_type": {"refresh_token"}, "refresh_token": {token}, "client_id": {oauthtest.ClientID},
			"resource": {oauthtest.Resource},
		}).Status()
	}

	for name, end := range map[string]func(*testing.T, *oauthtest.Stack, oauthtest.Person){
		"a password reset": func(t *testing.T, s *oauthtest.Stack, p oauthtest.Person) {
			acc, err := accounts.New(s.App.App)
			if err != nil {
				t.Fatal(err)
			}

			if err := acc.SetPassword(t.Context(), p.User.Email, "Correct-horse-battery-"+"9x"); err != nil {
				t.Fatal(err)
			}
		},
		"logout everywhere": func(t *testing.T, s *oauthtest.Stack, p oauthtest.Person) {
			acc, err := accounts.New(s.App.App)
			if err != nil {
				t.Fatal(err)
			}

			if err := acc.RevokeOtherSessions(t.Context(), p.User.ID, ""); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s := oauthtest.New(t, nil)
			ana := s.SignUp(t, "Ana", "ana@example.com")
			bob := s.SignUp(t, "Bob", "bob@example.com")

			anaTok := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")
			bobTok := s.Authorize(t, bob, bob.Workspace.ID, "accounts:read")

			s.Call(t, anaTok.Access, "tools/list", nil).AssertOK()

			end(t, s, ana)

			s.Call(t, anaTok.Access, "tools/list", nil).AssertUnauthorized()

			if code := refresh(s, anaTok.Refresh); code == 200 {
				t.Fatal("the refresh token still refreshes after the grants ended")
			}

			// Another person's grant is untouched.
			s.Call(t, bobTok.Access, "tools/list", nil).AssertOK()

			if code := refresh(s, bobTok.Refresh); code != 200 {
				t.Fatalf("another person's refresh answered %d", code)
			}
		})
	}
}

// A client identified by a metadata document can claim any name: the screen
// names it by its verified host and shows the claim apart, as unverified.
func TestOAuthConsentDoesNotTrustAClaimedName(t *testing.T) {
	t.Parallel()

	const id = "https://app.example.test/oauth/client.json"

	s := oauthtest.New(t, map[string]string{
		"MCP_OAUTH_CLIENTS": `[{"id":"` + id + `","name":"Official Bank Assistant","redirect_uris":["` + oauthtest.RedirectURI + `"]}]`,
	})
	ana := s.SignUp(t, "Ana", "ana@example.com")

	_, challenge := oauthtest.Verifier()
	q := oauthtest.AuthorizeQuery(challenge, "accounts:read")
	q.Set("client_id", id)

	res := s.Browser(t, ana, ana.Workspace.ID).Inertia("").Get("/authorize?" + q.Encode()).AssertOK()

	var page struct {
		Props struct {
			ClientName  string `json:"clientName"`
			ClaimedName string `json:"claimedName"`
			ClientID    string `json:"clientId"`
		} `json:"props"`
	}

	if err := json.Unmarshal([]byte(res.Body()), &page); err != nil {
		t.Fatal(err)
	}

	if p := page.Props; p.ClientName != "app.example.test" || p.ClaimedName != "Official Bank Assistant" || p.ClientID != id {
		t.Fatalf("props = %+v, want the verified host as the name and the claim apart", p)
	}
}

// A grant holds what was approved at consent. Promoting the person afterwards
// does not give the grant a scope it never had.
func TestMCPPromotionDoesNotWidenAGrant(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")
	dan := s.SignUp(t, "Dan", "dan@example.com")
	s.AddMember(t, ana.Workspace.ID, dan, models.RoleMember)

	tok := s.Authorize(t, dan, ana.Workspace.ID, "tags:read")

	if r := callTool(t, s, tok.Access, "tags_create", map[string]any{"name": "Before"}); !r.refused() {
		t.Fatalf("a read-only grant created a tag: %s", r.raw)
	}

	stackSQL(t, s, "UPDATE memberships SET role = ? WHERE user_id = ? AND workspace_id = ?", models.RoleAdmin, dan.User.ID, ana.Workspace.ID)

	if r := callTool(t, s, tok.Access, "tags_create", map[string]any{"name": "After"}); !r.refused() {
		t.Fatalf("a promotion widened the grant: %s", r.raw)
	}

	if list := decodeRPC(t, s.Call(t, tok.Access, "tools/list", nil)); strings.Contains(list.raw, "tags_create") {
		t.Fatalf("a tool outside the approved scopes is listed: %.200s", list.raw)
	}
}
