package alerts

import (
	"context"
	"fmt"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// TelegramEnabled reports whether the person gets alerts of the workspace of
// ctx in their linked Telegram chat.
func (s *Service) TelegramEnabled(ctx context.Context, userID int64) (bool, error) {
	return workspaces.Scoped[models.NotificationPreference](ctx, s.DB).Where("user_id", userID).Where("telegram", true).Exists(ctx)
}

// SetTelegram turns the Telegram channel on or off for every kind of alert of
// the person in the workspace of ctx. A kind with no saved choice gets one made
// of the defaults of the role, so turning Telegram on changes nothing else.
// The person is the caller's own id: there is no way to name another.
func (s *Service) SetTelegram(ctx context.Context, userID int64, role string, on bool) error {
	saved, err := s.savedPreferences(ctx)
	if err != nil {
		return err
	}

	for _, kind := range Kinds() {
		where := func() orm.Query[models.NotificationPreference] {
			return workspaces.Scoped[models.NotificationPreference](ctx, s.DB).Where("user_id", userID).Where("kind", kind)
		}

		n, err := where().Update(ctx, orm.Set{"telegram": on})
		if err != nil {
			return err
		}

		if n > 0 {
			continue
		}

		var pref *Channels
		if v, ok := saved[prefKey{userID, kind}]; ok {
			pref = &v
		}

		c := effective(role, pref)
		row := models.NotificationPreference{UserID: userID, Kind: kind, InApp: c.InApp, Email: c.Email, Telegram: on}

		if err := orm.Create(ctx, s.DB, &row); err != nil {
			if !s.DB.UniqueViolation(err) {
				return err
			}

			if _, err := where().Update(ctx, orm.Set{"telegram": on}); err != nil {
				return err
			}
		}
	}

	return nil
}

// sendTelegram sends the Telegram message of the notifications that want one
// and have not been claimed, with the same claim as the mail: one run sends
// it. A failure is logged and the claim is released for the next run (inside
// the same window as the mail); it never fails the run, so a chat that blocked
// the bot cannot hold back the in-app list or the mail.
func (s *Service) sendTelegram(ctx context.Context, members []workspaces.Member) {
	if s.Telegram == nil {
		return
	}

	byUser := make(map[int64]Recipient, len(members))
	for _, m := range members {
		byUser[m.UserID] = Recipient{UserID: m.UserID, Name: m.Name, Email: m.Email, Role: m.Role}
	}

	prefs, err := s.savedPreferences(ctx)
	if err != nil {
		s.Log.WarnContext(ctx, "alerts: telegram preferences not read", "err", err.Error())

		return
	}

	now := s.now().UTC()

	pending, err := workspaces.Scoped[models.Notification](ctx, s.DB).
		Where("telegram", true).WhereNull("telegrammed_at").Where("created_at", ">=", time.Now().UTC().Add(-mailWindow)).
		OrderBy("id", orm.Asc).Limit(mailBatch).Get(ctx)
	if err != nil {
		s.Log.WarnContext(ctx, "alerts: telegram notifications not read", "err", err.Error())

		return
	}

	for _, n := range pending {
		claimed, err := workspaces.Scoped[models.Notification](ctx, s.DB).Where("id", n.ID).WhereNull("telegrammed_at").
			Update(ctx, orm.Set{"telegrammed_at": now})
		if err != nil || claimed == 0 {
			continue
		}

		r, ok := byUser[n.UserID]
		if !ok {
			continue // left the workspace
		}

		// An opt-out since the row was made wins, like for the mail.
		var pref *Channels
		if v, saved := prefs[prefKey{n.UserID, n.Kind}]; saved {
			pref = &v
		}

		if !effective(r.Role, pref).Telegram {
			continue
		}

		if err := s.Telegram.Deliver(ctx, n, r); err != nil {
			s.Log.WarnContext(ctx, fmt.Sprintf("alerts: telegram for notification %d not sent", n.ID), "err", err.Error())

			_, _ = workspaces.Scoped[models.Notification](ctx, s.DB).Where("id", n.ID).Update(ctx, orm.Set{"telegrammed_at": nil})
		}
	}
}
