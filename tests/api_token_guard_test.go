package tests

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/auth"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
)

func TestAPITokenGuard(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")

	acc, err := accounts.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	if acc.API == nil || acc.API.Name() != "api" {
		t.Fatal("the app has no api token guard")
	}

	plain, tok, err := acc.Tokens.CreateWith(t.Context(), strconv.FormatInt(ana.User.ID, 10), "ci",
		[]string{"accounts:read"}, 24*time.Hour, authCreateOptions(ana.Workspace.ID))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(plain, "vf_"+tok.ID+".") {
		t.Fatalf("token %q does not look like vf_<id>.<secret>", plain)
	}

	secret := plain[strings.Index(plain, ".")+1:]

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var hash, resource string
	if err := db.QueryRowContext(t.Context(), "SELECT hash, resource FROM access_tokens WHERE id = ?", tok.ID).
		Scan(&hash, &resource); err != nil {
		t.Fatal(err)
	}

	if hash == secret || strings.Contains(hash, secret) || len(hash) != 64 {
		t.Errorf("the stored hash %q is not a SHA-256 of the secret", hash)
	}

	if resource != strconv.FormatInt(ana.Workspace.ID, 10) {
		t.Errorf("resource = %q, want the workspace id", resource)
	}

	var rows int64
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM access_tokens WHERE hash LIKE ? OR name LIKE ? OR abilities LIKE ?",
		"%"+secret+"%", "%"+secret+"%", "%"+secret+"%").Scan(&rows); err != nil {
		t.Fatal(err)
	}

	if rows != 0 {
		t.Error("the secret is stored in clear")
	}

	got, err := acc.Tokens.Find(t.Context(), plain)
	if err != nil || got.Resource != resource || !got.Can("accounts:read") || got.Can("*") {
		t.Errorf("Find = %+v, %v", got, err)
	}
}

func authCreateOptions(workspace int64) auth.CreateOptions {
	return auth.CreateOptions{Resource: strconv.FormatInt(workspace, 10)}
}
