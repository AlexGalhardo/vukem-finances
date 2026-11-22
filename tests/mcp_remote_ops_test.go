package tests

import (
	"net/url"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/tenancy"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func TestConnectedApps(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")
	bob := s.SignUp(t, "Bob", "bob@example.com")

	tok := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")
	bobTok := s.Authorize(t, bob, bob.Workspace.ID, "accounts:read")

	var grantID, bobGrantID string

	db, err := s.App.DB()
	if err != nil {
		t.Fatal(err)
	}

	if err := db.SQL().QueryRowContext(t.Context(), "SELECT grant_id FROM mcp_grants WHERE user_id = ?", ana.User.ID).Scan(&grantID); err != nil {
		t.Fatal(err)
	}

	if err := db.SQL().QueryRowContext(t.Context(), "SELECT grant_id FROM mcp_grants WHERE user_id = ?", bob.User.ID).Scan(&bobGrantID); err != nil {
		t.Fatal(err)
	}

	browser := s.Browser(t, ana, ana.Workspace.ID)

	t.Run("the page lists the client, its scopes and its last use", func(t *testing.T) {
		s.Call(t, tok.Access, "tools/list", nil).AssertOK()

		res := browser.Inertia("").Get("/connected-apps").AssertOK()
		for _, want := range []string{"Test client", "accounts:read", `"status":"live"`} {
			if !strings.Contains(res.Body(), want) {
				t.Errorf("the page lacks %q: %.300s", want, res.Body())
			}
		}

		if strings.Contains(res.Body(), bobGrantID) {
			t.Fatal("the page shows a grant of another person")
		}
	})

	t.Run("a grant of another person or workspace is a 404 and stays live", func(t *testing.T) {
		browser.WithHeader("Sec-Fetch-Site", "same-origin").Delete("/connected-apps/" + bobGrantID).AssertNotFound()
		s.Call(t, bobTok.Access, "tools/list", nil).AssertOK()
	})

	t.Run("revoking ends the access token and the refresh token at once", func(t *testing.T) {
		browser.Delete("/connected-apps/" + grantID).AssertStatus(303)

		s.Call(t, tok.Access, "tools/list", nil).AssertUnauthorized()

		res := s.Token(t, url.Values{
			"grant_type": {"refresh_token"}, "refresh_token": {tok.Refresh}, "client_id": {oauthtest.ClientID},
			"resource": {oauthtest.Resource},
		})
		if res.Status() == 200 {
			t.Fatal("a revoked grant still refreshes")
		}

		if page := browser.Inertia("").Get("/connected-apps").AssertOK(); !strings.Contains(page.Body(), `"status":"revoked"`) {
			t.Fatalf("the page does not show it revoked: %.300s", page.Body())
		}
	})

	t.Run("removing a member through the service revokes their grants for good", func(t *testing.T) {
		carl := s.SignUp(t, "Carl", "carl@example.com")
		s.AddMember(t, ana.Workspace.ID, carl, models.RoleMember)

		carlTok := s.Authorize(t, carl, ana.Workspace.ID, "accounts:read")
		s.Call(t, carlTok.Access, "tools/list", nil).AssertOK()

		ctx := workspaces.WithRole(tenancy.With(t.Context(), tenancy.ID(itoa(ana.Workspace.ID))), models.RoleOwner)
		if err := s.WS.RemoveMember(ctx, ana.User, carl.User.ID); err != nil {
			t.Fatal(err)
		}

		// Invited back, the old grant stays dead.
		s.AddMember(t, ana.Workspace.ID, carl, models.RoleMember)
		s.Call(t, carlTok.Access, "tools/list", nil).AssertUnauthorized()
	})
}

