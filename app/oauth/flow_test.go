package oauth_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
)

func jsonBody(t *testing.T, body string) map[string]any {
	t.Helper()

	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("not JSON: %v: %.200s", err, body)
	}

	return m
}

func TestDiscovery(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)

	as := jsonBody(t, s.App.Client(t).Get("/.well-known/oauth-authorization-server").AssertOK().Body())
	if as["issuer"] != oauthtest.Issuer || as["authorization_endpoint"] != oauthtest.Issuer+"/authorize" ||
		as["token_endpoint"] != oauthtest.Issuer+"/token" || as["authorization_response_iss_parameter_supported"] != true {
		t.Fatalf("authorization server metadata = %v", as)
	}

	if m, _ := as["code_challenge_methods_supported"].([]any); len(m) != 1 || m[0] != "S256" {
		t.Fatalf("code_challenge_methods_supported = %v, want only S256", m)
	}

	if as["client_id_metadata_document_supported"] != true {
		t.Fatal("client metadata documents are not advertised")
	}

	if _, dcr := as["registration_endpoint"]; dcr {
		t.Fatal("dynamic client registration is advertised")
	}

	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"} {
		pr := jsonBody(t, s.App.Client(t).Get(path).AssertOK().Body())
		if pr["resource"] != oauthtest.Resource {
			t.Fatalf("%s: resource = %v, want %s", path, pr["resource"], oauthtest.Resource)
		}

		if servers, _ := pr["authorization_servers"].([]any); len(servers) != 1 || servers[0] != oauthtest.Issuer {
			t.Fatalf("%s: authorization_servers = %v", path, servers)
		}
	}

	if keys, _ := jsonBody(t, s.App.Client(t).Get("/.well-known/jwks.json").AssertOK().Body())["keys"].([]any); len(keys) != 1 {
		t.Fatalf("jwks has %d keys", len(keys))
	}

	t.Run("a call with no token is challenged with the metadata URL", func(t *testing.T) {
		res := s.Call(t, "", "tools/list", nil).AssertUnauthorized()

		challenge := res.Header("WWW-Authenticate")
		want := "resource_metadata=\"" + oauthtest.Issuer + "/.well-known/oauth-protected-resource/mcp\""

		if !strings.HasPrefix(challenge, "Bearer") || !strings.Contains(challenge, want) {
			t.Fatalf("WWW-Authenticate = %q", challenge)
		}
	})
}

func TestDiscoveryIsAbsentInThePersonalEdition(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, map[string]string{"APP_EDITION": "personal"})

	for _, path := range []string{
		"/.well-known/oauth-authorization-server", "/.well-known/oauth-protected-resource",
		"/.well-known/oauth-protected-resource/mcp", "/.well-known/jwks.json", "/authorize", "/mcp",
	} {
		s.App.Client(t).Get(path).AssertNotFound()
	}

	s.App.Client(t).PostForm("/token", url.Values{}).AssertNotFound()
	s.App.Client(t).PostJSON("/mcp", map[string]any{}).AssertNotFound()
}

// refused asserts an authorization request produced no code and no consent screen.
func refused(t *testing.T, s *oauthtest.Stack, p oauthtest.Person, q url.Values) {
	t.Helper()

	res := s.Browser(t, p, p.Workspace.ID).Get("/authorize?" + q.Encode())

	if strings.Contains(res.Header("Location"), "code=") || res.Status() == 200 {
		t.Fatalf("a bad request was accepted (%d, %q) for %v", res.Status(), res.Header("Location"), q)
	}
}

