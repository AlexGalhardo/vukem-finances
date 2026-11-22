package telegram

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram/lang"
)

// Channel delivers an alert to the private chats a person linked. It
// implements alerts.Channel; the alert pipeline sends through it only for a
// person who turned Telegram on, and never fails because of it.
type Channel struct{ Bot *Bot }

var _ alerts.Channel = (*Channel)(nil)

// Channel returns the alert channel of the bot.
func (b *Bot) Channel() *Channel { return &Channel{Bot: b} }

// Name implements alerts.Channel.
func (*Channel) Name() string { return "telegram" }

// Deliver implements alerts.Channel. A person with no linked chat, or who is
// no longer a member of the workspace of the alert, gets nothing and that is
// not an error. Delivering again after a failure may repeat a message to a
// chat that had already got it, never to another person.
func (c *Channel) Deliver(ctx context.Context, n models.Notification, r alerts.Recipient) error {
	b := c.Bot

	links, err := b.Store.LinksOf(ctx, r.UserID)
	if err != nil || len(links) == 0 {
		return err
	}

	ac, ok, err := b.accessTo(ctx, r.UserID, sql.Null[int64]{V: n.WorkspaceID, Valid: true})
	if err != nil || !ok {
		return err
	}

	var failed error

	for _, l := range links {
		if err := b.Client.SendMessage(ctx, l.ChatID, alertText(l.Language, n, ac.ws), nil); err != nil {
			failed = errors.Join(failed, err)
		}
	}

	return failed
}

// alertText renders a notification in the language of the chat. The params are
// the ones the mail reads; every name is escaped.
func alertText(lg string, n models.Notification, ws models.Workspace) string {
	p := decodeParams(n.Params)
	money := func(key string) string { return finance.FormatAmount(p.num(key), ws.Currency) }

	var text string

	switch n.Kind {
	case alerts.KindBudget80, alerts.KindBudget100:
		text = lang.T(lg, lang.AlertBudget, EscapeHTML(p.str("name")), int(p.num("threshold")), money("spent_cents"), money("limit_cents"))
	case alerts.KindOverallBudget100:
		text = lang.T(lg, lang.AlertOverall, money("spent_cents"), money("limit_cents"))
	case alerts.KindCardLimit80:
		text = lang.T(lg, lang.AlertCard, EscapeHTML(p.str("name")), money("used_cents"), money("limit_cents"))
	case alerts.KindStatementDue:
		text = lang.T(lg, lang.AlertStatement, EscapeHTML(p.str("name")), EscapeHTML(p.str("due_on")), money("remaining_cents"))
	case alerts.KindGoalReached:
		text = lang.T(lg, lang.AlertGoal, EscapeHTML(p.str("name")), money("target_cents"))
	case alerts.KindRecurrenceFailed:
		text = lang.T(lg, lang.AlertRecurring, EscapeHTML(p.str("name")), EscapeHTML(p.str("date")))
	default:
		text = lang.T(lg, lang.AlertGeneric, EscapeHTML(ws.Name))
	}

	return text + "\n" + EscapeHTML(ws.Name)
}

type params map[string]any

func decodeParams(raw string) params {
	var m map[string]any

	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()

	if err := dec.Decode(&m); err != nil {
		return params{}
	}

	return m
}

func (p params) str(k string) string {
	v, _ := p[k].(string)

	return v
}

func (p params) num(k string) int64 {
	n, ok := p[k].(json.Number)
	if !ok {
		return 0
	}

	v, err := strconv.ParseInt(n.String(), 10, 64)
	if err != nil {
		return 0
	}

	return v
}

// Attach gives the alert service the Telegram channel when the bot is on.
func Attach(b *Bot, al *alerts.Service) *alerts.Service {
	if b != nil {
		al.Telegram = b.Channel()
	}

	return al
}
