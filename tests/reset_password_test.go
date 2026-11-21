package tests

import (
	"bytes"
	"strings"
	"testing"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

func TestUserResetPasswordCommand(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:        map[string]string{"APP_EDITION": "saas"},
		AppOptions: []vukem.Option{vukem.WithStdout(&out)},
	})

	register(t, app, app.Client(t).Inertia(""), "ana@example.com")

	if err := app.Call(t.Context(), "user:reset-password", []string{"-email", "nobody@example.com"}); err == nil {
		t.Fatal("an unknown address must fail")
	}

	if err := app.Call(t.Context(), "user:reset-password", nil); err == nil {
		t.Fatal("a missing -email must fail")
	}

	out.Reset()

	if err := app.Call(t.Context(), "user:reset-password", []string{"-email", "Ana@Example.com"}); err != nil {
		t.Fatal(err)
	}

	_, rest, found := strings.Cut(out.String(), "new password for Ana@Example.com: ")
	if !found {
		t.Fatalf("no password in the output: %q", out.String())
	}

	password, _, _ := strings.Cut(rest, "\n")
	if len(password) < 12 {
		t.Fatalf("password %q is shorter than the 12 characters the forms require", password)
	}

	// The old password is dead and the new one signs in.
	c := app.Client(t).Inertia("")
	c.PostJSON("/login", map[string]any{"email": "ana@example.com", "password": testPassword})
	c.Get("/dashboard").AssertRedirect("/login")
	c.PostJSON("/login", map[string]any{"email": "ana@example.com", "password": password}).AssertRedirect("/dashboard")
}
