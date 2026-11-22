package tests

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/desktopgate"
)

// gatedServer serves the personal app behind the desktop gate on a real
// loopback socket and returns the server, the entry URL and a client that
// does not follow redirects.
func gatedServer(t *testing.T) (*httptest.Server, string, *http.Client, *vukemtest.App) {
	t.Helper()

	app := newPersonalApp(t, map[string]string{"SETUP_TOKEN": setupToken})

	token, err := desktopgate.NewToken()
	if err != nil {
		t.Fatal(err)
	}

	gate, err := desktopgate.New(token)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(gate.Wrap(app.Handler()))
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	return srv, srv.URL + desktopgate.EntryPath(token), client, app
}

func status(t *testing.T, c *http.Client, req *http.Request) int {
	t.Helper()

	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	return res.StatusCode
}

func get(t *testing.T, target string) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatal(err)
	}

	return req
}

func TestSecurity_GateNoCookie(t *testing.T) {
	t.Parallel()

	srv, entry, client, _ := gatedServer(t)

	for _, p := range []string{"/", "/up", "/setup", "/login", "/build/manifest.json", "/api/v1/accounts", "/dashboard"} {
		if got := status(t, client, get(t, srv.URL+p)); got != http.StatusForbidden {
			t.Errorf("GET %s without cookie = %d, want 403", p, got)
		}
	}

	// After the exchange the same client is let in.
	if got := status(t, client, get(t, entry)); got != http.StatusSeeOther {
		t.Fatalf("exchange = %d, want 303", got)
	}

	if got := status(t, client, get(t, srv.URL+"/up")); got != http.StatusOK {
		t.Errorf("GET /up with cookie = %d, want 200", got)
	}

	// A client that never exchanged is still refused.
	if got := status(t, &http.Client{}, get(t, srv.URL+"/up")); got != http.StatusForbidden {
		t.Errorf("GET /up from another process = %d, want 403", got)
	}
}

func TestSecurity_GateRebinding(t *testing.T) {
	t.Parallel()

	srv, entry, client, _ := gatedServer(t)

	if got := status(t, client, get(t, entry)); got != http.StatusSeeOther {
		t.Fatalf("exchange = %d, want 303", got)
	}

	// An attacker domain rebound to 127.0.0.1 sends its own Host.
	req := get(t, srv.URL+"/up")
	req.Host = "attacker.example"

	if got := status(t, client, req); got != http.StatusForbidden {
		t.Errorf("rebound Host = %d, want 403", got)
	}

	// A proxy header means the request did not come straight from the window.
	req = get(t, srv.URL+"/up")
	req.Header.Set("X-Forwarded-For", "203.0.113.9")

	if got := status(t, client, req); got != http.StatusForbidden {
		t.Errorf("proxy header = %d, want 403", got)
	}

	// The exchange itself is guarded too.
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}

	other := &http.Client{Jar: jar, CheckRedirect: client.CheckRedirect}
	req = get(t, entry)
	req.Host = "attacker.example"

	if got := status(t, other, req); got != http.StatusForbidden {
		t.Errorf("exchange with a rebound Host = %d, want 403", got)
	}
}

func TestSecurity_GateTokenSpentOnce(t *testing.T) {
	t.Parallel()

	_, entry, _, _ := gatedServer(t)

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		codes = map[int]int{}
	)

	for range 50 {
		wg.Go(func() {
			got := status(t, &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, get(t, entry))

			mu.Lock()
			codes[got]++
			mu.Unlock()
		})
	}

	wg.Wait()

	if codes[http.StatusSeeOther] != 1 || codes[http.StatusUnauthorized] != 49 {
		t.Fatalf("exchanges = %v, want one 303 and 49 401", codes)
	}
}