func TestAuthorizationCode(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")

	exchange := func(code, verifier string, mutate func(url.Values)) *vukemtest.Response {
		form := url.Values{
			"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {oauthtest.RedirectURI},
			"client_id": {oauthtest.ClientID}, "code_verifier": {verifier}, "resource": {oauthtest.Resource},
		}
		if mutate != nil {
			mutate(form)
		}

		return s.Token(t, form)
	}

	t.Run("a code is spent once and its replay ends the grant", func(t *testing.T) {
		verifier, challenge := oauthtest.Verifier()
		code := s.Approve(t, ana, ana.Workspace.ID, oauthtest.AuthorizeQuery(challenge, "accounts:read"))

		first := exchange(code, verifier, nil).AssertOK()
		access := jsonBody(t, first.Body())["access_token"].(string)

		s.Call(t, access, "tools/list", nil).AssertOK()

		if res := exchange(code, verifier, nil); res.Status() != 400 || jsonBody(t, res.Body())["error"] != "invalid_grant" {
			t.Fatalf("the replay answered %d %s", res.Status(), res.Body())
		}

		s.Call(t, access, "tools/list", nil).AssertUnauthorized()
	})

	t.Run("a wrong verifier is refused and spends the code", func(t *testing.T) {
		verifier, challenge := oauthtest.Verifier()
		code := s.Approve(t, ana, ana.Workspace.ID, oauthtest.AuthorizeQuery(challenge, "accounts:read"))

		if res := exchange(code, "not-the-verifier-"+verifier, nil); res.Status() != 400 {
			t.Fatalf("a wrong verifier answered %d %s", res.Status(), res.Body())
		}

		exchange(code, verifier, nil).AssertStatus(400)
	})

	t.Run("a code is bound to its redirect, client and resource", func(t *testing.T) {
		for name, mutate := range map[string]func(url.Values){
			"another redirect": func(f url.Values) { f.Set("redirect_uri", "https://evil.example/callback") },
			"another client":   func(f url.Values) { f.Set("client_id", "someone-else") },
			"another resource": func(f url.Values) { f.Set("resource", "https://other.example/mcp") },
			"no verifier":      func(f url.Values) { f.Del("code_verifier") },
		} {
			verifier, challenge := oauthtest.Verifier()
			code := s.Approve(t, ana, ana.Workspace.ID, oauthtest.AuthorizeQuery(challenge, "accounts:read"))

			if res := exchange(code, verifier, mutate); res.Status() == 200 {
				t.Errorf("%s: a token was issued: %s", name, res.Body())
			}
		}
	})

	t.Run("a request that breaks a rule gets no code", func(t *testing.T) {
		_, challenge := oauthtest.Verifier()

		cases := map[string]func(url.Values){
			"no resource":              func(q url.Values) { q.Del("resource") },
			"another resource":         func(q url.Values) { q.Set("resource", "https://other.example/mcp") },
			"no challenge":             func(q url.Values) { q.Del("code_challenge") },
			"a plain challenge":        func(q url.Values) { q.Set("code_challenge_method", "plain") },
			"an unknown scope":         func(q url.Values) { q.Set("scope", "accounts:read admin:everything") },
			"a redirect not known":     func(q url.Values) { q.Set("redirect_uri", "https://evil.example/callback") },
			"a redirect with a suffix": func(q url.Values) { q.Set("redirect_uri", oauthtest.RedirectURI+"/../x") },
			"an unknown client":        func(q url.Values) { q.Set("client_id", "someone-else") },
			"the implicit grant":       func(q url.Values) { q.Set("response_type", "token") },
		}

		for name, mutate := range cases {
			q := oauthtest.AuthorizeQuery(challenge, "accounts:read")
			mutate(q)

			t.Run(name, func(t *testing.T) { refused(t, s, ana, q) })
		}
	})

	t.Run("denying sends the client back without a code", func(t *testing.T) {
		_, challenge := oauthtest.Verifier()
		action, form := s.Consent(t, ana, ana.Workspace.ID, oauthtest.AuthorizeQuery(challenge, "accounts:read"))
		form.Set("decision", "deny")

		res := s.Browser(t, ana, ana.Workspace.ID).PostForm(action, form)

		loc, _ := url.Parse(res.Header("Location"))
		if loc == nil || loc.Query().Get("error") != "access_denied" || loc.Query().Get("code") != "" {
			t.Fatalf("deny answered %d %q", res.Status(), res.Header("Location"))
		}
	})

	t.Run("the token endpoint takes no GET and no other grant", func(t *testing.T) {
		s.App.Client(t).Get("/token").AssertStatus(405)

		for _, grant := range []string{"password", "client_credentials", "implicit"} {
			if res := s.Token(t, url.Values{"grant_type": {grant}, "client_id": {oauthtest.ClientID}}); res.Status() != 400 {
				t.Errorf("grant_type %s answered %d", grant, res.Status())
			}
		}
	})
}

