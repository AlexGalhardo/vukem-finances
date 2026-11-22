package alerts

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Recipient is who a notification is for.
type Recipient struct {
	UserID int64
	Name   string
	Email  string
	Role   string
}

// Channel delivers a notification to a person. Deliver may be called again
// for a notification after a failure, so it must be safe to repeat.
type Channel interface {
	Name() string
	Deliver(ctx context.Context, n models.Notification, r Recipient) error
}

// InApp is the list in the app. The stored row is the delivery (it shows
// while its in_app flag is set), so there is nothing left to do.
type InApp struct{}

// Name implements Channel.
func (InApp) Name() string { return "in_app" }

// Deliver implements Channel.
func (InApp) Deliver(context.Context, models.Notification, Recipient) error { return nil }

// Email sends the notification as a mail through the mail queue.
type Email struct {
	WS  *workspaces.Service
	URL string
}

// Name implements Channel.
func (*Email) Name() string { return "email" }

// errNoTemplate marks a kind that has no mail.
var errNoTemplate = errors.New("alerts: no mail for this kind")

// Deliver implements Channel. An address that is suppressed (a bounce) counts
// as delivered.
func (e *Email) Deliver(ctx context.Context, n models.Notification, r Recipient) error {
	ws, err := e.WS.Current(ctx)
	if err != nil {
		return err
	}

	p := decode(n.Params)

	switch n.Kind {
	case KindBudget80, KindBudget100, KindOverallBudget100:
		name := p.str("name")
		if n.Kind == KindOverallBudget100 {
			name = "Overall"
		}

		return e.WS.Notify(ctx, appmail.BudgetAlert{
			To: r.Email, Workspace: ws.Name, Category: name, Percent: int(p.num("threshold")),
			Spent: finance.FormatAmount(p.num("spent_cents"), ws.Currency),
			Limit: finance.FormatAmount(p.num("limit_cents"), ws.Currency),
			URL:   e.URL + "/budgets",
		})
	default:
		return errNoTemplate
	}
}

// params is the decoded JSON object of a notification.
type params map[string]any

func decode(raw string) params {
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
