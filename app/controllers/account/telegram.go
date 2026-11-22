package account

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// maxTelegramLinks is how many chats one person links at a time.
const maxTelegramLinks = 5

var errTooManyChats = errs.New(http.StatusUnprocessableEntity, "Disconnect a chat before connecting another one.")

// TelegramHandlers serves the Telegram settings page. Everything on it is the
// signed-in person's own: no handler reads a user id or a chat id from the
// request, the only way to link a chat is the single-use code shown here, and
// a revoke names a link of the person or is a 404.
type TelegramHandlers struct {
	Bot    *telegram.Bot
	Alerts *alerts.Service
}

// NewTelegram returns the handlers; bot must not be nil (the routes exist only while the bot is on).
func NewTelegram(bot *telegram.Bot, al *alerts.Service) *TelegramHandlers {
	return &TelegramHandlers{Bot: bot, Alerts: al}
}

func me(c *web.Ctx) (models.User, error) { return auth.UserFrom[models.User](c.Context(), "web") }

func (h *TelegramHandlers) props(c *web.Ctx, u models.User, created *pages.TelegramCode) (pages.TelegramProps, error) {
	ctx := c.Context()

	links, err := h.Bot.Store.LinksOf(ctx, u.ID)
	if err != nil {
		return pages.TelegramProps{}, err
	}

	views := make([]pages.TelegramLinkView, 0, len(links))

	for _, l := range links {
		hint := strconv.FormatInt(l.ChatID, 10)
		if len(hint) > 4 {
			hint = hint[len(hint)-4:]
		}

		v := pages.TelegramLinkView{
			ID: l.ID, ChatHint: hint, Language: l.Language, LinkedAt: l.CreatedAt.UTC().Format(time.RFC3339),
		}
		if l.LastSeenAt.Valid {
			v.LastSeenAt = l.LastSeenAt.V.UTC().Format(time.RFC3339)
		}

		views = append(views, v)
	}

	on, err := h.Alerts.TelegramEnabled(ctx, u.ID)
	if err != nil {
		return pages.TelegramProps{}, err
	}

	return pages.TelegramProps{
		Username: h.Bot.Cfg.Username, Links: views, AlertsOn: on, MaxLinks: maxTelegramLinks, Created: created,
	}, nil
}

// Show renders the page.
func (h *TelegramHandlers) Show(c *web.Ctx) error {
	u, err := me(c)
	if err != nil {
		return err
	}

	p, err := h.props(c, u, nil)
	if err != nil {
		return err
	}

	return pages.Telegram.Render(c, p)
}

// Connect makes a single-use code for the signed-in person and answers with
// the page that shows its deep link once: no redirect and no flash, so the code
// never goes through the session, and the response is not cached.
func (h *TelegramHandlers) Connect(c *web.Ctx) error {
	ctx := c.Context()

	u, err := me(c)
	if err != nil {
		return err
	}

	links, err := h.Bot.Store.LinksOf(ctx, u.ID)
	if err != nil {
		return err
	}

	if len(links) >= maxTelegramLinks {
		return back(c, errTooManyChats)
	}

	ws, _ := workspaces.TenantID(ctx)

	code, expires, err := h.Bot.Store.NewCode(ctx, u.ID, ws)
	if err != nil {
		return err
	}

	p, err := h.props(c, u, &pages.TelegramCode{
		Link: "https://t.me/" + h.Bot.Cfg.Username + "?start=" + code, Code: code,
		ExpiresAt: expires.UTC().Format(time.RFC3339), Seconds: int(time.Until(expires).Seconds()),
	})
	if err != nil {
		return err
	}

	c.Writer().Header().Set("Cache-Control", "no-store")

	return pages.Telegram.Render(c, p)
}

// Revoke ends a link of the signed-in person.
func (h *TelegramHandlers) Revoke(c *web.Ctx) error {
	u, err := me(c)
	if err != nil {
		return err
	}

	id, err := c.ParamInt("link")
	if err != nil {
		return err
	}

	ok, err := h.Bot.Store.RevokeOf(c.Context(), u.ID, id)
	if err != nil {
		return err
	}

	if !ok {
		return errs.ErrNotFound
	}

	return flashAndGo(c, "Chat disconnected.")
}

// SetAlerts turns the Telegram channel of the alerts on or off for the person.
func (h *TelegramHandlers) SetAlerts(c *web.Ctx) error {
	u, err := me(c)
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.TelegramAlertsSchema)
	if err != nil {
		return err
	}

	if in.Enabled != "true" && in.Enabled != "false" {
		return errs.New(http.StatusUnprocessableEntity, "The switch must be on or off.")
	}

	if err := h.Alerts.SetTelegram(c.Context(), u.ID, workspaces.Role(c.Context()), in.Enabled == "true"); err != nil {
		return err
	}

	return flashAndGo(c, "Preferences saved.")
}

func flashAndGo(c *web.Ctx, message string) error {
	inertia.Flash(c, "success", message)

	return c.Redirect("/settings/telegram")
}

func back(c *web.Ctx, err error) error {
	if e, ok := errors.AsType[*errs.HTTPError](err); ok && !c.WantsJSON() {
		inertia.Flash(c, "error", e.Message)

		return c.Redirect("/settings/telegram")
	}

	return err
}
