package finance

import (
	"context"
	"database/sql"
	"strings"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// UpdateTransaction edits a transaction belonging to the current workspace.
// Moving it into another month observes that month's quota atomically.
func (s *Service) UpdateTransaction(ctx context.Context, id int64, in TxInput) (models.Transaction, error) {
	return s.saveTransaction(ctx, id, in)
}

// transactionInput validates in. keepAccount is the account the row already
// uses (0 for a new one): its history stays editable after it is archived.
func transactionInput(ctx context.Context, ex database.Executor, in TxInput, keepAccount, keepCategory int64) (models.Transaction, error) {
	if in.Kind != models.KindIncome && in.Kind != models.KindExpense {
		return models.Transaction{}, errKind
	}
	if !validDate(in.Date) {
		return models.Transaction{}, errDate
	}
	cents, err := ParseAmount(in.Amount)
	if err != nil || cents <= 0 {
		return models.Transaction{}, errAmountIn
	}
	if ok, err := workspaces.Scoped[models.Account](ctx, ex).Where("id", in.AccountID).Exists(ctx); err != nil || !ok {
		return models.Transaction{}, firstErr(err, errAccount)
	}
	if in.AccountID != keepAccount {
		if err := refuseArchived(ctx, ex, in.AccountID); err != nil {
			return models.Transaction{}, err
		}
	}
	out := models.Transaction{
		AccountID:   in.AccountID,
		Kind:        in.Kind,
		AmountCents: cents,
		Description: strings.TrimSpace(in.Description),
		OccurredOn:  in.Date,
	}
	if in.CategoryID > 0 {
		q := workspaces.Scoped[models.Category](ctx, ex).
			Where("id", in.CategoryID).
			Where("kind", in.Kind)
		// An archived category takes no new entries; a row that already has it keeps it.
		if in.CategoryID != keepCategory {
			q = q.WhereNull("archived_at")
		}
		ok, err := q.Exists(ctx)
		if err != nil || !ok {
			return models.Transaction{}, firstErr(err, errCategory)
		}
		out.CategoryID = sql.Null[int64]{V: in.CategoryID, Valid: true}
	}
	return out, nil
}

func (s *Service) saveTransaction(ctx context.Context, id int64, in TxInput) (models.Transaction, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return models.Transaction{}, err
	}

	var out, previous models.Transaction

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		return s.persistTransaction(ctx, tx, ws.ID, id, in, &out, &previous)
	})
	if err != nil {
		return models.Transaction{}, err
	}

	if err := s.notifyBudget(ctx, out, previous); err != nil {
		return out, err
	}

	return out, nil
}

func (s *Service) persistTransaction(
	ctx context.Context,
	tx *database.Tx,
	workspaceID int64,
	id int64,
	in TxInput,
	out, previous *models.Transaction,
) error {
	// Shared by inserts and edits. SQLite uses BEGIN IMMEDIATE; PostgreSQL
	// and MySQL hold the workspace row lock until the quota write commits.
	if _, err := orm.QueryOf[models.Workspace](tx).LockForUpdate().Find(ctx, workspaceID); err != nil {
		return err
	}

	if id != 0 {
		row, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("id", id).First(ctx)
		if err != nil {
			return err
		}

		if row.TransferUID.Valid {
			return errIsTransfer
		}

		*previous = row
	}

	row, err := transactionInput(ctx, tx, in, previous.AccountID, previous.CategoryID.V)
	if err != nil {
		return err
	}

	if id == 0 || MonthOf(previous.OccurredOn) != MonthOf(row.OccurredOn) {
		// The plan is read here, inside the transaction and after the workspace
		// lock, never before it: a plan that changes while the request waits
		// (a refund, a payment) is the plan the quota is checked against.
		lim, err := s.WS.LimitsIn(ctx, tx)
		if err != nil {
			return err
		}

		if err := checkMonthQuota(ctx, tx, lim, row.OccurredOn); err != nil {
			return err
		}
	}

	if id == 0 {
		*out = row

		if err := orm.Create(ctx, tx, out); err != nil {
			return err
		}

		saved, err := reload[models.Transaction](ctx, tx, out.ID)
		if err != nil {
			return err
		}

		*out = saved

		return nil
	}

	if err := updateTransaction(ctx, tx, id, row); err != nil {
		return err
	}

	saved, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("id", id).First(ctx)
	if err != nil {
		return err
	}

	*out = saved

	return nil
}

func checkMonthQuota(ctx context.Context, tx *database.Tx, lim workspaces.Limits, date string) error {
	from, to := monthBounds(MonthOf(date))

	// A transfer moves money between accounts; it is not an entry of the plan's monthly quota.
	n, err := workspaces.Scoped[models.Transaction](ctx, tx).WhereNull("transfer_uid").
		WhereBetween("occurred_on", from, to).Count(ctx)
	if err != nil {
		return err
	}

	return lim.CheckTransactions(int(n) + 1)
}

func updateTransaction(ctx context.Context, tx *database.Tx, id int64, row models.Transaction) error {
	var category any
	if row.CategoryID.Valid {
		category = row.CategoryID.V
	}

	_, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("id", id).Update(ctx, orm.Set{
		"account_id":   row.AccountID,
		"category_id":  category,
		"kind":         row.Kind,
		"amount_cents": row.AmountCents,
		"description":  row.Description,
		"occurred_on":  row.OccurredOn,
	})

	return err
}

func (s *Service) notifyBudget(ctx context.Context, out, previous models.Transaction) error {
	if out.Kind != models.KindExpense || !out.CategoryID.Valid {
		return nil
	}

	delta := out.AmountCents
	sameCategory := previous.Kind == out.Kind && previous.CategoryID == out.CategoryID
	// MonthOf needs a real date. A new transaction has no previous row.
	sameMonth := previous.ID != 0 && MonthOf(previous.OccurredOn) == MonthOf(out.OccurredOn)

	if previous.ID != 0 && sameMonth && sameCategory {
		delta -= previous.AmountCents
	}

	if delta <= 0 {
		return nil
	}

	return s.alertBudget(ctx, out.CategoryID.V, MonthOf(out.OccurredOn), delta)
}
