package tests

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/oauth/oauthtest"
)

// TestOAuthDiscoveryAndDecision covers the two protected resource metadata
// documents (the root one and the one under the /mcp path), the consent screen
// as a page, and the POST that decides it: a visitor's decision issues no code.
func TestOAuthDiscoveryAndDecision(t *testing.T) {
	t.Parallel()

	s := oauthtest.New(t, nil)
	ana := s.SignUp(t, "Ana", "ana@example.com")

	for _, rt := range []string{"GET /.well-known/oauth-protected-resource", "GET /.well-known/oauth-protected-resource/mcp"} {
		_, path, _ := strings.Cut(rt, " ")
		res := s.App.Client(t).Get(path).AssertOK()

		var doc struct {
			Resource   string   `json:"resource"`
			Authorizes []string `json:"authorization_servers"`
		}

		if err := json.Unmarshal([]byte(res.Body()), &doc); err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		if !strings.HasSuffix(doc.Resource, "/mcp") || len(doc.Authorizes) == 0 {
			t.Errorf("%s = %+v, want the /mcp resource and its authorization server", path, doc)
		}
	}

	_, challenge := oauthtest.Verifier()
	q := oauthtest.AuthorizeQuery(challenge, "accounts:read")

	s.Browser(t, ana, ana.Workspace.ID).Inertia("").Get("/authorize?" + q.Encode()).AssertInertia("OAuthConsent")

	_, form := s.Consent(t, ana, ana.Workspace.ID, q)
	form.Set("decision", "approve")

	res := s.App.Client(t).WithHeader("Sec-Fetch-Site", "same-origin").PostForm("/authorize", form)
	if loc := res.Header("Location"); strings.Contains(loc, "code=") {
		t.Fatalf("a visitor's decision issued a code: %d %q", res.Status(), loc)
	}

	approved := s.Browser(t, ana, ana.Workspace.ID).PostForm("/authorize", form)
	if loc := approved.Header("Location"); !strings.Contains(loc, "code=") {
		t.Fatalf("the signed-in person's approval gave %d %q, want a redirect with a code", approved.Status(), loc)
	}
}
