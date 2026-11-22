package alerts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

const (
	// mailWindow is how old a notification can be and still be mailed: a mail
	// that kept failing for two days is not sent late.
	mailWindow = 48 * time.Hour
	// mailBatch bounds the mails of one run.
	mailBatch = 200
)

// Run makes the notifications the workspace of ctx is due and sends the mail
// that is still owed. It is safe to run twice, or at the same time as itself:
// a notification is unique per (person, kind, subject), so the second insert
// is refused, and a mail is claimed with a conditional update before it is
// sent, so one run sends it. A claim is released when the send fails and the
// next run tries again; a crash between the claim and the send loses that
// one mail, the price of never sending it twice.
func (s *Service) Run(ctx context.Context) error {
	cands, evalErr := s.Evaluate(ctx)

	members, err := s.WS.Members(ctx)
	if err != nil {
		return errors.Join(evalErr, err)
	}

	saved, err := s.savedPreferences(ctx)
	if err != nil {
		return errors.Join(evalErr, err)
	}

	failed := evalErr

	for _, c := range cands {
		raw, err := json.Marshal(c.Params)
		if err != nil {
			failed = errors.Join(failed, err)

			continue
		}

		for _, m := range members {
			var pref *Channels
			if v, ok := saved[prefKey{m.UserID, c.Kind}]; ok {
				pref = &v
			}

			ch := effective(m.Role, pref)
			if !ch.InApp && !ch.Email {
				continue
			}

			n := models.Notification{
				UserID: m.UserID, Kind: c.Kind, SubjectKey: c.Subject, TitleKey: TitleKey(c.Kind),
				Params: string(raw), InApp: ch.InApp, Email: ch.Email,
			}

			// The unique key is the dedupe: an existing row is the answer.
			if err := orm.Create(ctx, s.DB, &n); err != nil && !s.DB.UniqueViolation(err) {
				failed = errors.Join(failed, err)
			}
		}
	}

	return errors.Join(failed, s.sendMails(ctx, members))
}

// sendMails sends the mail of the notifications that want one and have not
// been claimed.
func (s *Service) sendMails(ctx context.Context, members []workspaces.Member) error {
	byUser := make(map[int64]Recipient, len(members))
	for _, m := range members {
		byUser[m.UserID] = Recipient{UserID: m.UserID, Name: m.Name, Email: m.Email, Role: m.Role}
	}

	prefs, err := s.savedPreferences(ctx)
	if err != nil {
		return err
	}

	// created_at is stamped by the ORM from the wall clock, so the window is
	// measured on it too, not on the app clock a test may have moved.
	now := s.now().UTC()

	pending, err := workspaces.Scoped[models.Notification](ctx, s.DB).
		Where("email", true).WhereNull("emailed_at").Where("created_at", ">=", time.Now().UTC().Add(-mailWindow)).
		OrderBy("id", orm.Asc).Limit(mailBatch).Get(ctx)
	if err != nil {
		return err
	}

	var failed error

	for _, n := range pending {
		claimed, err := workspaces.Scoped[models.Notification](ctx, s.DB).Where("id", n.ID).WhereNull("emailed_at").
			Update(ctx, orm.Set{"emailed_at": now})
		if err != nil {
			failed = errors.Join(failed, err)

			continue
		}

		if claimed == 0 {
			continue // another run has it
		}

		r, ok := byUser[n.UserID]
		if !ok {
			continue // left the workspace: no mail
		}

		// The row holds the choice from when it was made; an opt-out since then
		// wins, and the claim above keeps the dropped mail from coming back if
		// the person opts in again.
		var pref *Channels
		if v, saved := prefs[prefKey{n.UserID, n.Kind}]; saved {
			pref = &v
		}

		if !effective(r.Role, pref).Email {
			continue
		}

		if err := s.Email.Deliver(ctx, n, r); err != nil {
			if errors.Is(err, errNoTemplate) {
				continue
			}

			_, relErr := workspaces.Scoped[models.Notification](ctx, s.DB).Where("id", n.ID).
				Update(ctx, orm.Set{"emailed_at": nil})
			failed = errors.Join(failed, fmt.Errorf("alerts: mail for notification %d: %w", n.ID, err), relErr)
		}
	}

	return failed
}

// RunAll runs every workspace, each under its own tenant; one that fails does
// not starve the others.
func (s *Service) RunAll(ctx context.Context) error {
	all, err := orm.QueryOf[models.Workspace](s.DB).Get(ctx)
	if err != nil {
		return err
	}

	var failed error

	for _, ws := range all {
		if err := s.Run(workspaces.WithTenant(ctx, ws.ID)); err != nil {
			failed = errors.Join(failed, fmt.Errorf("alerts of workspace %d: %w", ws.ID, err))
		}
	}

	return failed
}
