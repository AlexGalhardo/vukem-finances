package cli_test

import (
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestDestructive(t *testing.T) {
	t.Parallel()

	app := newEdition(t, "personal")
	signUp(t, app, "Ana", "ana@example.com")

	c, tm := newCLI(t, nil, localOf(app))

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	archived := func() int {
		t.Helper()

		n, err := orm.QueryOf[models.Account](db).WhereNotNull("archived_at").Count(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		return int(n)
	}

	create := func(name string) string {
		t.Helper()

		if code := run(t, c, tm, "accounts", "create", "--name", name, "--kind", "checking"); code != 0 {
			t.Fatalf("create exit %d: %s", code, tm.err.String())
		}

		return jsonField(t, tm.out.Bytes(), "uid")
	}

	uid := create("Old")

	t.Run("no terminal and no --yes exits 2 and changes nothing", func(t *testing.T) {
		if code := run(t, c, tm, "accounts", "archive", uid); code != cli.ExitUsage {
			t.Fatalf("exit %d, want %d", code, cli.ExitUsage)
		}

		if !strings.Contains(tm.err.String(), "--yes") || archived() != 0 {
			t.Fatalf("stderr %q, archived %d", tm.err.String(), archived())
		}
	})

	t.Run("a terminal asks and a no changes nothing", func(t *testing.T) {
		c.Env.StdinTTY, c.Env.Stdin = true, strings.NewReader("n\n")

		t.Cleanup(func() { c.Env.StdinTTY = false })

		if code := run(t, c, tm, "accounts", "archive", uid); code != cli.ExitFailure {
			t.Fatalf("exit %d, want %d", code, cli.ExitFailure)
		}

		if archived() != 0 {
			t.Fatal("a declined command changed data")
		}
	})

	t.Run("an empty answer is a no", func(t *testing.T) {
		c.Env.StdinTTY, c.Env.Stdin = true, strings.NewReader("")

		t.Cleanup(func() { c.Env.StdinTTY = false })

		if code := run(t, c, tm, "accounts", "archive", uid); code != cli.ExitFailure || archived() != 0 {
			t.Fatalf("exit %d, archived %d", code, archived())
		}
	})

	t.Run("a terminal and y runs it", func(t *testing.T) {
		c.Env.StdinTTY, c.Env.Stdin = true, strings.NewReader("y\n")

		t.Cleanup(func() { c.Env.StdinTTY = false })

		if code := run(t, c, tm, "accounts", "archive", uid); code != 0 || archived() != 1 {
			t.Fatalf("exit %d, archived %d: %s", code, archived(), tm.err.String())
		}
	})

	t.Run("--yes runs it with no terminal", func(t *testing.T) {
		other := create("Other")

		if code := run(t, c, tm, "accounts", "archive", other, "--yes"); code != 0 || archived() != 2 {
			t.Fatalf("exit %d, archived %d: %s", code, archived(), tm.err.String())
		}
	})

	t.Run("a command that is not destructive never asks", func(t *testing.T) {
		if code := run(t, c, tm, "accounts", "list"); code != 0 {
			t.Fatalf("exit %d", code)
		}
	})
}

func TestExitCodes(t *testing.T) {
	t.Parallel()

	for code, want := range map[string]int{
		"forbidden": cli.ExitAuth, "missing_scope": cli.ExitAuth, "not_found": cli.ExitNotFound,
		"conflict": cli.ExitConflict, "version_conflict": cli.ExitConflict,
		"rate_limited": cli.ExitLimit, "plan_limit": cli.ExitLimit, "validation_failed": cli.ExitUsage, "other": cli.ExitFailure,
	} {
		if got := cli.ExitForCode(code); got != want {
			t.Errorf("ExitForCode(%q) = %d, want %d", code, got, want)
		}
	}

	if cli.ExitOf(nil) != 0 {
		t.Error("nil is not 0")
	}

	app := newEdition(t, "personal")
	signUp(t, app, "Ana", "ana@example.com")

	c, tm := newCLI(t, nil, localOf(app))

	for name, tc := range map[string]struct {
		args []string
		want int
	}{
		"unknown verb":  {[]string{"accounts", "nope"}, cli.ExitUsage},
		"no verb":       {[]string{"accounts"}, cli.ExitUsage},
		"bad flag":      {[]string{"accounts", "list", "--nope"}, cli.ExitUsage},
		"missing field": {[]string{"accounts", "create", "--name", "x"}, cli.ExitUsage},
		"not found":     {[]string{"accounts", "show", "01JD5Z8K3M4N5P6Q7R8S9T0V1W"}, cli.ExitNotFound},
		"ok":            {[]string{"accounts", "list"}, 0},
	} {
		if got := run(t, c, tm, tc.args...); got != tc.want {
			t.Errorf("%s: exit %d, want %d (%s)", name, got, tc.want, tm.err.String())
		}
	}
}
