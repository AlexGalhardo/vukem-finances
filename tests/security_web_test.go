package tests

import (
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/billing"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// The web attack suite (epic 24 task 6): request forgery, cookies and sessions,
// headers, enumeration and redirects, attacked as a hostile page or a hostile
// visitor would.

// secWebNoCSRF are the routes whose credential is not a cookie, so no
// cross-site check applies to them: the bearer API, the webhooks (a secret or a
// signature) and the remote MCP endpoints.
func secWebNoCSRF(path string) bool {
	for _, prefix := range []string{
		"/api/", "/mcp", "/token", "/revoke", "/.well-known/", billing.WebhookPath, billing.PixWebhookPath,
		config.WebhookPath + "/", "/resend", "/abacatepay/", "/stripe/",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

func TestSecurity_Web_CSRF(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	// A signed-in person whose browser sends the request on behalf of another site.
	forged := func(site string) *vukemtest.Client {
		return app.Client(t).ActingAs("web", ana.User).WithHeader("Sec-Fetch-Site", site).
			WithSession(map[string]any{"workspace_id": itoa(ana.Workspace.ID)})
	}

	before := rowCount(t, app, "transactions")

	for _, route := range app.Router.Routes() {
		if route.Method == http.MethodGet || route.Method == http.MethodHead || secWebNoCSRF(route.Path) {
			continue
		}

		target := pathParam.ReplaceAllString(route.Path, "1")

		for _, site := range []string{"cross-site", "same-site"} {
			t.Run(route.Method+" "+route.Path+" from "+site, func(t *testing.T) {
				res := forged(site).Do(route.Method, target, nil, "")
				if res.Status() != http.StatusForbidden {
					t.Fatalf("answered %d, want 403", res.Status())
				}
			})
		}
	}

	t.Run("a forged form changes nothing", func(t *testing.T) {
		res := forged("cross-site").PostForm("/transactions", url.Values{
			"accountId": {itoa(firstAccount(t, app, ana))}, "kind": {"expense"}, "amount": {"1.00"}, "date": {today()}, "description": {"forged"},
		})
		if res.Status() != http.StatusForbidden || rowCount(t, app, "transactions") != before {
			t.Fatalf("answered %d, %d rows", res.Status(), rowCount(t, app, "transactions"))
		}
	})

	t.Run("a browser that sends no fetch metadata but a foreign Origin is refused", func(t *testing.T) {
		c := app.Client(t).ActingAs("web", ana.User).WithHeader("Origin", "https://evil.example")

		res := c.PostForm("/transactions", url.Values{"accountId": {"1"}, "kind": {"expense"}, "amount": {"1.00"}, "date": {today()}})
		if res.Status() != http.StatusForbidden {
			t.Fatalf("answered %d, want 403", res.Status())
		}
	})

	t.Run("logout and locale cannot be forged either", func(t *testing.T) {
		for _, target := range []string{"/logout", "/locale"} {
			if res := forged("cross-site").PostForm(target, url.Values{"language": {"es"}}); res.Status() != http.StatusForbidden {
				t.Errorf("POST %s answered %d", target, res.Status())
			}
		}
	})

	t.Run("a safe method is never a state change", func(t *testing.T) {
		for _, route := range app.Router.Routes() {
			if route.Method != http.MethodGet || secWebNoCSRF(route.Path) || strings.Contains(route.Path, "{") {
				continue
			}

			// GET answers a page or a download; none of them may write. The row counts prove it below.
			forged("cross-site").Get(route.Path)
		}

		if got := rowCount(t, app, "transactions"); got != before {
			t.Fatalf("a GET changed the transactions: %d, was %d", got, before)
		}
	})
}

func TestSecurity_Web_Headers(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	token := fullToken(t, app, ana)

	nonceOf := regexp.MustCompile(`script-src 'self' 'nonce-([A-Za-z0-9_-]+)'`)

	classes := map[string]*vukemtest.Response{
		"a public page":         app.Client(t).Get("/login"),
		"a missing page":        app.Client(t).Get("/no-such-page"),
		"a signed-in page":      ana.Client.Get("/dashboard"),
		"an inertia answer":     ana.Client.Inertia("").Get("/dashboard"),
		"a redirect":            app.Client(t).Get("/dashboard"),
		"the csv download":      ana.Client.Get("/transactions/export.csv"),
		"the json export":       ana.Client.Get("/workspace/export.json"),
		"the api":               apiClient(t, app, token).Get("/api/v1/accounts"),
		"an api refusal":        app.Client(t).Get("/api/v1/accounts"),
		"the openapi document":  app.Client(t).Get("/api/v1/openapi.json"),
		"a refused webhook":     app.Client(t).Do(http.MethodPost, billing.WebhookPath, strings.NewReader("{}"), "application/json"),
		"a health check":        app.Client(t).Get("/up"),
		"a method not allowed":  app.Client(t).Do(http.MethodDelete, "/login", nil, ""),
		"a static file missing": app.Client(t).Get("/build/assets/missing.js"),
	}

	for name, res := range classes {
		t.Run(name, func(t *testing.T) {
			csp := res.Header("Content-Security-Policy")
			for _, want := range []string{"default-src 'self'", "frame-ancestors 'none'", "object-src 'none'", "base-uri 'self'", "form-action 'self'"} {
				if !strings.Contains(csp, want) {
					t.Errorf("CSP %q lacks %s", csp, want)
				}
			}

			if strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "unsafe-eval") || strings.Contains(csp, "*") {
				t.Errorf("CSP %q is loose", csp)
			}

			if m := nonceOf.FindStringSubmatch(csp); m == nil || len(m[1]) < 16 {
				t.Errorf("CSP %q has no nonce of enough entropy", csp)
			}

			if got := res.Header("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q", got)
			}

			if got := res.Header("X-Frame-Options"); got != "DENY" {
				t.Errorf("X-Frame-Options = %q", got)
			}

			if got := res.Header("Referrer-Policy"); got != "strict-origin-when-cross-origin" && got != "no-referrer" {
				t.Errorf("Referrer-Policy = %q", got)
			}
		})
	}

	t.Run("the nonce is new on every response", func(t *testing.T) {
		a := nonceOf.FindStringSubmatch(app.Client(t).Get("/login").Header("Content-Security-Policy"))
		b := nonceOf.FindStringSubmatch(app.Client(t).Get("/login").Header("Content-Security-Policy"))

		if a == nil || b == nil || a[1] == b[1] {
			t.Fatalf("nonces %v and %v", a, b)
		}
	})

	t.Run("the nonce of the policy is the one of the page scripts", func(t *testing.T) {
		res := app.Client(t).Get("/login")

		m := nonceOf.FindStringSubmatch(res.Header("Content-Security-Policy"))
		if m == nil {
			t.Fatal("no nonce")
		}

		for _, tag := range regexp.MustCompile(`<script[^>]*>`).FindAllString(res.Body(), -1) {
			// A data block (type="application/json") is never executed, so it needs no nonce.
			if !strings.Contains(tag, `nonce="`+m[1]+`"`) && !strings.Contains(tag, "src=") && !strings.Contains(tag, `type="application/json"`) {
				t.Errorf("an inline script has no nonce: %s", tag)
			}
		}
	})

	t.Run("a download is an attachment and a json answer is not sniffed", func(t *testing.T) {
		if cd := classes["the csv download"].Header("Content-Disposition"); !strings.HasPrefix(cd, "attachment") {
			t.Errorf("the csv is served as %q", cd)
		}

		if ct := classes["the api"].Header("Content-Type"); !strings.Contains(ct, "json") {
			t.Errorf("the api answers %q", ct)
		}
	})

	t.Run("HSTS is absent outside production", func(t *testing.T) {
		if got := classes["a public page"].Header("Strict-Transport-Security"); got != "" {
			t.Errorf("Strict-Transport-Security = %q outside production", got)
		}
	})
}

func TestSecurity_Web_Production(t *testing.T) {
	t.Parallel()

	f := newFakeStripe(t)
	app := newApp(t, map[string]string{
		"APP_ENV": "production", "HSTS_FORCE": "true", "TRUSTED_PROXIES": "10.0.0.0/8", "APP_URL": "https://finances.example.test",
		"STRIPE_KEY": stripeKey, "STRIPE_WEBHOOK_SECRET": stripeSecret, "STRIPE_API_BASE": f.srv.URL,
		"STRIPE_PRICE_PRO": "price_pro", "STRIPE_CURRENCIES": "usd,brl",
	})

	t.Run("HSTS covers subdomains and lasts a year or more", func(t *testing.T) {
		hsts := app.Client(t).Get("/login").Header("Strict-Transport-Security")
		if !strings.Contains(hsts, "includeSubDomains") || !regexp.MustCompile(`max-age=(\d{8,})`).MatchString(hsts) {
			t.Fatalf("Strict-Transport-Security = %q", hsts)
		}
	})

	t.Run("the session cookie is host bound, secure, http only and lax", func(t *testing.T) {
		cookie := app.Client(t).WithHeader("Sec-Fetch-Site", "same-origin").PostForm("/locale", url.Values{"language": {"es"}}).Header("Set-Cookie")
		for _, want := range []string{"__Host-", "Secure", "HttpOnly", "SameSite=Lax", "Path=/"} {
			if !strings.Contains(cookie, want) {
				t.Errorf("cookie %q lacks %s", cookie, want)
			}
		}

		if strings.Contains(cookie, "Domain=") {
			t.Errorf("cookie %q names a Domain", cookie)
		}
	})
}

func TestSecurity_Web_Sessions(t *testing.T) {
	t.Parallel()

	app := authTestApp(t)
	register(t, app, app.Client(t).Inertia(""), "bob@example.com")

	cookieOf := func(res *vukemtest.Response) string {
		m := regexp.MustCompile(`vukem_session=([^;]+)`).FindStringSubmatch(res.Header("Set-Cookie"))
		if m == nil {
			return ""
		}

		return m[1]
	}

	withCookie := func(value string) *vukemtest.Client {
		return app.Client(t).Inertia("").WithHeader("Cookie", "vukem_session="+value)
	}

	t.Run("the cookie is http only and lax", func(t *testing.T) {
		res := app.Client(t).Inertia("").PostJSON("/login", map[string]any{"email": "bob@example.com", "password": testPassword})
		for _, want := range []string{"HttpOnly", "SameSite=Lax", "Path=/"} {
			if !strings.Contains(res.Header("Set-Cookie"), want) {
				t.Errorf("cookie %q lacks %s", res.Header("Set-Cookie"), want)
			}
		}
	})

	t.Run("signing in changes the session id (no fixation)", func(t *testing.T) {
		c := app.Client(t).Inertia("")

		pre := cookieOf(c.WithHeader("Sec-Fetch-Site", "same-origin").PostForm("/locale", url.Values{"language": {"es"}}))
		if pre == "" {
			t.Fatal("no session before the login")
		}

		post := cookieOf(withCookie(pre).WithHeader("Sec-Fetch-Site", "same-origin").
			PostJSON("/login", map[string]any{"email": "bob@example.com", "password": testPassword}))
		if post == pre {
			t.Fatal("the session id survived the login: a planted id would be signed in")
		}

		withCookie(pre).Get("/dashboard").AssertRedirect("/login")
	})

	t.Run("an id the server never issued is no session", func(t *testing.T) {
		for _, forged := range []string{"1", "admin", strings.Repeat("A", 128), "../../etc/passwd"} {
			withCookie(forged).Get("/dashboard").AssertRedirect("/login")
		}
	})

	t.Run("logging out ends the session of the browser that logged out", func(t *testing.T) {
		c := app.Client(t).Inertia("").WithHeader("Sec-Fetch-Site", "same-origin")
		c.PostJSON("/login", map[string]any{"email": "bob@example.com", "password": testPassword}).AssertRedirect("/dashboard")
		c.Get("/dashboard").AssertOK()
		c.PostJSON("/logout", nil).AssertRedirect("/")
		c.Get("/dashboard").AssertRedirect("/login")
	})

	t.Run("the cookie is not accepted by the API", func(t *testing.T) {
		login := app.Client(t).Inertia("").PostJSON("/login", map[string]any{"email": "bob@example.com", "password": testPassword})

		app.Client(t).WithHeader("Cookie", "vukem_session="+cookieOf(login)).Get("/api/v1/accounts").AssertUnauthorized()
	})
}

func TestSecurity_Web_Enumeration(t *testing.T) {
	t.Parallel()

	app := authTestApp(t)
	register(t, app, app.Client(t).Inertia(""), "known@example.com")

	t.Run("login answers the same for an unknown address and a wrong password", func(t *testing.T) {
		known := app.Client(t).Inertia("").WithHeader("Referer", "/login").
			PostJSON("/login", map[string]any{"email": "known@example.com", "password": "not the password"})
		unknown := app.Client(t).Inertia("").WithHeader("Referer", "/login").
			PostJSON("/login", map[string]any{"email": "unknown@example.com", "password": "not the password"})

		if known.Status() != unknown.Status() || known.Header("Location") != unknown.Header("Location") || known.Body() != unknown.Body() {
			t.Fatalf("login tells them apart: %d %q and %d %q", known.Status(), known.Header("Location"), unknown.Status(), unknown.Header("Location"))
		}
	})

	t.Run("the login error names no account", func(t *testing.T) {
		c := app.Client(t).Inertia("").WithHeader("Referer", "/login")
		c.PostJSON("/login", map[string]any{"email": "known@example.com", "password": "nope nope nope"})

		page := c.Get("/login").Body()
		if strings.Contains(strings.ToLower(page), "no account") || strings.Contains(strings.ToLower(page), "not registered") ||
			strings.Contains(page, "nope nope nope") {
			t.Fatalf("the login page tells too much: %.300s", page)
		}
	})

	t.Run("password reset answers the same for a known and an unknown address", func(t *testing.T) {
		known := app.Client(t).Inertia("").PostJSON("/forgot-password", map[string]any{"email": "known@example.com"})
		unknown := app.Client(t).Inertia("").PostJSON("/forgot-password", map[string]any{"email": "unknown@example.com"})

		if known.Status() != unknown.Status() || known.Header("Location") != unknown.Header("Location") || known.Body() != unknown.Body() {
			t.Fatalf("reset tells them apart: %d %q and %d %q", known.Status(), known.Header("Location"), unknown.Status(), unknown.Header("Location"))
		}

		if len(app.Mails().SentTo("unknown@example.com")) != 0 {
			t.Fatal("a mail went to an address with no account")
		}
	})

	t.Run("registration answers the same for a taken and a free address", func(t *testing.T) {
		taken := app.Client(t).Inertia("").PostJSON("/register", signUpForm("Mallory", "known@example.com", otherPassword))
		free := app.Client(t).Inertia("").PostJSON("/register", signUpForm("Mallory", "free@example.com", otherPassword))

		if taken.Status() != free.Status() || taken.Header("Location") != free.Header("Location") {
			t.Fatalf("registration tells them apart: %d %q and %d %q", taken.Status(), taken.Header("Location"), free.Status(), free.Header("Location"))
		}
	})
}

func TestSecurity_Web_OpenRedirect(t *testing.T) {
	t.Parallel()

	app := authTestApp(t)
	register(t, app, app.Client(t).Inertia(""), "bob@example.com")

	external := func(loc string) bool {
		u, err := url.Parse(loc)

		return err != nil || u.Host != "" || u.Scheme != "" || strings.HasPrefix(loc, "//") || strings.HasPrefix(loc, `/\`)
	}

	t.Run("Back never follows a foreign Referer", func(t *testing.T) {
		for _, ref := range []string{
			"https://evil.example/x", "//evil.example", `/\evil.example`, "http://localhost:80@evil.example",
			"javascript:alert(1)", "https://evil.example\r\nX-Injected: 1",
		} {
			res := app.Client(t).WithHeader("Referer", ref).WithHeader("Sec-Fetch-Site", "same-origin").
				PostForm("/locale", url.Values{"language": {"es"}})
			if loc := res.Header("Location"); external(loc) {
				t.Errorf("Referer %q redirected to %q", ref, loc)
			}

			if res.Header("X-Injected") != "" {
				t.Errorf("Referer %q injected a header", ref)
			}
		}
	})

	t.Run("a login ignores a redirect parameter", func(t *testing.T) {
		for _, param := range []string{"next", "redirect", "redirect_to", "return", "returnTo", "intended", "continue", "url"} {
			res := app.Client(t).Inertia("").PostJSON("/login?"+param+"=https://evil.example/", map[string]any{
				"email": "bob@example.com", "password": testPassword, param: "https://evil.example/",
			})
			if loc := res.Header("Location"); external(loc) {
				t.Errorf("%s redirected to %q", param, loc)
			}
		}
	})

	t.Run("the intended page is a local path", func(t *testing.T) {
		c := app.Client(t).Inertia("")
		c.Get("/dashboard").AssertRedirect("/login")

		res := c.PostJSON("/login", map[string]any{"email": "bob@example.com", "password": testPassword})
		if loc := res.Header("Location"); external(loc) {
			t.Fatalf("redirected to %q", loc)
		}
	})

	t.Run("a path that looks like a redirect is a page, not a hop", func(t *testing.T) {
		for _, p := range []string{"//evil.example/", "/\\evil.example", "/%2f%2fevil.example", "/login/..%2f..%2f"} {
			res := app.Client(t).Get(p)
			if loc := res.Header("Location"); loc != "" && external(loc) {
				t.Errorf("GET %s redirected to %q", p, loc)
			}
		}
	})

	t.Run("a safe local target is still followed", func(t *testing.T) {
		res := app.Client(t).WithHeader("Referer", "/ok").WithHeader("Sec-Fetch-Site", "same-origin").
			PostForm("/locale", url.Values{"language": {"es"}})
		if !slices.Contains([]string{"/ok", "/"}, res.Header("Location")) {
			t.Fatalf("Location = %q", res.Header("Location"))
		}
	})
}

func TestSecurity_Web_NoSecretInPages(t *testing.T) {
	t.Parallel()

	app := authTestApp(t)

	const password = "Tr0ub4dor-and-three-horses"

	c := app.Client(t).Inertia("").WithHeader("Referer", "/register")
	c.PostJSON("/register", map[string]any{"name": "Mallory", "email": "m@example.com", "password": password, "password_confirmation": password + "x"})

	for _, page := range []string{"/register", "/login", "/register/check-email"} {
		if body := c.Get(page).Body(); strings.Contains(body, password) {
			t.Errorf("%s shows the password the person typed", page)
		}
	}
}
