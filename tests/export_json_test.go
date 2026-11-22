package tests

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
)

const foreignEmail = "viewer-private@example.com"

// richWorkspace fills a workspace with one of everything the export holds, plus
// the private data around it that the export must never carry.
func richWorkspace(t *testing.T) (*vukemtest.App, who) {
	t.Helper()

	app, ana, acc := proWorkspace(t)
	ctx := wsCtx(t, ana)
	fin := softService(t, app)

	viewer := newVerifiedUser(t, app, "Vera Viewer", foreignEmail)
	join(t, app, ana, viewer, models.RoleViewer)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	// A Stripe customer on the workspace and a token of the user: both must stay out.
	if _, err := orm.QueryOf[models.Workspace](db).Where("id", ana.Workspace.ID).
		Update(t.Context(), orm.Set{"stripe_customer_id": sql.NullString{String: "cus_PRIVATE123", Valid: true}}); err != nil {
		t.Fatal(err)
	}

	makeToken(t, app, ana, 30, "transactions:read")

	food := categoryOf(t, app, ana, models.KindExpense)
	pay := categoryOf(t, app, ana, models.KindIncome)

	record(t, app, ana, acc, pay.ID, "income", "2500.00", "2026-01-05", "Salary")
	record(t, app, ana, acc, food.ID, "expense", "82.40", "2026-01-06", "Market", "weekly", "home")
	record(t, app, ana, acc, 0, "expense", "10.00", "2026-01-07", `=HYPERLINK("x")`)

	gone := record(t, app, ana, acc, 0, "expense", "999.00", "2026-01-08", "DELETED ROW")
	if err := fin.DeleteTransaction(ctx, gone.ID); err != nil {
		t.Fatal(err)
	}

	extra := insertAccount(t, app, ana, "Extra", 5000)

	if _, err := fin.CreateTransfer(ctx, finance.TransferInput{
		FromAccountID: acc, ToAccountID: extra.ID, Amount: "300.00", Description: "move", Date: "2026-01-10",
	}); err != nil {
		t.Fatal(err)
	}

	if err := fin.SaveBudget(ctx, finance.BudgetInput{CategoryID: food.ID, Limit: "400.00"}); err != nil {
		t.Fatal(err)
	}

	goal, err := fin.CreateGoal(ctx, finance.GoalInput{Name: "Trip", Target: "1000.00", TargetDate: "2027-01-01"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.AddContribution(ctx, goal.ID, finance.ContributionInput{Amount: "50.00", Date: "2026-01-20"}); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.CreateRecurrence(ctx, finance.RecurrenceInput{
		AccountID: acc, Kind: "expense", Amount: "30.00", Description: "Gym", Frequency: "monthly", Interval: 1,
		StartDate: "2027-03-01", Active: true,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := fin.CreateCard(ctx, finance.CardInput{Name: "Visa", Limit: "2000.00", ClosingDay: 10, DueDay: 20, Brand: "visa", Last4: "4242"}); err != nil {
		t.Fatal(err)
	}

	return app, ana
}

func loadSchema(t *testing.T) map[string]any {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "docs", "export-format.schema.json"))
	if err != nil {
		t.Fatal(err)
	}

	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}

	return schema
}

func TestJSONExportMatchesSchema(t *testing.T) {
	t.Parallel()

	app, ana := richWorkspace(t)

	res := ana.Client.Get("/workspace/export.json").AssertOK()
	res.AssertHeaderContains("Content-Type", "application/json")
	res.AssertHeader("X-Content-Type-Options", "nosniff")
	res.AssertHeader("Content-Disposition", `attachment; filename="vukem-finances-export-`+today()+`.json"`)

	doc, err := parseJSON(res.Body())
	if err != nil {
		t.Fatalf("not JSON: %v", err)
	}

	schema := loadSchema(t)

	if problems := validateSchema(schema, doc); len(problems) > 0 {
		t.Fatalf("the export does not match docs/export-format.schema.json:\n%s", strings.Join(problems, "\n"))
	}

	top := doc.(map[string]any)

	for key, want := range map[string]int{
		"accounts": 3, "tags": 2, "transactions": 5, "budgets": 1, "goals": 1, "recurrences": 1, "cards": 1,
	} {
		if got := len(top[key].([]any)); key != "accounts" && got != want || key == "accounts" && got < want {
			t.Errorf("%s: %d rows, want %d", key, got, want)
		}
	}

	if strings.Contains(res.Body(), "DELETED ROW") {
		t.Error("a deleted transaction was exported")
	}

	members := top["workspace"].(map[string]any)["members"].([]any)
	if len(members) != 2 {
		t.Errorf("members = %v", members)
	}

	goal := top["goals"].([]any)[0].(map[string]any)
	if len(goal["contributions"].([]any)) != 1 {
		t.Errorf("goal = %v", goal)
	}

	t.Run("the validator is not vacuous", func(t *testing.T) {
		bad, _ := parseJSON(res.Body())
		m := bad.(map[string]any)
		m["leak"] = "x"
		m["transactions"].([]any)[0].(map[string]any)["amount_cents"] = "12"
		delete(m["accounts"].([]any)[0].(map[string]any), "uid")

		if problems := validateSchema(schema, bad); len(problems) != 3 {
			t.Errorf("problems = %v, want an unexpected field, a wrong type and a missing uid", problems)
		}
	})

	t.Run("the transactions agree with the reports and the CSV", func(t *testing.T) {
		var income, expense int64

		for _, row := range top["transactions"].([]any) {
			r := row.(map[string]any)
			if r["transfer_uid"] != nil {
				continue
			}

			if r["kind"] == "income" {
				income += int64(r["amount_cents"].(float64))
			} else {
				expense += int64(r["amount_cents"].(float64))
			}
		}

		cf, err := reports.New(softService(t, app)).CashFlow(wsCtx(t, ana), reports.Period{Preset: reports.Custom, From: "2020-01-01", To: "2030-12-31"})
		if err != nil {
			t.Fatal(err)
		}

		if cf.Totals.IncomeCents != income || cf.Totals.ExpenseCents != expense {
			t.Errorf("reports = %+v, JSON = %d income and %d expense", cf.Totals, income, expense)
		}

		var csvIncome, csvExpense int64

		for _, rec := range csvRecords(t, ana.Client.Get("/transactions/export.csv").AssertOK().Body())[1:] {
			if rec[8] != "" {
				continue
			}

			cents, err := finance.ParseAmount(rec[6])
			if err != nil {
				t.Fatal(err)
			}

			if cents > 0 {
				csvIncome += cents
			} else {
				csvExpense -= cents
			}
		}

		if csvIncome != income || csvExpense != expense {
			t.Errorf("CSV = %d income and %d expense, JSON = %d and %d", csvIncome, csvExpense, income, expense)
		}
	})
}

// forbiddenNames are fragments no key of the export may hold.
var forbiddenNames = []string{
	"hash", "token", "secret", "password", "email", "stripe", "totp", "two_factor", "api_key", "apikey", "recovery", "session",
}

func walkKeys(v any, fn func(key string)) {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			fn(k)
			walkKeys(e, fn)
		}
	case []any:
		for _, e := range x {
			walkKeys(e, fn)
		}
	}
}

func TestSecurity_JSONExportHasNoSecrets(t *testing.T) {
	t.Parallel()

	app, ana := richWorkspace(t)
	bob := signUp(t, app, "Bob", "bob@example.com")
	bobAcc := firstAccount(t, app, bob)
	record(t, app, bob, bobAcc, 0, "expense", "5.55", "2026-01-09", "BOB PRIVATE")

	body := ana.Client.Get("/workspace/export.json").AssertOK().Body()

	doc, err := parseJSON(body)
	if err != nil {
		t.Fatal(err)
	}

	walkKeys(doc, func(key string) {
		for _, bad := range forbiddenNames {
			if strings.Contains(strings.ToLower(key), bad) {
				t.Errorf("the export has a field named %q", key)
			}
		}
	})

	for _, secret := range []string{
		foreignEmail, ana.User.Email, "cus_PRIVATE123", "sub_test_", "price_pro", "vf_", "BOB PRIVATE", "bob@example.com", "$2a$", "argon",
	} {
		if strings.Contains(body, secret) {
			t.Errorf("the export holds %q", secret)
		}
	}

	if !strings.Contains(body, "Vera Viewer") {
		t.Error("members are listed by display name and role")
	}
}