// The first-run form looked dead in the desktop app: without a Referer a failed
// validation went to / and on to /setup, and that extra stop lost the error.
func TestDesktopFormErrorReturnsToItsPage(t *testing.T) {
	t.Parallel()

	srv, entry, client, _ := gatedServer(t)

	if got := status(t, client, get(t, entry)); got != http.StatusSeeOther {
		t.Fatalf("exchange = %d, want 303", got)
	}

	form := url.Values{
		"name": {"Alexandra"}, "email": {"owner@example.com"}, "currency": {"BRL"},
		"password": {"short"}, "password_confirmation": {"short"},
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/setup", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("Origin", srv.URL)
	req.Header.Set("Referer", srv.URL+"/setup")

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	_ = res.Body.Close()

	if got := res.Header.Get("Location"); got != "/setup" && got != srv.URL+"/setup" {
		t.Fatalf("a failed form went to %q, want its own page", got)
	}

	page, err := client.Do(get(t, srv.URL+"/setup"))
	if err != nil {
		t.Fatal(err)
	}

	body, _ := io.ReadAll(page.Body)
	_ = page.Body.Close()

	unescaped := strings.NewReplacer("&#34;", `"`, "&quot;", `"`).Replace(string(body))
	if !strings.Contains(unescaped, `"errors":{"password`) {
		t.Errorf("the page does not show the password error: %s", unescaped)
	}
}

func TestSecurity_GateCookieSatisfiesSetupSecret(t *testing.T) {
	t.Parallel()

	srv, entry, client, app := gatedServer(t)

	if got := status(t, client, get(t, entry)); got != http.StatusSeeOther {
		t.Fatalf("exchange = %d, want 303", got)
	}

	page := get(t, srv.URL+"/setup")

	res, err := client.Do(page)
	if err != nil {
		t.Fatal(err)
	}

	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()

	unescaped := strings.NewReplacer("&#34;", `"`, "&quot;", `"`).Replace(string(body))
	if !strings.Contains(unescaped, `"codeRequired":false`) {
		t.Errorf("setup page in the desktop app = %s, want codeRequired false", body)
	}

	form := url.Values{
		"name": {"Alexandra"}, "email": {"owner@example.com"}, "currency": {"BRL"},
		"password": {testPassword}, "password_confirmation": {testPassword},
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/setup", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", srv.URL)

	if got := status(t, client, req); got != http.StatusSeeOther && got != http.StatusFound {
		t.Fatalf("setup with the desktop cookie and no code = %d, want a redirect", got)
	}

	vukemtest.AssertDatabaseCount(t, app, "users", 1)
}

func TestSecurity_GateCookieAloneCannotBypassSetupSecret(t *testing.T) {
	t.Parallel()

	// No gate is mounted, so a cookie named like the desktop one is only a
	// cookie: the setup secret is still required.
	app := newPersonalApp(t, map[string]string{"SETUP_TOKEN": setupToken})
	c := app.Client(t).WithCookie(desktopgate.CookieName, "anything").Inertia("")

	c.PostJSON("/setup", setupForm("owner@example.com", "")).AssertForbidden()
	vukemtest.AssertDatabaseCount(t, app, "users", 0)

	c.Get("/setup").AssertOK().AssertInertia("Setup").PropEquals("codeRequired", true)
}

func TestSecurity_GateHeadersSurviveTheApp(t *testing.T) {
	t.Parallel()

	srv, entry, client, _ := gatedServer(t)

	if got := status(t, client, get(t, entry)); got != http.StatusSeeOther {
		t.Fatalf("exchange = %d, want 303", got)
	}

	res, err := client.Do(get(t, srv.URL+"/setup"))
	if err != nil {
		t.Fatal(err)
	}

	_ = res.Body.Close()

	// The app sets strict-origin-when-cross-origin itself; the gate wins.
	if got := res.Header.Get("Referrer-Policy"); got != "same-origin" {
		t.Errorf("Referrer-Policy = %q, want same-origin", got)
	}

	if csp := strings.Join(res.Header.Values("Content-Security-Policy"), "; "); !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("Content-Security-Policy = %q, want frame-ancestors 'none'", csp)
	}
}
