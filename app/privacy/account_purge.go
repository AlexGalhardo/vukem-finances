package privacy

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// PurgeAccount erases one account whose grace period ended. In the workspaces
// that remain it removes what is private to the person (assistant conversations,
// notifications and their preferences, API tokens, OAuth grants, sync devices,
// pairing codes and the Telegram link), anonymizes the author references that
// stay on shared rows, and takes the membership; the shared financial rows are
// kept. Then the user row goes, with its sign-in methods.
//
// It waits (errPurgeBlocked) while a workspace that entered its grace period
// with the account is still there, for instance because its subscription could
// not be cancelled: the account stays in the same state, retried by the next run.
func (s *Service) PurgeAccount(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("privacy: refusing to purge user %d", id)
	}

	u, err := orm.QueryOf[models.User](s.DB).Find(ctx, id)
	if errors.Is(err, orm.ErrNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	if !u.PendingDeletion() || !u.PurgeAt.Valid || u.PurgeAt.V.After(s.now()) {
		return errors.New("privacy: the account is not due for purge")
	}

	waiting, err := orm.QueryOf[models.Workspace](s.DB).Where("state", models.StatePendingDeletion).
		Where("deletion_requested_by", id).Where("deletion_requested_at", u.DeletionRequestedAt.V).Count(ctx)
	if err != nil {
		return err
	}

	if waiting > 0 {
		s.Log.ErrorContext(ctx, "alert: an account is due for purge but a workspace that goes with it is still waiting",
			"user_ref", s.Ref("user", id))

		return errPurgeBlocked
	}

	ref := s.Ref("user", id)

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		uid := strconv.FormatInt(id, 10)

		statements := []struct {
			sql  string
			args []any
		}{
			// Private to the person, in every workspace that remains.
			{"DELETE FROM assistant_messages WHERE user_id = ?", []any{id}},
			{"DELETE FROM assistant_conversations WHERE user_id = ?", []any{id}},
			{"DELETE FROM notification_preferences WHERE user_id = ?", []any{id}},
			{"DELETE FROM notifications WHERE user_id = ?", []any{id}},
			{"DELETE FROM sync_devices WHERE user_id = ?", []any{id}},
			{"DELETE FROM pairing_codes WHERE user_id = ?", []any{id}},
			{"DELETE FROM oauth_codes WHERE grant_id IN (SELECT grant_id FROM mcp_grants WHERE user_id = ?)", []any{id}},
			{"DELETE FROM oauth_refresh_tokens WHERE grant_id IN (SELECT grant_id FROM mcp_grants WHERE user_id = ?)", []any{id}},
			{"DELETE FROM oauth_grants WHERE id IN (SELECT grant_id FROM mcp_grants WHERE user_id = ?)", []any{id}},
			{"DELETE FROM mcp_grants WHERE user_id = ?", []any{id}},
			{"DELETE FROM access_tokens WHERE user_id = ?", []any{uid}},
			{"DELETE FROM telegram_pending WHERE user_id = ?", []any{id}},
			{"DELETE FROM telegram_link_codes WHERE user_id = ?", []any{id}},
			{"DELETE FROM telegram_links WHERE user_id = ?", []any{id}},
			// The author of rows that stay is forgotten, the rows are kept.
			{"UPDATE audit_events SET actor_user_id = 0 WHERE actor_user_id = ?", []any{id}},
			{"UPDATE ai_credentials SET created_by = 0 WHERE created_by = ?", []any{id}},
			{"UPDATE workspace_ai_settings SET enabled_by = 0 WHERE enabled_by = ?", []any{id}},
			{"UPDATE pix_payments SET created_by = 0 WHERE created_by = ?", []any{id}},
			{"UPDATE workspaces SET deletion_requested_by = 0 WHERE deletion_requested_by = ?", []any{id}},
			{"UPDATE admin_audit_logs SET actor_id = '0' WHERE actor_id = ?", []any{uid}},
			{"DELETE FROM admin_audit_logs WHERE resource = 'users' AND record_id = ?", []any{uid}},
			// What names the person by address.
			{"DELETE FROM invites WHERE lower(email) = lower(?)", []any{u.Email}},
			{"DELETE FROM password_resets WHERE lower(email) = lower(?)", []any{u.Email}},
			{"DELETE FROM email_suppressions WHERE lower(address) = lower(?)", []any{u.Email}},
			// Their place in the workspaces and how they sign in.
			{"DELETE FROM memberships WHERE user_id = ?", []any{id}},
			{"DELETE FROM remember_tokens WHERE user_id = ?", []any{id}},
			{"DELETE FROM social_accounts WHERE user_id = ?", []any{id}},
			{"DELETE FROM users WHERE id = ?", []any{id}},
		}

		for _, st := range statements {
			if _, err := tx.ExecContext(ctx, st.sql, st.args...); err != nil {
				return fmt.Errorf("purging an account: %w", err)
			}
		}

		return s.record(ctx, tx, EventAccountPurged, ref)
	})
	if err != nil {
		return err
	}

	s.Log.InfoContext(ctx, "account purged", "user_ref", ref)

	return nil
}