func TestClientMetadataSSRF(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")

	_, challenge := oauthtest.Verifier()

	for _, id := range []string{
		"http://client.example.test/meta.json",
		"https://127.0.0.1/client.json",
		"https://localhost/client.json",
		"https://10.0.0.5/client.json",
		"https://192.168.1.10/client.json",
		"https://169.254.169.254/latest/meta-data",
		"https://[::1]/client.json",
		"https://[::ffff:127.0.0.1]/client.json",
		"https://user:" + "word" + "@client.example.test/meta.json",
		"https://client.example.test/",
		"https://client.example.test/a/../meta.json",
		"https://client.example.test/meta.json#fragment",
		"file:///etc/passwd",
		"ftp://client.example.test/meta.json",
	} {
		q := oauthtest.AuthorizeQuery(challenge, "accounts:read")
		q.Set("client_id", id)

		res := s.Browser(t, ana, ana.Workspace.ID).Get("/authorize?" + q.Encode())
		if res.Status() != 400 || res.Header("Location") != "" {
			t.Errorf("client_id %q answered %d (Location %q), want a plain 400", id, res.Status(), res.Header("Location"))
		}
	}
}

func TestRefresh(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")

	refresh := func(token string) *vukemtest.Response {
		return s.Token(t, url.Values{
			"grant_type": {"refresh_token"}, "refresh_token": {token}, "client_id": {oauthtest.ClientID},
			"resource": {oauthtest.Resource},
		})
	}

	tok := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")

	db, err := s.App.DB()
	if err != nil {
		t.Fatal(err)
	}

	var stored int

	err = db.SQL().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM oauth_refresh_tokens WHERE hash = ?", tok.Refresh).Scan(&stored)
	if err != nil || stored != 0 {
		t.Fatalf("the refresh token is stored in the clear (%d rows, %v)", stored, err)
	}

	next := jsonBody(t, refresh(tok.Refresh).AssertOK().Body())
	newAccess, newRefresh := next["access_token"].(string), next["refresh_token"].(string)

	if newRefresh == tok.Refresh || newAccess == tok.Access {
		t.Fatal("the refresh did not rotate the tokens")
	}

	s.Call(t, newAccess, "tools/list", nil).AssertOK()

	t.Run("a used refresh token revokes the whole grant", func(t *testing.T) {
		if res := refresh(tok.Refresh); res.Status() != 400 || jsonBody(t, res.Body())["error"] != "invalid_grant" {
			t.Fatalf("reuse answered %d %s", res.Status(), res.Body())
		}

		refresh(newRefresh).AssertStatus(400)
		s.Call(t, newAccess, "tools/list", nil).AssertUnauthorized()
		s.Call(t, tok.Access, "tools/list", nil).AssertUnauthorized()
	})

	t.Run("a refresh token is for its client", func(t *testing.T) {
		other := s.Authorize(t, ana, ana.Workspace.ID, "accounts:read")

		res := s.Token(t, url.Values{
			"grant_type": {"refresh_token"}, "refresh_token": {other.Refresh}, "client_id": {"someone-else"},
		})
		if res.Status() == 200 {
			t.Fatalf("another client refreshed: %s", res.Body())
		}
	})

	t.Run("an unknown refresh token is refused", func(t *testing.T) {
		refresh("not-a-token").AssertStatus(400)
	})
}
