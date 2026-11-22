package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// aiMarker is the unique part of the made-up key of these tests: finding it
// anywhere means the key leaked. The "placeholder" word keeps the secret
// scanner quiet; none of these keys is real.
const (
	aiMarker = "ZQXMARKER0123456789"
	aiKey    = "sk-placeholder-" + aiMarker + "-HINT"
)

func aiKeyForm(provider string) url.Values {
	return url.Values{"provider": {provider}, "apiSecret": {aiKey}, "label": {"main"}}
}

// aiTables are every table the AI layer writes; the key must be in none of them.
var aiTables = []string{"ai_credentials", "workspace_ai_settings", "ai_usage", "audit_events"}

func dumpTables(t *testing.T, app *vukemtest.App, tables ...string) string {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var out strings.Builder

	for _, table := range tables {
		rows, err := db.QueryContext(t.Context(), "SELECT * FROM "+table)
		if err != nil {
			t.Fatal(err)
		}

		cols, _ := rows.Columns()

		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))

			for i := range vals {
				ptrs[i] = &vals[i]
			}

			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}

			fmt.Fprintln(&out, table, vals)
		}

		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}

		_ = rows.Close()
	}

	return out.String()
}

func aiCredentials(t *testing.T, app *vukemtest.App, w who) []models.AICredential {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	rows, err := workspaces.Scoped[models.AICredential](workspaces.WithTenant(t.Context(), w.Workspace.ID), db).
		OrderBy("id", orm.Asc).Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return rows
}

func aiSettings(t *testing.T, app *vukemtest.App, w who) models.WorkspaceAISettings {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	row, err := workspaces.Scoped[models.WorkspaceAISettings](workspaces.WithTenant(t.Context(), w.Workspace.ID), db).
		First(t.Context())
	if err != nil && err != orm.ErrNotFound {
		t.Fatal(err)
	}

	return row
}

func TestAIOwnerOnly(t *testing.T) {
	t.Parallel()

	tm := newTeam(t)
	tm.owner.Client.PostForm("/ai/key", aiKeyForm("anthropic")).AssertRedirect("/ai")

	cred := aiCredentials(t, tm.app, tm.owner)[0]
	audits := rowCount(t, tm.app, "audit_events")

	settingsForm := url.Values{"enabled": {"true"}, "consent": {"true"}, "categorize": {"true"}, "limit": {"500"}}

	for name, user := range map[string]who{"viewer": tm.viewer, "member": tm.member, "admin": tm.admin} {
		t.Run(name, func(t *testing.T) {
			user.Client.PostForm("/ai/key", aiKeyForm("openai")).AssertForbidden()
			putForm(user.Client, "/ai/settings", settingsForm).AssertForbidden()
			user.Client.Delete("/ai/key/" + idString(cred.ID)).AssertForbidden()
			user.Client.PostForm("/ai/check", nil).AssertForbidden()
		})
	}

	// None of the refused calls changed anything.
	creds := aiCredentials(t, tm.app, tm.owner)
	if len(creds) != 1 || creds[0].RevokedAt.Valid || creds[0].Provider != "anthropic" {
		t.Errorf("credentials changed: %+v", creds)
	}

	if s := aiSettings(t, tm.app, tm.owner); s.Enabled {
		t.Error("a non-owner turned AI on")
	}

	if got := rowCount(t, tm.app, "audit_events"); got != audits {
		t.Errorf("audit events %d -> %d: a refused call wrote one", audits, got)
	}

	// Everyone reads the status, but only the owner sees the saved key's details.
	vp := tm.viewer.Client.Inertia("").Get("/ai").AssertInertia("AI").PropEquals("canManage", false)
	if cr, _ := vp.Prop("credential"); cr != nil {
		t.Errorf("a viewer received the credential: %v", cr)
	}

	tm.owner.Client.Inertia("").Get("/ai").AssertInertia("AI").PropEquals("canManage", true)

	// The owner can do all of it.
	putForm(tm.owner.Client, "/ai/settings", settingsForm).AssertRedirect("/ai")
	tm.owner.Client.Delete("/ai/key/" + idString(cred.ID)).AssertRedirect("/ai")

	if s := aiSettings(t, tm.app, tm.owner); !s.Enabled || s.MonthlyCallLimit != 500 {
		t.Errorf("settings = %+v", s)
	}

	if aiCredentials(t, tm.app, tm.owner)[0].KeyCiphertext != "" {
		t.Error("a deleted key kept its ciphertext")
	}
}

