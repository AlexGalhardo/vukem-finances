package finance

import (
	"context"
	"database/sql"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var (
	errSameAccount    = errs.New(http.StatusUnprocessableEntity, "Choose two different accounts.")
	errArchivedTarget = errs.New(http.StatusUnprocessableEntity, "An archived account cannot take part in a transfer.")
	errIsTransfer     = errs.New(http.StatusUnprocessableEntity, "This is one side of a transfer: edit the transfer instead.")
	errArchivedEntry  = errs.New(http.StatusUnprocessableEntity, "This account is archived: unarchive it to record new entries.")
)

// refuseArchived stops new entries on an archived account. An archived account
// frees its place in the plan's account limit, so accepting them would let a
// Free workspace keep using more accounts than the plan allows.
func refuseArchived(ctx context.Context, ex database.Executor, accountID int64) error {
	archived, err := workspaces.Scoped[models.Account](ctx, ex).Where("id", accountID).WhereNotNull("archived_at").Exists(ctx)
	if err != nil {
		return err
	}

	if archived {
		return errArchivedEntry
	}

	return nil
}

// TransferInput is a transfer as the form sends it.
type TransferInput struct {
	FromAccountID int64
	ToAccountID   int64
	Amount        string
	Description   string
	Date          string
}

// Transfer is the two legs of a transfer: an expense on the source account
// and an income on the destination, equal in amount and sharing TransferUID.
// Every account of a workspace uses the workspace currency, so a transfer has
// one amount and no exchange rate.
type Transfer struct {
	UID  string             `json:"uid"`
	From models.Transaction `json:"from"`
	To   models.Transaction `json:"to"`
}

type transferFields struct {
	from, to    models.Account
	cents       int64
	description string
	date        string
}

// checkTransfer validates in against the accounts of the workspace. An
// account outside the workspace (or deleted) is a 404, so ids cannot be
// probed; an archived one is refused unless it is one of keep, the accounts
// an edited transfer already uses.
func checkTransfer(ctx context.Context, ex database.Executor, in TransferInput, keep ...int64) (transferFields, error) {
	var f transferFields

	cents, err := ParseAmount(in.Amount)
	if err != nil || cents <= 0 {
		return f, errAmountIn
	}

	if !validDate(in.Date) {
		return f, errDate
	}

	load := func(id int64) (models.Account, error) {
		a, err := workspaces.Scoped[models.Account](ctx, ex).Where("id", id).First(ctx)
		if err != nil {
			return a, errs.ErrNotFound
		}

		return a, nil
	}

	if f.from, err = load(in.FromAccountID); err != nil {
		return f, err
	}

	if f.to, err = load(in.ToAccountID); err != nil {
		return f, err
	}

	if f.from.ID == f.to.ID {
		return f, errSameAccount
	}

	for _, a := range []models.Account{f.from, f.to} {
		if a.ArchivedAt.Valid && !slices.Contains(keep, a.ID) {
			return f, errArchivedTarget
		}
	}

	f.cents, f.description, f.date = cents, strings.TrimSpace(in.Description), in.Date

	return f, nil
}

// CreateTransfer records a transfer: both legs are written in one database
// transaction, so either both exist or neither does. A transfer is not
// income or expense and does not count toward the monthly quota.
func (s *Service) CreateTransfer(ctx context.Context, in TransferInput) (Transfer, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return Transfer{}, err
	}

	var out Transfer

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		out, err = insertTransfer(ctx, tx, in, "")

		return err
	})

	return out, err
}

// insertTransfer writes the two legs of a transfer inside a transaction that
// holds the workspace lock. statementRef, when set, tags the leg on the
// destination account: the payment of a card statement.
func insertTransfer(ctx context.Context, tx *database.Tx, in TransferInput, statementRef string) (Transfer, error) {
	f, err := checkTransfer(ctx, tx, in)
	if err != nil {
		return Transfer{}, err
	}

	uid := models.NewUID()
	leg := func(account int64, kind string) models.Transaction {
		return models.Transaction{
			AccountID: account, Kind: kind, AmountCents: f.cents, Description: f.description,
			OccurredOn: f.date, TransferUID: sql.Null[string]{V: uid, Valid: true},
		}
	}

	source, dest := leg(f.from.ID, models.KindExpense), leg(f.to.ID, models.KindIncome)
	dest.StatementRef = sql.Null[string]{V: statementRef, Valid: statementRef != ""}

	if err := orm.Create(ctx, tx, &source); err != nil {
		return Transfer{}, err
	}

	if err := orm.Create(ctx, tx, &dest); err != nil {
		return Transfer{}, err
	}

	return readTransfer(ctx, tx, uid)
}

