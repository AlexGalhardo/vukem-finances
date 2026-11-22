package jobs

import (
	"context"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/telegram"
)

// TelegramPrune drops old update ids, expired link codes and drafts and
// finished rate windows of the Telegram bot. routes/schedule.go runs it daily;
// it touches only the bot's tables and needs no token.
type TelegramPrune struct{}

// Kind names the job's handler; keep it stable across deploys.
func (TelegramPrune) Kind() string { return "telegram_prune" }

// HandleTelegramPrune prunes the bot's tables.
func HandleTelegramPrune(app *vukem.App) func(ctx context.Context, j TelegramPrune) error {
	return func(ctx context.Context, _ TelegramPrune) error {
		db, err := app.DB()
		if err != nil {
			return err
		}

		return (&telegram.Store{DB: db, Now: app.Now}).Prune(ctx)
	}
}
