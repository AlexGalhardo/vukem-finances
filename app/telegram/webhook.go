package telegram

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem-finances/config"
)

// SecretHeader is the header Telegram echoes the secret_token of setWebhook in.
const SecretHeader = "X-Telegram-Bot-Api-Secret-Token"

// handleTimeout bounds the handling of one update once it has begun: a client
// that hangs up must not leave an entry half made.
const handleTimeout = 25 * time.Second

// equal compares two secrets in constant time (over their hashes, so the
// length does not leak either). An empty want never matches.
func equal(got, want string) bool {
	if want == "" {
		return false
	}

	a, b := sha256.Sum256([]byte(got)), sha256.Sum256([]byte(want))

	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

// WebhookHandler serves POST /telegram/webhook/{secret}. The path segment is
// derived from the webhook secret and the header must equal the secret itself;
// both are compared in constant time. A wrong path is a 404 (the route does
// not exist for the caller), a wrong or missing header a 401, a body over 64
// KiB a 413. Register it with SecretParams("secret") so the segment stays out
// of the logs. A handled or repeated update is a 200; an internal failure is a
// 500, which makes Telegram deliver the update again.
func (b *Bot) WebhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !equal(r.PathValue("secret"), b.Cfg.WebhookPathSecret()) {
			http.NotFound(w, r)

			return
		}

		if !equal(r.Header.Get(SecretHeader), b.Cfg.WebhookSecret.Reveal()) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, config.TelegramWebhookMaxBody))
		if err != nil {
			if _, tooBig := errors.AsType[*http.MaxBytesError](err); tooBig {
				http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)

				return
			}

			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		var u Update

		// Typed fields only: Telegram adds fields to updates over time, so an
		// unknown one is ignored, but the body must be one JSON object.
		dec := json.NewDecoder(bytes.NewReader(body))
		if err := dec.Decode(&u); err != nil || dec.More() || u.UpdateID <= 0 {
			http.Error(w, "bad request", http.StatusBadRequest)

			return
		}

		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), handleTimeout)
		defer cancel()

		if err := b.HandleUpdate(ctx, u); err != nil {
			http.Error(w, "try again", http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
