package privacy

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/schema"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// purgeBatch is how many rows one DELETE removes, so a large workspace never
// holds SQLite's one writer for long.
const purgeBatch = 500

// tombstoneRetention is how long the public id of a purged workspace token is
// remembered. A token lives at most a year; after that no phone can still hold it.
const tombstoneRetention = 400 * 24 * time.Hour

// purgedTables is the classification of every table that holds a workspace_id:
// all of them are purged, children before parents (a transaction before its
// category, the changes log last). A table that is not listed stops the purge
// with an error, so a future table cannot escape it: add it here, or, when the
// law asks to keep something of it, copy that part to payment_records first.
//
// Retained on purpose (no workspace_id, never touched by a workspace purge):
// privacy_audit, payment_records (until retain_until), purged_tokens.
var purgedTables = []string{
	"transaction_tags", "transactions", "goal_contributions", "goals", "recurrences", "credit_cards", "budgets",
	"description_rules", "suggestions", "tags", "categories", "accounts", "imports",
	"assistant_messages", "assistant_conversations", "notification_preferences", "notifications",
	"sync_conflicts", "sync_ops", "sync_devices", "sync_state", "pairing_codes",
	"ai_credentials", "workspace_ai_settings", "ai_usage",
	"audit_events", "billing_events", "pix_payments", "pix_entitlements", "workspace_subscriptions",
	"monthly_summary_sent", "invites", "mcp_grants", "memberships", "changes",
}

// PurgedTables lists the tables the purge removes a workspace from, in order.
func PurgedTables() []string { return slices.Clone(purgedTables) }

// Report counts what one purge pass did.
type Report struct {
	Workspaces int
	Accounts   int
	// Blocked are subjects that are due but wait (a subscription that could not
	// be cancelled, a workspace of the account that is still waiting).
	Blocked int
	// Records are payment records and token tombstones removed after their retention.
	Records int
}

// Purge runs one idempotent pass: the workspaces whose grace period ended, then
// the accounts, then what the retention periods let go. Running it twice, or
// after a pass that stopped half way, gives the same final state and no error.
// A subject that cannot be purged yet is counted as blocked and the pass goes on.
func (s *Service) Purge(ctx context.Context) (Report, error) {
	var rep Report

	if err := s.verifyClassification(ctx); err != nil {
		return rep, err
	}

	now := s.now().UTC()

	due, err := orm.QueryOf[models.Workspace](s.DB).Where("state", models.StatePendingDeletion).
		Where("purge_at", "<=", now).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return rep, err
	}

	for _, ws := range due {
		switch err := s.PurgeWorkspace(ctx, ws.ID); {
		case err == nil:
			rep.Workspaces++
		case errors.Is(err, errPurgeBlocked):
			rep.Blocked++
		default:
			return rep, err
		}
	}

	users, err := orm.QueryOf[models.User](s.DB).Where("purge_at", "<=", now).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return rep, err
	}

	for _, u := range users {
		switch err := s.PurgeAccount(ctx, u.ID); {
		case err == nil:
			rep.Accounts++
		case errors.Is(err, errPurgeBlocked):
			rep.Blocked++
		default:
			return rep, err
		}
	}

	if rep.Records, err = s.pruneRetained(ctx, now); err != nil {
		return rep, err
	}

	return rep, nil
}

var errPurgeBlocked = errors.New("privacy: the purge waits")

