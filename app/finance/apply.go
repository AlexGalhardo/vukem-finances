package finance

import (
	"context"
	"fmt"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// DeleteIn soft-deletes row id of a synced entity inside tx, with the same
// cascades as the Delete services (an account takes its transactions, a
// transfer leg its partner, a category its budget, a goal its contributions).
// The caller owns the transaction and the workspace lock: the sync push
// applies a whole batch in one. ctx carries the tenant.
func (s *Service) DeleteIn(ctx context.Context, tx *database.Tx, entity string, id int64) error {
	at := s.now().UTC()

	switch entity {
	case "account":
		return deleteAccountIn(ctx, tx, id, at)
	case "category":
		return deleteCategoryIn(ctx, tx, id, at)
	case "transaction":
		return deleteTransactionIn(ctx, tx, id, at)
	case "tag":
		return deleteTagIn(ctx, tx, id, at)
	case "goal":
		return deleteGoalIn(ctx, tx, id, at)
	case "budget":
		return softDelete[models.Budget](ctx, tx, id, at)
	case "recurrence":
		return softDelete[models.Recurrence](ctx, tx, id, at)
	case "goal_contribution":
		return softDelete[models.GoalContribution](ctx, tx, id, at)
	case "credit_card":
		c, err := loadCard(ctx, tx, id)
		if err != nil {
			return err
		}

		return deleteAccountIn(ctx, tx, c.AccountID, at)
	}

	return fmt.Errorf("finance: no delete for entity %s", entity)
}

// RestoreIn brings back a row of a synced entity inside tx, at most
// RestoreWindow after its delete, with what was deleted together with it.
// A row whose parent is still deleted is refused (errParentDeleted).
func (s *Service) RestoreIn(ctx context.Context, tx *database.Tx, entity string, id int64) error {
	now := s.now()

	switch entity {
	case "account":
		return restoreAccountIn(ctx, tx, now, id)
	case "category":
		return restoreCategoryIn(ctx, tx, now, id)
	case "transaction":
		return restoreTransactionIn(ctx, tx, now, id)
	case "budget":
		return restoreBudgetIn(ctx, tx, now, id)
	case "goal":
		return restoreGoalIn(ctx, tx, now, id)
	case "recurrence":
		// Read through tx: the single connection of an in-memory database is ours.
		wsID, _ := workspaces.TenantID(ctx)

		ws, err := orm.QueryOf[models.Workspace](tx).Find(ctx, wsID)
		if err != nil {
			return err
		}

		return s.restoreRecurrenceIn(ctx, tx, ws, id)
	case "tag":
		if _, err := trashed[models.Tag](ctx, tx, now, id); err != nil {
			return err
		}

		_, err := workspaces.Scoped[models.Tag](ctx, tx).OnlyTrashed().Where("id", id).Update(ctx, orm.Set{"deleted_at": nil})
		if tx.UniqueViolation(err) {
			return errName
		}

		return err
	case "goal_contribution":
		c, err := trashed[models.GoalContribution](ctx, tx, now, id)
		if err != nil {
			return err
		}

		if err := requireLive[models.Goal](ctx, tx, c.GoalID); err != nil {
			return err
		}

		_, err = workspaces.Scoped[models.GoalContribution](ctx, tx).OnlyTrashed().Where("id", id).
			Update(ctx, orm.Set{"deleted_at": nil})

		return err
	case "credit_card":
		// A card is trashed with its account, at the same instant: bring the account back.
		c, err := workspaces.Scoped[models.CreditCard](ctx, tx).OnlyTrashed().Where("id", id).First(ctx)
		if err != nil {
			return errNotRestorable
		}

		return restoreAccountIn(ctx, tx, now, c.AccountID)
	}

	return fmt.Errorf("finance: no restore for entity %s", entity)
}

// IsParentDeleted reports whether err is the refusal to restore a row whose
// account, category or goal is still deleted.
func IsParentDeleted(err error) bool { return err == errParentDeleted } //nolint:errorlint // a sentinel

// IsNotRestorable reports whether err says the row is not in the trash or is
// past the restore window.
func IsNotRestorable(err error) bool { return err == errNotRestorable } //nolint:errorlint // a sentinel

// IsNameTaken reports whether err is the conflict of a restore over a live
// row with the same name.
func IsNameTaken(err error) bool { return err == errName } //nolint:errorlint // a sentinel