func TestAIConsentRequired(t *testing.T) {
	t.Parallel()

	tm := newTeam(t)
	o := tm.owner.Client

	noConsent := url.Values{"enabled": {"true"}, "categorize": {"true"}, "limit": {"100"}}
	putForm(o, "/ai/settings", noConsent).AssertRedirect("/ai")

	if s := aiSettings(t, tm.app, tm.owner); s.Enabled || s.ID != 0 {
		t.Fatalf("AI was turned on without the consent checkbox: %+v", s)
	}

	vukemtest.AssertDatabaseCount(t, tm.app, "audit_events", 0)

	// A limit out of range is refused as well.
	putForm(o, "/ai/settings", url.Values{"consent": {"true"}, "limit": {"0"}}).AssertRedirect("/ai")

	if s := aiSettings(t, tm.app, tm.owner); s.ID != 0 {
		t.Fatalf("a limit of 0 was saved: %+v", s)
	}

	noConsent.Set("consent", "true")
	putForm(o, "/ai/settings", noConsent).AssertRedirect("/ai")

	s := aiSettings(t, tm.app, tm.owner)
	if !s.Enabled || s.EnabledBy != tm.owner.User.ID || !s.EnabledAt.Valid {
		t.Fatalf("settings = %+v", s)
	}

	vukemtest.AssertDatabaseHas(t, tm.app, "audit_events", map[string]any{"action": ai.AuditEnabled, "actor_user_id": tm.owner.User.ID})

	// Turning it off needs no consent and is audited.
	putForm(o, "/ai/settings", url.Values{"limit": {"100"}}).AssertRedirect("/ai")

	if aiSettings(t, tm.app, tm.owner).Enabled {
		t.Error("AI is still on")
	}

	vukemtest.AssertDatabaseHas(t, tm.app, "audit_events", map[string]any{"action": ai.AuditDisabled})
}

// aiFakeServer is a local OpenAI-compatible server. It records what it was asked.
type aiFakeServer struct {
	*httptest.Server
	hits    atomic.Int32
	status  atomic.Int32
	lastKey atomic.Value
}

func newAIFakeServer(t *testing.T, status int) *aiFakeServer {
	t.Helper()

	f := &aiFakeServer{}
	f.status.Store(int32(status))
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits.Add(1)
		f.lastKey.Store(r.Header.Get("Authorization"))

		body, _ := io.ReadAll(r.Body)
		if code := int(f.status.Load()); code != http.StatusOK {
			w.WriteHeader(code)
			// The upstream echoes the key and the prompt, as a bad provider might.
			_, _ = fmt.Fprintf(w, `{"error":{"type":"invalid_request_error","message":"bad key %s in %s"}}`, aiKey, body)

			return
		}

		_, _ = io.WriteString(w, `{"model":"local","choices":[{"message":{"content":"pong"}}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`)
	}))
	t.Cleanup(f.Close)

	return f
}

func TestSecurity_AIKeyNeverReturned(t *testing.T) {
	t.Parallel()

	// The personal edition with the private-host switch reaches the fake server.
	app := newPersonalApp(t, map[string]string{"AI_ALLOW_PRIVATE_HOSTS": "true"})
	ana := signUp(t, app, "Ana", "ana@example.com")
	fake := newAIFakeServer(t, http.StatusUnauthorized)

	leaks := func(where, text string) {
		t.Helper()

		for _, secret := range []string{aiKey, aiMarker} {
			if strings.Contains(text, secret) {
				t.Errorf("%s leaks the key: %.300s", where, text)
			}
		}
	}

	form := aiKeyForm("openai_compatible")
	form.Set("baseUrl", fake.URL+"/v1")

	res := ana.Client.PostForm("/ai/key", form)
	res.AssertRedirect("/ai")
	leaks("save response", res.Body()+fmt.Sprint(res.Recorder().Header())+fmt.Sprint(res.Session()))

	// Page props (Inertia JSON) and the HTML shell.
	page := ana.Client.Inertia("").Get("/ai")
	page.AssertInertia("AI").PropEquals("credential.keyHint", "HINT")
	leaks("inertia page", page.Body())
	leaks("html page", ana.Client.Get("/ai").Body())
	leaks("json page", ana.Client.GetJSON("/ai").Body())

	// The provider rejects the key and echoes it with the prompt: none of it passes on.
	res = ana.Client.PostForm("/ai/check", nil)
	res.AssertRedirect("/ai")
	leaks("check response", res.Body()+fmt.Sprint(res.Session()))
	leaks("check flash", fmt.Sprint(res.Session()))

	if fmt.Sprint(res.Session()) == "" || !strings.Contains(fmt.Sprint(res.Session()), "The provider rejected the key.") {
		t.Errorf("the owner got no generic message: %v", res.Session())
	}

	js := browser(t, app, ana.User, ana.Workspace.ID).AcceptJSON().PostJSON("/ai/check", map[string]any{})
	js.AssertOK()
	leaks("check json", js.Body())

	if !strings.Contains(js.Body(), `"ok":false`) || strings.Contains(js.Body(), "invalid_request_error") {
		t.Errorf("check json = %s", js.Body())
	}

	if got, _ := fake.lastKey.Load().(string); got != "Bearer "+aiKey {
		t.Error("the provider did not receive the key it should have")
	}

	// A failed form must not flash the key back as old input.
	bad := aiKeyForm("")
	bad.Set("provider", "")
	res = ana.Client.PostForm("/ai/key", bad)
	leaks("validation failure", res.Body()+fmt.Sprint(res.Session()))

	res = ana.Client.PostForm("/ai/key", aiKeyForm("not-a-provider"))
	leaks("unknown provider", res.Body()+fmt.Sprint(res.Session()))

	// Audit entries, the database and the logs.
	leaks("database", dumpTables(t, app, aiTables...))
	leaks("logs", app.Logs())

	if strings.Contains(dumpTables(t, app, "audit_events"), "sk-") {
		t.Error("an audit event carries key-like text")
	}

	for _, c := range aiCredentials(t, app, ana) {
		b, _ := json.Marshal(c)
		leaks("credential JSON", string(b))
	}
}

