package reports

import (
	"context"
	"database/sql"
	"math/big"
	"slices"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// capCents is where sums stop growing: 2^55 cents, far above any real total
// (the largest amount is 2^53 - 1) and low enough that adding two capped
// values never overflows an int64. SUM would stop with an error instead.
const capCents int64 = 1 << 55

// Service computes the reports of the workspace in the context.
type Service struct {
	DB  *database.DB
	WS  *workspaces.Service
	Fin *finance.Service
}

// New builds the service on the finance service.
func New(fin *finance.Service) *Service { return &Service{DB: fin.DB, WS: fin.WS, Fin: fin} }

// Period resolves a preset (or from and to) against today in the timezone of
// the workspace of ctx.
func (s *Service) Period(ctx context.Context, preset, from, to string) (Period, error) {
	return NewPeriod(preset, from, to, s.Fin.Today(ctx))
}

// add adds b to a, clamping at +-capCents.
func add(a, b int64) int64 { return max(-capCents, min(capCents, a+b)) }

// mul multiplies a positive amount by a count, clamping at capCents.
func mul(amount, n int64) int64 {
	if amount <= 0 || n <= 0 {
		return 0
	}

	if amount > capCents/n {
		return capCents
	}

	return amount * n
}

// group is one (key, kind, amount) group of transactions with its row count.
type group struct {
	Date       string          `db:"occurred_on"`
	AccountID  int64           `db:"account_id"`
	CategoryID sql.Null[int64] `db:"category_id"`
	TagID      int64           `db:"tag_id"`
	Kind       string          `db:"kind"`
	Amount     int64           `db:"amount_cents"`
	N          int64           `db:"n"`
}

// Sum is income, expense and their difference.
type Sum struct {
	IncomeCents  int64 `json:"incomeCents"`
	ExpenseCents int64 `json:"expenseCents"`
	NetCents     int64 `json:"netCents"`
}

// put adds cents to the income or expense side.
func (m *Sum) put(kind string, cents int64) {
	if kind == models.KindIncome {
		m.IncomeCents = add(m.IncomeCents, cents)
	} else {
		m.ExpenseCents = add(m.ExpenseCents, cents)
	}

	m.NetCents = add(m.IncomeCents, -m.ExpenseCents)
}

// bucket holds the exact income and expense of a set of summary rows.
type bucket struct{ income, expense big.Int }

func (b *bucket) add(r models.TransactionMonthTotal) {
	if r.Kind == models.KindIncome {
		b.income.Add(&b.income, r.Sum())
	} else {
		b.expense.Add(&b.expense, r.Sum())
	}
}

// sum is the bucket as a Sum, clamped at capCents.
func (b *bucket) sum() Sum {
	in, out := finance.ClampCents(&b.income), finance.ClampCents(&b.expense)

	return Sum{IncomeCents: in, ExpenseCents: out, NetCents: add(in, -out)}
}

// ordinaryTotals reads the live rows of the period from the monthly summary,
// never a transfer leg. Future-dated rows count only when the period reaches
// their date.
func (s *Service) ordinaryTotals(ctx context.Context, p Period, by finance.Dim) ([]models.TransactionMonthTotal, error) {
	rows, err := s.Fin.MonthTotals(ctx, p.From, p.To, by)
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(rows, func(r models.TransactionMonthTotal) bool { return r.Transfer != 0 }), nil
}

// ordinary is the base query of the income and expense reports: live rows of
// the workspace in the period, never a transfer leg. Future-dated rows count
// only when the period reaches their date.
func (s *Service) ordinary(ctx context.Context, p Period) orm.Query[models.Transaction] {
	return workspaces.Scoped[models.Transaction](ctx, s.DB).
		WhereNull("transfer_uid").WhereBetween("occurred_on", p.From, p.To)
}

// bps is part over whole in basis points (1/100 of a percent), rounded half
// up, with integer arithmetic only; zero when whole is not positive.
func bps(part, whole int64) int64 {
	if whole <= 0 || part <= 0 {
		return 0
	}

	return ratio(part, whole)
}

// ratio is num * 10000 / den rounded half up, for num >= 0 and den > 0,
// clamped to the int64 range.
func ratio(num, den int64) int64 {
	n := new(big.Int).Mul(big.NewInt(num), big.NewInt(10000))
	d := big.NewInt(den)
	n.Mul(n, big.NewInt(2))
	n.Add(n, d)
	n.Quo(n, d.Mul(d, big.NewInt(2)))

	if !n.IsInt64() {
		return 1<<63 - 1
	}

	return n.Int64()
}
