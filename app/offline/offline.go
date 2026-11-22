// Package offline holds the rules that make the personal edition offline-first
// (decision D-027): the desktop app and the local Docker stack boot, migrate,
// render and run every core feature with no network. The edition is read here
// and nowhere else for these rules; see docs/offline.md for the inventory of
// every outbound path and what each does in personal.
package offline

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/analytics"
	"github.com/AlexGalhardo/vukem/mail"

	"github.com/AlexGalhardo/vukem-finances/config"
)

// MailTimeout caps one delivery attempt in the personal edition, so a mail
// server that does not answer cannot hold a request that sends mail inline
// (QUEUE_CONNECTION=sync, the default).
const MailTimeout = 10 * time.Second

// Gate applies the offline rules when the edition is personal. Build it once
// per application with NewGate, list Provider among the providers and
// ViewFuncs among the view functions.
type Gate struct {
	personal atomic.Bool
}

// NewGate returns a Gate that does nothing until its provider registers.
func NewGate() *Gate { return &Gate{} }

// Provider reads the edition once, while the application is built. In personal
// it drops the analytics settings (no tracker script, no CSP allowance, no
// verification route) and makes mail best effort. In saas it changes nothing.
// A bad APP_EDITION is left to the edition check, which stops the boot.
func (g *Gate) Provider() vukem.Provider {
	return vukem.ProviderFunc(func(app *vukem.App) error {
		edition, err := config.LoadEdition(app.Source)
		if err != nil || !edition.Personal() {
			return nil //nolint:nilerr // checkEdition reports the invalid value
		}

		g.personal.Store(true)
		app.Config.Analytics = analytics.Config{}

		return app.Instrument(vukem.Instrumentation{
			Mail: func(next mail.Transport) mail.Transport { return BestEffort(next, app.Log) },
		})
	})
}

// ViewFuncs adds "networkTags", which the root view asks before it renders the
// analytics slot: false in personal, whatever the environment says.
func (g *Gate) ViewFuncs() template.FuncMap {
	return template.FuncMap{"networkTags": func() bool { return !g.personal.Load() }}
}

// BestEffort wraps a mail transport so that a delivery that cannot happen (no
// mail server reachable, no network, a refused connection) is logged and
// dropped instead of failing the action that sent the mail. Only the kind of
// failure is logged, never an address or a body.
func BestEffort(next mail.Transport, log *slog.Logger) mail.Transport {
	return bestEffort{next: next, log: log}
}

type bestEffort struct {
	next mail.Transport
	log  *slog.Logger
}

// Send implements mail.Transport.
func (b bestEffort) Send(ctx context.Context, msg *mail.Message) error {
	ctx, cancel := context.WithTimeout(ctx, MailTimeout)
	defer cancel()

	err := b.next.Send(ctx, msg)
	if err == nil {
		return nil
	}

	b.log.WarnContext(ctx, "mail not delivered: personal mail is best effort", "kind", failureKind(err))

	return nil
}

func failureKind(err error) string {
	var netErr net.Error

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.As(err, &netErr):
		return "unreachable"
	default:
		return "refused"
	}
}
