package tests

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/config"
	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

func hookPath(bot *telegram.Bot) string {
	return config.WebhookPath + "/" + bot.Cfg.WebhookPathSecret()
}

func hookPost(c *vukemtest.Client, path, body string) *vukemtest.Response {
	return c.Do(http.MethodPost, path, strings.NewReader(body), "application/json")
}

func updateJSON(t *testing.T, chat int64, text string) string {
	t.Helper()

	raw, err := json.Marshal(fakes.TextUpdate(nextUpdate(), chat, text))
	if err != nil {
		t.Fatal(err)
	}

	return string(raw)
}

func TestSecurity_TelegramWebhook(t *testing.T) {
	t.Parallel()

	app, f, bot := tgApp(t)
	w := signUp(t, app, "Ana", "ana@example.com")
	linkChat(t, bot, w, 6001)

	path := hookPath(bot)
	good := app.Client(t).WithHeader(telegram.SecretHeader, tgWebhookSecret)

	t.Run("a valid request is handled without a session or a CSRF token", func(t *testing.T) {
		f.Reset()

		// Browser headers a CSRF check would act on: the webhook skips it.
		c := good.WithHeader("Sec-Fetch-Site", "cross-site").WithHeader("Origin", "https://evil.example")
		hookPost(c, path, updateJSON(t, 6001, "/help")).AssertStatus(http.StatusOK)

		if got := f.Sent(6001); len(got) != 1 {
			t.Fatalf("%d replies, want 1", len(got))
		}
	})

	t.Run("a wrong or missing secret header is 401", func(t *testing.T) {
		f.Reset()

		body := updateJSON(t, 6001, "/help")

		hookPost(app.Client(t), path, body).AssertStatus(http.StatusUnauthorized)
		hookPost(app.Client(t).WithHeader(telegram.SecretHeader, ""), path, body).AssertStatus(http.StatusUnauthorized)
		hookPost(app.Client(t).WithHeader(telegram.SecretHeader, tgWebhookSecret+"x"), path, body).AssertStatus(http.StatusUnauthorized)
		hookPost(app.Client(t).WithHeader(telegram.SecretHeader, tgWebhookSecret[:len(tgWebhookSecret)-1]), path, body).AssertStatus(http.StatusUnauthorized)
		// The path secret is not the header secret.
		hookPost(app.Client(t).WithHeader(telegram.SecretHeader, bot.Cfg.WebhookPathSecret()), path, body).AssertStatus(http.StatusUnauthorized)

		if got := f.Sent(6001); len(got) != 0 {
			t.Fatalf("a refused request was handled: %v", got)
		}
	})

	t.Run("a wrong path secret is 404", func(t *testing.T) {
		body := updateJSON(t, 6001, "/help")

		hookPost(good, config.WebhookPath+"/wrong", body).AssertNotFound()
		hookPost(good, config.WebhookPath+"/"+tgWebhookSecret, body).AssertNotFound()
		hookPost(good, config.WebhookPath+"/", body).AssertNotFound()
	})

	t.Run("a body over 64 KiB is 413", func(t *testing.T) {
		big := `{"update_id":1,"message":{"text":"` + strings.Repeat("a", 70<<10) + `"}}`
		hookPost(good, path, big).AssertStatus(http.StatusRequestEntityTooLarge)
	})

	t.Run("a body that is not one update is 400", func(t *testing.T) {
		for _, body := range []string{"", "nope", "[]", `{"update_id":0}`, `{"update_id":-4}`, `{}`, updateJSON(t, 6001, "/help") + "{}"} {
			hookPost(good, path, body).AssertStatus(http.StatusBadRequest)
		}
	})

	t.Run("a field Telegram adds later is ignored", func(t *testing.T) {
		body := `{"update_id":` + itoa(nextUpdate()) + `,"future_field":{"a":1},"edited_message":{"message_id":1}}`
		hookPost(good, path, body).AssertStatus(http.StatusOK)
	})

	t.Run("only POST", func(t *testing.T) {
		good.Get(path).AssertStatus(http.StatusMethodNotAllowed)
	})

	t.Run("a redelivery is a 200 and handled once", func(t *testing.T) {
		f.Reset()

		body := updateJSON(t, 6001, "/help")
		hookPost(good, path, body).AssertStatus(http.StatusOK)
		hookPost(good, path, body).AssertStatus(http.StatusOK)

		if got := f.Sent(6001); len(got) != 1 {
			t.Fatalf("%d replies for a redelivered update, want 1", len(got))
		}
	})
}

