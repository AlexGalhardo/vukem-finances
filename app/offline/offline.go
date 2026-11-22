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
	"syscall"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/analytics"
	"github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/queue"

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
// verification route) and makes a mail that cannot reach its server non-blocking (see BestEffort). In saas it changes nothing.
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
			Mail: func(next mail.Transport) mail.Transport { return BestEffort(next, app.Log, queueIsInline(app)) },
		})
	})
}

// ViewFuncs adds "networkTags", which the root view asks before it renders the
// analytics slot: false in personal, whatever the environment says.
func (g *Gate) ViewFuncs() template.FuncMap {
	return template.FuncMap{"networkTags": func() bool { return !g.personal.Load() }}
}

// queueIsInline reports whether the queue runs jobs during the call that
// dispatches them (QUEUE_CONNECTION=sync, the default), so a failed job is not
// retried and its error would reach the person's request.
func queueIsInline(app *vukem.App) func() bool {
	return func() bool {
		q, err := app.Queue()
		if err != nil {
			return true
		}

		in, ok := q.Driver().(queue.Inline)

		return ok && in.Inline()
	}
}

// BestEffort wraps a mail transport so that a delivery that cannot happen for
// lack of a connection (no DNS, a refused or timed out connection, an
// unreachable network) never fails the action that sent the mail.
//
//   - A connectivity failure on an inline queue (nobody will retry it) is logged
//     and dropped: the request answers as if the mail had gone, so the answer
//     of a flow that mails only some people (a reset link for an address that
//     exists) stays the same for everyone.
//   - A connectivity failure on a real queue (database, memory) is returned: the
//     job is retried with the queue's backoff and shows up as failed, and the
//     request had already answered when the job ran.
//   - Any other failure (authentication, TLS, a refused recipient, a bad
//     message) is returned as it is: it is a configuration or content error that
//     the person must see, not a missing network.
//
// Only the kind of failure is logged, never an address or a body.
func BestEffort(next mail.Transport, log *slog.Logger, inline func() bool) mail.Transport {
	// The log and the in-memory transports never touch a network: leave them as they are,
	// which also lets the test harness read what was sent.
	switch next.(type) {
	case mail.Log, *mail.Array:
		return next
	}

	return bestEffort{next: next, log: log, inline: inline}
}

type bestEffort struct {
	next   mail.Transport
	log    *slog.Logger
	inline func() bool
}

// Send implements mail.Transport.
func (b bestEffort) Send(ctx context.Context, msg *mail.Message) error {
	ctx, cancel := context.WithTimeout(ctx, MailTimeout)
	defer cancel()

	err := b.next.Send(ctx, msg)
	if err == nil || !IsConnectivity(err) {
		return err
	}

	b.log.WarnContext(ctx, "mail not delivered: no connection to the mail server", "kind", failureKind(err))

	if b.inline() {
		return nil
	}

	return err
}

// IsConnectivity reports whether err means the network or the server could not
// be reached: DNS, a refused or unreachable connection, a timeout. A server that
// answered and said no (authentication, TLS, recipient, content) is not one.
func IsConnectivity(err error) bool {
	var (
		dns  *net.DNSError
		op   *net.OpError
		nerr net.Error
	)

	switch {
	case errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, syscall.ECONNREFUSED), errors.Is(err, syscall.ENETUNREACH),
		errors.Is(err, syscall.EHOSTUNREACH), errors.Is(err, syscall.ETIMEDOUT):
		return true
	case errors.As(err, &dns):
		return true
	case errors.As(err, &op):
		return op.Op == "dial"
	case errors.As(err, &nerr):
		return nerr.Timeout()
	default:
		return false
	}
}

func failureKind(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, syscall.ECONNREFUSED):
		return "refused"
	default:
		return "unreachable"
	}
}
