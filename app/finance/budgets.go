package finance

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// spentCap bounds every spending sum. It is above any limit (2^53 - 1 cents)
// and low enough that spent*100 still fits an int64, so a percentage can be
// computed from a sum that was clamped. A month can only reach it by holding
// more money than the app can represent.
const spentCap int64 = 1 << 55

var errMonth = errs.New(http.StatusUnprocessableEntity, "The month must look like 2026-11.")

// ValidMonth reports whether month is a real "YYYY-MM" in the years the app
// accepts for dates (the same range as saneYear).
func ValidMonth(month string) bool {
	t, err := time.Parse("2006-01", month)

	return err == nil && len(month) == len("2006-01") && t.Year() >= minRuleYear && t.Year() <= maxRuleYear
}

// satAdd adds two non-negative amounts, clamping at spentCap.
func satAdd(a, b int64) int64 {
	if a >= spentCap-b {
		return spentCap
	}

	return a + b
}

// satMul multiplies an amount by a count, clamping at spentCap.
func satMul(amount, n int64) int64 {
	if amount <= 0 || n <= 0 {
		return 0
	}

	if amount > spentCap/n {
		return spentCap
	}

	return amount * n
}

// BudgetStatus is a budget with what the month has spent against it. A budget
// of the whole workspace has Overall set and no category. Available is the
// limit plus Carry, the part of the limit the previous month left unspent
// when Rollover is on.
type BudgetStatus struct {
	ID             int64  `json:"id"`
	UID            string `json:"uid"`
	CategoryID     int64  `json:"categoryId"`
	Category       string `json:"category"`
	Overall        bool   `json:"overall"`
	Rollover       bool   `json:"rollover"`
	LimitCents     int64  `json:"limitCents"`
	CarryCents     int64  `json:"carryCents"`
	AvailableCents int64  `json:"availableCents"`
	SpentCents     int64  `json:"spentCents"`
	Percent        int    `json:"percent"`
}

// BudgetInput is the form of a budget. CategoryID 0 is the overall budget. A
// nil Rollover leaves the setting of an existing budget alone (off for a new
// one).
type BudgetInput struct {
	CategoryID int64
	Limit      string
	Rollover   *bool
}

// SetBudget sets the monthly limit of an expense category (Pro).
func (s *Service) SetBudget(ctx context.Context, categoryID int64, limit string) error {
	if categoryID <= 0 {
		return errExpense
	}

	return s.SaveBudget(ctx, BudgetInput{CategoryID: categoryID, Limit: limit})
}

// SaveBudget creates or updates the budget of an expense category, or the
// overall budget when CategoryID is 0 (Pro). Writing twice, or two writers at
// once, leave one live budget.
func (s *Service) SaveBudget(ctx context.Context, in BudgetInput) error {
	lim, err := s.WS.Limits(ctx)
	if err != nil {
		return err
	}

	if err := lim.CheckBudgets(); err != nil {
		return err
	}

	cents, err := ParseAmount(in.Limit)
	if err != nil || cents <= 0 {
		return errAmountIn
	}

	if in.CategoryID != 0 {
		ok, err := workspaces.Scoped[models.Category](ctx, s.DB).Where("id", in.CategoryID).
			Where("kind", models.KindExpense).Exists(ctx)
		if err != nil || !ok {
			return firstErr(err, errExpense)
		}
	}

	set := orm.Set{"limit_cents": cents}
	if in.Rollover != nil {
		set["rollover"] = *in.Rollover
	}

	where := func() orm.Query[models.Budget] {
		q := workspaces.Scoped[models.Budget](ctx, s.DB)
		if in.CategoryID == 0 {
			return q.WhereNull("category_id")
		}

		return q.Where("category_id", in.CategoryID)
	}

	n, err := where().Update(ctx, set)
	if err != nil || n > 0 {
		return err
	}

	row := models.Budget{LimitCents: cents, Rollover: in.Rollover != nil && *in.Rollover}
	if in.CategoryID != 0 {
		row.CategoryID = sql.Null[int64]{V: in.CategoryID, Valid: true}
	}

	err = orm.Create(ctx, s.DB, &row)
	if s.DB.UniqueViolation(err) {
		// A concurrent writer created the budget first: this one is an edit.
		_, err = where().Update(ctx, set)
	}

	return err
}

// amountGroup is one (category, amount) group of a month's expenses.
type amountGroup struct {
	CategoryID sql.Null[int64] `db:"category_id"`
	Amount     int64           `db:"amount_cents"`
	N          int64           `db:"n"`
}

