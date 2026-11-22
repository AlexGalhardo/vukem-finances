// Package oauthtest is the test harness of the remote MCP endpoint: a saas app
// with an https public URL, people in separate workspaces, and a function that
// runs the whole authorization code flow over HTTP and returns the tokens. It is
// imported by tests only.
package oauthtest

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// The identity of the stack: the public URL of the app, the resource tokens are
// for, and the one registered client.
const (
	Issuer      = "https://finances.example.test"
	Resource    = Issuer + "/mcp"
	ClientID    = "test-client"
	RedirectURI = "https://client.example.test/callback"
)

// Person is a signed-up user with the workspace they own.
type Person struct {
	User      models.User
	Workspace models.Workspace
}

// Stack is the app and the people in it.
type Stack struct {
	App *vukemtest.App
	WS  *workspaces.Service
}

// Tokens are what the token endpoint answers with.
type Tokens struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
	Scope   string `json:"scope"`
	Type    string `json:"token_type"`
	Expires int64  `json:"expires_in"`
}

// New builds the saas app with remote MCP on. env adds or overrides settings.
func New(t *testing.T, env map[string]string) *Stack {
	t.Helper()

	e := map[string]string{
		"APP_EDITION": "saas", "APP_URL": Issuer, "MCP_TOOLSETS": "all", "MCP_ALLOW_DESTRUCTIVE": "true",
		"MCP_OAUTH_CLIENTS": `[{"id":"` + ClientID + `","name":"Test client","redirect_uris":["` + RedirectURI + `"]}]`,
	}
	for k, v := range env {
		e[k] = v
	}

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: e})

	ws, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	return &Stack{App: app, WS: ws}
}

// SignUp makes a verified user with a workspace of their own.
func (s *Stack) SignUp(t *testing.T, name, email string) Person {
	t.Helper()

	db, err := s.App.DB()
	if err != nil {
		t.Fatal(err)
	}

	u := models.User{Name: name, Email: email, Password: strings.Repeat("!", 8)}
	u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = time.Now().UTC(), true

	if err := orm.Create(t.Context(), db, &u); err != nil {
		t.Fatal(err)
	}

	w, err := s.WS.Create(t.Context(), u, name+" household", "USD")
	if err != nil {
		t.Fatal(err)
	}

	return Person{User: u, Workspace: w}
}

// AddMember puts a person in another's workspace with a role.
func (s *Stack) AddMember(t *testing.T, workspaceID int64, p Person, role string) {
	t.Helper()

	db, err := s.App.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.SQL().ExecContext(t.Context(),
		"INSERT INTO memberships (workspace_id, user_id, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		workspaceID, p.User.ID, role, time.Now().UTC(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
}

// Browser is a signed-in browser of p, working in workspace id.
func (s *Stack) Browser(t *testing.T, p Person, workspaceID int64) *vukemtest.Client {
	t.Helper()

	return s.App.Client(t).WithHeader("Sec-Fetch-Site", "same-origin").ActingAs("web", p.User).
		WithSession(map[string]any{workspaces.SessionKey: strconv.FormatInt(workspaceID, 10)})
}

// Verifier makes a PKCE code verifier and its S256 challenge.
func Verifier() (verifier, challenge string) {
	verifier = rand.Text() + rand.Text()
	sum := sha256.Sum256([]byte(verifier))

	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

// AuthorizeQuery is the query of an authorization request of the test client.
func AuthorizeQuery(challenge string, scopes ...string) url.Values {
	return url.Values{
		"response_type": {"code"}, "client_id": {ClientID}, "redirect_uri": {RedirectURI}, "state": {"state-1"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"}, "scope": {strings.Join(scopes, " ")},
		"resource": {Resource},
	}
}

// Consent asks for consent as p working in workspace id and returns the form
// the screen shows: the action and the hidden fields.
func (s *Stack) Consent(t *testing.T, p Person, workspaceID int64, q url.Values) (action string, form url.Values) {
	t.Helper()

	res := s.Browser(t, p, workspaceID).Inertia("").Get("/authorize?" + q.Encode()).AssertOK()

	var props struct {
		Props struct {
			Action string `json:"action"`
			Fields []struct{ Name, Value string }
		} `json:"props"`
	}

	if err := json.Unmarshal([]byte(res.Body()), &props); err != nil {
		t.Fatalf("the consent page is not an Inertia response: %v: %.300s", err, res.Body())
	}

	form = url.Values{}
	for _, f := range props.Props.Fields {
		form.Add(f.Name, f.Value)
	}

	return props.Props.Action, form
}

// Approve runs the consent as p and returns the code the client is sent back
// with.
func (s *Stack) Approve(t *testing.T, p Person, workspaceID int64, q url.Values) (code string) {
	t.Helper()

	action, form := s.Consent(t, p, workspaceID, q)
	form.Set("decision", "approve")

	res := s.Browser(t, p, workspaceID).PostForm(action, form)

	loc, err := url.Parse(res.Header("Location"))
	if err != nil || loc.Query().Get("code") == "" {
		t.Fatalf("approval answered %d with Location %q: %.200s", res.Status(), res.Header("Location"), res.Body())
	}

	if loc.Query().Get("iss") != Issuer || loc.Query().Get("state") != "state-1" {
		t.Fatalf("the redirect lacks iss or state: %s", loc)
	}

	return loc.Query().Get("code")
}

// Token posts a request to the token endpoint as the client.
func (s *Stack) Token(t *testing.T, form url.Values) *vukemtest.Response {
	t.Helper()

	return s.App.Client(t).PostForm("/token", form)
}

// Authorize runs the whole flow and returns the tokens.
func (s *Stack) Authorize(t *testing.T, p Person, workspaceID int64, scopes ...string) Tokens {
	t.Helper()

	verifier, challenge := Verifier()
	code := s.Approve(t, p, workspaceID, AuthorizeQuery(challenge, scopes...))

	res := s.Token(t, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {RedirectURI},
		"client_id": {ClientID}, "code_verifier": {verifier}, "resource": {Resource},
	}).AssertOK()

	var tok Tokens
	if err := json.Unmarshal([]byte(res.Body()), &tok); err != nil || tok.Access == "" || tok.Refresh == "" {
		t.Fatalf("token response: %v %.200s", err, res.Body())
	}

	return tok
}

// Call posts one JSON-RPC request to /mcp with the access token.
func (s *Stack) Call(t *testing.T, access, method string, params map[string]any) *vukemtest.Response {
	t.Helper()

	c := s.App.Client(t).WithHeader("Accept", "application/json").
		WithHeader("MCP-Protocol-Version", "2026-07-28").WithHeader("Mcp-Method", method)

	if access != "" {
		c = c.WithToken(access)
	}

	for _, key := range []string{"name", "uri"} {
		if name, _ := params[key].(string); name != "" {
			c = c.WithHeader("Mcp-Name", name)
		}
	}

	p := map[string]any{"_meta": map[string]any{
		"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
	}}
	for k, v := range params {
		p[k] = v
	}

	return c.PostJSON("/mcp", map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": p})
}
