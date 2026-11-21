package finance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var (
	errStatement    = errs.New(http.StatusUnprocessableEntity, "Choose one of the statements of this card.")
	errNothingOwed  = errs.New(http.StatusUnprocessableEntity, "This statement has nothing left to pay.")
	errPayFromCard  = errs.New(http.StatusUnprocessableEntity, "Pay a statement from a bank account, not from a credit card.")
	errNoPayAccount = errs.New(http.StatusUnprocessableEntity, "Choose the account the payment comes from.")
)

// cardOfAccount finds the live card whose account is accountID.
func cardOfAccount(ctx context.Context, ex database.Executor, accountID int64) (models.CreditCard, bool, error) {
	c, err := workspaces.Scoped[models.CreditCard](ctx, ex).Where("account_id", accountID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return c, false, nil
	}

	return c, err == nil, err
}

// refForRow is the statement a purchase on accountID made on date belongs to:
// null unless the account is a credit card.
func refForRow(ctx context.Context, ex database.Executor, accountID int64, date string) (sql.Null[string], error) {
	c, ok, err := cardOfAccount(ctx, ex, accountID)
	if err != nil || !ok {
		return sql.Null[string]{}, err
	}

	ref := statementRefFor(date, c.ClosingDay)

	return sql.Null[string]{V: ref, Valid: ref != ""}, nil
}

// StatementSummary is one statement of a card: what was bought, what was
// paid and the status, all derived from the rows tagged with its closing date.
type StatementSummary struct {
	// Ref is the closing date, the identity of the statement.
	Ref            string `json:"ref"`
	ClosingOn      string `json:"closingOn"`
	DueOn          string `json:"dueOn"`
	PurchasesCents int64  `json:"purchasesCents"`
	CreditsCents   int64  `json:"creditsCents"`
	// TotalCents is purchases minus credits (refunds); PaidCents the payments.
	TotalCents     int64  `json:"totalCents"`
	PaidCents      int64  `json:"paidCents"`
	RemainingCents int64  `json:"remainingCents"`
	Status         string `json:"status"`
}

type refSum struct {
	Ref   string `db:"ref"`
	Kind  string `db:"kind"`
	Cents int64  `db:"cents"`
}

// statementSums sums the live rows of the card account by statement: the
// purchases and refunds (not transfers) and the payments (income legs of a
// transfer).
func statementSums(ctx context.Context, ex database.Executor, c models.CreditCard) (map[string]*StatementSummary, error) {
	out := map[string]*StatementSummary{}

	get := func(ref string) *StatementSummary {
		if out[ref] == nil {
			out[ref] = &StatementSummary{Ref: ref}
		}

		return out[ref]
	}

	for _, transfer := range []bool{false, true} {
		q := workspaces.Scoped[models.Transaction](ctx, ex).Where("account_id", c.AccountID).WhereNotNull("statement_ref")
		if transfer {
			q = q.WhereNotNull("transfer_uid").Where("kind", models.KindIncome)
		} else {
			q = q.WhereNull("transfer_uid")
		}

		sums, err := q.GroupBy("statement_ref", "kind").
			SelectAs("statement_ref", "ref").SelectAs("kind", "kind").
			SelectExpr(orm.Sum("amount_cents"), "cents").Scan[refSum](ctx)
		if err != nil {
			return nil, err
		}

		for _, r := range sums {
			st := get(r.Ref)

			switch {
			case transfer:
				st.PaidCents += r.Cents
			case r.Kind == models.KindExpense:
				st.PurchasesCents += r.Cents
			default:
				st.CreditsCents += r.Cents
			}
		}
	}

	return out, nil
}

// finish fills the derived fields of st.
func (st *StatementSummary) finish(c models.CreditCard, today string) {
	closing, err := time.Parse(isoDate, st.Ref)
	if err != nil {
		return
	}

	st.ClosingOn = st.Ref
	st.DueOn = dueDate(closing, c.DueDay).Format(isoDate)
	st.TotalCents = st.PurchasesCents - st.CreditsCents
	st.RemainingCents = max(0, st.TotalCents-st.PaidCents)
	st.Status = statementStatus(today, st.ClosingOn, st.DueOn, st.TotalCents, st.PaidCents)
}

// cardStatements lists the statements of a card, newest closing first: every
// one that has rows plus the one open today.
func cardStatements(ctx context.Context, ex database.Executor, c models.CreditCard, today string) ([]StatementSummary, error) {
	sums, err := statementSums(ctx, ex, c)
	if err != nil {
		return nil, err
	}

	if cur := statementRefFor(today, c.ClosingDay); cur != "" && sums[cur] == nil {
		sums[cur] = &StatementSummary{Ref: cur}
	}

	out := make([]StatementSummary, 0, len(sums))

	for _, st := range sums {
		st.finish(c, today)
		out = append(out, *st)
	}

	slices.SortFunc(out, func(a, b StatementSummary) int { return strings.Compare(b.Ref, a.Ref) })

	return out, nil
}

