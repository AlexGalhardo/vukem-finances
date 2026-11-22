package tests

import (
	"context"
	"errors"
	"maps"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// The offline guard (decision D-027). Go has no hook on every outbound
// connection, so it combines the two seams that exist:
//
//   - http.DefaultTransport: every http.Client the app builds without its own
//     transport (the Telegram and AbacatePay clients, the CLI, webhooks) ends
//     there, and the guard sees the request URL.
//   - net.DefaultResolver: every connection to a host name, from any transport
//     (the AI client, SMTP, OAuth), resolves the name first, and the guard's
//     resolver refuses it.
//
// It does not see a dial to an IP literal made by a custom transport or by
// net.Dialer directly, nor anything outside Go (the browser, a wget in a
// health check). The browser spec (e2e/offline.spec.ts) and the container run
// on a network with no route out (docs/offline.md) cover those.
type offlineGuard struct {
	mu         sync.Mutex
	violations []string
}

func (g *offlineGuard) note(what string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.violations = append(g.violations, what)
}

func (g *offlineGuard) seen() []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	return append([]string(nil), g.violations...)
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(strings.Trim(host, "[]"))

	return ip != nil && ip.IsLoopback()
}

// RoundTrip implements http.RoundTripper.
func (g *offlineGuard) RoundTrip(req *http.Request) (*http.Response, error) {
	if !isLoopbackHost(req.URL.Hostname()) {
		g.note("http " + req.Method + " " + req.URL.Host)

		return nil, errors.New("offline guard: outbound HTTP request refused")
	}

	return nil, errors.New("offline guard: loopback requests are not expected from the app")
}

// resolver refuses every DNS lookup: a lookup of a host name is the first step
// of every connection to one. localhost never reaches it (the hosts file).
func (g *offlineGuard) resolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(_ context.Context, _, address string) (net.Conn, error) {
			g.note("dns via " + address)

			return nil, errors.New("offline guard: DNS lookup refused")
		},
	}
}

// install puts the guard on the process-wide seams. Tests that call it must
// not run in parallel: the globals are shared (parallel tests only resume
// after every serial test of the package is done, so a serial one is alone).
func (g *offlineGuard) install(t *testing.T) {
	t.Helper()

	transport, resolver := http.DefaultTransport, net.DefaultResolver
	http.DefaultTransport, net.DefaultResolver = g, g.resolver()

	t.Cleanup(func() { http.DefaultTransport, net.DefaultResolver = transport, resolver })
}

// TestOfflineGuardSeesOutboundConnections proves the guard is not blind: the two
// paths it watches each record an attempt to leave the machine.
func TestOfflineGuardSeesOutboundConnections(t *testing.T) {
	t.Parallel()

	g := &offlineGuard{}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.telegram.org/bot/getMe", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := (&http.Client{Transport: g}).Do(req); err == nil { //nolint:bodyclose // refused, no response
		t.Error("the guard let an HTTP request to a public host through")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	if _, err := (&net.Dialer{Resolver: g.resolver()}).DialContext(ctx, "tcp", "api.openai.com:443"); err == nil {
		t.Error("the guard let a connection to a public host name through")
	}

	if got := g.seen(); len(got) < 2 {
		t.Errorf("violations = %v, want the HTTP request and the DNS lookup", got)
	}
}

// TestPersonalMakesNoOutboundConnection boots the personal edition, migrates,
// runs every scheduled job once and walks a core journey while the guard is on.
// No integration is configured, so nothing may leave the machine.
func TestPersonalMakesNoOutboundConnection(t *testing.T) {
	g := &offlineGuard{}
	g.install(t)

	app := newPersonalApp(t, map[string]string{"SETUP_TOKEN": setupToken})
	c := app.Client(t).Inertia("").WithHeader("Sec-Fetch-Site", "same-origin")

	// Boot and migrate happened in newPersonalApp; the first screens come next.
	c.Get("/setup").AssertOK().AssertInertia("Setup")
	c.PostJSON("/setup", setupForm("owner@example.com", setupToken)).AssertRedirect("/login")
	c.Get("/login").AssertOK().AssertInertia("Auth/Login")
	c.PostJSON("/login", map[string]any{"email": "owner@example.com", "password": testPassword}).
		AssertRedirect("/dashboard")

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	ws, err := orm.QueryOf[models.Workspace](db).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	owner := who{Workspace: ws, Client: c}

	c.PostForm("/accounts", url.Values{"name": {"Savings"}, "kind": {"savings"}}).AssertRedirect("/accounts")
	addTransaction(t, owner, firstAccount(t, app, owner), categoryID(t, app, owner, "Groceries"),
		"expense", "12.50", today()).AssertRedirect("/transactions")

	for _, page := range []string{
		"/dashboard", "/accounts", "/transactions", "/categories", "/budgets", "/goals", "/reports",
		"/recurrences", "/cards", "/import", "/notifications", "/workspace", "/ai", "/tokens", "/settings/telegram",
	} {
		if got := c.Get(page).Status(); got >= http.StatusInternalServerError {
			t.Errorf("GET %s = %d, want no server error", page, got)
		}
	}

	c.Get("/transactions/export.csv").AssertOK().AssertSee("12.50")
	c.Get("/workspace/export.json").AssertOK()

	runEveryScheduledJob(t, app)

	if got := g.seen(); len(got) > 0 {
		t.Fatalf("the personal edition tried to leave the machine: %v", got)
	}
}

