package seeders

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// FinanceDemoSeeder creates a USD demo for the local administrator once.
// Any existing membership is left intact; reruns preserve all user edits.
type FinanceDemoSeeder struct{}

// SeedApp implements vukem.AppSeeder. Run FinanceAdminSeeder first.
func (FinanceDemoSeeder) SeedApp(ctx context.Context, app *vukem.App) error {
	if err := localSeedOnly(app); err != nil {
		return err
	}
	db, err := app.DB()
	if err != nil {
		return err
	}
	u, err := orm.QueryOf[models.User](db).Where("email", financeAdminEmail(app)).First(ctx)
	if err != nil {
		return err
	}
	return db.Tx(ctx, func(tx *database.Tx) error {
		exists, err := orm.QueryOf[models.Membership](tx).Where("user_id", u.ID).Exists(ctx)
		if err != nil || exists {
			return err
		}

		return seedDemoWorkspace(ctx, tx, u.ID)
	})
}

type demoFixture struct {
	Name, Kind string
	Amount     int64
	Note       string
	Limit      int64
}

func demoFixtures() []demoFixture {
	return []demoFixture{
		{"Salary", models.KindIncome, 480000, "Monthly salary", 0},
		{"Housing", models.KindExpense, 145000, "Rent", 160000},
		{"Groceries", models.KindExpense, 18940, "Groceries", 60000},
		{"Transport", models.KindExpense, 6500, "Transit pass", 15000},
	}
}

func seedDemoWorkspace(ctx context.Context, tx *database.Tx, userID int64) error {
	ws := models.Workspace{Name: "My finances", Currency: "USD", Timezone: workspaces.DefaultTimezone}
	if err := orm.Create(ctx, tx, &ws); err != nil {
		return err
	}

	member := models.Membership{WorkspaceID: ws.ID, UserID: userID, Role: models.RoleOwner}
	if err := orm.Create(ctx, tx, &member); err != nil {
		return err
	}

	tenant := workspaces.WithTenant(ctx, ws.ID)
	account := models.Account{
		WorkspaceID: ws.ID, Name: "Checking", Kind: models.AccountChecking, OpeningCents: 125000, Position: 1,
	}
	if err := orm.Create(tenant, tx, &account); err != nil {
		return err
	}

	savings := models.Account{
		WorkspaceID: ws.ID, Name: "Savings", Kind: models.AccountSavings, OpeningCents: 300000, Position: 2,
	}
	if err := orm.Create(tenant, tx, &savings); err != nil {
		return err
	}

	ids, err := seedDemoCategories(tenant, tx, ws.ID)
	if err != nil {
		return err
	}

	if err := seedDemoMonths(tenant, tx, ws.ID, account.ID, ids); err != nil {
		return err
	}

	if err := seedDemoTransfer(tenant, tx, ws.ID, account.ID, savings.ID); err != nil {
		return err
	}

	return seedDemoCard(tenant, tx, ws.ID, account.ID)
}

// seedDemoCard adds a credit card (closing day 10, due day 20) with two
// statements: October, paid from checking, and November, closed and unpaid.
// The purchases keep the statement they belong to, the payment is a transfer
// whose card leg carries the October statement.
func seedDemoCard(ctx context.Context, tx *database.Tx, workspaceID, payFrom int64) error {
	account := models.Account{
		WorkspaceID: workspaceID, Name: "Demo Visa", Kind: models.AccountCreditCard, Position: 3,
	}
	if err := orm.Create(ctx, tx, &account); err != nil {
		return err
	}

	card := models.CreditCard{
		WorkspaceID: workspaceID, AccountID: account.ID, LimitCents: 200000, ClosingDay: 10, DueDay: 20,
		PaymentAccountID: sql.Null[int64]{V: payFrom, Valid: true}, Brand: "visa", Last4: "4242",
	}
	if err := orm.Create(ctx, tx, &card); err != nil {
		return err
	}

	ref := func(v string) sql.Null[string] { return sql.Null[string]{V: v, Valid: true} }

	for _, p := range []struct {
		note, date, statement string
		cents                 int64
	}{
		{"Online store", "2026-10-05", "2026-10-10", 8990},
		{"Restaurant", "2026-10-08", "2026-10-10", 4210},
		{"Pharmacy", "2026-10-20", "2026-11-10", 3550},
		{"Gas station", "2026-11-04", "2026-11-10", 6000},
	} {
		t := models.Transaction{
			WorkspaceID: workspaceID, AccountID: account.ID, Kind: models.KindExpense, AmountCents: p.cents,
			Description: p.note, OccurredOn: p.date, StatementRef: ref(p.statement),
		}
		if err := orm.Create(ctx, tx, &t); err != nil {
			return err
		}
	}

	uid := sql.Null[string]{V: models.NewUID(), Valid: true}

	for _, leg := range []models.Transaction{
		{AccountID: payFrom, Kind: models.KindExpense},
		{AccountID: account.ID, Kind: models.KindIncome, StatementRef: ref("2026-10-10")},
	} {
		leg.WorkspaceID = workspaceID
		leg.AmountCents = 13200
		leg.Description = "Credit card payment 2026-10-10"
		leg.OccurredOn = "2026-10-18"
		leg.TransferUID = uid

		if err := orm.Create(ctx, tx, &leg); err != nil {
			return err
		}
	}

	return nil
}

// seedDemoTransfer moves 500.00 from checking to savings: two legs sharing a
// transfer uid, so the demo shows a transfer that is neither income nor expense.
func seedDemoTransfer(ctx context.Context, tx *database.Tx, workspaceID, from, to int64) error {
	uid := sql.Null[string]{V: models.NewUID(), Valid: true}

	for _, leg := range []models.Transaction{
		{AccountID: from, Kind: models.KindExpense, Description: "Move to savings"},
		{AccountID: to, Kind: models.KindIncome, Description: "Move to savings"},
	} {
		leg.WorkspaceID = workspaceID
		leg.AmountCents = 50000
		leg.OccurredOn = "2026-11-02"
		leg.TransferUID = uid

		if err := orm.Create(ctx, tx, &leg); err != nil {
			return err
		}
	}

	return nil
}

func seedDemoCategories(ctx context.Context, tx *database.Tx, workspaceID int64) ([]int64, error) {
	items := demoFixtures()
	ids := make([]int64, len(items))

	for i, item := range items {
		category := models.Category{WorkspaceID: workspaceID, Name: item.Name, Kind: item.Kind}
		if err := orm.Create(ctx, tx, &category); err != nil {
			return nil, err
		}

		ids[i] = category.ID
		if item.Limit == 0 {
			continue
		}

		budget := models.Budget{WorkspaceID: workspaceID, CategoryID: category.ID, LimitCents: item.Limit}
		if err := orm.Create(ctx, tx, &budget); err != nil {
			return nil, err
		}
	}

	return ids, nil
}

// June through November 2026: six months for the dashboard chart.
func seedDemoMonths(ctx context.Context, tx *database.Tx, workspaceID, accountID int64, ids []int64) error {
	items := demoFixtures()

	for month := 6; month <= 11; month++ {
		for i, item := range items {
			transaction := models.Transaction{
				WorkspaceID: workspaceID,
				AccountID:   accountID,
				CategoryID:  sql.Null[int64]{V: ids[i], Valid: true},
				Kind:        item.Kind,
				AmountCents: item.Amount,
				Description: item.Note,
				OccurredOn:  fmt.Sprintf("2026-%02d-%02d", month, 1+i),
			}
			if err := orm.Create(ctx, tx, &transaction); err != nil {
				return err
			}
		}
	}

	return nil
}
