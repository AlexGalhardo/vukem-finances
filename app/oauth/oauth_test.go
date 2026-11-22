package oauth_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
)

func TestSchema(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)

	db, err := s.App.DB()
	if err != nil {
		t.Fatal(err)
	}

	snap, err := schema.Inspect(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"oauth_grants":         {"id", "client_id", "subject", "scopes", "resource", "created_at", "revoked_at"},
		"oauth_codes":          {"hash", "grant_id", "client_id", "redirect_uri", "challenge", "expires_at", "used_at"},
		"oauth_refresh_tokens": {"hash", "grant_id", "expires_at", "used_at"},
		"mcp_grants":           {"workspace_id", "user_id", "grant_id", "client_id", "client_name", "last_used_at"},
	}

	for table, cols := range want {
		var found bool

		for _, tbl := range snap.Tables {
			if tbl.Name != table {
				continue
			}

			found = true

			for _, c := range cols {
				if !slices.ContainsFunc(tbl.Columns, func(ci schema.ColumnInfo) bool { return ci.Name == c }) {
					t.Errorf("%s has no column %s", table, c)
				}
			}

			if table == "mcp_grants" && !slices.ContainsFunc(tbl.Indexes, func(ix schema.IndexInfo) bool {
				return len(ix.Columns) > 0 && ix.Columns[0] == "workspace_id"
			}) {
				t.Error("mcp_grants has no index that starts with workspace_id")
			}

			// A secret is never stored: codes and refresh tokens are hashes.
			for _, c := range tbl.Columns {
				if strings.Contains(c.Name, "secret") || c.Name == "token" || c.Name == "code" {
					t.Errorf("%s stores %s in the clear", table, c.Name)
				}
			}
		}

		if !found {
			t.Errorf("table %s does not exist", table)
		}
	}
}

// The grant row binds the grant to the workspace the person chose, and the
// subject of the access token says the same.
func TestGrantIsBoundToTheWorkspace(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")

	tok := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")
	if tok.Type != "Bearer" || tok.Expires != 900 || tok.Scope != "accounts:read" {
		t.Fatalf("tokens = %+v, want a 15 minute bearer token for the scope asked", tok)
	}

	db, err := s.App.DB()
	if err != nil {
		t.Fatal(err)
	}

	var ws, user int64

	err = db.SQL().QueryRowContext(t.Context(), "SELECT workspace_id, user_id FROM mcp_grants").Scan(&ws, &user)
	if err != nil || ws != ana.Workspace.ID || user != ana.User.ID {
		t.Fatalf("binding = workspace %d user %d (%v), want %d and %d", ws, user, err, ana.Workspace.ID, ana.User.ID)
	}
}