func TestAIBaseURLInSaaSIsRefused(t *testing.T) {
	t.Parallel()

	// The variable is ignored in saas, whatever it says.
	app := newSaaSApp(t, map[string]string{"AI_ALLOW_PRIVATE_HOSTS": "true"})
	ana := signUp(t, app, "Ana", "ana@example.com")
	fake := newAIFakeServer(t, http.StatusOK)

	for _, base := range []string{fake.URL + "/v1", "https://169.254.169.254/v1", "http://example.com/v1", "https://localhost/v1"} {
		form := aiKeyForm("openai_compatible")
		form.Set("baseUrl", base)

		res := ana.Client.PostForm("/ai/key", form)
		res.AssertRedirect("/ai")

		if got := aiCredentials(t, app, ana); len(got) != 0 {
			t.Fatalf("%s was accepted in saas: %+v", base, got)
		}

		if strings.Contains(fmt.Sprint(res.Session()), aiMarker) {
			t.Error("the refusal echoes the key")
		}
	}

	if fake.hits.Load() != 0 {
		t.Error("the server reached the fake local server")
	}
}

func TestAIExcludedFromExportAndAPI(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	ana.Client.PostForm("/ai/key", aiKeyForm("anthropic")).AssertRedirect("/ai")

	// No route of the export, the API or the sync names the AI tables or pages.
	// Epics 05, 12 and 14 add those routes; extend this list with the paths they use.
	for _, route := range app.Router.Routes() {
		for _, prefix := range []string{"/api", "/export", "/sync", "/v1"} {
			if strings.HasPrefix(route.Path, prefix) && strings.Contains(strings.ToLower(route.Path+route.Name), "ai") {
				t.Errorf("route %s %s exposes the AI layer", route.Method, route.Path)
			}
		}
	}

	// The model never serializes its ciphertext, so a generic JSON export of
	// the row cannot carry the key.
	cred := aiCredentials(t, app, ana)[0]
	if cred.KeyCiphertext == "" {
		t.Fatal("the credential has no ciphertext")
	}

	b, err := json.Marshal(cred)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(b), cred.KeyCiphertext) || strings.Contains(string(b), "key_ciphertext") {
		t.Errorf("the credential JSON carries the ciphertext: %s", b)
	}

	// The real exports (epic 12) carry neither the key nor anything of the AI tables.
	for _, path := range []string{"/workspace/export.json", "/transactions/export.csv"} {
		body := ana.Client.Get(path).AssertOK().Body()
		if strings.Contains(body, cred.KeyCiphertext) || strings.Contains(body, aiMarker) ||
			strings.Contains(strings.ToLower(body), "ai_credentials") || strings.Contains(strings.ToLower(body), "ai_usage") {
			t.Errorf("%s carries AI data", path)
		}
	}

	// The shared props sent with every page hold no AI data.
	page := ana.Client.Inertia("").Get("/dashboard")
	if strings.Contains(page.Body(), cred.KeyCiphertext) || strings.Contains(page.Body(), aiMarker) {
		t.Error("the dashboard props carry the key")
	}
}
