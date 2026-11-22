package cli_test

import (
	"bytes"
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

func newEdition(t *testing.T, edition string) *vukemtest.App {
	t.Helper()

	return vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"APP_EDITION": edition}})
}

type person struct {
	User      models.User
	Workspace models.Workspace
}

func signUp(t *testing.T, app *vukemtest.App, name, email string) person {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u := models.User{Name: name, Email: email, Password: unusable()}
	u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = time.Now().UTC(), true

	if err := orm.Create(t.Context(), db, &u); err != nil {
		t.Fatal(err)
	}

	ws, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	w, err := ws.Create(t.Context(), u, name+" household", "USD")
	if err != nil {
		t.Fatal(err)
	}

	return person{u, w}
}

// makeToken creates a token with every scope through the Tokens page and
// returns the plain secret the page shows once.
func makeToken(t *testing.T, app *vukemtest.App, p person) string {
	t.Helper()

	var scopes []string
	for _, s := range operations.Scopes() {
		scopes = append(scopes, s.Name)
	}

	c := app.Client(t).WithHeader("Sec-Fetch-Site", "same-origin").
		ActingAs("web", p.User).WithSession(map[string]any{workspaces.SessionKey: strconv.FormatInt(p.Workspace.ID, 10)})

	res := c.Inertia("").PostForm("/tokens", url.Values{"name": {"cli"}, "scopes": scopes, "days": {"90"}}).AssertOK()

	secret, ok := res.JSONPath("props.created.secret")
	if !ok {
		t.Fatalf("the Tokens page shows no secret: %s", res.Body())
	}

	return secret.(string)
}

type term struct {
	out, err bytes.Buffer
	in       *strings.Reader
}

// newCLI builds a CLI over a fake terminal.
func newCLI(t *testing.T, env map[string]string, local func(context.Context) (cli.Backend, error)) (*cli.CLI, *term) {
	t.Helper()

	tree, err := cli.DefaultTree()
	if err != nil {
		t.Fatal(err)
	}

	tm := &term{in: strings.NewReader("")}

	return &cli.CLI{Tree: tree, Env: cli.Env{
		Stdin: tm.in, Stdout: &tm.out, Stderr: &tm.err, OpenLocal: local,
		Getenv: func(k string) (string, bool) { v, ok := env[k]; return v, ok },
	}}, tm
}

func localOf(app *vukemtest.App) func(context.Context) (cli.Backend, error) {
	return func(ctx context.Context) (cli.Backend, error) {
		l, err := cli.NewLocal(ctx, app.App)
		if err != nil {
			return nil, err
		}

		return l, nil
	}
}

// run runs a command and returns its exit code; the output stays in tm.
func run(t *testing.T, c *cli.CLI, tm *term, args ...string) int {
	t.Helper()

	tm.out.Reset()
	tm.err.Reset()

	err := c.Run(t.Context(), args[0], args[1:])
	if err != nil {
		tm.err.WriteString(err.Error() + "\n")
	}

	return cli.ExitOf(err)
}

// unusable is a password hash no one can log in with.
func unusable() string { return strings.Repeat("!", 8) }
