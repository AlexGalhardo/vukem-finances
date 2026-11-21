package tests

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

// makeToken creates an API token through the Tokens page, as a person would,
// and returns the plain token the page shows once.
func makeToken(t *testing.T, app *vukemtest.App, w who, days int, scopes ...string) string {
	t.Helper()

	// A fresh client: Inertia() would turn the caller's client into an Inertia one.
	res := browser(t, app, w.User, w.Workspace.ID).Inertia("").PostForm("/tokens", url.Values{
		"name": {"test token"}, "scopes": scopes, "days": {strconv.Itoa(days)},
	}).AssertOK()

	secret, ok := res.JSONPath("props.created.secret")
	if !ok {
		t.Fatalf("the Tokens page shows no secret: %s", res.Body())
	}

	return secret.(string)
}

// apiClient is a client that sends only a bearer token, no cookie.
func apiClient(t *testing.T, app *vukemtest.App, token string) *vukemtest.Client {
	t.Helper()

	return app.Client(t).WithToken(token)
}
