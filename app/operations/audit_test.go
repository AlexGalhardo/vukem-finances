package operations_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/config"
)

type moneyIn struct {
	Description string `json:"description" validate:"max=40"`
	AmountCents int64  `json:"amount_cents"`
}

type moneyOut struct {
	Description string `json:"description"`
}

func TestAudit(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	log := slog.New(slog.NewTextHandler(&buf, nil))
	reg := operations.NewRegistry(operations.Config{Edition: config.EditionSaaS, Log: log})
	run := func(_ context.Context, _ operations.Actor, in moneyIn) (moneyOut, error) {
		return moneyOut{Description: in.Description}, nil
	}

	write := operations.Define(reg, operations.Operation[moneyIn, moneyOut]{
		Name: "money.create", Scope: operations.ScopeTransactionsWrite, Kind: operations.KindWrite, Run: run,
	})
	destroy := operations.Define(reg, operations.Operation[moneyIn, moneyOut]{
		Name: "money.delete", Scope: operations.ScopeTransactionsWrite, Kind: operations.KindDestructive, Run: run,
	})
	read := operations.Define(reg, operations.Operation[moneyIn, moneyOut]{
		Name: "money.list", Scope: operations.ScopeTransactionsRead, Kind: operations.KindRead, Run: run,
	})

	owner := actor(models.RoleOwner, operations.ScopeTransactionsWrite, operations.ScopeTransactionsRead)
	in := moneyIn{Description: "secret lunch with Ana", AmountCents: 987654}

	for _, call := range []func() error{
		func() error { _, err := operations.Invoke(t.Context(), write, owner, in); return err },
		func() error { _, err := operations.Invoke(t.Context(), destroy, owner, in); return err },
		func() error { _, err := operations.Invoke(t.Context(), read, owner, in); return err },
		func() error {
			_, err := operations.Invoke(t.Context(), write, actor(models.RoleViewer, operations.ScopeTransactionsWrite), in)
			return err
		},
	} {
		_ = call()
	}

	out := buf.String()

	for _, want := range []string{"operation=money.create", "operation=money.delete", "kind=destructive", "outcome=ok", "outcome=refused", "code=forbidden", "workspace=3", "user=7", "surface=rest"} {
		if !strings.Contains(out, want) {
			t.Errorf("the audit log lacks %q:\n%s", want, out)
		}
	}

	if strings.Contains(out, "money.list") {
		t.Errorf("a read was audited:\n%s", out)
	}

	for _, leak := range []string{"secret", "lunch", "Ana", "987654", "9876"} {
		if strings.Contains(out, leak) {
			t.Errorf("the audit log holds %q:\n%s", leak, out)
		}
	}

	if n := strings.Count(out, "msg=operation"); n != 3 {
		t.Errorf("%d entries, want 3 (write, destructive, refused write):\n%s", n, out)
	}
}
