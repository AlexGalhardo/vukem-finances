package ledger

import (
	"net/http"
	"slices"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var errBadKind = errs.New(http.StatusUnprocessableEntity, "Choose one of the alert kinds.")

// me is the signed-in person. Notifications and their choices are theirs
// alone, so no handler below reads a user id from the request.
func me(c *web.Ctx) (models.User, error) { return auth.UserFrom[models.User](c.Context(), "web") }

// Notifications is the full list of the person and their channel choices.
func (h *Handlers) Notifications(c *web.Ctx) error {
	u, err := me(c)
	if err != nil {
		return err
	}

	list, err := h.Alerts.Views(c.Context(), u.ID, alerts.ListLimit)
	if err != nil {
		return err
	}

	prefs, err := h.Alerts.Preferences(c.Context(), u.ID, workspaces.Role(c.Context()))
	if err != nil {
		return err
	}

	return pages.Notifications.Render(c, pages.NotificationsProps{Notifications: list, Preferences: prefs})
}

// ReadNotification marks one notification of the person as read.
func (h *Handlers) ReadNotification(c *web.Ctx) error {
	u, err := me(c)
	if err != nil {
		return err
	}

	nid, err := c.ParamInt("notification")
	if err != nil {
		return err
	}

	if err := h.Alerts.MarkRead(c.Context(), u.ID, nid); err != nil {
		return err
	}

	return c.Back("/notifications")
}

// ReadAllNotifications marks every notification of the person as read.
func (h *Handlers) ReadAllNotifications(c *web.Ctx) error {
	u, err := me(c)
	if err != nil {
		return err
	}

	if err := h.Alerts.MarkAllRead(c.Context(), u.ID); err != nil {
		return err
	}

	return c.Back("/notifications")
}

// SavePreferences saves the channel choices of the person.
func (h *Handlers) SavePreferences(c *web.Ctx) error {
	u, err := me(c)
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.NotificationPreferencesSchema)
	if err != nil {
		return err
	}

	for _, kind := range slices.Concat(in.InApp, in.Email) {
		if !alerts.ValidKind(kind) {
			return back(c, "/notifications", errBadKind)
		}
	}

	rows := make([]alerts.PreferenceInput, 0, len(alerts.Kinds()))
	for _, kind := range alerts.Kinds() {
		rows = append(rows, alerts.PreferenceInput{
			Kind: kind, InApp: slices.Contains(in.InApp, kind), Email: slices.Contains(in.Email, kind),
		})
	}

	if err := h.Alerts.SetPreferences(c.Context(), u.ID, workspaces.Role(c.Context()), rows); err != nil {
		return back(c, "/notifications", err)
	}

	return done(c, "/notifications", "Preferences saved.")
}
