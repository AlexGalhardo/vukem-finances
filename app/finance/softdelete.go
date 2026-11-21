package finance

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// RestoreWindow is how long a deleted row can be restored
// (docs/specs/sync-data-model.md).
const RestoreWindow = 30 * 24 * time.Hour

var (
	errParentDeleted = errs.New(http.StatusConflict, "Restore the account or category it belongs to first.")
	errNotRestorable = errs.New(http.StatusNotFound, "There is nothing to restore here.")
)

// reload reads a row after a write: the database bumps version in a trigger,
// so the struct the caller holds is stale.
func reload[T any](ctx context.Context, ex database.Executor, id int64) (T, error) {
	return workspaces.Scoped[T](ctx, ex).Where("id", id).First(ctx)
}

// softDelete marks a live row of the workspace as deleted at at.
func softDelete[T any](ctx context.Context, ex database.Executor, id int64, at time.Time) error {
	n, err := workspaces.Scoped[T](ctx, ex).Where("id", id).Update(ctx, orm.Set{"deleted_at": at})
	if err == nil && n == 0 {
		return errs.ErrNotFound
	}

	return err
}

// trashed finds a deleted row of the workspace that is still inside the
// restore window.
func trashed[T any](ctx context.Context, ex database.Executor, now time.Time, id int64) (T, error) {
	row, err := workspaces.Scoped[T](ctx, ex).OnlyTrashed().Where("id", id).
		Where("deleted_at", ">=", now.UTC().Add(-RestoreWindow)).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return row, errNotRestorable
	}

	return row, err
}

// DeleteAccount deletes an account of the workspace and, in the same
// transaction, its transactions (each one logged by the triggers). The other
// leg of every transfer it took part in goes too, so no transfer is left with
// one side. Its CSV imports are server-side artifacts of the account and go
// with it.
func (s *Service) DeleteAccount(ctx context.Context, id int64) error {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	at := s.now().UTC()

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		if err := softDelete[models.Account](ctx, tx, id, at); err != nil {
			return err
		}

		uids, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("account_id", id).
			WhereNotNull("transfer_uid").Pluck[string](ctx, "transfer_uid")
		if err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("account_id", id).
			Update(ctx, orm.Set{"deleted_at": at}); err != nil {
			return err
		}

		if len(uids) > 0 {
			if _, err := workspaces.Scoped[models.Transaction](ctx, tx).WhereIn("transfer_uid", anys(uids)...).
				Update(ctx, orm.Set{"deleted_at": at}); err != nil {
				return err
			}
		}

		_, err = workspaces.Scoped[models.Import](ctx, tx).Where("account_id", id).Delete(ctx)

		return err
	})
}

// DeleteCategory deletes a category with its budget; the transactions that
// used it become uncategorized (their category is cleared, and logged).
func (s *Service) DeleteCategory(ctx context.Context, id int64) error {
	at := s.now().UTC()

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := softDelete[models.Category](ctx, tx, id, at); err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.Budget](ctx, tx).Where("category_id", id).
			Update(ctx, orm.Set{"deleted_at": at}); err != nil {
			return err
		}

		_, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("category_id", id).
			Update(ctx, orm.Set{"category_id": nil})

		return err
	})
}

// DeleteTransaction deletes a transaction of the workspace. A leg of a
// transfer takes the other leg with it.
func (s *Service) DeleteTransaction(ctx context.Context, id int64) error {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return err
	}

	at := s.now().UTC()

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		leg, err := isTransferLeg(ctx, tx, id)
		if err != nil {
			return err
		}

		if leg {
			return deleteTransferOf(ctx, tx, id, at)
		}

		return softDelete[models.Transaction](ctx, tx, id, at)
	})
}

// DeleteBudget deletes the budget of a category.
func (s *Service) DeleteBudget(ctx context.Context, id int64) error {
	return softDelete[models.Budget](ctx, s.DB, id, s.now().UTC())
}

// RestoreAccount brings back an account deleted less than RestoreWindow ago,
// with the transactions that were deleted together with it.
func (s *Service) RestoreAccount(ctx context.Context, id int64) (models.Account, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Account{}, err
	}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		a, err := trashed[models.Account](ctx, tx, s.now(), id)
		if err != nil {
			return err
		}

		// Read before the legs of this account come back: the transfers whose
		// other leg lives on another account have to come back with them.
		uids, err := workspaces.Scoped[models.Transaction](ctx, tx).OnlyTrashed().Where("account_id", id).
			Where("deleted_at", a.DeletedAt.V).WhereNotNull("transfer_uid").Pluck[string](ctx, "transfer_uid")
		if err != nil {
			return err
		}

		if err := checkPairedAccounts(ctx, tx, id, a.DeletedAt.V, uids); err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.Account](ctx, tx).OnlyTrashed().Where("id", id).
			Update(ctx, orm.Set{"deleted_at": nil}); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.Transaction](ctx, tx).OnlyTrashed().Where("account_id", id).
			Where("deleted_at", a.DeletedAt.V).Update(ctx, orm.Set{"deleted_at": nil})
		if err != nil {
			return err
		}

		return restorePairedLegs(ctx, tx, a.DeletedAt.V, uids)
	})
	if err != nil {
		return models.Account{}, err
	}

	return reload[models.Account](ctx, s.DB, id)
}

