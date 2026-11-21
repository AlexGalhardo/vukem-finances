package billing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// ReconcileWindow is how far back a paid PIX payment is re-read: a refund or a
// dispute of an older payment is not expected to appear.
const ReconcileWindow = 35 * 24 * time.Hour

// ReconcilePix is the job billing_reconcile (every 6 hours). It does not rely
// on the names of the provider's events: it re-reads the charges that can still
// change and applies what the provider says.
//
//   - payments paid in the last 35 days: refunded or disputed ones lose their months;
//   - disputed payments: a dispute that was won gives them back;
//   - pending charges past their timer: expired only when the provider says so,
//     or activated if the provider says they were paid after all.
//
// Every move is a conditional UPDATE, so a second run changes nothing. A charge
// that cannot be read is logged by reference and skipped; the others are still
// processed, and the first failures are returned so the job shows as failed.
func (b *Billing) ReconcilePix(ctx context.Context) error {
	if b.Abacate == nil {
		return nil
	}

	now := b.Now().UTC()

	payments, err := orm.QueryOf[models.PixPayment](b.DB).Where("status", models.PixPaid).
		Where("paid_at", ">=", now.Add(-ReconcileWindow)).Get(ctx)
	if err != nil {
		return err
	}

	disputed, err := orm.QueryOf[models.PixPayment](b.DB).Where("status", models.PixDisputed).Get(ctx)
	if err != nil {
		return err
	}

	late, err := orm.QueryOf[models.PixPayment](b.DB).Where("status", models.PixPending).
		Where("expires_at", "<", now).Get(ctx)
	if err != nil {
		return err
	}

	var failures []error

	for _, group := range [][]models.PixPayment{payments, disputed, late} {
		for _, p := range group {
			if err := ctx.Err(); err != nil {
				return err
			}

			if err := b.applyPixState(ctx, p); err != nil {
				b.Log.WarnContext(ctx, "billing: reconcile failed for a payment", "payment", p.PublicRef, "err", err.Error())

				if len(failures) < 3 {
					failures = append(failures, fmt.Errorf("payment %s: %w", p.PublicRef, err))
				}
			}
		}
	}

	return errors.Join(failures...)
}

var errNotRefundable = errs.New(http.StatusConflict, "Only a paid or disputed payment can be marked as refunded.")

// AdminMarkRefunded is the support action "mark as refunded": the same effect
// as RevokePixPayment, plus an audit row naming the admin by user id. It
// reads across workspaces on purpose; the route is behind the admin gate.
func (b *Billing) AdminMarkRefunded(ctx context.Context, publicRef string, adminID int64) error {
	p, err := orm.QueryOf[models.PixPayment](b.DB).Where("public_ref", publicRef).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return errs.ErrNotFound
	}

	if err != nil {
		return err
	}

	switch p.Status {
	case models.PixRefunded:
		return nil // already done: nothing changes twice
	case models.PixPaid, models.PixDisputed:
	default:
		return errNotRefundable
	}

	if err := b.RevokePixPayment(ctx, p, models.RevokedRefund, SourceAdmin); err != nil {
		return err
	}

	return record(ctx, b.DB, p.WorkspaceID, SourceAdmin, "admin_mark_refunded", p.PublicRef,
		fmt.Sprintf("Marked as refunded by admin user %d", adminID))
}
