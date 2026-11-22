package cli_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestLocalMode(t *testing.T) {
	t.Parallel()

	t.Run("personal runs through Invoke as the owner", func(t *testing.T) {
		t.Parallel()

		app := newEdition(t, "personal")
		c, tm := newCLI(t, nil, localOf(app))

		if code := run(t, c, tm, "accounts", "list"); code != cli.ExitNotFound {
			t.Fatalf("with no owner: exit %d, want %d", code, cli.ExitNotFound)
		}

		signUp(t, app, "Ana", "ana@example.com")

		if code := run(t, c, tm, "accounts", "create", "--name", "Checking", "--kind", "checking", "--opening", "1.234,56"); code != 0 {
			t.Fatalf("create exit %d: %s", code, tm.err.String())
		}

		var created map[string]any
		if err := json.Unmarshal(tm.out.Bytes(), &created); err != nil {
			t.Fatalf("not JSON: %v: %s", err, tm.out.String())
		}

		if created["name"] != "Checking" || created["opening_cents"].(float64) != 123456 {
			t.Fatalf("created = %v", created)
		}

		if code := run(t, c, tm, "accounts", "show", created["uid"].(string)); code != 0 || !strings.Contains(tm.out.String(), "Checking") {
			t.Fatalf("show exit %d: %s", code, tm.out.String())
		}

		if code := run(t, c, tm, "accounts", "show", "01JD5Z8K3M4N5P6Q7R8S9T0V1W"); code != cli.ExitNotFound {
			t.Fatalf("a uid that is not here: exit %d, want %d", code, cli.ExitNotFound)
		}

		if code := run(t, c, tm, "accounts", "create", "--name", "x"); code != cli.ExitUsage {
			t.Fatalf("a create with no kind: exit %d, want %d (%s)", code, cli.ExitUsage, tm.err.String())
		}
	})

	t.Run("saas has no local owner and nothing changes", func(t *testing.T) {
		t.Parallel()

		app := newEdition(t, "saas")
		signUp(t, app, "Ana", "ana@example.com")

		c, tm := newCLI(t, nil, localOf(app))

		db, err := app.DB()
		if err != nil {
			t.Fatal(err)
		}

		before, err := orm.QueryOf[models.Account](db).Count(t.Context())
		if err != nil {
			t.Fatal(err)
		}

		if code := run(t, c, tm, "accounts", "create", "--name", "Checking", "--kind", "checking"); code != cli.ExitAuth {
			t.Fatalf("exit %d, want %d", code, cli.ExitAuth)
		}

		n, err := orm.QueryOf[models.Account](db).Count(t.Context())
		if err != nil || n != before {
			t.Fatalf("accounts = %d, %v, want %d: the refused call changed data", n, err, before)
		}
	})
}
