package tests

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

// Neither the token, the webhook secrets, a link code nor what a person wrote
// reaches a log line or a response, on the happy path and on failures.
func TestSecurity_TokenNeverInLogsOrResponses(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")

	const note = "ZXQ-PRIVATE-NOTE-4471"

	code, _, _ := bot.Store.NewCode(t.Context(), w.User.ID, w.Workspace.ID)
	hook := app.Client(t).WithHeader(telegram.SecretHeader, tgWebhookSecret)
	path := hookPath(bot)

	var bodies []string

	post := func(text string) {
		res := hookPost(hook, path, updateJSON(t, 9901, text))
		bodies = append(bodies, res.Body())
	}

	post("/start " + code)
	post("-12,50 " + note)
	post("/last")
	post("/start NOTACODE" + note)

	// A send that fails, so the failure path logs too.
	f.FailSend(true)
	post("-7 " + note)
	post("/balance")
	f.FailSend(false)

	// Refused webhook calls: the response and the log must not echo the secrets.
	bodies = append(bodies,
		hookPost(app.Client(t), path, "{}").Body(),
		hookPost(hook, "/telegram/webhook/wrong-"+tgWebhookSecret, "{}").Body(),
		hookPost(app.Client(t).WithHeader(telegram.SecretHeader, "guess-"+tgWebhookSecret), path, "{}").Body(),
	)

	logs := app.Logs()
	secrets := []string{fakes.TelegramToken, "fake-token-for-tests-only", tgWebhookSecret, bot.Cfg.WebhookPathSecret(), code, note}

	for _, s := range secrets {
		if strings.Contains(logs, s) {
			t.Errorf("a log line contains %q:\n%s", s, logs)
		}

		for _, body := range bodies {
			if strings.Contains(body, s) {
				t.Errorf("a response contains %q: %s", s, body)
			}
		}
	}

	// The same through the bot's own logger on direct calls with a failing Telegram.
	var buf bytes.Buffer

	bot.Log = newBufferLogger(&buf)
	f.FailSend(true)
	say(t, bot, 9901, "-1 "+note)
	say(t, bot, 9901, "/last")

	for _, s := range secrets {
		if strings.Contains(buf.String(), s) {
			t.Errorf("the bot log contains %q: %s", s, buf.String())
		}
	}

	if !strings.Contains(buf.String(), "send failed") {
		t.Fatalf("the failure path did not log (the assertion above is empty): %q", buf.String())
	}
}

func TestTelegramCommands(t *testing.T) {
	t.Parallel()

	f := fakes.NewTelegram(t)

	var out bytes.Buffer

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env: tgEnv(f, map[string]string{"APP_EDITION": "saas"}), AppOptions: []vukem.Option{vukem.WithStdout(&out)},
	})

	bot, err := telegram.New(app.App, service(t, app), softService(t, app))
	if err != nil {
		t.Fatal(err)
	}

	if err := app.Call(t.Context(), "telegram:set-webhook", nil); err != nil {
		t.Fatal(err)
	}

	url, secret := f.Webhook()
	if url != "https://finance.example.test"+hookPath(bot) || secret != tgWebhookSecret {
		t.Fatalf("registered %q with secret %q", url, secret)
	}

	if err := app.Call(t.Context(), "telegram:info", nil); err != nil {
		t.Fatal(err)
	}

	for _, s := range []string{fakes.TelegramToken, tgWebhookSecret, bot.Cfg.WebhookPathSecret()} {
		if strings.Contains(out.String(), s) {
			t.Errorf("a command printed a secret (%q):\n%s", s, out.String())
		}
	}

	mustContain(t, out.String(), "@fake_finance_bot", "webhook", "[redacted]", "finance.example.test")

	// The registered path is the one the route serves.
	hookPost(app.Client(t).WithHeader(telegram.SecretHeader, secret), strings.TrimPrefix(url, "https://finance.example.test"), "{}").
		AssertStatus(http.StatusBadRequest)
}

func TestTelegramCommandsNeedTheBot(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	for _, name := range []string{"telegram:set-webhook", "telegram:info"} {
		if err := app.Call(t.Context(), name, nil); err == nil || !strings.Contains(err.Error(), "TELEGRAM_BOT_TOKEN") {
			t.Errorf("%s with the bot off: %v", name, err)
		}
	}
}
