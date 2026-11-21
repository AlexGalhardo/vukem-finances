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

func TestInstallmentCreatesTheParts(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-01-05")
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	parts, err := fin.CreateInstallments(ctx, finance.InstallmentInput{
		AccountID: firstAccount(t, app, ana), Total: "100.00", Parts: 3, FirstDate: "2027-01-31", Description: "Sofa",
	})
	if err != nil {
		t.Fatal(err)
	}

	wantCents, wantDates := []int64{3334, 3333, 3333}, []string{"2027-01-31", "2027-02-28", "2027-03-31"}

	for i, p := range parts {
		if p.AmountCents != wantCents[i] || p.OccurredOn != wantDates[i] || p.InstallmentNo.V != int64(i+1) ||
			p.InstallmentTotal.V != 3 || p.InstallmentGroupUID.V != parts[0].InstallmentGroupUID.V {
			t.Errorf("part %d = %+v", i+1, p)
		}
	}
}

func TestInstallmentRefusals(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-01-05")
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	acc := firstAccount(t, app, ana)

	for name, in := range map[string]finance.InstallmentInput{
		"total below the parts": {AccountID: acc, Total: "0.02", Parts: 3, FirstDate: "2027-01-31"},
		"one part":              {AccountID: acc, Total: "10.00", Parts: 1, FirstDate: "2027-01-31"},
		"sixty one parts":       {AccountID: acc, Total: "100.00", Parts: 61, FirstDate: "2027-01-31"},
		"bad date":              {AccountID: acc, Total: "10.00", Parts: 2, FirstDate: "31/01/2027"},
	} {
		_, err := fin.CreateInstallments(ctx, in)

		e, ok := errors.AsType[*errs.HTTPError](err)
		if !ok || e.Status != http.StatusUnprocessableEntity {
			t.Errorf("%s: err = %v, want 422", name, err)
		}
	}

	if live, err := workspaces.Scoped[models.Transaction](ctx, mustDB(t, app)).Count(ctx); err != nil || live != 0 {
		t.Errorf("%d transactions written by refused purchases, %v", live, err)
	}
}

// A part over its month's quota refuses the whole group.
func TestInstallmentQuotaRefusesTheGroup(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-01-05")
	ana := signUp(t, app, "Ana", "ana@example.com")
	acc := firstAccount(t, app, ana)
	fillMonth(t, app, ana, acc, 100, "2027-03")

	_, err := softService(t, app).CreateInstallments(wsCtx(t, ana), finance.InstallmentInput{
		AccountID: acc, Total: "30.00", Parts: 3, FirstDate: "2027-01-31", Description: "Sofa",
	})
	if err == nil {
		t.Fatal("the March part exceeded the Free quota and the purchase was accepted")
	}

	live, err := workspaces.Scoped[models.Transaction](wsCtx(t, ana), mustDB(t, app)).Where("description", "Sofa").Count(t.Context())
	if err != nil || live != 0 {
		t.Errorf("%d parts written, %v; want none", live, err)
	}
}

func TestInstallmentOnCard(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-01-05")
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	card, err := fin.CreateCard(ctx, finance.CardInput{Name: "Visa", Limit: "5000.00", ClosingDay: 10, DueDay: 20})
	if err != nil {
		t.Fatal(err)
	}

	parts, err := fin.CreateInstallments(ctx, finance.InstallmentInput{
		AccountID: card.AccountID, Total: "90.00", Parts: 3, FirstDate: "2027-01-08", Description: "Phone",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Closing day 10: Jan 8 closes Jan 10, Feb 8 Feb 10, Mar 8 Mar 10.
	for i, want := range []string{"2027-01-10", "2027-02-10", "2027-03-10"} {
		if parts[i].StatementRef.V != want {
			t.Errorf("part %d statement = %q, want %s", i+1, parts[i].StatementRef.V, want)
		}
	}

	// A purchase on the 15th is already in the next statement.
	late, err := fin.CreateInstallments(ctx, finance.InstallmentInput{
		AccountID: card.AccountID, Total: "20.00", Parts: 2, FirstDate: "2027-01-15", Description: "Late",
	})
	if err != nil || late[0].StatementRef.V != "2027-02-10" || late[1].StatementRef.V != "2027-03-10" {
		t.Errorf("late parts = %+v, %v", late, err)
	}
}

func TestInstallmentDelete(t *testing.T) {
	t.Parallel()

	app, _ := clockApp(t, "2027-02-15")
	ana := signUp(t, app, "Ana", "ana@example.com")
	fin := softService(t, app)
	ctx := wsCtx(t, ana)
	db := mustDB(t, app)

	buy := func(desc string) []models.Transaction {
		t.Helper()

		parts, err := fin.CreateInstallments(ctx, finance.InstallmentInput{
			AccountID: firstAccount(t, app, ana), Total: "60.00", Parts: 6, FirstDate: "2027-01-15", Description: desc,
		})
		if err != nil {
			t.Fatal(err)
		}

		return parts
	}

	live := func(desc string) int64 {
		t.Helper()

		n, err := workspaces.Scoped[models.Transaction](ctx, db).Where("description", desc).Count(ctx)
		if err != nil {
			t.Fatal(err)
		}

		return n
	}

	// Jan, Feb (today) are past; Mar to Jun are future.
	parts := buy("A")
	if n, err := fin.DeleteInstallments(ctx, parts[2].ID, false, false); err != nil || n != 4 || live("A") != 2 {
		t.Errorf("this and the following: deleted %d, %v, live %d; want 4 and 2", n, err, live("A"))
	}

	parts = buy("B")
	if n, err := fin.DeleteInstallments(ctx, parts[3].ID, false, false); err != nil || n != 3 || live("B") != 3 {
		t.Errorf("from part 4: deleted %d, %v, live %d; want 3 and 3", n, err, live("B"))
	}

	parts = buy("C")
	if n, err := fin.DeleteInstallments(ctx, parts[0].ID, true, false); err != nil || n != 4 || live("C") != 2 {
		t.Errorf("whole group, future only: deleted %d, %v, live %d; want 4 and 2", n, err, live("C"))
	}

	parts = buy("D")
	if n, err := fin.DeleteInstallments(ctx, parts[0].ID, true, true); err != nil || n != 6 || live("D") != 0 {
		t.Errorf("whole group with the past: deleted %d, %v, live %d; want 6 and 0", n, err, live("D"))
	}

	plain, err := fin.CreateTransaction(ctx, finance.TxInput{
		AccountID: firstAccount(t, app, ana), Kind: models.KindExpense, Amount: "1.00", Date: "2027-02-15",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.DeleteInstallments(ctx, plain.ID, true, true); err == nil {
		t.Error("a plain transaction was deleted as an installment group")
	}
}