// verifyClassification stops the purge when a table holds a workspace_id that
// purgedTables does not know: better no purge than a purge that leaves rows.
func (s *Service) verifyClassification(ctx context.Context) error {
	snap, err := schema.Inspect(ctx, s.DB)
	if err != nil {
		return err
	}

	var missing []string

	for _, tbl := range snap.Tables {
		has := slices.ContainsFunc(tbl.Columns, func(c schema.ColumnInfo) bool { return c.Name == "workspace_id" })
		if has && !slices.Contains(purgedTables, tbl.Name) {
			missing = append(missing, tbl.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("privacy: tables with a workspace_id are not classified for the purge: %v", missing)
	}

	return nil
}

// PurgeWorkspace erases one workspace for good. It does nothing for a workspace
// that does not exist (already purged) and refuses one that is not pending or
// whose grace period has not ended.
//
// Every statement is scoped by the workspace id, which is validated first; the
// rows go in batches, bypassing soft delete on purpose (a trashed row is a row).
func (s *Service) PurgeWorkspace(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("privacy: refusing to purge workspace %d", id)
	}

	ws, err := orm.QueryOf[models.Workspace](s.DB).Find(ctx, id)
	if errors.Is(err, orm.ErrNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	if !ws.PendingDeletion() || !ws.PurgeAt.Valid || ws.PurgeAt.V.After(s.now()) {
		return errors.New("privacy: the workspace is not due for purge")
	}

	ref := s.Ref("workspace", id)

	if s.Billing != nil {
		if err := s.Billing.CancelWorkspaceSubscriptions(ctx, id); err != nil {
			s.markBlocked(ctx, id, PurgeBlockedBilling)

			s.Log.ErrorContext(ctx, "alert: a workspace is due for purge but its subscription could not be cancelled",
				"workspace_ref", ref)

			return errPurgeBlocked
		}
	}

	if err := s.retainPayments(ctx, ws); err != nil {
		return err
	}

	// Remember the public ids of the tokens, so a phone that syncs late hears 410.
	if _, err := s.DB.ExecContext(ctx, `INSERT OR IGNORE INTO purged_tokens (token_id, created_at, updated_at)
SELECT id, ?, ? FROM access_tokens WHERE resource = ?`, s.now().UTC(), s.now().UTC(), strconv.FormatInt(id, 10)); err != nil {
		return err
	}

	// What lives outside the tenant tables and points at the workspace.
	if err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		// A cancel that committed just before the grace period ended wins.
		still, err := orm.QueryOf[models.Workspace](tx).Where("id", id).Where("state", models.StatePendingDeletion).Count(ctx)
		if err != nil {
			return err
		}

		if still == 0 {
			return errPurgeBlocked
		}

		return purgeWorkspaceEdges(ctx, tx, id)
	}); err != nil {
		return err
	}

	for _, table := range purgedTables {
		if err := deleteInBatches(ctx, s.DB, table, id); err != nil {
			return fmt.Errorf("purging %s: %w", table, err)
		}
	}

	// The workspace row last; the foreign keys would take anything left with it.
	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM workspaces WHERE id = ?", id); err != nil {
			return err
		}

		return s.record(ctx, tx, EventWorkspacePurged, ref)
	})
	if err != nil {
		return err
	}

	s.Log.InfoContext(ctx, "workspace purged", "workspace_ref", ref)

	return nil
}

// purgeWorkspaceEdges removes what points at the workspace from tables that carry
// no workspace_id: OAuth state behind the MCP grants, API tokens bound to it
// (resource), and the Telegram rows that name it.
func purgeWorkspaceEdges(ctx context.Context, tx *database.Tx, id int64) error {
	ws := strconv.FormatInt(id, 10)

	for _, q := range []string{
		"DELETE FROM oauth_codes WHERE grant_id IN (SELECT grant_id FROM mcp_grants WHERE workspace_id = ?)",
		"DELETE FROM oauth_refresh_tokens WHERE grant_id IN (SELECT grant_id FROM mcp_grants WHERE workspace_id = ?)",
		"DELETE FROM oauth_grants WHERE id IN (SELECT grant_id FROM mcp_grants WHERE workspace_id = ?)",
		"DELETE FROM telegram_pending WHERE for_workspace_id = ?",
		"DELETE FROM telegram_link_codes WHERE for_workspace_id = ?",
		"UPDATE telegram_links SET current_workspace_id = NULL WHERE current_workspace_id = ?",
	} {
		if _, err := tx.ExecContext(ctx, q, id); err != nil {
			return err
		}
	}

	_, err := tx.ExecContext(ctx, "DELETE FROM access_tokens WHERE resource = ?", ws)

	return err
}

