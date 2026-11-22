package desktopgate_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/desktopgate"
)

func newGate(t *testing.T) (*desktopgate.Gate, string, http.Handler) {
	t.Helper()

	token, err := desktopgate.NewToken()
	if err != nil {
		t.Fatal(err)
	}

	g, err := desktopgate.New(token)
	if err != nil {
		t.Fatal(err)
	}

	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if desktopgate.Verified(r.Context()) {
			_, _ = w.Write([]byte("verified"))
			return
		}

		_, _ = w.Write([]byte("unverified"))
	})

	return g, token, g.Wrap(app)
}

func do(h http.Handler, method, target, host, remote string, hdr map[string]string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://"+host+target, nil)
	r.RemoteAddr = remote

	for k, v := range hdr {
		r.Header.Set(k, v)
	}

	if cookie != nil {
		r.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	return w
}

func enter(t *testing.T, h http.Handler, token string) *http.Cookie {
	t.Helper()

	w := do(h, http.MethodGet, desktopgate.EntryPath(token), "127.0.0.1:5000", "127.0.0.1:4000", nil, nil)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/" {
		t.Fatalf("enter = %d %q, want 303 to /", w.Code, w.Header().Get("Location"))
	}

	for _, c := range w.Result().Cookies() {
		if c.Name == desktopgate.CookieName {
			return c
		}
	}

	t.Fatal("no cookie minted")

	return nil
}

func TestGateRejectsWeakToken(t *testing.T) {
	t.Parallel()

	for _, tok := range []string{"", "short", strings.Repeat("a", 42), strings.Repeat("a/", 30)} {
		if _, err := desktopgate.New(tok); err == nil {
			t.Errorf("New(%q) accepted a weak token", tok)
		}
	}
}

func TestGateCookieAttributes(t *testing.T) {
	t.Parallel()

	_, token, h := newGate(t)
	c := enter(t, h, token)

	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" {
		t.Errorf("cookie = %+v, want HttpOnly, SameSite=Strict, Path=/", c)
	}
}

func TestGateServesWithCookieAndSetsHeaders(t *testing.T) {
	t.Parallel()

	_, token, h := newGate(t)
	c := enter(t, h, token)

	w := do(h, http.MethodGet, "/dashboard", "127.0.0.1:5000", "127.0.0.1:4000", nil, c)
	if w.Code != http.StatusOK || w.Body.String() != "verified" {
		t.Fatalf("with cookie = %d %q, want 200 verified", w.Code, w.Body.String())
	}

	if got := w.Header().Get("Content-Security-Policy"); got != "frame-ancestors 'none'" {
		t.Errorf("CSP = %q", got)
	}

	if got := w.Header().Get("Referrer-Policy"); got != "same-origin" {
		t.Errorf("Referrer-Policy = %q", got)
	}
}

func TestGateWrongCookieAndWrongToken(t *testing.T) {
	t.Parallel()

	_, token, h := newGate(t)

	w := do(h, http.MethodGet, desktopgate.EntryPath(strings.Repeat("x", 43)), "127.0.0.1:5000", "127.0.0.1:4000", nil, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong token = %d, want 401", w.Code)
	}

	// The wrong guess must not have spent the real token.
	c := enter(t, h, token)

	forged := &http.Cookie{Name: desktopgate.CookieName, Value: "forged"}
	if w := do(h, http.MethodGet, "/", "127.0.0.1:5000", "127.0.0.1:4000", nil, forged); w.Code != http.StatusForbidden {
		t.Errorf("forged cookie = %d, want 403", w.Code)
	}

	if w := do(h, http.MethodGet, "/", "127.0.0.1:5000", "127.0.0.1:4000", nil, c); w.Code != http.StatusOK {
		t.Errorf("real cookie = %d, want 200", w.Code)
	}
}

func TestGateRefusesForeignRequests(t *testing.T) {
	t.Parallel()

	_, token, h := newGate(t)
	c := enter(t, h, token)

	cases := []struct {
		name   string
		host   string
		remote string
		hdr    map[string]string
	}{
		{"wrong host", "evil.example:5000", "127.0.0.1:4000", nil},
		{"non-loopback peer", "127.0.0.1:5000", "192.168.1.20:4000", nil},
		{"forwarded for", "127.0.0.1:5000", "127.0.0.1:4000", map[string]string{"X-Forwarded-For": "203.0.113.9"}},
		{"forwarded", "127.0.0.1:5000", "127.0.0.1:4000", map[string]string{"Forwarded": "for=203.0.113.9"}},
		{"via", "127.0.0.1:5000", "127.0.0.1:4000", map[string]string{"Via": "1.1 proxy"}},
		{"same-site fetch", "127.0.0.1:5000", "127.0.0.1:4000", map[string]string{"Sec-Fetch-Site": "same-site"}},
		{"cross-site fetch", "127.0.0.1:5000", "127.0.0.1:4000", map[string]string{"Sec-Fetch-Site": "cross-site"}},
		{"other origin", "127.0.0.1:5000", "127.0.0.1:4000", map[string]string{"Origin": "http://127.0.0.1:9999"}},
		{"null origin", "127.0.0.1:5000", "127.0.0.1:4000", map[string]string{"Origin": "null"}},
	}

	for _, tc := range cases {
		if w := do(h, http.MethodPost, "/transactions", tc.host, tc.remote, tc.hdr, c); w.Code != http.StatusForbidden {
			t.Errorf("%s with cookie = %d, want 403", tc.name, w.Code)
		}
	}

	same := map[string]string{"Origin": "http://127.0.0.1:5000", "Sec-Fetch-Site": "same-origin"}
	if w := do(h, http.MethodPost, "/transactions", "127.0.0.1:5000", "[::1]:4000", same, c); w.Code != http.StatusOK {
		t.Errorf("same origin = %d, want 200", w.Code)
	}
}

func TestGateEnterIsAlsoGuarded(t *testing.T) {
	t.Parallel()

	_, token, h := newGate(t)

	for _, host := range []string{"evil.example", "10.0.0.5:5000"} {
		if w := do(h, http.MethodGet, desktopgate.EntryPath(token), host, "127.0.0.1:4000", nil, nil); w.Code != http.StatusForbidden {
			t.Errorf("enter with Host %q = %d, want 403", host, w.Code)
		}
	}

	// Neither refusal spent the token.
	enter(t, h, token)
}

func TestGateTokenIsSpentOnce(t *testing.T) {
	t.Parallel()

	_, token, h := newGate(t)

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		codes = map[int]int{}
	)

	for range 50 {
		wg.Go(func() {
			w := do(h, http.MethodGet, desktopgate.EntryPath(token), "127.0.0.1:5000", "127.0.0.1:4000", nil, nil)

			mu.Lock()
			codes[w.Code]++
			mu.Unlock()
		})
	}

	wg.Wait()

	if codes[http.StatusSeeOther] != 1 || codes[http.StatusUnauthorized] != 49 {
		t.Fatalf("codes = %v, want one 303 and 49 401", codes)
	}
}

func TestGateNothingAnswersWithoutCookie(t *testing.T) {
	t.Parallel()

	_, _, h := newGate(t)

	for _, p := range []string{"/", "/up", "/build/app.js", "/api/v1/accounts", "/desktop/enter"} {
		if w := do(h, http.MethodGet, p, "127.0.0.1:5000", "127.0.0.1:4000", nil, nil); w.Code != http.StatusForbidden {
			t.Errorf("GET %s = %d, want 403", p, w.Code)
		}
	}
}
