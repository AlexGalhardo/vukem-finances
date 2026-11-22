package telegram

import (
	"context"
	"crypto/rand"
	"time"
)

const (
	// leaseTTL is how long the polling lease lasts without a renewal: longer
	// than one long poll, so a healthy worker never loses it between polls.
	leaseTTL = 2 * time.Minute
	// retryAfterError waits before polling again after a failure.
	retryAfterError = 3 * time.Second
	// retryAfterHeld waits while another instance polls.
	retryAfterHeld = 10 * time.Second
)

// NewOwner makes the name a polling worker takes the lease with.
func NewOwner() string { return rand.Text() }

// PollOnce takes or renews the lease, long-polls once from the stored offset
// and handles the updates one by one, storing the next offset after each, so a
// restart in the middle of a batch resumes at the first update not handled.
// leased is false while another instance holds the lease.
func (b *Bot) PollOnce(ctx context.Context) (handled int, leased bool, err error) {
	offset, ok, err := b.Store.AcquireLease(ctx, b.Owner, leaseTTL)
	if err != nil || !ok {
		return 0, false, err
	}

	updates, err := b.Client.GetUpdates(ctx, offset)
	if err != nil {
		return 0, true, err
	}

	for _, u := range updates {
		if ctx.Err() != nil {
			break
		}

		if err := b.HandleUpdate(ctx, u); err != nil {
			return handled, true, err
		}

		handled++

		// A cancelled context must not lose the offset of an update just handled.
		if _, err := b.Store.SaveOffset(context.WithoutCancel(ctx), b.Owner, u.UpdateID+1); err != nil {
			return handled, true, err
		}

		if b.afterUpdate != nil {
			b.afterUpdate(u.UpdateID)
		}
	}

	return handled, true, nil
}

// Run is the polling worker, a vukem Runner: it loops until ctx is done and
// frees the lease on the way out. Webhook mode never starts it.
func (b *Bot) Run(ctx context.Context) error {
	defer func() {
		if err := b.Store.ReleaseLease(context.WithoutCancel(ctx), b.Owner); err != nil {
			b.Log.Warn("telegram: lease not released", "err", err.Error())
		}
	}()

	// Telegram refuses getUpdates while a webhook is set.
	if err := b.Client.DeleteWebhook(ctx); err != nil && ctx.Err() == nil {
		b.Log.Warn("telegram: webhook not removed", "err", err.Error())
	}

	for ctx.Err() == nil {
		_, leased, err := b.PollOnce(ctx)

		switch {
		case err != nil && ctx.Err() == nil:
			b.Log.Warn("telegram: poll failed", "err", err.Error())
			sleep(ctx, retryAfterError)
		case !leased:
			sleep(ctx, retryAfterHeld)
		}
	}

	return nil
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// SetAfterUpdate sets a function called after each update of a poll batch.
// Tests use it to stop a poll in the middle of a batch.
func (b *Bot) SetAfterUpdate(fn func(updateID int64)) { b.afterUpdate = fn }