// deleteInBatches removes every row of table that belongs to workspaceID,
// purgeBatch at a time. The table name comes from purgedTables, never from input.
func deleteInBatches(ctx context.Context, db *database.DB, table string, workspaceID int64) error {
	q := "DELETE FROM " + table + " WHERE rowid IN (SELECT rowid FROM " + table + " WHERE workspace_id = ? LIMIT " +
		strconv.Itoa(purgeBatch) + ")"

	for {
		res, err := db.ExecContext(ctx, q, workspaceID)
		if err != nil {
			return err
		}

		n, err := res.RowsAffected()
		if err != nil {
			return err
		}

		if n == 0 {
			return nil
		}
	}
}

func (s *Service) markBlocked(ctx context.Context, workspaceID int64, reason string) {
	if _, err := orm.QueryOf[models.Workspace](s.DB).Where("id", workspaceID).Update(ctx, orm.Set{"purge_error": reason}); err != nil {
		s.Log.ErrorContext(ctx, "privacy: could not store why a purge waits", "err", err)
	}
}

// retainPayments copies what the law asks to keep of the payments of the
// workspace into payment_records: the amount, the date, the provider reference
// and a subject reference that cannot be turned back into the workspace. The
// copy is idempotent (the provider reference is unique).
func (s *Service) retainPayments(ctx context.Context, ws models.Workspace) error {
	ref := s.Ref("workspace", ws.ID)
	now := s.now().UTC()

	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		payments, err := orm.QueryOf[models.PixPayment](tx).Where("workspace_id", ws.ID).
			WhereIn("status", models.PixPaid, models.PixRefunded, models.PixDisputed).Get(ctx)
		if err != nil {
			return err
		}

		for _, p := range payments {
			if !p.PaidAt.Valid {
				continue
			}

			if err := s.insertRecord(ctx, tx, models.PaymentRecord{
				SubjectRef: ref, Provider: "abacatepay", ProviderRef: p.ProviderRef, AmountCents: p.AmountCents,
				Currency: p.Currency, PaidAt: p.PaidAt.V.UTC(), RetainUntil: p.PaidAt.V.UTC().AddDate(s.RetentionYears, 0, 0),
			}); err != nil {
				return err
			}
		}

		subs, err := orm.QueryOf[models.WorkspaceSubscription](tx).Where("workspace_id", ws.ID).Get(ctx)
		if err != nil {
			return err
		}

		for _, sub := range subs {
			// A subscription is not a payment: the record keeps the date it began and
			// its price, which is all the local row knows. Stripe holds the invoices.
			if err := s.insertRecord(ctx, tx, models.PaymentRecord{
				SubjectRef: ref, Provider: "stripe", ProviderRef: sub.StripeSubscriptionID, AmountCents: sub.UnitAmount,
				PaidAt: sub.CreatedAt.UTC(), RetainUntil: now.AddDate(s.RetentionYears, 0, 0),
			}); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Service) insertRecord(ctx context.Context, tx *database.Tx, r models.PaymentRecord) error {
	now := s.now().UTC()

	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO payment_records
(subject_ref, provider, provider_ref, amount_cents, currency, paid_at, retain_until, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.SubjectRef, r.Provider, r.ProviderRef, r.AmountCents, r.Currency, r.PaidAt, r.RetainUntil, now, now)

	return err
}

// pruneRetained removes the payment records whose retention ended and the token
// tombstones old enough that no device can still hold the token.
func (s *Service) pruneRetained(ctx context.Context, now time.Time) (int, error) {
	total := 0

	for _, q := range []struct {
		sql string
		arg time.Time
	}{
		{"DELETE FROM payment_records WHERE retain_until < ?", now},
		{"DELETE FROM purged_tokens WHERE created_at < ?", now.Add(-tombstoneRetention)},
	} {
		res, err := s.DB.ExecContext(ctx, q.sql, q.arg)
		if err != nil {
			return total, err
		}

		n, err := res.RowsAffected()
		if err != nil {
			return total, err
		}

		total += int(n)
	}

	return total, nil
}
