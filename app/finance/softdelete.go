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
// transaction, its transactions (each one logged by the triggers). Its CSV
// imports are server-side artifacts of the account and go with it.
func (s *Service) DeleteAccount(ctx context.Context, id int64) error {
	at := s.now().UTC()

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := softDelete[models.Account](ctx, tx, id, at); err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("account_id", id).
			Update(ctx, orm.Set{"deleted_at": at}); err != nil {
			return err
		}

		_, err := workspaces.Scoped[models.Import](ctx, tx).Where("account_id", id).Delete(ctx)

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

// DeleteTransaction deletes a transaction of the workspace.
func (s *Service) DeleteTransaction(ctx context.Context, id int64) error {
	return softDelete[models.Transaction](ctx, s.DB, id, s.now().UTC())
}

// DeleteBudget deletes the budget of a category.
func (s *Service) DeleteBudget(ctx context.Context, id int64) error {
	return softDelete[models.Budget](ctx, s.DB, id, s.now().UTC())
}

// RestoreAccount brings back an account deleted less than RestoreWindow ago,
// with the transactions that were deleted together with it.
func (s *Service) RestoreAccount(ctx context.Context, id int64) (models.Account, error) {
	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		a, err := trashed[models.Account](ctx, tx, s.now(), id)
		if err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.Account](ctx, tx).OnlyTrashed().Where("id", id).
			Update(ctx, orm.Set{"deleted_at": nil}); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.Transaction](ctx, tx).OnlyTrashed().Where("account_id", id).
			Where("deleted_at", a.DeletedAt.V).Update(ctx, orm.Set{"deleted_at": nil})

		return err
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

// RestoreTransaction brings back a transaction whose account is alive.
func (s *Service) RestoreTransaction(ctx context.Context, id int64) (models.Transaction, error) {
	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		t, err := trashed[models.Transaction](ctx, tx, s.now(), id)
		if err != nil {
			return err
		}

		if err := requireLive[models.Account](ctx, tx, t.AccountID); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.Transaction](ctx, tx).OnlyTrashed().Where("id", id).
			Update(ctx, orm.Set{"deleted_at": nil})

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
