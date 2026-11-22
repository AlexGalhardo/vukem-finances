package mcp_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/auth/jwt"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
)

const listBody = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`

// raw sends a request to /mcp with exactly the headers given.
func raw(t *testing.T, s *oauthtest.Stack, method string, headers map[string]string, body string) *vukemtest.Response {
	t.Helper()

	c := s.App.Client(t)
	for k, v := range headers {
		c = c.WithHeader(k, v)
	}

	ct := headers["Content-Type"]
	if ct == "" {
		ct = "application/json"
	}

	return c.Do(method, "/mcp", strings.NewReader(body), ct)
}

func good(access string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + access, "Accept": "application/json",
		"MCP-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/list",
	}
}

func with(h map[string]string, k, v string) map[string]string {
	out := map[string]string{}
	for hk, hv := range h {
		out[hk] = hv
	}

	if v == "" {
		delete(out, k)
	} else {
		out[k] = v
	}

	return out
}

func TestHTTPTransport(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")
	tok := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")
	h := good(tok.Access)

	t.Run("a good call is served and opens no session", func(t *testing.T) {
		res := raw(t, s, http.MethodPost, h, listBody).AssertOK()
		if res.Header("Mcp-Session-Id") != "" {
			t.Fatal("the server issued a session id")
		}
	})

	t.Run("only POST", func(t *testing.T) {
		for _, m := range []string{http.MethodGet, http.MethodDelete, http.MethodPut, http.MethodPatch} {
			res := raw(t, s, m, h, "")
			if res.Status() != 405 || res.Header("Allow") != "POST" {
				t.Errorf("%s answered %d (Allow %q), want 405 and POST", m, res.Status(), res.Header("Allow"))
			}
		}
	})

	t.Run("the origin is checked on every request", func(t *testing.T) {
		for _, origin := range []string{"https://evil.example", "null", "http://finances.example.test", "https://finances.example.test.evil.example"} {
			if res := raw(t, s, http.MethodPost, with(h, "Origin", origin), listBody); res.Status() != 403 {
				t.Errorf("Origin %q answered %d, want 403", origin, res.Status())
			}
		}

		for _, m := range []string{http.MethodGet, http.MethodDelete} {
			if res := raw(t, s, m, with(h, "Origin", "https://evil.example"), ""); res.Status() != 403 {
				t.Errorf("%s with a bad Origin answered %d, want 403", m, res.Status())
			}
		}

		raw(t, s, http.MethodPost, with(h, "Origin", oauthtest.Issuer), listBody).AssertOK()
	})

	t.Run("the protocol headers must agree with the request", func(t *testing.T) {
		for name, headers := range map[string]map[string]string{
			"no version":         with(h, "MCP-Protocol-Version", ""),
			"another version":    with(h, "MCP-Protocol-Version", "2025-11-25"),
			"an unknown version": with(h, "MCP-Protocol-Version", "1999-01-01"),
			"no method header":   with(h, "Mcp-Method", ""),
			"another method":     with(h, "Mcp-Method", "tools/call"),
		} {
			if res := raw(t, s, http.MethodPost, headers, listBody); res.Status() != 400 {
				t.Errorf("%s answered %d, want 400", name, res.Status())
			}
		}

		call := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"accounts_list","arguments":{},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
		base := with(h, "Mcp-Method", "tools/call")

		if res := raw(t, s, http.MethodPost, base, call); res.Status() != 400 {
			t.Errorf("a call with no Mcp-Name answered %d, want 400", res.Status())
		}

		if res := raw(t, s, http.MethodPost, with(base, "Mcp-Name", "accounts_show"), call); res.Status() != 400 {
			t.Errorf("a call with another Mcp-Name answered %d, want 400", res.Status())
		}

		raw(t, s, http.MethodPost, with(base, "Mcp-Name", "accounts_list"), call).AssertOK()
	})

	t.Run("a token is never read from the query", func(t *testing.T) {
		res := s.App.Client(t).WithHeader("Accept", "application/json").WithHeader("MCP-Protocol-Version", "2026-07-28").
			WithHeader("Mcp-Method", "tools/list").Do(http.MethodPost, "/mcp?access_token="+tok.Access, strings.NewReader(listBody), "application/json")
		if res.Status() != 400 {
			t.Fatalf("a token in the query answered %d, want 400", res.Status())
		}

		if strings.Contains(s.App.Logs(), tok.Access) {
			t.Fatal("the token reached the logs")
		}
	})

	t.Run("the body and its type are checked", func(t *testing.T) {
		if res := raw(t, s, http.MethodPost, with(h, "Content-Type", "text/plain"), listBody); res.Status() != 415 {
			t.Errorf("text/plain answered %d, want 415", res.Status())
		}

		if res := raw(t, s, http.MethodPost, with(h, "Accept", "text/html"), listBody); res.Status() != 406 {
			t.Errorf("Accept text/html answered %d, want 406", res.Status())
		}

		if res := raw(t, s, http.MethodPost, h, "["+listBody+"]"); res.Status() != 400 {
			t.Errorf("a batch answered %d, want 400", res.Status())
		}

		if res := raw(t, s, http.MethodPost, h, "not json"); res.Status() != 400 {
			t.Errorf("garbage answered %d, want 400", res.Status())
		}

		if res := raw(t, s, http.MethodPost, h, strings.Repeat("x", 2<<20)); res.Status() != 413 {
			t.Errorf("a 2 MiB body answered %d, want 413", res.Status())
		}
	})
}

// forge signs a token with the real key of the server for the claims given.
func forge(t *testing.T, s *oauthtest.Stack, mutate func(*jwt.Claims)) string {
	t.Helper()

	seed, err := s.App.DeriveKey("mcp-oauth-ed25519", ed25519.SeedSize)
	if err != nil {
		t.Fatal(err)
	}

	db, err := s.App.DB()
	if err != nil {
		t.Fatal(err)
	}

	var gid, subject string
	if err := db.SQL().QueryRowContext(t.Context(), "SELECT id, subject FROM oauth_grants LIMIT 1").Scan(&gid, &subject); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	claims := jwt.Claims{
		Issuer: oauthtest.Issuer, Subject: subject, Audience: []string{oauthtest.Resource}, IssuedAt: now, NotBefore: now,
		ExpiresAt: now.Add(time.Minute), ID: "forged",
		Extra: map[string]any{"scope": "accounts:read", "client_id": oauthtest.ClientID, "gid": gid},
	}
	mutate(&claims)

	token, err := jwt.SignTyped(jwt.EdDSA("mcp-1", ed25519.NewKeyFromSeed(seed)), "at+jwt", claims)
	if err != nil {
		t.Fatal(err)
	}

	return token
}

func TestHTTPTransportRefusesBadTokens(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")
	tok := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")

	// The control: the same claims, signed with the real key, are accepted.
	same := forge(t, s, func(*jwt.Claims) {})
	s.Call(t, same, "tools/list", nil).AssertOK()

	none := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"at+jwt"}`)) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"`+oauthtest.Issuer+`","aud":"`+oauthtest.Resource+`","sub":"1:1","exp":9999999999}`)) + "."

	for name, token := range map[string]string{
		"none":                        "",
		"garbage":                     "not-a-token",
		"alg none":                    none,
		"a tampered signature":        tok.Access[:len(tok.Access)-4] + "AAAA",
		"a refresh token":             tok.Refresh,
		"another audience":            forge(t, s, func(c *jwt.Claims) { c.Audience = []string{"https://other.example/mcp"} }),
		"two audiences":               forge(t, s, func(c *jwt.Claims) { c.Audience = []string{oauthtest.Resource, "https://other.example/mcp"} }),
		"another issuer":              forge(t, s, func(c *jwt.Claims) { c.Issuer = "https://other.example" }),
		"an expired token":            forge(t, s, func(c *jwt.Claims) { c.ExpiresAt = time.Now().Add(-time.Hour) }),
		"an unknown grant":            forge(t, s, func(c *jwt.Claims) { c.Extra["gid"] = "no-such-grant" }),
		"another subject":             forge(t, s, func(c *jwt.Claims) { c.Subject = "999:999" }),
		"a subject with no workspace": forge(t, s, func(c *jwt.Claims) { c.Subject = "1" }),
	} {
		res := s.Call(t, token, "tools/list", nil)
		if res.Status() != 401 {
			t.Errorf("%s answered %d, want 401", name, res.Status())
		}

		if !strings.Contains(res.Header("WWW-Authenticate"), "resource_metadata") {
			t.Errorf("%s: no challenge with the metadata URL", name)
		}
	}
}
