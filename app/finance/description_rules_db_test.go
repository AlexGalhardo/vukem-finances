package finance_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

type memSite struct {
	app     *vukemtest.App
	fin     *finance.Service
	ctx     context.Context
	account int64
}

func newMemSite(t *testing.T, email string, app *vukemtest.App) memSite {
	t.Helper()

	if app == nil {
		app = vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"APP_EDITION": "saas"}})
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u := models.User{Name: "Ana", Email: email, Password: strings.Repeat("h", 20)}
	u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = time.Now().UTC(), true

	if err := orm.Create(t.Context(), db, &u); err != nil {
		t.Fatal(err)
	}

	ws, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	w, err := ws.Create(t.Context(), u, "Ana household", "USD")
	if err != nil {
		t.Fatal(err)
	}

	fin, err := finance.New(app.App, ws)
	if err != nil {
		t.Fatal(err)
	}

	ctx := workspaces.WithTenant(t.Context(), w.ID)

	acc, err := workspaces.Scoped[models.Account](ctx, db).First(ctx)
	if err != nil {
		t.Fatal(err)
	}

	return memSite{app: app, fin: fin, ctx: ctx, account: acc.ID}
}

func (s memSite) category(t *testing.T, name string) models.Category {
	t.Helper()

	db, _ := s.app.DB()

	c, err := workspaces.Scoped[models.Category](s.ctx, db).Where("name", name).First(s.ctx)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func (s memSite) add(t *testing.T, description string, category int64) models.Transaction {
	t.Helper()

	tx, err := s.fin.CreateTransaction(s.ctx, finance.TxInput{
		AccountID: s.account, CategoryID: category, Kind: models.KindExpense, Amount: "10.00",
		Description: description, Date: "2026-11-10",
	})
	if err != nil {
		t.Fatal(err)
	}

	return tx
}

func TestRulesMemory(t *testing.T) {
	t.Parallel()

	s := newMemSite(t, "ana@example.com", nil)
	db, _ := s.app.DB()
	groceries, rent := s.category(t, "Groceries"), s.category(t, "Housing")

	if _, ok, err := s.fin.RuleCategory(s.ctx, db, "Padaria Sao Joao", models.KindExpense); err != nil || ok {
		t.Fatalf("nothing is remembered yet: ok=%v err=%v", ok, err)
	}

	// A manual categorization teaches the memory; a variant of the line hits it.
	s.add(t, "PADARIA SÃO JOÃO 1234", groceries.ID)

	got, ok, err := s.fin.RuleCategory(s.ctx, db, "padaria sao joao 12/11", models.KindExpense)
	if err != nil || !ok || got.ID != groceries.ID {
		t.Fatalf("rule = %v %v %v, want groceries", got.Name, ok, err)
	}

	// An entry without a category teaches nothing.
	s.add(t, "Mystery shop", 0)

	if _, ok, _ := s.fin.RuleCategory(s.ctx, db, "Mystery shop", models.KindExpense); ok {
		t.Error("an uncategorized entry must not create a rule")
	}

	// The last correction wins.
	s.add(t, "Padaria Sao Joao", rent.ID)

	if got, _, _ := s.fin.RuleCategory(s.ctx, db, "Padaria Sao Joao", models.KindExpense); got.ID != rent.ID {
		t.Errorf("rule = %s, want Housing", got.Name)
	}

	// A rule is never used for a transaction of the other kind.
	if _, ok, _ := s.fin.RuleCategory(s.ctx, db, "Padaria Sao Joao", models.KindIncome); ok {
		t.Error("an expense category was offered for an income")
	}

	// An archived category is not offered.
	if _, err := s.fin.ArchiveCategory(s.ctx, rent.ID); err != nil {
		t.Fatal(err)
	}

	if _, ok, _ := s.fin.RuleCategory(s.ctx, db, "Padaria Sao Joao", models.KindExpense); ok {
		t.Error("an archived category was offered")
	}
}

func TestRulesMemoryStaysInTheWorkspace(t *testing.T) {
	t.Parallel()

	a := newMemSite(t, "ana@example.com", nil)
	b := newMemSite(t, "bob@example.com", a.app)
	db, _ := a.app.DB()

	a.add(t, "Padaria Sao Joao", a.category(t, "Groceries").ID)

	if _, ok, _ := b.fin.RuleCategory(b.ctx, db, "Padaria Sao Joao", models.KindExpense); ok {
		t.Error("a rule crossed to another workspace")
	}
}
