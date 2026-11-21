package finance

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/timeutil"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Installment limits: docs/specs/recurrence.md.
const (
	minParts = 2
	maxParts = 60
)

var (
	errParts      = errs.New(http.StatusUnprocessableEntity, "An installment purchase has 2 to 60 parts.")
	errPartsTotal = errs.New(http.StatusUnprocessableEntity, "The total is too small: every part must be at least one cent.")
	errNotPart    = errs.New(http.StatusUnprocessableEntity, "This transaction is not part of an installment purchase.")
)

// installmentMark says which part of a group a transaction is.
type installmentMark struct {
	group string
	no    int
	total int
}

// SplitInstallments divides total cents into parts amounts that add up to
// total exactly: the first total%parts parts get one extra cent (100.00 in 3
// is 33.34, 33.33, 33.33). It needs parts >= 1 and total >= parts.
func SplitInstallments(total int64, parts int) []int64 {
	base, extra := total/int64(parts), int(total%int64(parts))
	out := make([]int64, parts)

	for i := range out {
		out[i] = base
		if i < extra {
			out[i]++
		}
	}

	return out
}

// InstallmentDates are the dates of the parts: AddMonths of the first date,
// so a purchase starting on the 31st keeps month ends.
func InstallmentDates(first time.Time, parts int) []string {
	out := make([]string, parts)
	for i := range out {
		out[i] = timeutil.AddMonths(first, i).Format(isoDate)
	}

	return out
}

// InstallmentInput is an installment purchase as the form sends it: an
// expense of Total split into Parts monthly parts.
type InstallmentInput struct {
	AccountID   int64
	CategoryID  int64
	Total       string
	Parts       int
	FirstDate   string
	Description string
	Tags        []string
}

// CreateInstallments writes the parts in one database transaction, each one
// through the normal path (its own month's quota, its card statement). A part
// that exceeds a quota refuses the whole group and nothing is written.
func (s *Service) CreateInstallments(ctx context.Context, in InstallmentInput) ([]models.Transaction, error) {
	if in.Parts < minParts || in.Parts > maxParts {
		return nil, errParts
	}

	total, err := ParseAmount(in.Total)
	if err != nil || total <= 0 {
		return nil, errAmountIn
	}

	if total < int64(in.Parts) {
		return nil, errPartsTotal
	}

	first, err := time.Parse(isoDate, in.FirstDate)
	if err != nil {
		return nil, errDate
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return nil, err
	}

	amounts, dates, group := SplitInstallments(total, in.Parts), InstallmentDates(first, in.Parts), models.NewUID()
	out := make([]models.Transaction, 0, in.Parts)

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		out = out[:0]

		for i := range amounts {
			var made, previous models.Transaction

			part := TxInput{
				AccountID: in.AccountID, CategoryID: in.CategoryID, Kind: models.KindExpense, Amount: plainAmount(amounts[i]),
				Description: in.Description, Date: dates[i], Tags: in.Tags,
				installment: installmentMark{group: group, no: i + 1, total: in.Parts},
			}

			if err := s.persistTransaction(ctx, tx, ws.ID, 0, part, &made, &previous); err != nil {
				return err
			}

			out = append(out, made)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, t := range out {
		_ = s.notifyBudget(ctx, t, models.Transaction{})
	}

	return out, nil
}

// InstallmentLabel is how a part reads in a list: "Description (2/6)".
func InstallmentLabel(description string, no, total int64) string {
	return fmt.Sprintf("%s (%d/%d)", description, no, total)
}

// DeleteInstallments deletes the part id and the ones after it (wholeGroup
// false) or every part of its group (true). Parts dated up to today are kept
// unless includePast: they are history. It returns how many it deleted.
func (s *Service) DeleteInstallments(ctx context.Context, id int64, wholeGroup, includePast bool) (int64, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return 0, err
	}

	at := s.now().UTC()
	today := s.nowIn(ws).Format(isoDate)

	var n int64

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx, ws.ID); err != nil {
			return err
		}

		part, err := workspaces.Scoped[models.Transaction](ctx, tx).Where("id", id).First(ctx)
		if err != nil {
			return err
		}

		if !part.InstallmentGroupUID.Valid {
			return errNotPart
		}

		q := workspaces.Scoped[models.Transaction](ctx, tx).Where("installment_group_uid", part.InstallmentGroupUID.V)
		if !wholeGroup {
			q = q.Where("installment_no", ">=", part.InstallmentNo.V)
		}

		if !includePast {
			q = q.Where("occurred_on", ">", today)
		}

		n, err = q.Update(ctx, orm.Set{"deleted_at": at})

		return err
	})

	return n, err
}
