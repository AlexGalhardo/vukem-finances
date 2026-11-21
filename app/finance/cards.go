package finance

import (
	"context"
	"database/sql"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// CardBrands are the brands a card may carry; empty means none.
var CardBrands = []string{"visa", "mastercard", "amex", "elo", "hipercard", "diners", "other"} //nolint:gochecknoglobals // constant list

var (
	errCardName     = errs.New(http.StatusUnprocessableEntity, "Give the card a name of up to 80 characters.")
	errCardLimit    = errs.New(http.StatusUnprocessableEntity, "The limit must be a positive number like 1500.00.")
	errCardDay      = errs.New(http.StatusUnprocessableEntity, "The closing day and the due day must be between 1 and 31.")
	errCardLast4    = errs.New(http.StatusUnprocessableEntity, "Enter only the last 4 digits of the card.")
	errCardBrand    = errs.New(http.StatusUnprocessableEntity, "Choose one of the listed brands.")
	errCardPayment  = errs.New(http.StatusUnprocessableEntity, "Choose a bank account that is not a credit card to pay from.")
	errCardArchived = errs.New(http.StatusUnprocessableEntity, "An archived account cannot be the account a card is paid from.")
)

// CardInput is a card as the form sends it. Limit is an amount like "1500.00".
type CardInput struct {
	Name       string
	Limit      string
	ClosingDay int
	DueDay     int
	// PaymentAccountID is the account statements are paid from by default; 0 means none.
	PaymentAccountID int64
	Brand            string
	Last4            string
}

// CardView is a card with the name and archive state of its account.
type CardView struct {
	models.CreditCard

	Name     string `json:"name"`
	Archived bool   `json:"archived"`
}

type cardFields struct {
	name    string
	limit   int64
	closing int
	due     int
	payment sql.Null[int64]
	brand   string
	last4   string
}

// checkCard validates in. keepPayment is the payment account the card
// already has (0 for a new card): it stays valid after it is archived.
func checkCard(ctx context.Context, ex database.Executor, in CardInput, keepPayment int64) (cardFields, error) {
	var f cardFields

	f.name = strings.TrimSpace(in.Name)
	if f.name == "" || utf8.RuneCountInString(f.name) > 80 {
		return f, errCardName
	}

	cents, err := ParseAmount(in.Limit)
	if err != nil || cents <= 0 {
		return f, errCardLimit
	}

	f.limit = cents

	if in.ClosingDay < 1 || in.ClosingDay > 31 || in.DueDay < 1 || in.DueDay > 31 {
		return f, errCardDay
	}

	f.closing, f.due = in.ClosingDay, in.DueDay

	// Only ASCII digits, never more than four: a longer value could be a card number.
	f.last4 = strings.TrimSpace(in.Last4)
	if f.last4 != "" && (len(f.last4) != 4 || strings.Trim(f.last4, "0123456789") != "") {
		return f, errCardLast4
	}

	f.brand = strings.ToLower(strings.TrimSpace(in.Brand))
	if f.brand != "" && !slices.Contains(CardBrands, f.brand) {
		return f, errCardBrand
	}

	if in.PaymentAccountID > 0 {
		a, err := workspaces.Scoped[models.Account](ctx, ex).Where("id", in.PaymentAccountID).First(ctx)
		if err != nil || a.Kind == models.AccountCreditCard {
			return f, errCardPayment
		}

		if a.ArchivedAt.Valid && a.ID != keepPayment {
			return f, errCardArchived
		}

		f.payment = sql.Null[int64]{V: a.ID, Valid: true}
	}

	return f, nil
}

func (f cardFields) set() orm.Set {
	var payment any
	if f.payment.Valid {
		payment = f.payment.V
	}

	return orm.Set{
		"limit_cents": f.limit, "closing_day": f.closing, "due_day": f.due,
		"payment_account_id": payment, "brand": f.brand, "last4": f.last4,
	}
}

// loadCard reads a live card of the workspace; another workspace's card, or a
// deleted one, is a 404.
func loadCard(ctx context.Context, ex database.Executor, id int64) (models.CreditCard, error) {
	c, err := workspaces.Scoped[models.CreditCard](ctx, ex).Where("id", id).First(ctx)
	if err != nil {
		return c, errs.ErrNotFound
	}

	return c, nil
}

// Cards lists the live cards of the workspace in the order of their accounts,
// archived ones included.
func (s *Service) Cards(ctx context.Context) ([]CardView, error) {
	cards, err := workspaces.Scoped[models.CreditCard](ctx, s.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := workspaces.Scoped[models.Account](ctx, s.DB).Where("kind", models.AccountCreditCard).Get(ctx)
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]models.Account, len(accounts))
	for _, a := range accounts {
		byID[a.ID] = a
	}

	out := make([]CardView, 0, len(cards))

	for _, c := range cards {
		a := byID[c.AccountID]
		out = append(out, CardView{CreditCard: c, Name: a.Name, Archived: a.ArchivedAt.Valid})
	}

	slices.SortStableFunc(out, func(a, b CardView) int { return byID[a.AccountID].Position - byID[b.AccountID].Position })

	return out, nil
}

