package finance

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// CapCents is where a sum stops growing: 2^55 cents, far above any real total
// (the largest amount is 2^53 - 1) and low enough that adding two capped
// values never overflows an int64.
const CapCents int64 = 1 << 55

// ClampCents narrows an exact sum to +-CapCents.
func ClampCents(v *big.Int) int64 {
	switch {
	case v.Cmp(big.NewInt(CapCents)) > 0:
		return CapCents
	case v.Cmp(big.NewInt(-CapCents)) < 0:
		return -CapCents
	}

	return v.Int64()
}

// Dim is a dimension of the summary a reader keeps; the kind and the transfer
// flag are always kept apart.
type Dim int

// The dimensions a MonthTotals read can keep apart.
const (
	ByMonth Dim = 1 << iota
	ByAccount
	ByCategory
)

// FirstSummaryDate is before every date the app accepts.
const FirstSummaryDate = "0001-01-01"

// monthSpan is the first and the last real day of a valid "YYYY-MM".
func monthSpan(month string) (from, to string) {
	t, _ := time.Parse("2006-01", month)

	return month + "-01", fmt.Sprintf("%s-%02d", month, timeutil.LastDayOfMonth(t.Year(), t.Month()))
}

// MonthTotals sums the live transactions of the workspace of ctx dated from
// to to (ISO dates, both included), keeping apart the dimensions of by (the
// others come back zero) and always the kind and the transfer flag. Whole
// months are read from the summary the database triggers keep, the days of a
// month the range only touches from the transactions themselves (never more
// than two months of rows), so the cost does not grow with the history. The
// result may hold the same key twice (a summary row and an edge row): add
// them. Sums are exact: use Sum on a total, never ClampCents on a part.
func (s *Service) MonthTotals(ctx context.Context, from, to string, by Dim) ([]models.TransactionMonthTotal, error) {
	id, ok := workspaces.TenantID(ctx)
	if !ok || from > to {
		return nil, nil
	}

	whole, edges, err := splitRange(from, to)
	if err != nil {
		return nil, err
	}

	var out []models.TransactionMonthTotal

	if whole != nil {
		rows, err := readSummary(ctx, s.DB, id, whole[0], whole[1], by)
		if err != nil {
			return nil, err
		}

		out = rows
	}

	for _, e := range edges {
		rows, err := readEdge(ctx, s.DB, id, e[0], e[1], by)
		if err != nil {
			return nil, err
		}

		out = append(out, rows...)
	}

	return out, nil
}

// splitRange cuts from..to into the whole months the summary answers
// ("YYYY-MM" bounds, or nil) and the date ranges it does not.
func splitRange(from, to string) (whole []string, edges [][2]string, err error) {
	a, err := time.Parse(isoDate, from)
	if err != nil {
		return nil, nil, err
	}

	b, err := time.Parse(isoDate, to)
	if err != nil {
		return nil, nil, err
	}

	first, last := a, b

	if a.Day() != 1 {
		first = timeutil.AddMonths(time.Date(a.Year(), a.Month(), 1, 0, 0, 0, 0, time.UTC), 1)
	}

	if b.Day() != timeutil.LastDayOfMonth(b.Year(), b.Month()) {
		last = timeutil.AddMonths(time.Date(b.Year(), b.Month(), 1, 0, 0, 0, 0, time.UTC), -1)
	}

	if first.After(last) {
		return nil, [][2]string{{from, to}}, nil
	}

	whole = []string{first.Format("2006-01"), last.Format("2006-01")}

	if a.Before(first) {
		edges = append(edges, [2]string{from, MonthOf(from) + "-31"})
	}

	if b.After(last) {
		edges = append(edges, [2]string{timeutil.AddMonths(last, 1).Format("2006-01") + "-01", to})
	}

	return whole, edges, nil
}

// dimList renders the select list and the group by of a read; expr says how
// each dimension is written in the table behind it.
func dimList(by Dim, kind, transfer string, expr map[Dim]string) (list, group string) {
	list = kind + ", " + transfer + " AS transfer"
	cols := []string{"kind", "transfer"}

	for _, d := range []struct {
		dim   Dim
		alias string
		zero  string
	}{{ByMonth, "month", "''"}, {ByAccount, "account_id", "0"}, {ByCategory, "category_id", "0"}} {
		if by&d.dim != 0 {
			list = expr[d.dim] + " AS " + d.alias + ", " + list
			cols = append(cols, d.alias)
		} else {
			list = d.zero + " AS " + d.alias + ", " + list
		}
	}

	return list, strings.Join(cols, ", ")
}