// readTransfer reads the two live legs of a transfer; anything but exactly
// two (one expense, one income) is not a transfer the caller may use.
func readTransfer(ctx context.Context, ex database.Executor, uid string) (Transfer, error) {
	legs, err := workspaces.Scoped[models.Transaction](ctx, ex).Where("transfer_uid", uid).Get(ctx)
	if err != nil {
		return Transfer{}, err
	}

	t := Transfer{UID: uid}
	seenFrom, seenTo := false, false

	for _, l := range legs {
		switch l.Kind {
		case models.KindExpense:
			t.From, seenFrom = l, true
		case models.KindIncome:
			t.To, seenTo = l, true
		}
	}

	if len(legs) != 2 || !seenFrom || !seenTo {
		return Transfer{}, errs.ErrNotFound
	}

	return t, nil
}

// transferOfLeg finds the transfer one of whose legs is the transaction id.
// A transaction that is not a transfer leg is a 404 here.
func transferOfLeg(ctx context.Context, ex database.Executor, id int64) (Transfer, error) {
	leg, err := workspaces.Scoped[models.Transaction](ctx, ex).Where("id", id).WhereNotNull("transfer_uid").First(ctx)
	if err != nil {
		return Transfer{}, errs.ErrNotFound
	}

	return readTransfer(ctx, ex, leg.TransferUID.V)
}

// UpdateTransfer edits a transfer through either of its legs: both legs
// change together, in one database transaction.
func (s *Service) UpdateTransfer(ctx context.Context, id int64, in TransferInput) (Transfer, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return Transfer{}, err
	}

	var out Transfer

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		current, err := transferOfLeg(ctx, tx, id)
		if err != nil {
			return err
		}

		f, err := checkTransfer(ctx, tx, in, current.From.AccountID, current.To.AccountID)
		if err != nil {
			return err
		}

		for _, leg := range []struct {
			id      int64
			account int64
			was     models.Transaction
		}{{current.From.ID, f.from.ID, current.From}, {current.To.ID, f.to.ID, current.To}} {
			set := orm.Set{
				"account_id": leg.account, "amount_cents": f.cents, "description": f.description, "occurred_on": f.date,
			}

			// A statement payment stays a payment of its statement while it
			// stays on the same card; moved to another account it is no longer one.
			if leg.was.StatementRef.Valid && leg.was.AccountID != leg.account {
				set["statement_ref"] = nil
			}

			if _, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("id", leg.id).Update(ctx, set); err != nil {
				return err
			}
		}

		out, err = readTransfer(ctx, tx, current.UID)

		return err
	})

	return out, err
}

// DeleteTransfer deletes both legs of the transfer one of whose legs is id.
func (s *Service) DeleteTransfer(ctx context.Context, id int64) error {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	at := s.now().UTC()

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		return deleteTransferOf(ctx, tx, id, at)
	})
}

func deleteTransferOf(ctx context.Context, tx *database.Tx, id int64, at time.Time) error {
	t, err := transferOfLeg(ctx, tx, id)
	if err != nil {
		return err
	}

	_, err = workspaces.Scoped[models.Transaction](ctx, tx).Where("transfer_uid", t.UID).
		Update(ctx, orm.Set{"deleted_at": at})

	return err
}

// isTransferLeg reports whether the live transaction id is a transfer leg.
func isTransferLeg(ctx context.Context, ex database.Executor, id int64) (bool, error) {
	return workspaces.Scoped[models.Transaction](ctx, ex).Where("id", id).WhereNotNull("transfer_uid").Exists(ctx)
}
