package reports_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// fx is a Pro workspace with the services the reports read through.
type fx struct {
	t       *testing.T
	app     *vukemtest.App
	ctx     context.Context
	fin     *finance.Service
	rep     *reports.Service
	ws      models.Workspace
	account models.Account
}

func newFx(t *testing.T) *fx {
	t.Helper()

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"APP_EDITION": "saas"}})

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u := models.User{Name: "Ana", Email: "ana@example.com"}
	u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = time.Now().UTC(), true

	if err := orm.Create(t.Context(), db, &u); err != nil {
		t.Fatal(err)
	}

	wsvc, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	ws, err := wsvc.Create(t.Context(), u, "Ana household", "USD")
	if err != nil {
		t.Fatal(err)
	}

	sub := models.WorkspaceSubscription{
		WorkspaceID: ws.ID, StripeSubscriptionID: "sub_test_" + strconv.FormatInt(ws.ID, 10),
		Status: "active", PriceID: "price_pro", UnitAmount: 900,
	}
	if err := orm.Create(t.Context(), db, &sub); err != nil {
		t.Fatal(err)
	}

	fin, err := finance.New(app.App, wsvc)
	if err != nil {
		t.Fatal(err)
	}

	ctx := workspaces.WithTenant(t.Context(), ws.ID)

	accounts, err := fin.Accounts(ctx)
	if err != nil || len(accounts) == 0 {
		t.Fatalf("accounts: %v, %v", accounts, err)
	}

	return &fx{t: t, app: app, ctx: ctx, fin: fin, rep: reports.New(fin), ws: ws, account: accounts[0]}
}

func (f *fx) tx(account, category int64, kind, amount, date string, tags ...string) models.Transaction {
	f.t.Helper()

	row, err := f.fin.CreateTransaction(f.ctx, finance.TxInput{
		AccountID: account, CategoryID: category, Kind: kind, Amount: amount, Description: "x", Date: date, Tags: tags,
	})
	if err != nil {
		f.t.Fatalf("%s %s on %s: %v", kind, amount, date, err)
	}

	return row
}

func (f *fx) newAccount(name, kind, opening string) models.Account {
	f.t.Helper()

	a, err := f.fin.CreateAccount(f.ctx, name, kind, opening)
	if err != nil {
		f.t.Fatal(err)
	}

	return a
}

func (f *fx) category(name, kind string, parent int64) models.Category {
	f.t.Helper()

	c, err := f.fin.AddCategory(f.ctx, finance.CategoryInput{Name: name, Kind: kind, ParentID: parent})
	if err != nil {
		f.t.Fatal(err)
	}

	return c
}

func (f *fx) period(from, to string) reports.Period {
	f.t.Helper()

	p, err := reports.NewPeriod(reports.Custom, from, to, "2026-06-15")
	if err != nil {
		f.t.Fatal(err)
	}

	return p
}