// Statements lists the statements of a card, newest first.
func (s *Service) Statements(ctx context.Context, cardID int64) ([]StatementSummary, error) {
	c, err := loadCard(ctx, s.DB, cardID)
	if err != nil {
		return nil, err
	}

	return cardStatements(ctx, s.DB, c, s.Today(ctx))
}

// StatementDetail is a statement with its rows (purchases, refunds and
// payments), newest first.
type StatementDetail struct {
	StatementSummary

	Rows []TxRow `json:"rows"`
}

// Statement reads one statement of a card: a ref that is not one of the
// statements of the card is a 404.
func (s *Service) Statement(ctx context.Context, cardID int64, ref string) (StatementDetail, error) {
	c, err := loadCard(ctx, s.DB, cardID)
	if err != nil {
		return StatementDetail{}, err
	}

	list, err := cardStatements(ctx, s.DB, c, s.Today(ctx))
	if err != nil {
		return StatementDetail{}, err
	}

	i := slices.IndexFunc(list, func(st StatementSummary) bool { return st.Ref == ref })
	if i < 0 {
		return StatementDetail{}, errs.ErrNotFound
	}

	txs, err := workspaces.Scoped[models.Transaction](ctx, s.DB).Where("account_id", c.AccountID).
		Where("statement_ref", ref).OrderBy("occurred_on", orm.Desc).OrderBy("id", orm.Desc).Get(ctx)
	if err != nil {
		return StatementDetail{}, err
	}

	rows, err := s.rowsOf(ctx, txs)
	if err != nil {
		return StatementDetail{}, err
	}

	return StatementDetail{StatementSummary: list[i], Rows: rows}, nil
}

// CardOverview is what a card tile shows.
type CardOverview struct {
	CardView

	// BalanceCents is negative while money is owed. UsedCents is the debt
	// over all dates (so purchases dated ahead count as committed limit) and
	// AvailableCents what is left of the limit; both are plain amounts.
	BalanceCents   int64 `json:"balanceCents"`
	UsedCents      int64 `json:"usedCents"`
	AvailableCents int64 `json:"availableCents"`
	OverLimit      bool  `json:"overLimit"`
	// Current is the statement open today; Due the oldest closed statement
	// still unpaid, when there is one.
	Current StatementSummary  `json:"current"`
	Due     *StatementSummary `json:"due"`
}

// cardBalance is the balance of the card account over all dates.
func cardBalance(ctx context.Context, ex database.Executor, accountID int64) (int64, error) {
	var balance int64

	for kind, sign := range map[string]int64{models.KindIncome: 1, models.KindExpense: -1} {
		sum, err := workspaces.Scoped[models.Transaction](ctx, ex).Where("account_id", accountID).
			Where("kind", kind).Sum[int64](ctx, "amount_cents")
		if err != nil {
			return 0, err
		}

		var err2 error
		if balance, err2 = addCents(balance, sign*sum); err2 != nil {
			return 0, err2
		}
	}

	return balance, nil
}

// CardOverviews returns one overview per card of the workspace.
func (s *Service) CardOverviews(ctx context.Context) ([]CardOverview, error) {
	cards, err := s.Cards(ctx)
	if err != nil {
		return nil, err
	}

	today := s.Today(ctx)
	out := make([]CardOverview, 0, len(cards))

	for _, cv := range cards {
		o, err := overviewOf(ctx, s.DB, cv, today)
		if err != nil {
			return nil, err
		}

		out = append(out, o)
	}

	return out, nil
}

// CardOverview is the overview of one card.
func (s *Service) CardOverview(ctx context.Context, id int64) (CardOverview, error) {
	cv, err := s.Card(ctx, id)
	if err != nil {
		return CardOverview{}, err
	}

	return overviewOf(ctx, s.DB, cv, s.Today(ctx))
}

func overviewOf(ctx context.Context, ex database.Executor, cv CardView, today string) (CardOverview, error) {
	list, err := cardStatements(ctx, ex, cv.CreditCard, today)
	if err != nil {
		return CardOverview{}, err
	}

	balance, err := cardBalance(ctx, ex, cv.AccountID)
	if err != nil {
		return CardOverview{}, err
	}

	o := CardOverview{CardView: cv, BalanceCents: balance, UsedCents: max(0, -balance)}
	o.AvailableCents = cv.LimitCents - o.UsedCents
	o.OverLimit = o.UsedCents > cv.LimitCents

	cur := statementRefFor(today, cv.ClosingDay)

	for i := len(list) - 1; i >= 0; i-- {
		st := list[i]

		switch {
		case st.Ref == cur:
			o.Current = st
		case o.Due == nil && (st.Status == StatementClosed || st.Status == StatementOverdue):
			o.Due = &st
		}
	}

	return o, nil
}

