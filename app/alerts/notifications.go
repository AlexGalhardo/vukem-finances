package alerts

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var errKind = errs.New(http.StatusUnprocessableEntity, "Choose one of the alert kinds.")

// ListLimit is how many notifications a list shows.
const ListLimit = 100

// own is the notifications the person sees in the workspace of ctx. Every
// read and write below starts from it, so a row of another person or another
// workspace is simply not there (a 404), never a 403.
func (s *Service) own(ctx context.Context, userID int64) orm.Query[models.Notification] {
	return workspaces.Scoped[models.Notification](ctx, s.DB).Where("user_id", userID).Where("in_app", true)
}

// List returns the newest notifications of the person; ReadAt tells the unread
// ones.
func (s *Service) List(ctx context.Context, userID int64, limit int) ([]models.Notification, error) {
	if limit <= 0 || limit > ListLimit {
		limit = ListLimit
	}

	return s.own(ctx, userID).OrderBy("id", orm.Desc).Limit(limit).Get(ctx)
}

// Unread counts the notifications the person has not read.
func (s *Service) Unread(ctx context.Context, userID int64) (int64, error) {
	return s.own(ctx, userID).WhereNull("read_at").Count(ctx)
}

// MarkRead marks one notification of the person as read; reading it twice is
// fine. Another person's id is a 404.
func (s *Service) MarkRead(ctx context.Context, userID, id int64) error {
	ok, err := s.own(ctx, userID).Where("id", id).Exists(ctx)
	if err != nil || !ok {
		return firstErr(err, errs.ErrNotFound)
	}

	_, err = s.own(ctx, userID).Where("id", id).WhereNull("read_at").Update(ctx, orm.Set{"read_at": s.now().UTC()})

	return err
}

// MarkAllRead marks every notification of the person as read.
func (s *Service) MarkAllRead(ctx context.Context, userID int64) error {
	_, err := s.own(ctx, userID).WhereNull("read_at").Update(ctx, orm.Set{"read_at": s.now().UTC()})

	return err
}

func firstErr(err, fallback error) error {
	if err != nil {
		return err
	}

	return fallback
}

// PreferenceRow is one kind with the person's choice. EmailAllowed is false
// for a viewer, who never gets mail.
type PreferenceRow struct {
	Kind         string `json:"kind"`
	InApp        bool   `json:"inApp"`
	Email        bool   `json:"email"`
	EmailAllowed bool   `json:"emailAllowed"`
}

// Preferences returns the choice of the person for every kind, saved or
// default.
func (s *Service) Preferences(ctx context.Context, userID int64, role string) ([]PreferenceRow, error) {
	saved, err := s.savedPreferences(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]PreferenceRow, 0, len(Kinds()))

	for _, kind := range Kinds() {
		var pref *Channels
		if v, ok := saved[prefKey{userID, kind}]; ok {
			pref = &v
		}

		c := effective(role, pref)
		out = append(out, PreferenceRow{Kind: kind, InApp: c.InApp, Email: c.Email, EmailAllowed: role != models.RoleViewer})
	}

	return out, nil
}

// PreferenceInput is a choice as the form sends it.
type PreferenceInput struct {
	Kind  string
	InApp bool
	Email bool
}

// SetPreferences saves the choices of the person; every kind is checked before
// anything is saved. The person is
// the caller's own id: there is no way to name another. A viewer's mail choice
// is stored off.
func (s *Service) SetPreferences(ctx context.Context, userID int64, role string, in []PreferenceInput) error {
	for _, p := range in {
		if !ValidKind(p.Kind) {
			return errKind
		}
	}

	for _, p := range in {
		email := p.Email && role != models.RoleViewer
		set := orm.Set{"in_app": p.InApp, "email": email}

		where := func() orm.Query[models.NotificationPreference] {
			return workspaces.Scoped[models.NotificationPreference](ctx, s.DB).Where("user_id", userID).Where("kind", p.Kind)
		}

		n, err := where().Update(ctx, set)
		if err != nil {
			return err
		}

		if n > 0 {
			continue
		}

		row := models.NotificationPreference{UserID: userID, Kind: p.Kind, InApp: p.InApp, Email: email}
		if err := orm.Create(ctx, s.DB, &row); err != nil {
			if !s.DB.UniqueViolation(err) {
				return err
			}

			// A concurrent save made the row first: this one is an edit.
			if _, err := where().Update(ctx, set); err != nil {
				return err
			}
		}
	}

	return nil
}

// View is a notification as the client reads it: the key and the params are
// rendered there, in the language of the person.
type View struct {
	ID        int64           `json:"id"`
	Kind      string          `json:"kind"`
	TitleKey  string          `json:"titleKey"`
	Params    json.RawMessage `json:"params"`
	CreatedAt string          `json:"createdAt"`
	Read      bool            `json:"read"`
}

// Views lists the newest notifications of the person for the client.
func (s *Service) Views(ctx context.Context, userID int64, limit int) ([]View, error) {
	rows, err := s.List(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	out := make([]View, 0, len(rows))

	for _, n := range rows {
		params := json.RawMessage(n.Params)
		if !json.Valid(params) {
			params = json.RawMessage("{}")
		}

		out = append(out, View{
			ID: n.ID, Kind: n.Kind, TitleKey: n.TitleKey, Params: params,
			CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339), Read: n.ReadAt.Valid,
		})
	}

	return out, nil
}