// runEveryScheduledJob runs each task of the real schedule once, the way
// schedule:run does when its time comes.
func runEveryScheduledJob(t *testing.T, app *vukemtest.App) {
	t.Helper()

	s, err := app.Schedule()
	if err != nil {
		t.Fatal(err)
	}

	tasks := s.Tasks()
	if len(tasks) == 0 {
		t.Fatal("the schedule has no task")
	}

	ran := map[string]bool{}
	anchor := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)

	for _, task := range tasks {
		due := s.Next(task, anchor)
		for _, d := range s.Due(due) {
			ran[d.Name()] = true
		}

		if err := s.RunDue(t.Context(), due); err != nil {
			t.Errorf("scheduled run at %s: %v", due.Format(time.RFC3339), err)
		}
	}

	for _, task := range tasks {
		if !ran[task.Name()] {
			t.Errorf("task %s never ran", task.Name())
		}
	}
}

// TestPersonalDropsAnalytics: a tracker set in the environment is a script from
// another host, so personal renders none and allows none in the CSP, while saas
// keeps what it always had.
func TestPersonalDropsAnalytics(t *testing.T) {
	t.Parallel()

	env := map[string]string{"UMAMI_WEBSITE_ID": "123e4567-e89b-12d3-a456-426614174000"}

	personal := newPersonalApp(t, env).Client(t).Get("/setup")
	personal.AssertOK().AssertDontSee("umami")

	if csp := personal.Header("Content-Security-Policy"); strings.Contains(csp, "umami") {
		t.Errorf("personal CSP allows the tracker: %s", csp)
	}

	newSaaSApp(t, env).Client(t).Get("/login").AssertOK().AssertSee("umami")
}

// unreachableMail points the mailer at a loopback port nobody listens on: the
// connection is refused at once, and it is not "outbound".
func unreachableMail(extra map[string]string) map[string]string {
	env := map[string]string{
		"SETUP_TOKEN": setupToken,
		"MAIL_MAILER": "smtp", "SMTP_HOST": "127.0.0.1", "SMTP_PORT": "1", "SMTP_INSECURE": "true",
	}
	maps.Copy(env, extra)

	return env
}

// TestPersonalMailNeverBlocksAFlow: a mail server that cannot be reached must not
// fail the action that sent the mail, and the answer must not tell an address
// that exists from one that does not. The reset link is sent inline
// (QUEUE_CONNECTION is sync by default), so a refused connection would reach the
// person, and only for the address that exists.
func TestPersonalMailNeverBlocksAFlow(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t, unreachableMail(nil))
	c := app.Client(t).Inertia("").WithHeader("Sec-Fetch-Site", "same-origin")

	c.PostJSON("/setup", setupForm("owner@example.com", setupToken)).AssertRedirect("/login")

	for _, email := range []string{"owner@example.com", "nobody@example.com"} {
		c.PostJSON("/forgot-password", map[string]any{"email": email}).AssertRedirect("/forgot-password")
	}
}

// TestPersonalMailOnARealQueueIsRetried: with a database queue the flow answers
// as before and the mail waits in the queue, to be retried when a server answers.
func TestPersonalMailOnARealQueueIsRetried(t *testing.T) {
	t.Parallel()

	app := newPersonalApp(t, unreachableMail(map[string]string{"QUEUE_CONNECTION": "database", "QUEUE_IN_SERVE": "false"}))
	c := app.Client(t).Inertia("").WithHeader("Sec-Fetch-Site", "same-origin")

	c.PostJSON("/setup", setupForm("owner@example.com", setupToken)).AssertRedirect("/login")

	for _, email := range []string{"owner@example.com", "nobody@example.com"} {
		c.PostJSON("/forgot-password", map[string]any{"email": email}).AssertRedirect("/forgot-password")
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var queued int
	if err := db.SQL().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM jobs").Scan(&queued); err != nil {
		t.Fatal(err)
	}

	if queued != 1 {
		t.Errorf("queued jobs = %d, want the one reset mail of the address that exists", queued)
	}
}