func itoa(n int) string {
	b, _ := json.Marshal(n)

	return string(b)
}

// With no token the bot is off: the webhook route is 404 whatever is sent to
// it, and in polling mode there is no webhook route either.
func TestTelegramOffMeansNoRoutes(t *testing.T) {
	t.Parallel()

	f := fakes.NewTelegram(t)

	for name, env := range map[string]map[string]string{
		"no token":     {"TELEGRAM_MODE": "webhook", "TELEGRAM_WEBHOOK_SECRET": tgWebhookSecret},
		"polling mode": tgEnv(f, map[string]string{"TELEGRAM_MODE": "polling"}),
	} {
		app := newApp(t, env)

		cfg, err := config.LoadTelegram(app.Source, config.EditionSaaS)
		if err != nil {
			t.Fatal(err)
		}

		secret := config.Telegram{WebhookSecret: cfg.WebhookSecret}.WebhookPathSecret()

		for _, p := range []string{config.WebhookPath + "/" + secret, config.WebhookPath + "/x"} {
			c := app.Client(t).WithHeader(telegram.SecretHeader, tgWebhookSecret)
			hookPost(c, p, updateJSON(t, 1, "/help")).AssertNotFound()
		}

		if name == "no token" {
			bot, err := telegram.New(app.App, service(t, app), softService(t, app))
			if bot != nil || err != nil {
				t.Fatalf("telegram.New with no token = %v, %v; want nil, nil", bot, err)
			}
		}
	}
}

func TestTelegramConfigIsValidated(t *testing.T) {
	t.Parallel()

	src := func(kv map[string]string) lookup { return lookup(kv) }

	cases := map[string]map[string]string{
		"bad mode":             {"TELEGRAM_BOT_TOKEN": "1:x", "TELEGRAM_BOT_USERNAME": "good_bot", "TELEGRAM_MODE": "carrier-pigeon"},
		"no username":          {"TELEGRAM_BOT_TOKEN": "1:x", "TELEGRAM_MODE": "polling"},
		"webhook no secret":    {"TELEGRAM_BOT_TOKEN": "1:x", "TELEGRAM_BOT_USERNAME": "good_bot", "TELEGRAM_MODE": "webhook"},
		"webhook short secret": {"TELEGRAM_BOT_TOKEN": "1:x", "TELEGRAM_BOT_USERNAME": "good_bot", "TELEGRAM_MODE": "webhook", "TELEGRAM_WEBHOOK_SECRET": "short"},
		"webhook http url": {
			"TELEGRAM_BOT_TOKEN": "1:x", "TELEGRAM_BOT_USERNAME": "good_bot", "TELEGRAM_MODE": "webhook",
			"TELEGRAM_WEBHOOK_SECRET": tgWebhookSecret, "TELEGRAM_WEBHOOK_URL": "http://x.test",
		},
	}

	for name, kv := range cases {
		if _, err := config.LoadTelegram(src(kv), config.EditionSaaS); err == nil {
			t.Errorf("%s: no error", name)
		}
	}

	// The default mode follows the edition.
	cfg, err := config.LoadTelegram(src(map[string]string{"TELEGRAM_BOT_TOKEN": "1:x", "TELEGRAM_BOT_USERNAME": "good_bot"}), config.EditionPersonal)
	if err != nil || cfg.Mode != config.TelegramPolling {
		t.Errorf("personal default = %q, %v; want polling", cfg.Mode, err)
	}

	cfg, err = config.LoadTelegram(src(map[string]string{
		"TELEGRAM_BOT_TOKEN": "1:x", "TELEGRAM_BOT_USERNAME": "good_bot", "TELEGRAM_WEBHOOK_SECRET": tgWebhookSecret,
	}), config.EditionSaaS)
	if err != nil || cfg.Mode != config.TelegramWebhook {
		t.Errorf("saas default = %q, %v; want webhook", cfg.Mode, err)
	}

	if cfg.WebhookPathSecret() == tgWebhookSecret || len(cfg.WebhookPathSecret()) != 32 {
		t.Errorf("the path secret leaks or is short: %q", cfg.WebhookPathSecret())
	}
}

type lookup map[string]string

func (l lookup) Lookup(k string) (string, bool) { v, ok := l[k]; return v, ok }
