// Package aitest builds a workspace with AI wired to the fake provider, for the
// tests of the AI features. No test that uses it can reach a real provider or
// needs a real key.
package aitest

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/queue"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/ai/fake"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// Key is a made-up key: no test needs, or may use, a real one.
const Key = "sk-placeholder-0123456789abcdefghij-WXYZ"

// Rig is one user with a workspace, a saved key and the fake provider.
type Rig struct {
	App      *vukemtest.App
	DB       *database.DB
	User     models.User
	Ws       models.Workspace
	Ctx      context.Context
	Account  int64
	Fin      *finance.Service
	Client   *ai.Client
	Provider *fake.Provider
	Settings *ai.SettingsStore
	// Log holds what the client logged.
	Log *strings.Builder
}

// NewApp builds the app of a test, in the saas edition (several workspaces).
func NewApp(t *testing.T) *vukemtest.App {
	t.Helper()

	// A recording queue: a job a write enqueues is not run behind the test's back.
	return vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:        map[string]string{"APP_EDITION": "saas"},
		AppOptions: []vukem.Option{vukem.WithQueueDriver(queue.NewFake())},
	})
}

// New makes a user with a workspace in app (a new app when nil), saves the key
// and leaves AI off: call Enable.
func New(t *testing.T, app *vukemtest.App, email string) *Rig {
	t.Helper()

	if app == nil {
		app = NewApp(t)
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

	wsvc, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	ws, err := wsvc.Create(t.Context(), u, "Ana household", "USD")
	if err != nil {
		t.Fatal(err)
	}

	fin, err := finance.New(app.App, wsvc)
	if err != nil {
		t.Fatal(err)
	}

	ctx := workspaces.WithTenant(t.Context(), ws.ID)

	acc, err := workspaces.Scoped[models.Account](ctx, db).First(ctx)
	if err != nil {
		t.Fatal(err)
	}

	vault, err := ai.NewVault(app.App, false)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := vault.Save(ctx, u.ID, ai.SaveInput{Provider: ai.KindAnthropic, Key: ai.NewKey(Key)}); err != nil {
		t.Fatal(err)
	}

	provider := &fake.Provider{}

	// Whatever else builds the shared client in this app must not reach a real provider either.
	shared, err := ai.Shared(app.App)
	if err != nil {
		t.Fatal(err)
	}

	shared.Client.Factory = provider.Factory()

	settings := ai.NewSettingsStore(db, nil)
	log := &strings.Builder{}

	return &Rig{
		App: app, DB: db, User: u, Ws: ws, Ctx: ctx, Account: acc.ID, Fin: fin, Provider: provider, Settings: settings, Log: log,
		Client: ai.NewClient(db, vault, settings, provider.Factory(), slog.New(slog.NewTextHandler(log, nil)), nil),
	}
}

// Enable turns AI on with the given features (consent given, a high limit).
func (r *Rig) Enable(t *testing.T, autoApply bool, features ...string) {
	t.Helper()

	on := map[string]bool{}
	for _, f := range features {
		on[f] = true
	}

	_, err := r.Settings.Update(r.Ctx, r.User.ID, ai.UpdateInput{
		Enabled: true, Consent: true, MonthlyCallLimit: 1000, Features: on, AutoApply: autoApply,
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Category returns the category of the workspace with this name.
func (r *Rig) Category(t *testing.T, name string) models.Category {
	t.Helper()

	c, err := workspaces.Scoped[models.Category](r.Ctx, r.DB).Where("name", name).First(r.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

// Add records an expense on the first account; category 0 leaves it uncategorized.
func (r *Rig) Add(t *testing.T, description string, category int64) models.Transaction {
	t.Helper()

	return r.AddOn(t, models.KindExpense, "10.00", description, "2026-11-10", category)
}

// AddOn records a transaction of any kind, amount and date.
func (r *Rig) AddOn(t *testing.T, kind, amount, description, date string, category int64) models.Transaction {
	t.Helper()

	tx, err := r.Fin.CreateTransaction(r.Ctx, finance.TxInput{
		AccountID: r.Account, CategoryID: category, Kind: kind, Amount: amount, Description: description, Date: date,
	})
	if err != nil {
		t.Fatal(err)
	}

	return tx
}

// Reload reads a transaction again.
func (r *Rig) Reload(t *testing.T, id int64) models.Transaction {
	t.Helper()

	tx, err := workspaces.Scoped[models.Transaction](r.Ctx, r.DB).Where("id", id).First(r.Ctx)
	if err != nil {
		t.Fatal(err)
	}

	return tx
}
