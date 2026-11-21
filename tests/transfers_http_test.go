package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func patchForm(c *vukemtest.Client, target string, form url.Values) *vukemtest.Response {
	return c.Do(http.MethodPatch, target, strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
}

func transferForm(from, to int64, amount string) url.Values {
	return url.Values{
		"fromAccountId": {idStr(from)}, "toAccountId": {idStr(to)}, "amount": {amount},
		"description": {"move"}, "date": {daysAgo(1)},
	}
}

func TestAccountsHTTP(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	vera := viewerOf(t, app, ana)
	main := firstAccount(t, app, ana)
	extra := insertAccount(t, app, ana, "Extra", 2000)

	addTransaction(t, ana, main, 0, "expense", "10.00", daysAgo(1)).AssertRedirect("/transactions")

	t.Run("update changes the name, kind and opening balance", func(t *testing.T) {
		patchForm(ana.Client, "/accounts/"+idStr(extra.ID), url.Values{"name": {"Rainy day"}, "kind": {"savings"}, "opening": {"12.34"}}).
			AssertRedirect("/accounts")

		got := firstOf[models.Account](t, app, ana, "Rainy day")
		if got.Kind != "savings" || got.OpeningCents != 1234 || got.Version < 2 {
			t.Errorf("account = %+v, want savings, 1234 cents, version bumped", got)
		}
	})

	t.Run("a bad kind or amount is refused with a message and changes nothing", func(t *testing.T) {
		for _, form := range []url.Values{
			{"name": {"X"}, "kind": {"credit_card"}},
			{"name": {"X"}, "kind": {"cash"}, "opening": {"abc"}},
			{"name": {"X"}, "kind": {"cash"}, "opening": {"90071992547409.92"}},
		} {
			patchForm(ana.Client, "/accounts/"+idStr(extra.ID), form).AssertRedirect("/accounts")
		}

		if got := firstOf[models.Account](t, app, ana, "Rainy day"); got.Kind != "savings" || got.OpeningCents != 1234 {
			t.Errorf("a refused update changed the account: %+v", got)
		}

		ana.Client.PostForm("/accounts", url.Values{"name": {"Card"}, "kind": {"credit_card"}}).AssertRedirect("/accounts")

		if n := count[models.Account](t, app, ana); n != 2 {
			t.Errorf("%d accounts, want 2: credit_card is reserved", n)
		}
	})

	t.Run("the page shows balances, the net worth and the archived flag", func(t *testing.T) {
		ana.Client.PostForm("/accounts/"+idStr(extra.ID)+"/archive", nil).AssertRedirect("/accounts")

		if got := firstOf[models.Account](t, app, ana, "Rainy day"); !got.ArchivedAt.Valid {
			t.Fatal("the account was not archived")
		}

		page := ana.Client.Inertia("").Get("/accounts").AssertInertia("Accounts")
		// The seeded account at -10.00 is the only one that counts.
		page.PropEquals("netWorthCents", float64(firstOf[models.Account](t, app, ana, "").OpeningCents-1000))
		page.PropEquals("balances.1.archived", true)
		page.PropEquals("balances.1.balanceCents", float64(1234))
		page.PropEquals("balances.0.transactions", float64(1))

		ana.Client.PostForm("/accounts/"+idStr(extra.ID)+"/unarchive", nil).AssertRedirect("/accounts")

		if got := firstOf[models.Account](t, app, ana, "Rainy day"); got.ArchivedAt.Valid {
			t.Error("the account is still archived")
		}
	})

	t.Run("a viewer cannot update, archive or unarchive", func(t *testing.T) {
		before := firstOf[models.Account](t, app, ana, "Rainy day")

		patchForm(vera.Client, "/accounts/"+idStr(extra.ID), url.Values{"name": {"Hacked"}, "kind": {"cash"}}).AssertForbidden()
		vera.Client.PostForm("/accounts/"+idStr(extra.ID)+"/archive", nil).AssertForbidden()
		vera.Client.PostForm("/accounts/"+idStr(extra.ID)+"/unarchive", nil).AssertForbidden()
		vera.Client.Delete("/accounts/" + idStr(extra.ID)).AssertForbidden()

		if got := firstOf[models.Account](t, app, ana, "Rainy day"); got.Version != before.Version || got.ArchivedAt.Valid {
			t.Errorf("the viewer changed the account: %+v", got)
		}
	})

	t.Run("unarchiving past the Free plan sends the user to billing", func(t *testing.T) {
		app := newApp(t)
		ben := signUp(t, app, "Ben", "ben@example.com")
		first := firstAccount(t, app, ben)

		ben.Client.PostForm("/accounts/"+idStr(first)+"/archive", nil).AssertRedirect("/accounts")
		ben.Client.PostForm("/accounts", url.Values{"name": {"Second"}, "kind": {"cash"}}).AssertRedirect("/accounts")
		ben.Client.PostForm("/accounts/"+idStr(first)+"/unarchive", nil).AssertRedirect("/billing")
	})
}

func TestTransferHTTP(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	bob := signUp(t, app, "Bob", "bob@example.com")
	vera := viewerOf(t, app, ana)
	anaJSON := browser(t, app, ana.User, ana.Workspace.ID).AcceptJSON()
	bobJSON := browser(t, app, bob.User, bob.Workspace.ID).AcceptJSON()
	checking := insertAccount(t, app, ana, "Checking", 100000)
	savings := insertAccount(t, app, ana, "Savings", 0)
	foreign := insertAccount(t, app, bob, "Bob's", 0)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	fin := softService(t, app)

	legs := func() []models.Transaction {
		db, err := app.DB()
		if err != nil {
			t.Fatal(err)
		}

		got, err := workspaces.Scoped[models.Transaction](ctx, db).WhereNotNull("transfer_uid").Get(ctx)
		if err != nil {
			t.Fatal(err)
		}

		return got
	}

	ana.Client.PostForm("/transfers", transferForm(checking.ID, savings.ID, "100.00")).AssertRedirect("/transactions")

	made := legs()
	if len(made) != 2 || made[0].TransferUID != made[1].TransferUID {
		t.Fatalf("legs = %+v, want two sharing a transfer uid", made)
	}

	if bs, err := fin.AccountBalances(ctx, ""); err != nil || balanceOf(t, bs, checking.ID).BalanceCents != 90000 || balanceOf(t, bs, savings.ID).BalanceCents != 10000 {
		t.Errorf("balances = %+v, %v; want 90000 and 10000", bs, err)
	}

	ana.Client.Inertia("").Get("/transactions").AssertInertia("Transactions").PropEquals("transactions.0.transferAccountName", "Checking")

	t.Run("refused transfers keep the dialog open with a field error", func(t *testing.T) {
		ana.Client.PostForm("/transfers", transferForm(checking.ID, checking.ID, "1.00")).
			AssertRedirect("/transactions").AssertSessionHasErrors("toAccountId")
		ana.Client.PostForm("/transfers", transferForm(checking.ID, savings.ID, "0")).
			AssertRedirect("/transactions").AssertSessionHasErrors("amount")
		anaJSON.PostJSON("/transfers", map[string]any{
			"fromAccountId": idStr(checking.ID), "toAccountId": idStr(checking.ID), "amount": "1", "date": daysAgo(0),
		}).AssertUnprocessable()

		if n := len(legs()); n != 2 {
			t.Errorf("%d legs after refused transfers, want 2", n)
		}
	})

	t.Run("an account of another workspace is a 404, in either position", func(t *testing.T) {
		ana.Client.PostForm("/transfers", transferForm(checking.ID, foreign.ID, "1.00")).AssertNotFound()
		ana.Client.PostForm("/transfers", transferForm(foreign.ID, savings.ID, "1.00")).AssertNotFound()
		anaJSON.PostJSON("/transfers", map[string]any{
			"fromAccountId": idStr(checking.ID), "toAccountId": idStr(foreign.ID), "amount": "1", "date": daysAgo(0),
		}).AssertNotFound()

		if n := len(legs()); n != 2 {
			t.Errorf("%d legs after cross-workspace attempts, want 2", n)
		}
	})

	t.Run("an archived account is refused with 422", func(t *testing.T) {
		old := insertAccount(t, app, ana, "Old", 0)
		if _, err := fin.ArchiveAccount(ctx, old.ID); err != nil {
			t.Fatal(err)
		}

		anaJSON.PostJSON("/transfers", map[string]any{
			"fromAccountId": idStr(checking.ID), "toAccountId": idStr(old.ID), "amount": "1", "date": daysAgo(0),
		}).AssertUnprocessable()
	})

	t.Run("another workspace cannot read, edit or delete the legs", func(t *testing.T) {
		leg := idStr(made[0].ID)

		bobJSON.PatchJSON("/transfers/"+leg, map[string]any{
			"fromAccountId": idStr(foreign.ID), "toAccountId": idStr(foreign.ID), "amount": "1", "date": daysAgo(0),
		}).AssertNotFound()
		bob.Client.Delete("/transfers/" + leg).AssertNotFound()
		bob.Client.Delete("/transactions/" + leg).AssertNotFound()

		if n := len(legs()); n != 2 {
			t.Errorf("%d legs after Bob's calls, want 2", n)
		}
	})

	t.Run("a viewer cannot create, edit or delete a transfer", func(t *testing.T) {
		leg := idStr(made[0].ID)

		vera.Client.PostForm("/transfers", transferForm(checking.ID, savings.ID, "1.00")).AssertForbidden()
		patchForm(vera.Client, "/transfers/"+leg, transferForm(checking.ID, savings.ID, "2.00")).AssertForbidden()
		vera.Client.Delete("/transfers/" + leg).AssertForbidden()
		vera.Client.Delete("/transactions/" + leg).AssertForbidden()

		got := legs()
		if len(got) != 2 || got[0].AmountCents != 10000 || got[0].DeletedAt.Valid {
			t.Errorf("the viewer changed the transfer: %+v", got)
		}
	})

	t.Run("editing changes both legs and an ordinary edit of a leg is refused", func(t *testing.T) {
		leg := idStr(made[1].ID)

		patchForm(ana.Client, "/transfers/"+leg, transferForm(checking.ID, savings.ID, "40.00")).AssertRedirect("/transactions")

		for _, l := range legs() {
			if l.AmountCents != 4000 {
				t.Errorf("leg = %+v, want 4000", l)
			}
		}

		anaJSON.PatchJSON("/transactions/"+leg, map[string]any{
			"accountId": idStr(savings.ID), "kind": "income", "amount": "1", "date": daysAgo(0),
		}).AssertUnprocessable()
	})

	t.Run("deleting through the transactions route removes both legs", func(t *testing.T) {
		ana.Client.Delete("/transactions/" + idStr(made[0].ID)).AssertRedirect("/transactions")

		if n := len(legs()); n != 0 {
			t.Errorf("%d legs live after the delete, want 0", n)
		}

		ana.Client.Delete("/transfers/" + idStr(made[0].ID)).AssertNotFound()
	})
}

func TestSecurity_TransferFromViewerAndOtherWorkspaceChangesNothing(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	vera := viewerOf(t, app, ana)
	a := insertAccount(t, app, ana, "A", 100)
	b := insertAccount(t, app, ana, "B", 100)

	vera.Client.PostForm("/transfers", transferForm(a.ID, b.ID, "5.00")).AssertForbidden()

	if n := count[models.Transaction](t, app, ana); n != 0 {
		t.Errorf("a viewer created %d transactions", n)
	}
}