// monthSpending is what a month spent: Total counts every expense that is not
// a transfer leg (uncategorized ones too), and Tree maps a category to its own
// spending plus its subcategories'. Card purchases, installment parts and
// generated occurrences are ordinary expenses and count once; the payment of a
// statement is a transfer and counts never.
type monthSpending struct {
	Total int64
	Tree  map[int64]int64
}

// spendingOf reads the month in one grouped query. It groups by amount as well
// and multiplies in Go with clamping, because SUM would stop with an overflow
// error on a database that holds absurdly many huge amounts. Soft-deleted
// rows are out of the query; the category tree includes deleted categories,
// since a live transaction may still name one.
func (s *Service) spendingOf(ctx context.Context, month string) (monthSpending, error) {
	from, to := monthBounds(month)

	groups, err := workspaces.Scoped[models.Transaction](ctx, s.DB).
		WhereNull("transfer_uid").Where("kind", models.KindExpense).WhereBetween("occurred_on", from, to).
		GroupBy("category_id", "amount_cents").
		SelectAs("category_id", "category_id").
		SelectAs("amount_cents", "amount_cents").
		SelectExpr(orm.Count("*"), "n").
		Scan[amountGroup](ctx)
	if err != nil {
		return monthSpending{}, err
	}

	cats, err := workspaces.Scoped[models.Category](ctx, s.DB).WithTrashed().Get(ctx)
	if err != nil {
		return monthSpending{}, err
	}

	parent := make(map[int64]int64, len(cats))

	for _, c := range cats {
		if c.ParentID.Valid {
			parent[c.ID] = c.ParentID.V
		}
	}

	out := monthSpending{Tree: map[int64]int64{}}

	for _, g := range groups {
		cents := satMul(g.Amount, g.N)
		out.Total = satAdd(out.Total, cents)

		if !g.CategoryID.Valid {
			continue
		}

		out.Tree[g.CategoryID.V] = satAdd(out.Tree[g.CategoryID.V], cents)

		if p, ok := parent[g.CategoryID.V]; ok {
			out.Tree[p] = satAdd(out.Tree[p], cents)
		}
	}

	return out, nil
}

// previousMonth is the "YYYY-MM" before a valid month.
func previousMonth(month string) string {
	t, _ := time.Parse("2006-01", month)

	return timeutil.AddMonths(t, -1).Format("2006-01")
}

// statusOf turns a budget and the spending of its month into a status. The
// carry is the unspent part of the limit in the previous month, only for a
// budget that already existed then; it never goes negative and never chains
// (last month's carry is not carried again).
func statusOf(b models.Budget, name, month string, cur, prev *monthSpending) BudgetStatus {
	spendOf := func(m *monthSpending) int64 {
		if !b.CategoryID.Valid {
			return m.Total
		}

		return m.Tree[b.CategoryID.V]
	}

	st := BudgetStatus{
		ID: b.ID, UID: b.UID, CategoryID: b.CategoryID.V, Category: name, Overall: !b.CategoryID.Valid,
		Rollover: b.Rollover, LimitCents: b.LimitCents, SpentCents: spendOf(cur),
	}

	if b.Rollover && prev != nil && b.CreatedAt.UTC().Format("2006-01") < month {
		st.CarryCents = max(0, b.LimitCents-spendOf(prev))
	}

	st.AvailableCents = st.LimitCents + st.CarryCents
	st.Percent = int(st.SpentCents * 100 / st.AvailableCents)

	return st
}

// Budgets lists the budgets, the overall one first, with what month has spent
// against each. A month that is not a real "YYYY-MM" is refused.
func (s *Service) Budgets(ctx context.Context, month string) ([]BudgetStatus, error) {
	if !ValidMonth(month) {
		return nil, errMonth
	}

	budgets, err := workspaces.Scoped[models.Budget](ctx, s.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil || len(budgets) == 0 {
		return []BudgetStatus{}, err
	}

	cats, err := s.Categories(ctx)
	if err != nil {
		return nil, err
	}

	names := map[int64]string{}
	for _, c := range cats {
		names[c.ID] = c.Name
	}

	cur, err := s.spendingOf(ctx, month)
	if err != nil {
		return nil, err
	}

	var prev *monthSpending

	for _, b := range budgets {
		if b.Rollover {
			p, err := s.spendingOf(ctx, previousMonth(month))
			if err != nil {
				return nil, err
			}

			prev = &p

			break
		}
	}

	out := make([]BudgetStatus, 0, len(budgets))

	for _, b := range budgets {
		st := statusOf(b, names[b.CategoryID.V], month, &cur, prev)
		if st.Overall {
			out = append([]BudgetStatus{st}, out...)
		} else {
			out = append(out, st)
		}
	}

	return out, nil
}
