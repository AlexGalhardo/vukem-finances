package tests

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

const tgWebhookSecret = "webhook-secret-for-tests-0123456789"

// tgUpdateSeq hands out update ids that no two tests share a database for, and
// that never repeat inside one.
var tgUpdateSeq atomic.Int64

func nextUpdate() int { return int(tgUpdateSeq.Add(1)) }

// tgEnv is the configuration of a bot pointed at the fake Telegram server.
func tgEnv(f *fakes.Telegram, extra ...map[string]string) map[string]string {
	env := map[string]string{
		"TELEGRAM_BOT_TOKEN": fakes.TelegramToken, "TELEGRAM_BOT_USERNAME": "fake_finance_bot",
		"TELEGRAM_API_BASE": f.URL, "TELEGRAM_MODE": "webhook", "TELEGRAM_WEBHOOK_SECRET": tgWebhookSecret,
		"TELEGRAM_WEBHOOK_URL": "https://finance.example.test",
	}

	for _, e := range extra {
		for k, v := range e {
			env[k] = v
		}
	}

	return env
}

// tgApp builds a saas app with the bot on, the fake Telegram server and the bot.
func tgApp(t *testing.T, extra ...map[string]string) (*vukemtest.App, *fakes.Telegram, *telegram.Bot) {
	t.Helper()

	return tgAppAt(t, nil, extra...)
}

// tgAppAt is tgApp with a clock the test moves (nil keeps the wall clock).
func tgAppAt(t *testing.T, clock *vukemtest.FakeClock, extra ...map[string]string) (*vukemtest.App, *fakes.Telegram, *telegram.Bot) {
	t.Helper()

	f := fakes.NewTelegram(t)
	env := map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"}
	maps.Copy(env, tgEnv(f, extra...))

	opts := vukemtest.Options{Env: env}
	if clock != nil {
		opts.Clock = clock.Now
	}

	app := vukemtest.New(t, bootstrap.New, opts)

	bot, err := telegram.New(app.App, service(t, app), softService(t, app))
	if err != nil || bot == nil {
		t.Fatalf("telegram.New = %v, %v", bot, err)
	}

	return app, f, bot
}

func say(t *testing.T, bot *telegram.Bot, chat int64, text string) {
	t.Helper()

	if err := bot.HandleUpdate(t.Context(), mustUpdate(t, fakes.TextUpdate(nextUpdate(), chat, text))); err != nil {
		t.Fatalf("say %q: %v", text, err)
	}
}

func tap(t *testing.T, bot *telegram.Bot, chat int64, data string) {
	t.Helper()

	if err := bot.HandleUpdate(t.Context(), mustUpdate(t, fakes.CallbackUpdate(nextUpdate(), chat, data))); err != nil {
		t.Fatalf("tap %q: %v", data, err)
	}
}

// mustUpdate turns a fake update map into the typed update, as the webhook would decode it.
func mustUpdate(t *testing.T, m map[string]any) telegram.Update {
	t.Helper()

	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	var u telegram.Update
	if err := json.Unmarshal(raw, &u); err != nil {
		t.Fatal(err)
	}

	return u
}

// linkChat links chat to the person of w through the real flow: a code made
// for the signed-in person and a /start with it.
func linkChat(t *testing.T, bot *telegram.Bot, w who, chat int64) {
	t.Helper()

	code, _, err := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)
	if err != nil {
		t.Fatal(err)
	}

	say(t, bot, chat, "/start "+code)

	if _, ok, err := bot.Store.LinkByChat(t.Context(), chat); err != nil || !ok {
		t.Fatalf("chat %d did not link: %v %v", chat, ok, err)
	}
}

// lastText is the newest message the fake got for chat.
func lastText(t *testing.T, f *fakes.Telegram, chat int64) string {
	t.Helper()

	c, ok := f.LastSent(chat)
	if !ok {
		t.Fatalf("no message was sent to chat %d", chat)
	}

	text, _ := c.Params["text"].(string)

	return text
}

func mustContain(t *testing.T, text string, parts ...string) {
	t.Helper()

	for _, p := range parts {
		if !strings.Contains(text, p) {
			t.Errorf("%q does not contain %q", text, p)
		}
	}
}

func tgCtx(w who) context.Context {
	return workspaces.WithTenant(context.Background(), w.Workspace.ID)
}

func transactionsOf(t *testing.T, app *vukemtest.App, w who) []models.Transaction {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	rows, err := workspaces.Scoped[models.Transaction](tgCtx(w), db).OrderBy("id", orm.Asc).Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return rows
}

// removeMember takes the person out of the workspace of w, as the owner would.
func removeMember(t *testing.T, app *vukemtest.App, w who, userID int64) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := orm.QueryOf[models.Membership](db).Where("workspace_id", w.Workspace.ID).Where("user_id", userID).Delete(t.Context()); err != nil {
		t.Fatal(err)
	}
}

// onlyButton finds the button whose label contains label in the last keyboard sent to chat.
func buttonData(t *testing.T, f *fakes.Telegram, chat int64, label string) string {
	t.Helper()

	c, ok := f.LastSent(chat)
	if !ok {
		t.Fatalf("no message to chat %d", chat)
	}

	markup, _ := c.Params["reply_markup"].(map[string]any)
	rows, _ := markup["inline_keyboard"].([]any)

	for _, row := range rows {
		for _, cell := range row.([]any) {
			b := cell.(map[string]any)
			if strings.Contains(b["text"].(string), label) {
				return b["callback_data"].(string)
			}
		}
	}

	t.Fatalf("no button %q in the last message to chat %d", label, chat)

	return ""
}

// newBufferLogger logs every level to w, for the tests that read what was logged.
func newBufferLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
