package privacy

import (
	"context"
	"net/http"
	"strings"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

var (
	errAccountPending  = errs.New(http.StatusConflict, "This account is already scheduled for deletion.")
	errAccountNotQueue = errs.New(http.StatusConflict, "This account is not scheduled for deletion.")
)

// onlyOwnerError builds the refusal of an account deletion while the person is
// the only owner of a workspace other people still use. It is a 409 that names
// the workspaces, so the page can say which ones.
func onlyOwnerError(names []string) error {
	return errs.New(http.StatusConflict,
		"You are the only owner of "+strings.Join(names, ", ")+
			", which other people use: transfer the ownership or delete the workspace first.").
		WithExtension("workspaces", names)
}

// RequestAccountDeletion schedules the deletion of the account of u; the caller
// has already re-authenticated the person.
//
// It is refused, changing nothing, while u is the only owner of a workspace with
// other members (a workspace already waiting to be deleted does not count): they
// must transfer the ownership or delete that workspace first. Workspaces where
// u is the only member enter their own grace period with the account. In the
// workspaces that remain the purge removes what is private to u and keeps the
// shared financial rows.
//
// Every credential of u ends at once, and the subscriptions of the workspaces
// that go with the account are cancelled.
func (s *Service) RequestAccountDeletion(ctx context.Context, u models.User) error {
	now := s.now().UTC()
	purgeAt := now.Add(GracePeriod)

	var alone []int64

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		fresh, err := orm.QueryOf[models.User](tx).Find(ctx, u.ID)
		if err != nil {
			return err
		}

		if fresh.PendingDeletion() {
			return errAccountPending
		}

		memberships, err := orm.QueryOf[models.Membership](tx).Where("user_id", u.ID).Get(ctx)
		if err != nil {
			return err
		}

		var blockers []string

		for _, m := range memberships {
			ws, err := orm.QueryOf[models.Workspace](tx).Find(ctx, m.WorkspaceID)
			if err != nil {
				return err
			}

			if ws.PendingDeletion() {
				continue
			}

			members, err := orm.QueryOf[models.Membership](tx).Where("workspace_id", ws.ID).Count(ctx)
			if err != nil {
				return err
			}

			switch {
			case members <= 1:
				alone = append(alone, ws.ID)
			case m.Role == models.RoleOwner:
				blockers = append(blockers, ws.Name)
			}
		}

		if len(blockers) > 0 {
			return onlyOwnerError(blockers)
		}

		for _, id := range alone {
			if _, err := orm.QueryOf[models.Workspace](tx).Where("id", id).Update(ctx, orm.Set{
				"state": models.StatePendingDeletion, "purge_at": purgeAt, "deletion_requested_at": now,
				"deletion_requested_by": u.ID, "purge_error": "",
			}); err != nil {
				return err
			}

			if err := s.record(ctx, tx, EventWorkspaceRequested, s.Ref("workspace", id)); err != nil {
				return err
			}
		}

		if _, err := orm.QueryOf[models.User](tx).Where("id", u.ID).Update(ctx, orm.Set{
			"deletion_requested_at": now, "purge_at": purgeAt,
		}); err != nil {
			return err
		}

		return s.record(ctx, tx, EventAccountRequested, s.Ref("user", u.ID))
	})
	if err != nil {
		return err
	}

	if s.Access != nil {
		if err := s.Access.RevokeAllAccess(ctx, u.ID); err != nil {
			// The request stands. Say so loudly: a credential outlived it.
			s.Log.ErrorContext(ctx, "alert: the credentials of an account waiting for deletion were not all revoked",
				"user_ref", s.Ref("user", u.ID), "err", err)
		}
	}

	s.dropTelegram(ctx, u.ID)

	for _, id := range alone {
		s.cancelBilling(ctx, id)
	}

	// A mail failure never blocks the deletion (the personal edition may have no network).
	if err := s.WS.Notify(ctx, appmail.AccountDeletionScheduled{
		To: u.Email, Name: u.Name, PurgeDate: dateOf(purgeAt), URL: s.WS.URL + "/login",
	}); err != nil {
		s.Log.WarnContext(ctx, "privacy: the account deletion mail was not sent", "err", err)
	}

	return nil
}

// CancelAccountDeletion brings the account back, and the workspaces that
// entered their grace period with it. A workspace the person asked to delete on
// its own keeps waiting. The credentials revoked by the request stay revoked:
// the person signs in again and makes new ones.
func (s *Service) CancelAccountDeletion(ctx context.Context, u models.User) error {
	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		fresh, err := orm.QueryOf[models.User](tx).Find(ctx, u.ID)
		if err != nil {
			return err
		}

		if !fresh.PendingDeletion() {
			return errAccountNotQueue
		}

		// Its workspaces share this date, and the purge may have started on them.
		if fresh.PurgeAt.Valid && !fresh.PurgeAt.V.After(s.now()) {
			return errGraceEnded
		}

		// The workspaces of this request carry the very timestamp of the account.
		ws, err := orm.QueryOf[models.Workspace](tx).Where("state", models.StatePendingDeletion).
			Where("deletion_requested_by", u.ID).Where("deletion_requested_at", fresh.DeletionRequestedAt.V).Get(ctx)
		if err != nil {
			return err
		}

		for _, w := range ws {
			if _, err := orm.QueryOf[models.Workspace](tx).Where("id", w.ID).Update(ctx, orm.Set{
				"state": models.StateActive, "purge_at": nil, "deletion_requested_at": nil,
				"deletion_requested_by": 0, "purge_error": "",
			}); err != nil {
				return err
			}

			if err := s.record(ctx, tx, EventWorkspaceCancelled, s.Ref("workspace", w.ID)); err != nil {
				return err
			}
		}

		if _, err := orm.QueryOf[models.User](tx).Where("id", u.ID).Update(ctx, orm.Set{
			"deletion_requested_at": nil, "purge_at": nil,
		}); err != nil {
			return err
		}

		return s.record(ctx, tx, EventAccountCancelled, s.Ref("user", u.ID))
	})
}

// dropTelegram unlinks the Telegram chats of the person and forgets their codes
// and drafts: a chat is a credential too.
func (s *Service) dropTelegram(ctx context.Context, userID int64) {
	for _, table := range []string{"telegram_pending", "telegram_link_codes", "telegram_links"} {
		if _, err := s.DB.ExecContext(ctx, "DELETE FROM "+table+" WHERE user_id = "+s.DB.Dialect().Placeholder(1), userID); err != nil {
			s.Log.WarnContext(ctx, "privacy: could not unlink Telegram", "table", table, "err", err)
		}
	}
}
