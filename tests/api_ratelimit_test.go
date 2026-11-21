package tests

import (
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

func TestAPIRateLimit(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	token := makeToken(t, app, ana, 30, "tags:write")
	other := makeToken(t, app, ana, 30, "tags:write")
	c := apiClient(t, app, token)

	var last *vukemtest.Response

	for i := range 120 {
		last = c.Get("/api/v1/me")
		if last.Status() != http.StatusOK {
			t.Fatalf("read %d answered %d, want 200", i+1, last.Status())
		}
	}

	res := c.Get("/api/v1/me")
	assertProblem(t, res, 429, "rate_limited")
	res.AssertHeaderContains("RateLimit-Limit", "120")

	if res.Header("Retry-After") == "" || res.Header("RateLimit-Remaining") != "0" {
		t.Errorf("Retry-After = %q, RateLimit-Remaining = %q", res.Header("Retry-After"), res.Header("RateLimit-Remaining"))
	}

	// Reads and writes are counted apart, and each token has its own budget.
	c.PostJSON("/api/v1/tags", map[string]any{"name": "one"}).AssertCreated()
	apiClient(t, app, other).Get("/api/v1/me").AssertOK()

	for i := range 59 {
		if s := c.PostJSON("/api/v1/tags", map[string]any{"name": ""}).Status(); s != http.StatusUnprocessableEntity {
			t.Fatalf("write %d answered %d, want 422", i+2, s)
		}
	}

	assertProblem(t, c.PostJSON("/api/v1/tags", map[string]any{"name": "late"}), 429, "rate_limited")
}

func TestAPIFailedTokenLimit(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	good := makeToken(t, app, ana, 30, "accounts:read")

	for i := range 60 {
		if s := apiClient(t, app, "vf_AAAA.BBBB").Get("/api/v1/me").Status(); s != http.StatusUnauthorized {
			t.Fatalf("failed lookup %d answered %d, want 401", i+1, s)
		}
	}

	// The address is out of failed lookups: refused before any token is checked.
	res := apiClient(t, app, "vf_AAAA.BBBB").Get("/api/v1/me")
	assertProblem(t, res, 429, "rate_limited")

	if res.Header("Retry-After") == "" {
		t.Error("no Retry-After on the 429")
	}

	assertProblem(t, apiClient(t, app, good).Get("/api/v1/me"), 429, "rate_limited")
}