// RestoreCategory brings back a category and the budget deleted with it. A
// live category with the same name refuses it (a conflict).
func (s *Service) RestoreCategory(ctx context.Context, id int64) (models.Category, error) {
	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		c, err := trashed[models.Category](ctx, tx, s.now(), id)
		if err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.Category](ctx, tx).OnlyTrashed().Where("id", id).
			Update(ctx, orm.Set{"deleted_at": nil}); err != nil {
			if s.DB.UniqueViolation(err) {
				return errName
			}

			return err
		}

		_, err = workspaces.Scoped[models.Budget](ctx, tx).OnlyTrashed().Where("category_id", id).
			Where("deleted_at", c.DeletedAt.V).Update(ctx, orm.Set{"deleted_at": nil})

		return err
	})
	if err != nil {
		return models.Category{}, err
	}

	return reload[models.Category](ctx, s.DB, id)
}

// RestoreTransaction brings back a transaction whose account is alive. A leg
// of a transfer brings the other leg back with it.
func (s *Service) RestoreTransaction(ctx context.Context, id int64) (models.Transaction, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Transaction{}, err
	}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		t, err := trashed[models.Transaction](ctx, tx, s.now(), id)
		if err != nil {
			return err
		}

		if err := requireLive[models.Account](ctx, tx, t.AccountID); err != nil {
			return err
		}

		if t.TransferUID.Valid {
			return restoreTransfer(ctx, tx, t)
		}

		set := orm.Set{"deleted_at": nil}

		// Its category may have been deleted while it was in the trash.
		if t.CategoryID.Valid {
			live, err := workspaces.Scoped[models.Category](ctx, tx).Where("id", t.CategoryID.V).Exists(ctx)
			if err != nil {
				return err
			}

			if !live {
				set["category_id"] = nil
			}
		}

		_, err = workspaces.Scoped[models.Transaction](ctx, tx).OnlyTrashed().Where("id", id).Update(ctx, set)

		return err
	})
	if err != nil {
		return models.Transaction{}, err
	}

	return reload[models.Transaction](ctx, s.DB, id)
}

// RestoreBudget brings back a budget whose category is alive.
func (s *Service) RestoreBudget(ctx context.Context, id int64) (models.Budget, error) {
	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		b, err := trashed[models.Budget](ctx, tx, s.now(), id)
		if err != nil {
			return err
		}

		if err := requireLive[models.Category](ctx, tx, b.CategoryID); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.Budget](ctx, tx).OnlyTrashed().Where("id", id).
			Update(ctx, orm.Set{"deleted_at": nil})
		if s.DB.UniqueViolation(err) {
			return errName
		}

		return err
	})
	if err != nil {
		return models.Budget{}, err
	}

	return reload[models.Budget](ctx, s.DB, id)
}

func requireLive[T any](ctx context.Context, ex database.Executor, id int64) error {
	ok, err := workspaces.Scoped[T](ctx, ex).Where("id", id).Exists(ctx)
	if err != nil {
		return err
	}

	if !ok {
		return errParentDeleted
	}

	return nil
}

func anys[T any](values []T) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}

	return out
}

// restoreTransfer brings back both legs of the transfer of t, the ones
// deleted together with it. The account of the other leg must be alive.
func restoreTransfer(ctx context.Context, tx *database.Tx, t models.Transaction) error {
	uids := []string{t.TransferUID.V}

	if err := checkPairedAccounts(ctx, tx, 0, t.DeletedAt.V, uids); err != nil {
		return err
	}

	return restorePairedLegs(ctx, tx, t.DeletedAt.V, uids)
}

// checkPairedAccounts refuses a restore that would bring back a transfer leg
// whose account is still deleted: that would leave a transfer with one side.
// skip is the account being restored itself.
func checkPairedAccounts(ctx context.Context, tx *database.Tx, skip int64, at time.Time, uids []string) error {
	if len(uids) == 0 {
		return nil
	}

	legs, err := workspaces.Scoped[models.Transaction](ctx, tx).OnlyTrashed().
		WhereIn("transfer_uid", anys(uids)...).Where("deleted_at", at).Get(ctx)
	if err != nil {
		return err
	}

	for _, l := range legs {
		if l.AccountID == skip {
			continue
		}

		if err := requireLive[models.Account](ctx, tx, l.AccountID); err != nil {
			return err
		}
	}

	return nil
}

func restorePairedLegs(ctx context.Context, tx *database.Tx, at time.Time, uids []string) error {
	if len(uids) == 0 {
		return nil
	}

	_, err := workspaces.Scoped[models.Transaction](ctx, tx).OnlyTrashed().
		WhereIn("transfer_uid", anys(uids)...).Where("deleted_at", at).
		Update(ctx, orm.Set{"deleted_at": nil})

	return err
}