func readSummary(ctx context.Context, db database.Executor, ws int64, from, to string, by Dim) ([]models.TransactionMonthTotal, error) {
	list, group := dimList(by, "kind", "transfer", map[Dim]string{ByMonth: "month", ByAccount: "account_id", ByCategory: "category_id"})

	return orm.Raw[models.TransactionMonthTotal](ctx, db,
		"SELECT "+list+", SUM(n) AS n, SUM(amount_hi) AS amount_hi, SUM(amount_lo) AS amount_lo "+
			"FROM transaction_month_totals WHERE workspace_id = ? AND month BETWEEN ? AND ? GROUP BY "+group,
		ws, from, to)
}

func readEdge(ctx context.Context, db database.Executor, ws int64, from, to string, by Dim) ([]models.TransactionMonthTotal, error) {
	list, group := dimList(by, "kind", "(transfer_uid IS NOT NULL)", map[Dim]string{
		ByMonth: "substr(occurred_on, 1, 7)", ByAccount: "account_id", ByCategory: "COALESCE(category_id, 0)",
	})

	return orm.Raw[models.TransactionMonthTotal](ctx, db,
		"SELECT "+list+", COUNT(*) AS n, SUM(amount_cents >> 32) AS amount_hi, "+
			"SUM(amount_cents - ((amount_cents >> 32) << 32)) AS amount_lo "+
			"FROM transactions WHERE workspace_id = ? AND deleted_at IS NULL AND occurred_on BETWEEN ? AND ? GROUP BY "+group,
		ws, from, to)
}

// CheckMonthTotals recomputes the summary of every workspace from the
// transactions, in exact integers, and returns an error naming the first
// bucket that differs, or one the summary should not hold. It is the oracle
// of the tests and a tool for an operator who suspects the triggers.
func CheckMonthTotals(ctx context.Context, db database.Executor) error {
	type key struct {
		ws, account, category int64
		month, kind           string
		transfer              int64
	}

	type sum struct {
		n   int64
		sum *big.Int
	}

	want := map[key]*sum{}

	txs, err := orm.Raw[struct {
		Workspace int64  `db:"workspace_id"`
		Account   int64  `db:"account_id"`
		Category  int64  `db:"category_id"`
		Month     string `db:"month"`
		Kind      string `db:"kind"`
		Transfer  int64  `db:"transfer"`
		Cents     int64  `db:"amount_cents"`
	}](ctx, db, "SELECT workspace_id, account_id, COALESCE(category_id, 0) AS category_id, substr(occurred_on, 1, 7) AS month, "+
		"kind, (transfer_uid IS NOT NULL) AS transfer, amount_cents FROM transactions WHERE deleted_at IS NULL")
	if err != nil {
		return err
	}

	for _, t := range txs {
		k := key{t.Workspace, t.Account, t.Category, t.Month, t.Kind, t.Transfer}
		if want[k] == nil {
			want[k] = &sum{sum: new(big.Int)}
		}

		want[k].n++
		want[k].sum.Add(want[k].sum, big.NewInt(t.Cents))
	}

	rows, err := orm.Raw[models.TransactionMonthTotal](ctx, db, "SELECT * FROM transaction_month_totals")
	if err != nil {
		return err
	}

	for _, r := range rows {
		k := key{r.WorkspaceID, r.AccountID, r.CategoryID, r.Month, r.Kind, r.Transfer}
		w := want[k]

		switch {
		case w == nil:
			return fmt.Errorf("finance: the summary holds %+v but no live transaction matches", k)
		case w.n != r.N || w.sum.Cmp(r.Sum()) != 0:
			return fmt.Errorf("finance: summary %+v is %d rows, %v cents; the transactions say %d, %v", k, r.N, r.Sum(), w.n, w.sum)
		}

		delete(want, k)
	}

	if len(want) > 0 {
		keys := make([]string, 0, len(want))
		for k := range want {
			keys = append(keys, fmt.Sprintf("%+v", k))
		}

		slices.Sort(keys)

		return fmt.Errorf("finance: the summary misses %d buckets, first %s", len(keys), keys[0])
	}

	return nil
}
