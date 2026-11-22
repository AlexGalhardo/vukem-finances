package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

// The Telegram bot modes (TELEGRAM_MODE).
const (
	TelegramPolling = "polling"
	TelegramWebhook = "webhook"
)

// Limits of the bot (docs/telegram.md). Each is per chat.
const (
	// TelegramMessagesPerMinute and TelegramMessagesPerDay bound the messages of one chat.
	TelegramMessagesPerMinute = 20
	TelegramMessagesPerDay    = 200
	// TelegramStartPerMinute bounds the link attempts (/start) of one chat.
	TelegramStartPerMinute = 5
	// TelegramCallbacksPerMinute bounds the button taps of one chat.
	TelegramCallbacksPerMinute = 60
	// TelegramInputMax is the longest message the bot reads, in characters.
	TelegramInputMax = 500
	// TelegramLinkCodeTTL is how long a link code works.
	TelegramLinkCodeTTL = 10 * time.Minute
	// TelegramPendingTTL is how long a draft waits for a tap.
	TelegramPendingTTL = 15 * time.Minute
	// TelegramUndoWindow is how long the Undo button of a saved entry works.
	TelegramUndoWindow = 60 * time.Second
	// TelegramUpdateRetention is how long processed update ids are kept.
	TelegramUpdateRetention = 7 * 24 * time.Hour
	// TelegramWebhookMaxBody caps a webhook body.
	TelegramWebhookMaxBody = 64 << 10
)

// Telegram configures the bot. With no token the bot is off: every bot route
// is 404 and no worker starts.
type Telegram struct {
	// Token is the bot token from BotFather (TELEGRAM_BOT_TOKEN). It prints as
	// [redacted] and is never logged.
	Token vconfig.Secret
	// Username is the bot's @name without the at sign (TELEGRAM_BOT_USERNAME),
	// used to build the t.me deep link.
	Username string
	// Mode is TelegramPolling or TelegramWebhook (TELEGRAM_MODE).
	Mode string
	// WebhookSecret is the value Telegram echoes in X-Telegram-Bot-Api-Secret-Token
	// (TELEGRAM_WEBHOOK_SECRET).
	WebhookSecret vconfig.Secret
	// WebhookURL is the public base URL of the app that setWebhook registers
	// (TELEGRAM_WEBHOOK_URL), without the path.
	WebhookURL string
	// APIBase replaces https://api.telegram.org; honored only by the tests
	// and outside production (TELEGRAM_API_BASE).
	APIBase string
}

var (
	usernameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{3,31}$`)
	// A header value Telegram accepts for secret_token: 1-256 of A-Z a-z 0-9 _ -.
	secretRe = regexp.MustCompile(`^[A-Za-z0-9_-]{16,256}$`)
)

// Enabled reports whether the bot is on.
func (t Telegram) Enabled() bool { return !t.Token.IsZero() }

// LoadTelegram reads the TELEGRAM_* settings. The default mode is polling in
// the personal edition and webhook in saas.
func LoadTelegram(src vconfig.Lookuper, edition Edition) (Telegram, error) {
	get := func(k string) string {
		v, _ := src.Lookup(k)

		return strings.TrimSpace(v)
	}

	t := Telegram{
		Token:         vconfig.NewSecret(get("TELEGRAM_BOT_TOKEN")),
		Username:      strings.TrimPrefix(get("TELEGRAM_BOT_USERNAME"), "@"),
		Mode:          strings.ToLower(get("TELEGRAM_MODE")),
		WebhookSecret: vconfig.NewSecret(get("TELEGRAM_WEBHOOK_SECRET")),
		WebhookURL:    strings.TrimSuffix(get("TELEGRAM_WEBHOOK_URL"), "/"),
		APIBase:       strings.TrimSuffix(get("TELEGRAM_API_BASE"), "/"),
	}

	if t.Mode == "" {
		t.Mode = TelegramPolling
		if edition.SaaS() {
			t.Mode = TelegramWebhook
		}
	}

	if !t.Enabled() {
		return t, nil
	}

	return t, t.validate()
}

func (t Telegram) validate() error {
	switch t.Mode {
	case TelegramPolling, TelegramWebhook:
	default:
		return fmt.Errorf("TELEGRAM_MODE=%q is invalid: use %q or %q", t.Mode, TelegramPolling, TelegramWebhook)
	}

	if !usernameRe.MatchString(t.Username) {
		return errors.New("TELEGRAM_BOT_USERNAME must be the bot name from BotFather (5 to 32 letters, digits or underscores)")
	}

	if t.Mode == TelegramWebhook {
		if !secretRe.MatchString(t.WebhookSecret.Reveal()) {
			return errors.New("TELEGRAM_WEBHOOK_SECRET must be 16 to 256 characters of A-Z a-z 0-9 _ - in webhook mode")
		}

		if t.WebhookURL != "" && !strings.HasPrefix(t.WebhookURL, "https://") {
			return errors.New("TELEGRAM_WEBHOOK_URL must start with https://")
		}
	}

	return nil
}

// WebhookPathSecret is the path segment of the webhook route, derived from
// the webhook secret so that the URL does not hold the secret itself: an HMAC
// of a fixed label, base64url. The header holds the secret.
func (t Telegram) WebhookPathSecret() string {
	mac := hmac.New(sha256.New, []byte(t.WebhookSecret.Reveal()))
	mac.Write([]byte("telegram-webhook-path"))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))[:32]
}

// WebhookPath is the route pattern of the webhook, without the method.
const WebhookPath = "/telegram/webhook"
