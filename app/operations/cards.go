package operations

import (
	"context"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// StatementOut is one statement of a card. Its ref is its closing date, the
// identity of the statement; every amount is in cents.
type StatementOut struct {
	Ref            string `json:"ref"             example:"2026-11-10"`
	ClosingOn      string `json:"closing_on"      example:"2026-11-10"`
	DueOn          string `json:"due_on"          example:"2026-11-20"`
	PurchasesCents int64  `json:"purchases_cents" example:"120000"`
	CreditsCents   int64  `json:"credits_cents"   example:"0"`
	TotalCents     int64  `json:"total_cents"     example:"120000"`
	PaidCents      int64  `json:"paid_cents"      example:"0"`
	RemainingCents int64  `json:"remaining_cents" example:"120000"`
	Status         string `json:"status"          doc:"open, closed, paid or overdue" example:"open"`
}

func statementOut(s finance.StatementSummary) StatementOut {
	return StatementOut{
		Ref: s.Ref, ClosingOn: s.ClosingOn, DueOn: s.DueOn, PurchasesCents: s.PurchasesCents, CreditsCents: s.CreditsCents,
		TotalCents: s.TotalCents, PaidCents: s.PaidCents, RemainingCents: s.RemainingCents, Status: s.Status,
	}
}

// CardOut is a credit card with its limit and the statements that matter now.
// A card is also an account of kind credit_card (account_uid); its balance is
// negative while money is owed.
type CardOut struct {
	UID               string        `json:"uid"                 example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Version           int64         `json:"version"             example:"1"`
	AccountUID        string        `json:"account_uid"         example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Name              string        `json:"name"                example:"Visa"`
	Archived          bool          `json:"archived"`
	Currency          string        `json:"currency"            example:"USD"`
	LimitCents        int64         `json:"limit_cents"         example:"150000"`
	ClosingDay        int           `json:"closing_day"         example:"10"`
	DueDay            int           `json:"due_day"             example:"20"`
	PaymentAccountUID *string       `json:"payment_account_uid" doc:"The account statements are paid from by default"`
	Brand             string        `json:"brand"               example:"visa"`
	Last4             string        `json:"last4"               example:"4242"`
	BalanceCents      int64         `json:"balance_cents"       example:"-120000"`
	UsedCents         int64         `json:"used_cents"          example:"120000"`
	AvailableCents    int64         `json:"available_cents"     example:"30000"`
	OverLimit         bool          `json:"over_limit"`
	Current           StatementOut  `json:"current"             doc:"The statement open today"`
	Due               *StatementOut `json:"due"                 doc:"The oldest closed statement still unpaid, when there is one"`
	CreatedAt         string        `json:"created_at"          example:"2026-11-21T12:00:00Z"`
	UpdatedAt         string        `json:"updated_at"          example:"2026-11-21T12:00:00Z"`
}

// CardCreateIn is the body of cards.create. Only the last four digits of a
// card are ever kept: never a number, an expiry or a security code.
type CardCreateIn struct {
	Name              string `json:"name"                validate:"required,max=80" example:"Visa"`
	LimitCents        int64  `json:"limit_cents"         doc:"The credit limit, a positive whole number of cents" example:"150000"`
	ClosingDay        int    `json:"closing_day"         validate:"min=1,max=31" doc:"Day of the month the statement closes" example:"10"`
	DueDay            int    `json:"due_day"             validate:"min=1,max=31" doc:"Day of the month the statement is due" example:"20"`
	PaymentAccountUID string `json:"payment_account_uid" validate:"max=26" doc:"The bank account statements are paid from by default; leave out for none"`
	Brand             string `json:"brand"               validate:"max=20" doc:"visa, mastercard, amex, elo, hipercard, diners or other" example:"visa"`
	Last4             string `json:"last4"               validate:"max=4" doc:"The last 4 digits, nothing more" example:"4242"`
}

// CardUpdateIn is the body of cards.update; a field left out stays as it is,
// and payment_account_uid "" removes the default account.
type CardUpdateIn struct {
	UID               string  `json:"uid"                 validate:"required,max=26"`
	Name              *string `json:"name"                validate:"max=80"`
	LimitCents        *int64  `json:"limit_cents"`
	ClosingDay        *int    `json:"closing_day"         validate:"min=0,max=31"`
	DueDay            *int    `json:"due_day"             validate:"min=0,max=31"`
	PaymentAccountUID *string `json:"payment_account_uid" validate:"max=26"`
	Brand             *string `json:"brand"               validate:"max=20"`
	Last4             *string `json:"last4"               validate:"max=4"`
}

// StatementsIn is the input of statements.list: the card.
type StatementsIn struct {
	UID string `json:"uid" validate:"required,max=26" doc:"The uid of the card" example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
}

// StatementIn is the input of statements.show: a statement of a card.
type StatementIn struct {
	UID string `json:"uid" validate:"required,max=26" doc:"The uid of the card" example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Ref string `json:"ref" validate:"required,max=10" doc:"The closing date of the statement, YYYY-MM-DD" example:"2026-11-10"`
}

// StatementDetailOut is a statement with its purchases, refunds and payments,
// newest first.
type StatementDetailOut struct {
	Statement StatementOut     `json:"statement"`
	Rows      []TransactionOut `json:"rows"`
}

// StatementPayIn is the body of statements.pay.
type StatementPayIn struct {
	UID            string `json:"uid"              validate:"required,max=26" doc:"The uid of the card" example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Ref            string `json:"ref"              validate:"required,max=10" doc:"The closing date of the statement, YYYY-MM-DD" example:"2026-11-10"`
	FromAccountUID string `json:"from_account_uid" validate:"required,max=26" doc:"The bank account the payment leaves"`
	AmountCents    int64  `json:"amount_cents"     doc:"What to pay, in cents; 0 pays what is left of the statement" example:"0"`
	Date           string `json:"date"             validate:"max=10" doc:"The payment date; today when left out" example:"2026-11-20"`
}

type cardOps struct {
	List      *Op[NoInput, List[CardOut]]
	Show      *Op[UIDIn, CardOut]
	Create    *Op[CardCreateIn, CardOut]
	Update    *Op[CardUpdateIn, CardOut]
	Archive   *Op[UIDIn, CardOut]
	Unarchive *Op[UIDIn, CardOut]

	Statements   *Op[StatementsIn, List[StatementOut]]
	Statement    *Op[StatementIn, StatementDetailOut]
	PayStatement *Op[StatementPayIn, TransferOut]
}

func (c *Catalog) defineCards() {
	c.Cards = cardOps{
		List: Define(c.Registry, Operation[NoInput, List[CardOut]]{
			Name: "cards.list", Summary: "List the credit cards with their limits and statements", Scope: ScopeCardsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, _ NoInput) (List[CardOut], error) { return c.cardsList(ctx) },
		}),
		Show: Define(c.Registry, Operation[UIDIn, CardOut]{
			Name: "cards.show", Summary: "Read a credit card", Scope: ScopeCardsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (CardOut, error) { return c.cardsShow(ctx, in) },
		}),
		Create: Define(c.Registry, Operation[CardCreateIn, CardOut]{
			Name: "cards.create", Summary: "Create a credit card with its account", Scope: ScopeCardsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in CardCreateIn) (CardOut, error) { return c.cardsCreate(ctx, in) },
		}),
		Update: Define(c.Registry, Operation[CardUpdateIn, CardOut]{
			Name: "cards.update", Summary: "Change a credit card", Scope: ScopeCardsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in CardUpdateIn) (CardOut, error) { return c.cardsUpdate(ctx, in) },
		}),
		Archive: Define(c.Registry, Operation[UIDIn, CardOut]{
			Name: "cards.archive", Summary: "Archive a credit card", Scope: ScopeCardsWrite, Kind: KindDestructive, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (CardOut, error) { return c.cardsArchive(ctx, in, true) },
		}),
		Unarchive: Define(c.Registry, Operation[UIDIn, CardOut]{
			Name: "cards.unarchive", Summary: "Bring an archived credit card back", Scope: ScopeCardsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (CardOut, error) { return c.cardsArchive(ctx, in, false) },
		}),

		Statements: Define(c.Registry, Operation[StatementsIn, List[StatementOut]]{
			Name: "statements.list", Summary: "List the statements of a credit card, newest first", Scope: ScopeStatementsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in StatementsIn) (List[StatementOut], error) {
				return c.statementsList(ctx, in)
			},
		}),
		Statement: Define(c.Registry, Operation[StatementIn, StatementDetailOut]{
			Name: "statements.show", Summary: "Read a statement of a credit card with its rows", Scope: ScopeStatementsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in StatementIn) (StatementDetailOut, error) {
				return c.statementsShow(ctx, in)
			},
		}),
		PayStatement: Define(c.Registry, Operation[StatementPayIn, TransferOut]{
			Name: "statements.pay", Summary: "Pay a statement from a bank account", Scope: ScopeStatementsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in StatementPayIn) (TransferOut, error) {
				return c.statementsPay(ctx, in)
			},
		}),
	}
}

func (c *Catalog) cardByUID(ctx context.Context, uid string) (models.CreditCard, error) {
	x, err := workspaces.Scoped[models.CreditCard](ctx, c.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

func cardOut(o finance.CardOverview, l lookups) CardOut {
	out := CardOut{
		UID: o.UID, Version: o.Version, AccountUID: l.accounts[o.AccountID], Name: o.Name, Archived: o.Archived, Currency: l.currency,
		LimitCents: o.LimitCents, ClosingDay: o.ClosingDay, DueDay: o.DueDay, Brand: o.Brand, Last4: o.Last4,
		BalanceCents: o.BalanceCents, UsedCents: o.UsedCents, AvailableCents: o.AvailableCents, OverLimit: o.OverLimit,
		Current: statementOut(o.Current), CreatedAt: Stamp(o.CreatedAt), UpdatedAt: Stamp(o.UpdatedAt),
	}

	if o.PaymentAccountID.Valid {
		if uid, ok := l.accounts[o.PaymentAccountID.V]; ok {
			out.PaymentAccountUID = &uid
		}
	}

	if o.Due != nil {
		due := statementOut(*o.Due)
		out.Due = &due
	}

	return out
}

func (c *Catalog) cardView(ctx context.Context, id int64) (CardOut, error) {
	o, err := c.Fin.CardOverview(ctx, id)
	if err != nil {
		return CardOut{}, err
	}

	l, err := c.lookups(ctx)
	if err != nil {
		return CardOut{}, err
	}

	return cardOut(o, l), nil
}

func (c *Catalog) cardsList(ctx context.Context) (List[CardOut], error) {
	all, err := c.Fin.CardOverviews(ctx)
	if err != nil {
		return List[CardOut]{}, err
	}

	l, err := c.lookups(ctx)
	if err != nil {
		return List[CardOut]{}, err
	}

	data := make([]CardOut, 0, len(all))
	for _, o := range all {
		data = append(data, cardOut(o, l))
	}

	return List[CardOut]{Data: data}, nil
}

func (c *Catalog) cardsShow(ctx context.Context, in UIDIn) (CardOut, error) {
	card, err := c.cardByUID(ctx, in.UID)
	if err != nil {
		return CardOut{}, err
	}

	return c.cardView(ctx, card.ID)
}

// paymentAccountID turns the uid of the default payment account into its id;
// "" is none.
func (c *Catalog) paymentAccountID(ctx context.Context, uid string) (int64, error) {
	if uid == "" {
		return 0, nil
	}

	return c.accountID(ctx, uid)
}

func (c *Catalog) cardsCreate(ctx context.Context, in CardCreateIn) (CardOut, error) {
	pay, err := c.paymentAccountID(ctx, in.PaymentAccountUID)
	if err != nil {
		return CardOut{}, err
	}

	v, err := c.Fin.CreateCard(ctx, finance.CardInput{
		Name: in.Name, Limit: centsText(in.LimitCents), ClosingDay: in.ClosingDay, DueDay: in.DueDay,
		PaymentAccountID: pay, Brand: in.Brand, Last4: in.Last4,
	})
	if err != nil {
		return CardOut{}, err
	}

	return c.cardView(ctx, v.ID)
}

func (c *Catalog) cardsUpdate(ctx context.Context, in CardUpdateIn) (CardOut, error) {
	card, err := c.cardByUID(ctx, in.UID)
	if err != nil {
		return CardOut{}, err
	}

	cur, err := c.Fin.Card(ctx, card.ID)
	if err != nil {
		return CardOut{}, err
	}

	next := finance.CardInput{
		Name: cur.Name, Limit: centsText(cur.LimitCents), ClosingDay: cur.ClosingDay, DueDay: cur.DueDay,
		PaymentAccountID: cur.PaymentAccountID.V, Brand: cur.Brand, Last4: cur.Last4,
	}

	if in.Name != nil {
		next.Name = *in.Name
	}

	if in.LimitCents != nil {
		next.Limit = centsText(*in.LimitCents)
	}

	if in.ClosingDay != nil && *in.ClosingDay != 0 {
		next.ClosingDay = *in.ClosingDay
	}

	if in.DueDay != nil && *in.DueDay != 0 {
		next.DueDay = *in.DueDay
	}

	if in.PaymentAccountUID != nil {
		if next.PaymentAccountID, err = c.paymentAccountID(ctx, *in.PaymentAccountUID); err != nil {
			return CardOut{}, err
		}
	}

	if in.Brand != nil {
		next.Brand = *in.Brand
	}

	if in.Last4 != nil {
		next.Last4 = *in.Last4
	}

	if _, err := c.Fin.UpdateCard(ctx, card.ID, next); err != nil {
		return CardOut{}, err
	}

	return c.cardView(ctx, card.ID)
}

func (c *Catalog) cardsArchive(ctx context.Context, in UIDIn, archive bool) (CardOut, error) {
	card, err := c.cardByUID(ctx, in.UID)
	if err != nil {
		return CardOut{}, err
	}

	if archive {
		_, err = c.Fin.ArchiveCard(ctx, card.ID)
	} else {
		_, err = c.Fin.UnarchiveCard(ctx, card.ID)
	}

	if err != nil {
		return CardOut{}, err
	}

	return c.cardView(ctx, card.ID)
}

func (c *Catalog) statementsList(ctx context.Context, in StatementsIn) (List[StatementOut], error) {
	card, err := c.cardByUID(ctx, in.UID)
	if err != nil {
		return List[StatementOut]{}, err
	}

	all, err := c.Fin.Statements(ctx, card.ID)
	if err != nil {
		return List[StatementOut]{}, err
	}

	data := make([]StatementOut, 0, len(all))
	for _, s := range all {
		data = append(data, statementOut(s))
	}

	return List[StatementOut]{Data: data}, nil
}

func (c *Catalog) statementsShow(ctx context.Context, in StatementIn) (StatementDetailOut, error) {
	card, err := c.cardByUID(ctx, in.UID)
	if err != nil {
		return StatementDetailOut{}, err
	}

	d, err := c.Fin.Statement(ctx, card.ID, in.Ref)
	if err != nil {
		return StatementDetailOut{}, err
	}

	l, err := c.lookups(ctx)
	if err != nil {
		return StatementDetailOut{}, err
	}

	rows := make([]TransactionOut, 0, len(d.Rows))
	for _, r := range d.Rows {
		rows = append(rows, l.transaction(r))
	}

	return StatementDetailOut{Statement: statementOut(d.StatementSummary), Rows: rows}, nil
}

func (c *Catalog) statementsPay(ctx context.Context, in StatementPayIn) (TransferOut, error) {
	card, err := c.cardByUID(ctx, in.UID)
	if err != nil {
		return TransferOut{}, err
	}

	from, err := c.accountID(ctx, in.FromAccountUID)
	if err != nil {
		return TransferOut{}, err
	}

	pay := finance.PaymentInput{FromAccountID: from, Date: in.Date}
	if in.AmountCents != 0 {
		pay.Amount = centsText(in.AmountCents)
	}

	t, err := c.Fin.PayStatement(ctx, card.ID, in.Ref, pay)
	if err != nil {
		return TransferOut{}, err
	}

	return c.transferOut(ctx, t)
}

// transferOut is the answer for a transfer the services made.
func (c *Catalog) transferOut(ctx context.Context, t finance.Transfer) (TransferOut, error) {
	rows, err := c.Fin.Rows(ctx, []models.Transaction{t.From, t.To})
	if err != nil {
		return TransferOut{}, err
	}

	l, err := c.lookups(ctx)
	if err != nil {
		return TransferOut{}, err
	}

	return TransferOut{UID: t.UID, From: l.transaction(rows[0]), To: l.transaction(rows[1])}, nil
}
