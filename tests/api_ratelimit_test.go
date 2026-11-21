package tests

import (
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

// untilLimited calls fn until it answers 429 and returns that answer and how many
// calls passed before it. The bucket refills while the calls run (a slow machine
// or -race), so the exact count is not fixed: it is at least the limit and the
// 429 comes soon after.
func untilLimited(t *testing.T, limit int, fn func() *vukemtest.Response) (*vukemtest.Response, int) {
	t.Helper()

	for passed := 0; passed < limit*4; passed++ {
		res := fn()
		if res.Status() == http.StatusTooManyRequests {
			return res, passed
		}
	}

	t.Fatalf("no 429 after %d calls", limit*4)

	return nil, 0
}

func TestAPIRateLimit(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	token := makeToken(t, app, ana, 30, "tags:write")
	other := makeToken(t, app, ana, 30, "tags:write")
	c := apiClient(t, app, token)

	res, passed := untilLimited(t, 120, func() *vukemtest.Response { return c.Get("/api/v1/me") })
	if passed < 120 {
		t.Errorf("%d reads passed before the 429, want at least 120", passed)
	}

	assertProblem(t, res, 429, "rate_limited")

	if res.Header("Retry-After") == "" || res.Header("RateLimit-Limit") != "120" || res.Header("RateLimit-Remaining") != "0" {
		t.Errorf("Retry-After = %q, RateLimit-Limit = %q, RateLimit-Remaining = %q",
			res.Header("Retry-After"), res.Header("RateLimit-Limit"), res.Header("RateLimit-Remaining"))
	}

	// Reads and writes are counted apart, and each token has its own budget.
	c.PostJSON("/api/v1/tags", map[string]any{"name": "one"}).AssertCreated()
	apiClient(t, app, other).Get("/api/v1/me").AssertOK()

	res, passed = untilLimited(t, 60, func() *vukemtest.Response { return c.PostJSON("/api/v1/tags", map[string]any{"name": ""}) })
	if passed < 59 {
		t.Errorf("%d writes passed before the 429, want at least 59 (one was spent above)", passed)
	}

	assertProblem(t, res, 429, "rate_limited")
	res.AssertHeaderContains("RateLimit-Limit", "60")
}

func TestAPIFailedTokenLimit(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	good := makeToken(t, app, ana, 30, "accounts:read")

	res, passed := untilLimited(t, 60, func() *vukemtest.Response { return apiClient(t, app, "vf_AAAA.BBBB").Get("/api/v1/me") })
	if passed < 60 {
		t.Errorf("%d failed lookups passed before the 429, want at least 60", passed)
	}

	assertProblem(t, res, 429, "rate_limited")

	if res.Header("Retry-After") == "" {
		t.Error("no Retry-After on the 429")
	}

	// The address is out of failed lookups: refused before any token is checked.
	assertProblem(t, apiClient(t, app, good).Get("/api/v1/me"), 429, "rate_limited")
}
