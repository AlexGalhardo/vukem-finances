package billing

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// PixWebhookPath is where AbacatePay sends events. The secret may also be the
// last path segment (PixWebhookPath + "/" + secret) when the provider only lets
// us configure a URL.
const PixWebhookPath = "/abacatepay/webhook"

// PixWebhookSecretPath is the pattern of PixWebhookPath with the secret segment,
// for Router.SecretPaths, which keeps the secret out of the logs.
const PixWebhookSecretPath = PixWebhookPath + "/{secret}"

// pixWebhookMaxBody caps a delivery: real ones are a few KiB.
const pixWebhookMaxBody = 64 << 10

var eventName = regexp.MustCompile(`^[a-z_]{1,30}(\.[a-z_]{1,30}){0,2}$`)

// pixEvent is the envelope of a delivery (docs.abacatepay.com, webhooks). Only
// the fields read are declared; the outcome is never taken from them.
type pixEvent struct {
	ID    string `json:"id"`
	Event string `json:"event"`
	Data  struct {
		Transparent struct {
			ID string `json:"id"`
		} `json:"transparent"`
	} `json:"data"`
}

// PixWebhook serves AbacatePay deliveries ahead of the router stack: the caller
// is a server, so there is no session or CSRF check. Its checks are, in order:
// the shared secret in constant time (header X-Webhook-Secret or a path
// segment, never the query string, which lands in access logs), the HMAC
// signature of the raw body, a body size cap and a strict decode. Even then the
// body decides nothing: the charge is re-read from the provider before access
// changes. Register it with r.UseServer, and PixWebhookSecretPath with
// r.SecretPaths so the segment never reaches a log line.
func (b *Billing) PixWebhook() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch segment, ok := strings.CutPrefix(r.URL.Path, PixWebhookPath+"/"); {
			case r.URL.Path == PixWebhookPath:
				b.servePixWebhook(w, r, "")
			case ok && segment != "" && !strings.Contains(segment, "/"):
				b.servePixWebhook(w, r, segment)
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

func (b *Billing) servePixWebhook(w http.ResponseWriter, r *http.Request, segment string) {
	if b.Abacate == nil || b.Config.Pix.WebhookSecret.IsZero() { // fail closed
		http.Error(w, "webhooks are not configured", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	given := r.Header.Get("X-Webhook-Secret")
	if given == "" {
		given = segment
	}

	if !SecretEqual(given, b.Config.Pix.WebhookSecret.Reveal()) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, pixWebhookMaxBody))
	if err != nil {
		if _, tooBig := errors.AsType[*http.MaxBytesError](err); tooBig {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "bad request", http.StatusBadRequest)

		return
	}

	if !VerifySignature(body, r.Header.Get("X-Webhook-Signature")) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var ev pixEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch ev.Event {
	case "transparent.completed", "transparent.refunded", "transparent.disputed", "transparent.lost":
		if ev.Data.Transparent.ID == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if err := b.HandlePixEvent(r.Context(), ev.Data.Transparent.ID); err != nil {
			// The reason is logged as a kind only; AbacatePay delivers again.
			b.Log.WarnContext(r.Context(), "pix webhook not applied", "event", ev.Event, "err", err.Error())
			http.Error(w, "try again", http.StatusInternalServerError)

			return
		}
	default:
		name := "invalid"
		if eventName.MatchString(ev.Event) {
			name = ev.Event
		}

		b.Log.InfoContext(r.Context(), "pix webhook: ignored event", "event", name)
	}

	w.WriteHeader(http.StatusOK)
}
