package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem/vukemtest"
)

// isoCall runs one operation as the actor and returns what it answered.
type isoCall func(ctx context.Context, a operations.Actor) (any, error)

func call[In, Out any](op *operations.Op[In, Out], in In) isoCall {
	return func(ctx context.Context, a operations.Actor) (any, error) {
		res, err := operations.Invoke(ctx, op, a, in)
		if err != nil {
			return nil, err
		}

		if d, ok := any(res.Value).(operations.Download); ok {
			var buf bytes.Buffer
			if err := d.Write(&buf); err != nil {
				return nil, err
			}

			return buf.String(), nil
		}

		return res.Value, nil
	}
}

// isoCase is an operation run in workspace B with what workspace A owns.
type isoCase struct {
	run isoCall
	// foreign: the input names a row of A, so the answer is not_found and nothing
	// of A changes. Otherwise the input names no row: the operation works in B
	// and nothing of A is in its answer or changed by it.
	foreign bool
}

// theirs are the rows of workspace A the cases point at.
type theirs struct {
	account, account2, category, tag, transaction, transfer, budget, card, statement, recurrence, goal, contribution string
}

func isolationCases(c *operations.Catalog, a theirs) map[string]isoCase {
	foreign := func(run isoCall) isoCase { return isoCase{run: run, foreign: true} }
	mine := func(run isoCall) isoCase { return isoCase{run: run} }
	uid := operations.UIDIn{UID: ""}

	with := func(id string) operations.UIDIn { u := uid; u.UID = id; return u }

	return map[string]isoCase{
		"accounts.list":    mine(call(c.Accounts.List, operations.NoInput{})),
		"accounts.show":    foreign(call(c.Accounts.Show, with(a.account))),
		"accounts.create":  mine(call(c.Accounts.Create, operations.AccountCreateIn{Name: "Ana's secret account", Kind: "cash"})),
		"accounts.update":  foreign(call(c.Accounts.Update, operations.AccountUpdateIn{UID: a.account, Name: ptr("Renamed")})),
		"accounts.archive": foreign(call(c.Accounts.Archive, with(a.account))),

		"categories.list":    mine(call(c.Categories.List, operations.NoInput{})),
		"categories.show":    foreign(call(c.Categories.Show, with(a.category))),
		"categories.create":  foreign(call(c.Categories.Create, operations.CategoryCreateIn{Name: "Sub", ParentUID: a.category})),
		"categories.update":  foreign(call(c.Categories.Update, operations.CategoryUpdateIn{UID: a.category, Color: ptr("red")})),
		"categories.archive": foreign(call(c.Categories.Archive, with(a.category))),

		"tags.list":   mine(call(c.Tags.List, operations.NoInput{})),
		"tags.create": mine(call(c.Tags.Create, operations.TagCreateIn{Name: "ana-secret-tag"})),
		"tags.update": foreign(call(c.Tags.Update, operations.TagUpdateIn{UID: a.tag, Name: ptr("Renamed")})),
		"tags.delete": foreign(call(c.Tags.Delete, with(a.tag))),

		"transactions.list": foreign(call(c.Transactions.List, operations.TransactionListIn{Account: a.account})),
		"transactions.show": foreign(call(c.Transactions.Show, with(a.transaction))),
		"transactions.create": foreign(call(c.Transactions.Create, operations.TransactionCreateIn{
			AccountUID: a.account, Kind: "expense", AmountCents: 5, Date: "2026-11-20",
		})),
		"transactions.update": foreign(call(c.Transactions.Update, operations.TransactionUpdateIn{UID: a.transaction, AmountCents: ptr(int64(7))})),
		"transactions.delete": foreign(call(c.Transactions.Delete, with(a.transaction))),

		"transfers.create": foreign(call(c.Transactions.Transfer, operations.TransferCreateIn{
			FromAccountUID: a.account, ToAccountUID: a.account2, AmountCents: 5, Date: "2026-11-20",
		})),
		"transfers.delete": foreign(call(c.Transactions.Untransfer, with(a.transfer))),

		"budgets.list":   mine(call(c.Budgets.List, operations.MonthIn{})),
		"budgets.set":    foreign(call(c.Budgets.Set, operations.BudgetSetIn{CategoryUID: a.category, LimitCents: 200})),
		"budgets.delete": foreign(call(c.Budgets.Delete, with(a.budget))),
		"summary.show":   mine(call(c.Budgets.Summary, operations.MonthIn{})),

		"cards.list":      mine(call(c.Cards.List, operations.NoInput{})),
		"cards.show":      foreign(call(c.Cards.Show, with(a.card))),
		"cards.create":    foreign(call(c.Cards.Create, operations.CardCreateIn{Name: "Visa", LimitCents: 100, ClosingDay: 10, DueDay: 20, PaymentAccountUID: a.account})),
		"cards.update":    foreign(call(c.Cards.Update, operations.CardUpdateIn{UID: a.card, Name: ptr("Renamed")})),
		"cards.archive":   foreign(call(c.Cards.Archive, with(a.card))),
		"cards.unarchive": foreign(call(c.Cards.Unarchive, with(a.card))),

		"statements.list": foreign(call(c.Cards.Statements, operations.StatementsIn{UID: a.card})),
		"statements.show": foreign(call(c.Cards.Statement, operations.StatementIn{UID: a.card, Ref: a.statement})),
		"statements.pay": foreign(call(c.Cards.PayStatement, operations.StatementPayIn{
			UID: a.card, Ref: a.statement, FromAccountUID: a.account, AmountCents: 100,
		})),

		"recurrences.list": mine(call(c.Recurrences.List, operations.NoInput{})),
		"recurrences.create": foreign(call(c.Recurrences.Create, operations.RecurrenceCreateIn{
			AccountUID: a.account, Kind: "expense", AmountCents: 100, Frequency: "monthly", Interval: 1, StartDate: "2099-01-01",
		})),
		"recurrences.update": foreign(call(c.Recurrences.Update, operations.RecurrenceUpdateIn{UID: a.recurrence, Description: ptr("Renamed")})),
		"recurrences.delete": foreign(call(c.Recurrences.Delete, operations.RecurrenceDeleteIn{UID: a.recurrence, Future: true})),

		"goals.list":              mine(call(c.Goals.List, operations.NoInput{})),
		"goals.create":            mine(call(c.Goals.Create, operations.GoalCreateIn{Name: "Ana's secret goal", TargetCents: 100})),
		"goals.update":            foreign(call(c.Goals.Update, operations.GoalUpdateIn{UID: a.goal, Name: ptr("Renamed")})),
		"goals.archive":           foreign(call(c.Goals.Archive, with(a.goal))),
		"goals.unarchive":         foreign(call(c.Goals.Unarchive, with(a.goal))),
		"goals.delete":            foreign(call(c.Goals.Delete, with(a.goal))),
		"goals.contributions.add": foreign(call(c.Goals.Add, operations.ContributionAddIn{UID: a.goal, AmountCents: 100})),
		"goals.contributions.delete": foreign(call(c.Goals.Remove, operations.ContributionDeleteIn{
			UID: a.goal, ContributionUID: a.contribution,
		})),

		"reports.cashflow":   mine(call(c.Reports.CashFlow, operations.PeriodIn{Period: "last_12_months"})),
		"reports.categories": mine(call(c.Reports.Categories, operations.CategoryReportIn{Period: "last_12_months"})),
		"reports.accounts":   mine(call(c.Reports.Accounts, operations.PeriodIn{Period: "last_12_months"})),
		"reports.tags":       mine(call(c.Reports.Tags, operations.PeriodIn{Period: "last_12_months"})),
		"reports.networth":   mine(call(c.Reports.NetWorth, operations.PeriodIn{Period: "last_12_months"})),
		"reports.compare":    mine(call(c.Reports.Compare, operations.PeriodIn{Period: "last_12_months"})),

		"export.transactions": foreign(call(c.Exports.Transactions, operations.ExportTransactionsIn{Account: a.account})),
		"export.workspace":    mine(call(c.Exports.Workspace, operations.ExportWorkspaceIn{})),
	}
}

