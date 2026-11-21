package tests

import (
	"errors"
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func httpStatus(err error) int {
	if e, ok := errors.AsType[*errs.HTTPError](err); ok {
		return e.Status
	}

	return 0
}

func TestAccountKinds(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

	for _, kind := range models.AccountKinds() {
		a, err := fin.CreateAccount(ctx, "A "+kind, kind, "10.00")
		if err != nil || a.Kind != kind || a.OpeningCents != 1000 {
			t.Errorf("CreateAccount(%s) = %+v, %v; want the kind kept and 1000 cents", kind, a, err)
		}
	}

	for _, kind := range []string{"credit_card", "bank", "", "Checking"} {
		if _, err := fin.CreateAccount(ctx, "Bad", kind, ""); httpStatus(err) != http.StatusUnprocessableEntity {
			t.Errorf("CreateAccount(%q) = %v, want 422", kind, err)
		}
	}

	acc := firstAccount(t, app, ana)

	if _, err := fin.UpdateAccount(ctx, acc, "Renamed", "credit_card", "0"); httpStatus(err) != http.StatusUnprocessableEntity {
		t.Errorf("UpdateAccount to credit_card = %v, want 422", err)
	}

	a, err := fin.UpdateAccount(ctx, acc, "  Renamed ", "wallet", "-5.50")
	if err != nil || a.Name != "Renamed" || a.Kind != "wallet" || a.OpeningCents != -550 || a.Version < 2 {
		t.Errorf("UpdateAccount = %+v, %v; want the new name, kind and opening, version bumped", a, err)
	}

	if _, err := fin.UpdateAccount(ctx, 999999, "X", "cash", ""); httpStatus(err) != http.StatusNotFound {
		t.Errorf("UpdateAccount of an unknown id = %v, want 404", err)
	}

	if _, err := fin.UpdateAccount(ctx, acc, "X", "cash", "9007199254740992"); err == nil {
		t.Error("an opening balance beyond 2^53-1 cents was accepted")
	}

	// Another workspace's account is a 404 for Bob.
	bob := signUp(t, app, "Bob", "bob@example.com")
	if _, err := fin.UpdateAccount(workspaces.WithTenant(t.Context(), bob.Workspace.ID), acc, "Mine", "cash", ""); httpStatus(err) != http.StatusNotFound {
		t.Errorf("UpdateAccount across workspaces = %v, want 404", err)
	}
}

func TestArchiveAccount(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	acc := firstAccount(t, app, ana)

	addTransaction(t, ana, acc, 0, "expense", "10.00", today()).AssertRedirect("/transactions")

	a, err := fin.ArchiveAccount(ctx, acc)
	if err != nil || !a.ArchivedAt.Valid || a.Version < 2 {
		t.Fatalf("ArchiveAccount = %+v, %v; want archived_at set and version bumped", a, err)
	}

	again, err := fin.ArchiveAccount(ctx, acc)
	if err != nil || again.Version != a.Version || again.ArchivedAt != a.ArchivedAt {
		t.Errorf("archiving twice = %+v, %v; want no change", again, err)
	}

	accounts, err := fin.Accounts(ctx)
	if err != nil || len(accounts) != 1 || !accounts[0].ArchivedAt.Valid {
		t.Errorf("Accounts = %+v, %v; want the archived account still listed with its flag", accounts, err)
	}

	rows, err := fin.Transactions(ctx, finance.TxFilter{AccountID: acc})
	if err != nil || len(rows) != 1 || rows[0].AccountName != a.Name {
		t.Errorf("history of the archived account = %+v, %v; want its transaction with the account name", rows, err)
	}

	u, err := fin.UnarchiveAccount(ctx, acc)
	if err != nil || u.ArchivedAt.Valid {
		t.Errorf("UnarchiveAccount = %+v, %v; want archived_at cleared", u, err)
	}

	for name, call := range map[string]func() error{
		"archive":   func() error { _, err := fin.ArchiveAccount(ctx, 999999); return err },
		"unarchive": func() error { _, err := fin.UnarchiveAccount(ctx, 999999); return err },
	} {
		if err := call(); err == nil {
			t.Errorf("%s of an unknown id succeeded", name)
		}
	}

	bob := signUp(t, app, "Bob", "bob@example.com")
	if _, err := fin.ArchiveAccount(workspaces.WithTenant(t.Context(), bob.Workspace.ID), acc); err == nil {
		t.Error("Bob archived Ana's account")
	}

	if got := firstOf[models.Account](t, app, ana, ""); got.ArchivedAt.Valid {
		t.Error("Ana's account was archived by Bob")
	}
}

// TestArchivedDoesNotCountForQuota proves an archived account frees its place
// on the Free plan, and that bringing it back is checked against the plan
// again, so archiving cannot be used to hold more accounts than allowed.
func TestArchivedDoesNotCountForQuota(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	first := firstAccount(t, app, ana)

	if _, err := fin.CreateAccount(ctx, "Second", "savings", ""); !workspaces.IsUpgrade(err) {
		t.Fatalf("a second account on Free = %v, want the upgrade answer", err)
	}

	if _, err := fin.ArchiveAccount(ctx, first); err != nil {
		t.Fatal(err)
	}

	second, err := fin.CreateAccount(ctx, "Second", "savings", "")
	if err != nil {
		t.Fatalf("an account after archiving the only one: %v", err)
	}

	if _, err := fin.CreateAccount(ctx, "Third", "cash", ""); !workspaces.IsUpgrade(err) {
		t.Errorf("a third account on Free = %v, want the upgrade answer", err)
	}

	if _, err := fin.UnarchiveAccount(ctx, first); !workspaces.IsUpgrade(err) {
		t.Errorf("unarchiving past the plan = %v, want the upgrade answer", err)
	}

	if got := firstOf[models.Account](t, app, ana, ""); !got.ArchivedAt.Valid {
		t.Error("a refused unarchive still cleared archived_at")
	}

	if _, err := fin.ArchiveAccount(ctx, second.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.UnarchiveAccount(ctx, first); err != nil {
		t.Errorf("unarchiving with room in the plan: %v", err)
	}

	makePro(t, app, ana)

	if _, err := fin.UnarchiveAccount(ctx, second.ID); err != nil {
		t.Errorf("Pro has no account limit: %v", err)
	}
}
