package alerts

import (
	"context"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Channels says which ways a person wants a kind of alert.
type Channels struct {
	InApp bool `json:"inApp"`
	Email bool `json:"email"`
	// Telegram is off until the person turns it on (docs/telegram.md): there
	// is no default for a role.
	Telegram bool `json:"telegram"`
}

// Default is what a person with this role gets when they never chose: the
// in-app list for everyone, mail for owners and admins. A viewer gets no mail.
func Default(role string) Channels {
	return Channels{InApp: true, Email: role == models.RoleOwner || role == models.RoleAdmin}
}

// effective applies a saved choice over the default. A viewer never gets
// mail, whatever a saved row says.
func effective(role string, saved *Channels) Channels {
	c := Default(role)
	if saved != nil {
		c = *saved
	}

	if role == models.RoleViewer {
		c.Email = false
	}

	return c
}

type prefKey struct {
	user int64
	kind string
}

// savedPreferences reads every saved choice of the workspace of ctx.
func (s *Service) savedPreferences(ctx context.Context) (map[prefKey]Channels, error) {
	rows, err := workspaces.Scoped[models.NotificationPreference](ctx, s.DB).Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make(map[prefKey]Channels, len(rows))
	for _, r := range rows {
		out[prefKey{r.UserID, r.Kind}] = Channels{InApp: r.InApp, Email: r.Email, Telegram: r.Telegram}
	}

	return out, nil
}