func ptr[T any](v T) *T { return &v }

// workspaceState fingerprints the rows of one workspace: the count and the sum
// of the versions of every tenant table the operations touch.
func workspaceState(t *testing.T, app *vukemtest.App, workspace int64) string {
	t.Helper()

	var parts []string

	for _, table := range []string{"accounts", "categories", "tags", "transactions", "budgets", "credit_cards", "recurrences", "goals", "goal_contributions"} {
		db, err := app.DB()
		if err != nil {
			t.Fatal(err)
		}

		var n, v int64
		if err := db.QueryRowContext(t.Context(),
			"SELECT COUNT(*), COALESCE(SUM(version), 0) FROM "+table+" WHERE workspace_id = ?", workspace).Scan(&n, &v); err != nil {
			t.Fatal(err)
		}

		parts = append(parts, table+":"+itoa(n)+"/"+itoa(v))
	}

	return strings.Join(parts, " ")
}

// TestOperationIsolation runs every operation of the inventory in workspace B
// with what workspace A owns. An operation that names a row of A answers
// not_found and changes nothing; one that names no row works in B, answers
// nothing of A's and changes nothing of A's. The cases must cover every entry
// of resources/operations.json.
func TestOperationIsolation(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	makePro(t, app, ana)
	makePro(t, app, bob)

	ws, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	fin, err := finance.New(app.App, ws)
	if err != nil {
		t.Fatal(err)
	}

	cache, err := app.App.Cache()
	if err != nil {
		t.Fatal(err)
	}

	cat := operations.NewCatalog(operations.Config{Edition: ws.Edition, Store: cache.Store(), Now: app.App.Now},
		operations.Services{WS: ws, Fin: fin, Rep: reports.New(fin)})

	var scopes []string
	for _, s := range api.Scopes() {
		scopes = append(scopes, s.Name)
	}

	actorOf := func(w who) operations.Actor {
		return operations.NewActor(operations.SurfaceREST, w.User, w.Workspace.ID, models.RoleOwner, scopes, "isolation")
	}

	// Workspace A owns one row of everything, with names that give it away.
	as := actorOf(ana)
	ctx := context.Background()

	must := func(res any, err error) {
		t.Helper()

		if err != nil {
			t.Fatal(err)
		}
	}

	var mine theirs

	acc, err := operations.Invoke(ctx, cat.Accounts.Create, as, operations.AccountCreateIn{Name: "Ana's secret account", Kind: "cash"})
	must(acc, err)

	acc2, err := operations.Invoke(ctx, cat.Accounts.Create, as, operations.AccountCreateIn{Name: "Ana's other account", Kind: "cash"})
	must(acc2, err)

	category, err := operations.Invoke(ctx, cat.Categories.Create, as, operations.CategoryCreateIn{Name: "Ana's secret category", Kind: "expense"})
	must(category, err)

	tag, err := operations.Invoke(ctx, cat.Tags.Create, as, operations.TagCreateIn{Name: "ana-secret-tag"})
	must(tag, err)

	tx, err := operations.Invoke(ctx, cat.Transactions.Create, as, operations.TransactionCreateIn{
		AccountUID: acc.Value.UID, CategoryUID: category.Value.UID, Kind: "expense", AmountCents: 777, Date: "2026-11-15",
		Description: "ana-secret-lunch", Tags: []string{"ana-secret-tag"},
	})
	must(tx, err)

	tr, err := operations.Invoke(ctx, cat.Transactions.Transfer, as, operations.TransferCreateIn{
		FromAccountUID: acc.Value.UID, ToAccountUID: acc2.Value.UID, AmountCents: 100, Date: "2026-11-15",
	})
	must(tr, err)

	budget, err := operations.Invoke(ctx, cat.Budgets.Set, as, operations.BudgetSetIn{CategoryUID: category.Value.UID, LimitCents: 5000})
	must(budget, err)

	card, err := operations.Invoke(ctx, cat.Cards.Create, as, operations.CardCreateIn{
		Name: "Ana's secret card", LimitCents: 100000, ClosingDay: 10, DueDay: 20,
	})
	must(card, err)

	_, err = operations.Invoke(ctx, cat.Transactions.Create, as, operations.TransactionCreateIn{
		AccountUID: card.Value.AccountUID, Kind: "expense", AmountCents: 1000, Date: "2026-11-15",
	})
	must(nil, err)

	statements, err := operations.Invoke(ctx, cat.Cards.Statements, as, operations.StatementsIn{UID: card.Value.UID})
	must(statements, err)

	rule, err := operations.Invoke(ctx, cat.Recurrences.Create, as, operations.RecurrenceCreateIn{
		AccountUID: acc.Value.UID, Kind: "expense", AmountCents: 100, Description: "ana-secret-rule", Frequency: "monthly",
		Interval: 1, StartDate: "2099-01-01",
	})
	must(rule, err)

	goal, err := operations.Invoke(ctx, cat.Goals.Create, as, operations.GoalCreateIn{Name: "Ana's secret goal", TargetCents: 100000})
	must(goal, err)

	part, err := operations.Invoke(ctx, cat.Goals.Add, as, operations.ContributionAddIn{UID: goal.Value.UID, AmountCents: 100})
	must(part, err)

	mine = theirs{
		account: acc.Value.UID, account2: acc2.Value.UID, category: category.Value.UID, tag: tag.Value.UID, transaction: tx.Value.UID,
		transfer: tr.Value.UID, budget: budget.Value.UID, card: card.Value.UID, statement: statements.Value.Data[0].Ref,
		recurrence: rule.Value.UID, goal: goal.Value.UID, contribution: part.Value.UID,
	}

	// What gives A away in an answer: its uids and the descriptions only it wrote
	// (B may make an account or a tag of the same name: that is not a leak).
	markers := []string{
		mine.account, mine.account2, mine.category, mine.tag, mine.transaction, mine.transfer, mine.budget, mine.card,
		mine.recurrence, mine.goal, mine.contribution, "ana-secret-lunch", "ana-secret-rule",
	}

	cases := isolationCases(cat, mine)

	// The cases cover the whole inventory, and nothing else.
	raw, err := os.ReadFile(inventoryFile)
	if err != nil {
		t.Fatal(err)
	}

	var inv operations.Inventory
	if err := json.Unmarshal(raw, &inv); err != nil {
		t.Fatal(err)
	}

	for _, e := range inv.Operations {
		if _, ok := cases[e.Name]; !ok {
			t.Errorf("operation %s has no isolation case", e.Name)
		}
	}

	if len(cases) != len(inv.Operations) {
		t.Errorf("%d isolation cases for %d operations of resources/operations.json", len(cases), len(inv.Operations))
	}

	bs := actorOf(bob)

	for _, e := range inv.Operations {
		tc, ok := cases[e.Name]
		if !ok {
			continue
		}

		t.Run(e.Name, func(t *testing.T) {
			before := workspaceState(t, app, ana.Workspace.ID)

			got, err := tc.run(ctx, bs)

			if tc.foreign {
				if !operations.IsCode(err, operations.CodeNotFound) {
					t.Fatalf("with a row of another workspace: err = %v, want not_found", err)
				}
			} else {
				if err != nil {
					t.Fatalf("the operation failed in its own workspace: %v", err)
				}

				body, _ := json.Marshal(got)
				for _, m := range markers {
					if strings.Contains(string(body), m) {
						t.Errorf("the answer holds %q of another workspace: %.300s", m, body)
					}
				}
			}

			if after := workspaceState(t, app, ana.Workspace.ID); after != before {
				t.Errorf("the other workspace changed:\n before %s\n after  %s", before, after)
			}
		})
	}
}