// PaymentInput is a statement payment as the form sends it. An empty Amount
// pays what is left of the statement; an empty Date means today.
type PaymentInput struct {
	FromAccountID int64
	Amount        string
	Date          string
}

// PayStatement pays a statement of the card from a bank account. The payment
// is a transfer: it moves money from the account to the card and is neither
// income nor expense, so the purchases stay counted once, on their dates.
// The card-side leg carries the statement. A partial or extra payment is
// allowed; an overpayment shows as credit on the card.
func (s *Service) PayStatement(ctx context.Context, cardID int64, ref string, in PaymentInput) (Transfer, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return Transfer{}, err
	}

	today := s.Today(ctx)

	var out Transfer

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		c, err := loadCard(ctx, tx, cardID)
		if err != nil {
			return err
		}

		list, err := cardStatements(ctx, tx, c, today)
		if err != nil {
			return err
		}

		i := slices.IndexFunc(list, func(st StatementSummary) bool { return st.Ref == ref })
		if i < 0 {
			return errStatement
		}

		if in.FromAccountID <= 0 {
			return errNoPayAccount
		}

		from, err := workspaces.Scoped[models.Account](ctx, tx).Where("id", in.FromAccountID).First(ctx)
		if err != nil {
			return errs.ErrNotFound
		}

		if from.Kind == models.AccountCreditCard {
			return errPayFromCard
		}

		amount := strings.TrimSpace(in.Amount)
		if amount == "" {
			if list[i].RemainingCents <= 0 {
				return errNothingOwed
			}

			amount = plainAmount(list[i].RemainingCents)
		}

		date := in.Date
		if date == "" {
			date = today
		}

		out, err = insertTransfer(ctx, tx, TransferInput{
			FromAccountID: from.ID, ToAccountID: c.AccountID, Amount: amount, Date: date,
			Description: "Credit card payment " + ref,
		}, ref)

		return err
	})

	return out, err
}

// recomputeStatements moves the rows of the statements of c that are not paid
// to the statements the closing day newDay gives them, inside the transaction
// of the change. A paid statement keeps its rows, so what the person already
// settled never moves. Trashed rows move too, so a restored purchase is on the
// right statement. Purchases follow their date; a payment moves to the
// statement that closes in the same month of the new closing day.
func recomputeStatements(ctx context.Context, tx *database.Tx, c models.CreditCard, newDay int, today string) error {
	list, err := cardStatements(ctx, tx, c, today)
	if err != nil {
		return err
	}

	var open []any

	for _, st := range list {
		if st.Status != StatementPaid {
			open = append(open, st.Ref)
		}
	}

	if len(open) == 0 {
		return nil
	}

	rows := func() orm.Query[models.Transaction] {
		return workspaces.Scoped[models.Transaction](ctx, tx).WithTrashed().Where("account_id", c.AccountID).
			WhereIn("statement_ref", open...)
	}

	dates, err := rows().WhereNull("transfer_uid").Pluck[string](ctx, "occurred_on")
	if err != nil {
		return err
	}

	slices.Sort(dates)

	for _, d := range slices.Compact(dates) {
		if _, err := rows().WhereNull("transfer_uid").Where("occurred_on", d).
			Update(ctx, orm.Set{"statement_ref": statementRefFor(d, newDay)}); err != nil {
			return err
		}
	}

	// By row, not by ref: a moved payment may land on a ref another payment
	// is about to leave.
	payments, err := rows().WhereNotNull("transfer_uid").Get(ctx)
	if err != nil {
		return err
	}

	for _, p := range payments {
		t, err := time.Parse(isoDate, p.StatementRef.V)
		if err != nil {
			continue
		}

		next := closingDate(t.Year(), t.Month(), newDay).Format(isoDate)

		if _, err := workspaces.Scoped[models.Transaction](ctx, tx).WithTrashed().Where("id", p.ID).
			Update(ctx, orm.Set{"statement_ref": next}); err != nil {
			return err
		}
	}

	return nil
}

// plainAmount renders cents as ParseAmount reads them: "1234.50".
func plainAmount(cents int64) string {
	return strconv.FormatInt(cents/100, 10) + "." + fmt.Sprintf("%02d", cents%100)
}
