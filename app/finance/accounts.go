package finance

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var errAccountKind = errs.New(
	http.StatusUnprocessableEntity,
	"The kind must be checking, savings, cash, investment or wallet.",
)

var errCardAccount = errs.New(http.StatusUnprocessableEntity, "This account is a credit card: edit it from the Cards page.")

// Accounts lists the live accounts of the workspace of ctx in display order,
// archived ones included: pickers hide them, histories and balances keep them.
func (s *Service) Accounts(ctx context.Context) ([]models.Account, error) {
	return workspaces.Scoped[models.Account](ctx, s.DB).
		OrderBy("position", orm.Asc).OrderBy("id", orm.Asc).Get(ctx)
}

// lockWorkspace serializes the writers that decide on a count or a pair
// (the account quota, the legs of a transfer). SQLite already runs the
// transaction as BEGIN IMMEDIATE; PostgreSQL and MySQL hold the row lock.
func lockWorkspace(ctx context.Context, tx *database.Tx, workspaceID int64) error {
	_, err := orm.QueryOf[models.Workspace](tx).LockForUpdate().Find(ctx, workspaceID)

	return err
}

// parseOpening reads an opening balance; empty means zero.
func parseOpening(opening string) (int64, error) {
	if strings.TrimSpace(opening) == "" {
		return 0, nil
	}

	cents, err := ParseAmount(opening)
	if err != nil {
		return 0, errAmountIn
	}

	return cents, nil
}

// liveAccounts counts the accounts that weigh on the plan: the archived ones
// do not.
func liveAccounts(ctx context.Context, ex database.Executor) (int, error) {
	n, err := workspaces.Scoped[models.Account](ctx, ex).WhereNull("archived_at").Count(ctx)

	return int(n), err
}

// CreateAccount adds an account; the Free plan allows one that is not archived.
func (s *Service) CreateAccount(ctx context.Context, name, kind, opening string) (models.Account, error) {
	return s.CreateAccountWithUID(ctx, "", name, kind, opening)
}

// CreateAccountWithUID is CreateAccount with the uid a client chose (empty
// generates one); the caller has checked its shape.
func (s *Service) CreateAccountWithUID(ctx context.Context, uid, name, kind, opening string) (models.Account, error) {
	if !slices.Contains(models.AccountKinds(), kind) {
		return models.Account{}, errAccountKind
	}

	cents, err := parseOpening(opening)
	if err != nil {
		return models.Account{}, err
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Account{}, err
	}

	lim, err := s.WS.Limits(ctx)
	if err != nil {
		return models.Account{}, err
	}

	var id int64

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		a, err := insertAccount(ctx, tx, lim, uid, name, kind, cents)
		id = a.ID

		return err
	})
	if err != nil {
		return models.Account{}, err
	}

	return reload[models.Account](ctx, s.DB, id)
}

// insertAccount adds an account inside a transaction that holds the workspace
// lock, after the plan's account quota allows one more. An empty uid gets a
// new one.
func insertAccount(
	ctx context.Context, tx *database.Tx, lim workspaces.Limits, uid, name, kind string, opening int64,
) (models.Account, error) {
	n, err := liveAccounts(ctx, tx)
	if err != nil {
		return models.Account{}, err
	}

	if err := lim.CheckAccounts(n + 1); err != nil {
		return models.Account{}, err
	}

	last, err := workspaces.Scoped[models.Account](ctx, tx).Max[int64](ctx, "position")
	if err != nil && !errors.Is(err, orm.ErrNotFound) {
		return models.Account{}, err
	}

	a := models.Account{UID: uid, Name: strings.TrimSpace(name), Kind: kind, OpeningCents: opening, Position: int(last) + 1}
	if err := orm.Create(ctx, tx, &a); err != nil {
		return models.Account{}, err
	}

	return a, nil
}

// UpdateAccount changes the name, the kind and the opening balance of an
// account (an archived one too). The balance is never edited: it follows.
func (s *Service) UpdateAccount(ctx context.Context, id int64, name, kind, opening string) (models.Account, error) {
	if !slices.Contains(models.AccountKinds(), kind) {
		return models.Account{}, errAccountKind
	}

	cents, err := parseOpening(opening)
	if err != nil {
		return models.Account{}, err
	}

	current, err := reload[models.Account](ctx, s.DB, id)
	if err != nil {
		return models.Account{}, errs.ErrNotFound
	}

	if current.Kind == models.AccountCreditCard {
		return models.Account{}, errCardAccount
	}

	n, err := workspaces.Scoped[models.Account](ctx, s.DB).Where("id", id).Update(ctx, orm.Set{
		"name": strings.TrimSpace(name), "kind": kind, "opening_cents": cents,
	})
	if err != nil {
		return models.Account{}, err
	}

	if n == 0 {
		return models.Account{}, errs.ErrNotFound
	}

	return reload[models.Account](ctx, s.DB, id)
}

// ArchiveAccount hides an account from pickers and frees its place in the
// plan; its history and balance stay. Archiving twice changes nothing.
func (s *Service) ArchiveAccount(ctx context.Context, id int64) (models.Account, error) {
	a, err := reload[models.Account](ctx, s.DB, id)
	if err != nil {
		return a, err
	}

	if a.ArchivedAt.Valid {
		return a, nil
	}

	if _, err := workspaces.Scoped[models.Account](ctx, s.DB).Where("id", id).WhereNull("archived_at").
		Update(ctx, orm.Set{"archived_at": s.now().UTC()}); err != nil {
		return models.Account{}, err
	}

	return reload[models.Account](ctx, s.DB, id)
}

// UnarchiveAccount brings an account back; it weighs on the plan again, so
// the Free plan refuses it while another account is live.
func (s *Service) UnarchiveAccount(ctx context.Context, id int64) (models.Account, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Account{}, err
	}

	lim, err := s.WS.Limits(ctx)
	if err != nil {
		return models.Account{}, err
	}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		a, err := reload[models.Account](ctx, tx, id)
		if err != nil {
			return err
		}

		if !a.ArchivedAt.Valid {
			return nil
		}

		n, err := liveAccounts(ctx, tx)
		if err != nil {
			return err
		}

		if err := lim.CheckAccounts(n + 1); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.Account](ctx, tx).Where("id", id).
			Update(ctx, orm.Set{"archived_at": nil})

		return err
	})
	if err != nil {
		return models.Account{}, err
	}

	return reload[models.Account](ctx, s.DB, id)
}
