package telegram

import (
	"context"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/ratelimit"

	"github.com/AlexGalhardo/vukem-finances/config"
)

// dbCounter is a ratelimit.Counter kept in the database, so a restart of the
// app does not give a chat a fresh budget. It does not depend on
// RATE_LIMIT_STORE or CACHE_STORE: the bot limits are durable on their own.
type dbCounter struct {
	db  *database.DB
	now func() time.Time
}

// Increment implements ratelimit.Counter. A delta of zero reads.
func (c dbCounter) Increment(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	ph := c.db.Dialect().Placeholder
	now := c.now().UTC()

	if delta == 0 {
		var n int64

		err := c.db.QueryRowContext(ctx,
			"SELECT count FROM telegram_counters WHERE bucket = "+ph(1)+" AND expires_at > "+ph(2), key, now).Scan(&n)
		if err != nil {
			return 0, nil //nolint:nilerr // no live row is a count of zero
		}

		return n, nil
	}

	var n int64

	err := c.db.Tx(ctx, func(tx *database.Tx) error {
		res, err := tx.ExecContext(ctx,
			"UPDATE telegram_counters SET count = count + "+ph(1)+", updated_at = "+ph(2)+
				" WHERE bucket = "+ph(3)+" AND expires_at > "+ph(4), delta, now, key, now)
		if err != nil {
			return err
		}

		if rows, _ := res.RowsAffected(); rows == 0 {
			// An expired row of the same bucket has no use: replace it.
			if _, err := tx.ExecContext(ctx, "DELETE FROM telegram_counters WHERE bucket = "+ph(1), key); err != nil {
				return err
			}

			if _, err := tx.ExecContext(ctx,
				"INSERT INTO telegram_counters (bucket, count, expires_at, created_at, updated_at) VALUES ("+
					ph(1)+", "+ph(2)+", "+ph(3)+", "+ph(4)+", "+ph(5)+")", key, delta, now.Add(ttl), now, now); err != nil {
				return err
			}
		}

		return tx.QueryRowContext(ctx, "SELECT count FROM telegram_counters WHERE bucket = "+ph(1), key).Scan(&n)
	})

	return n, err
}

// PruneCounters deletes the counters whose window is over.
func PruneCounters(ctx context.Context, db *database.DB, now time.Time) error {
	_, err := db.ExecContext(ctx, "DELETE FROM telegram_counters WHERE expires_at < "+db.Dialect().Placeholder(1), now.UTC())

	return err
}

// Limits are the rate limits of the bot, all per chat (config/telegram.go).
type Limits struct {
	minute, day, start, callback, notice ratelimit.Limiter
}

// NewLimits builds the limits over counter.
func NewLimits(counter ratelimit.Counter) *Limits {
	return &Limits{
		minute:   ratelimit.NewWindow(counter, "tg-minute", config.TelegramMessagesPerMinute, time.Minute),
		day:      ratelimit.NewWindow(counter, "tg-day", config.TelegramMessagesPerDay, 24*time.Hour),
		start:    ratelimit.NewWindow(counter, "tg-start", config.TelegramStartPerMinute, time.Minute),
		callback: ratelimit.NewWindow(counter, "tg-callback", config.TelegramCallbacksPerMinute, time.Minute),
		notice:   ratelimit.NewWindow(counter, "tg-notice", 1, time.Minute),
	}
}

func key(chat int64) string { return strconv.FormatInt(chat, 10) }

// Verdict says what to do with a message or a tap.
type Verdict int

// The verdicts.
const (
	// Allow: handle it.
	Allow Verdict = iota
	// Drop: refuse in silence.
	Drop
	// Notify: refuse and say so once (the notice itself is limited to one a minute).
	Notify
)

// Message counts one message of chat against the minute and the day budgets.
// A failing store drops the message: the limit fails closed.
func (l *Limits) Message(ctx context.Context, chat int64) Verdict {
	res, err := l.minute.Allow(ctx, key(chat))
	if err == nil && res.Allowed {
		res, err = l.day.Allow(ctx, key(chat))
		if err == nil && res.Allowed {
			return Allow
		}
	}

	if err != nil {
		return Drop
	}

	if note, err := l.notice.Allow(ctx, key(chat)); err == nil && note.Allowed {
		return Notify
	}

	return Drop
}

// Start counts one link attempt of chat.
func (l *Limits) Start(ctx context.Context, chat int64) bool {
	res, err := l.start.Allow(ctx, key(chat))

	return err == nil && res.Allowed
}

// Callback counts one button tap of chat.
func (l *Limits) Callback(ctx context.Context, chat int64) bool {
	res, err := l.callback.Allow(ctx, key(chat))

	return err == nil && res.Allowed
}

// NewDBLimits builds the limits over the database, the way the bot runs.
func NewDBLimits(db *database.DB) *Limits { return NewLimits(dbCounter{db: db, now: time.Now}) }