// Card reads one card with its account's name.
func (s *Service) Card(ctx context.Context, id int64) (CardView, error) {
	c, err := loadCard(ctx, s.DB, id)
	if err != nil {
		return CardView{}, err
	}

	a, err := workspaces.Scoped[models.Account](ctx, s.DB).Where("id", c.AccountID).First(ctx)
	if err != nil {
		return CardView{}, err
	}

	return CardView{CreditCard: c, Name: a.Name, Archived: a.ArchivedAt.Valid}, nil
}

// CreateCard adds a card with its account (kind credit_card, balance zero).
// The account weighs on the plan's account quota like any other.
func (s *Service) CreateCard(ctx context.Context, in CardInput) (CardView, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return CardView{}, err
	}

	lim, err := s.WS.Limits(ctx)
	if err != nil {
		return CardView{}, err
	}

	var out models.CreditCard

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		f, err := checkCard(ctx, tx, in, 0)
		if err != nil {
			return err
		}

		a, err := insertAccount(ctx, tx, lim, "", f.name, models.AccountCreditCard, 0)
		if err != nil {
			return err
		}

		out = models.CreditCard{
			AccountID: a.ID, LimitCents: f.limit, ClosingDay: f.closing, DueDay: f.due,
			PaymentAccountID: f.payment, Brand: f.brand, Last4: f.last4,
		}

		return orm.Create(ctx, tx, &out)
	})
	if err != nil {
		return CardView{}, err
	}

	return s.Card(ctx, out.ID)
}

// UpdateCard changes a card and the name of its account. A new closing day
// moves the purchases of the statements that are not paid yet to the
// statements the new day gives them; paid statements keep their rows.
func (s *Service) UpdateCard(ctx context.Context, id int64, in CardInput) (CardView, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return CardView{}, err
	}

	// Read outside the transaction: the workspace is not reachable from inside it.
	today := s.Today(ctx)

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		c, err := loadCard(ctx, tx, id)
		if err != nil {
			return err
		}

		f, err := checkCard(ctx, tx, in, c.PaymentAccountID.V)
		if err != nil {
			return err
		}

		if f.closing != c.ClosingDay {
			if err := recomputeStatements(ctx, tx, c, f.closing, today); err != nil {
				return err
			}
		}

		if _, err := workspaces.Scoped[models.Account](ctx, tx).Where("id", c.AccountID).
			Update(ctx, orm.Set{"name": f.name}); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.CreditCard](ctx, tx).Where("id", id).Update(ctx, f.set())

		return err
	})
	if err != nil {
		return CardView{}, err
	}

	return s.Card(ctx, id)
}

// ArchiveCard archives the account of the card: it leaves the pickers and the
// plan quota, and keeps its history.
func (s *Service) ArchiveCard(ctx context.Context, id int64) (CardView, error) {
	c, err := loadCard(ctx, s.DB, id)
	if err != nil {
		return CardView{}, err
	}

	if _, err := s.ArchiveAccount(ctx, c.AccountID); err != nil {
		return CardView{}, err
	}

	return s.Card(ctx, id)
}

// UnarchiveCard brings an archived card back; the plan's account quota
// applies again.
func (s *Service) UnarchiveCard(ctx context.Context, id int64) (CardView, error) {
	c, err := loadCard(ctx, s.DB, id)
	if err != nil {
		return CardView{}, err
	}

	if _, err := s.UnarchiveAccount(ctx, c.AccountID); err != nil {
		return CardView{}, err
	}

	return s.Card(ctx, id)
}

// DeleteCard deletes the card with its account and everything recorded on it
// (DeleteAccount does the work and takes the card row along).
func (s *Service) DeleteCard(ctx context.Context, id int64) error {
	c, err := loadCard(ctx, s.DB, id)
	if err != nil {
		return err
	}

	return s.DeleteAccount(ctx, c.AccountID)
}