func TestMCPRateLimit(t *testing.T) {
	t.Parallel()

	t.Run("a grant is limited per minute, on tools and on resources", func(t *testing.T) {
		t.Parallel()

		s := oauthtest.New(t, nil)
		ana := s.SignUp(t, "Ana", "ana@example.com")
		tok := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")

		// The limit is a count per minute: read resources until it bites, then the
		// tool of the same grant is limited too, in the same window.
		limited := false

		for range 130 {
			if r := readResource(t, s, tok.Access, "finances://accounts"); strings.Contains(r.raw, "rate_limited") {
				limited = true

				break
			} else if r.Error != nil {
				t.Fatalf("a read failed for another reason: %s", r.raw)
			}
		}

		if !limited {
			t.Fatal("130 resource reads in a minute were all served")
		}

		if r := callTool(t, s, tok.Access, "accounts_list", map[string]any{}); !strings.Contains(r.text(), "rate_limited") {
			t.Fatalf("the tool of a limited grant was served: %s", r.raw)
		}

		// Another person has their own budget (the key is the person, the workspace and the client).
		bob := s.SignUp(t, "Bob", "bob@example.com")
		other := s.Authorize(t, bob, bob.Workspace.ID, "accounts:read")
		if r := callTool(t, s, other.Access, "accounts_list", map[string]any{}); r.refused() {
			t.Fatalf("another grant shares the limit: %s", r.text())
		}
	})

	t.Run("an address is limited on the endpoint", func(t *testing.T) {
		t.Parallel()

		s := oauthtest.New(t, nil)

		var last int

		for range 320 {
			last = s.Call(t, "", "tools/list", nil).Status()
			if last == 429 {
				return
			}
		}

		t.Fatalf("320 calls from one address were all served (last %d)", last)
	})

	t.Run("the token endpoint is limited per address", func(t *testing.T) {
		t.Parallel()

		s := oauthtest.New(t, nil)

		for range 40 {
			if s.Token(t, url.Values{"grant_type": {"authorization_code"}}).Status() == 429 {
				return
			}
		}

		t.Fatal("40 token requests from one address were all served")
	})
}

// An entry for each call: the client, the tool and the outcome, never an
// amount or a description. A resource read writes the same entry as a tool.
func TestMCPAudit(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")
	tok := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read", "accounts:write", "transactions:read", "transactions:write")

	uids := accountUIDs(t, callTool(t, s, tok.Access, "accounts_list", map[string]any{}))

	const (
		description = "AUDIT-CANARY-DESCRIPTION"
		amount      = "987654"
	)

	created := callTool(t, s, tok.Access, "transactions_create", map[string]any{
		"account_uid": uids[0], "kind": "expense", "amount_cents": 987654, "date": "2026-11-20", "description": description,
	})
	if created.refused() {
		t.Fatalf("transactions_create: %s", created.text())
	}

	readResource(t, s, tok.Access, "finances://accounts")
	callTool(t, s, tok.Access, "accounts_show", map[string]any{"uid": "01JD5Z8K3M4N5P6Q7R8S9T0V1W"})

	var entries []string

	for _, line := range strings.Split(s.App.Logs(), "\n") {
		if strings.Contains(line, "mcp call") {
			entries = append(entries, line)
		}
	}

	if len(entries) != 4 {
		t.Fatalf("%d audit entries for 4 calls:\n%s", len(entries), strings.Join(entries, "\n"))
	}

	for _, e := range entries {
		for _, want := range []string{"client=" + oauthtest.ClientID, "tool=", "outcome=", "workspace=", "user="} {
			if !strings.Contains(e, want) {
				t.Errorf("an entry lacks %s: %s", want, e)
			}
		}

		if strings.Contains(e, description) || strings.Contains(e, amount) {
			t.Errorf("an entry holds data: %s", e)
		}
	}

	joined := strings.Join(entries, "\n")
	if strings.Count(joined, "tool=accounts_list") != 2 {
		t.Errorf("want one entry for the accounts_list tool and one for the accounts resource:\n%s", joined)
	}

	if !strings.Contains(joined, "tool=transactions_create") || !strings.Contains(joined, "outcome=refused code=not_found") {
		t.Errorf("the write and the refusal are not both audited:\n%s", joined)
	}

	for _, secret := range []string{tok.Access, tok.Refresh} {
		if strings.Contains(s.App.Logs(), secret) {
			t.Fatal("a token is in the logs")
		}
	}
}
