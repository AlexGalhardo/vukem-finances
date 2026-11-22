package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/httpclient"
)

// Limits of every outbound AI call.
const (
	callTimeout  = 30 * time.Second
	maxResponse  = 1 << 20
	retryBackoff = 500 * time.Millisecond
	maxRetryWait = 2 * time.Second
)

// localPrefixes are what AI_ALLOW_PRIVATE_HOSTS opens up in the personal
// edition: the machine itself and the private networks. Link-local addresses
// (the cloud metadata service) stay blocked.
var localPrefixes = []netip.Prefix{ //nolint:gochecknoglobals // read-only table
	netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("fc00::/7"),
}

// HTTPOptions tune NewHTTP.
type HTTPOptions struct {
	// AllowPrivate lets the client reach loopback and private networks
	// (personal edition, AI_ALLOW_PRIVATE_HOSTS).
	AllowPrivate bool
	// Allow adds prefixes to reach; for tests that talk to a local fake server.
	Allow []netip.Prefix
	// Retry is the base wait before the one retry (default 500ms).
	Retry time.Duration
}

// HTTP makes the outbound calls of the providers: the safe client (internal
// addresses refused at connect time), a 30 second timeout, a 1 MiB response
// cap, no redirects (a redirect would resend the key header to another host),
// and one retry on 429 and 5xx.
type HTTP struct {
	client *http.Client
	retry  time.Duration
}

// NewHTTP builds the client of the providers.
func NewHTTP(o HTTPOptions) *HTTP {
	allow := append([]netip.Prefix(nil), o.Allow...)
	if o.AllowPrivate {
		allow = append(allow, localPrefixes...)
	}

	client := httpclient.Safe(httpclient.Options{Timeout: callTimeout, MaxBody: maxResponse, Allow: allow})
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	if o.Retry <= 0 {
		o.Retry = retryBackoff
	}

	return &HTTP{client: client, retry: o.Retry}
}

// postJSON sends body as JSON with headers and decodes a 2xx answer into out.
// A failure is a *ProviderError: its text is generic and it carries the status
// and the provider's error type for logs, never the upstream message (which can
// echo the key or the prompt).
func (h *HTTP) postJSON(ctx context.Context, url string, headers map[string]string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return &ProviderError{Class: ErrBadRequest}
	}

	for attempt := 0; ; attempt++ {
		status, raw, err := h.once(ctx, url, headers, payload)
		if err != nil {
			return err
		}

		if status >= 200 && status < 300 {
			if json.Unmarshal(raw, out) != nil {
				return &ProviderError{Status: status, Type: "bad_response", Class: ErrUnavailable}
			}

			return nil
		}

		retryable := status == http.StatusTooManyRequests || status >= 500
		if retryable && attempt == 0 {
			if h.wait(ctx) != nil {
				return &ProviderError{Status: status, Class: ErrUnavailable}
			}

			continue
		}

		return &ProviderError{Status: status, Type: errorType(raw), Class: classify(status)}
	}
}

func (h *HTTP) once(ctx context.Context, url string, headers map[string]string, payload []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, &ProviderError{Type: "bad_url", Class: ErrBadRequest}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		// The error text of net/http names the URL and can carry more: drop it.
		// A blocked address and a failed connection share the class and so the message:
		// a client must not learn from the answer which addresses the server refuses
		// to reach. Only the logged Type (and the usage accounting that keys on it) differs.
		pe := &ProviderError{Type: "transport", Class: ErrUnreachable}
		if errors.Is(err, httpclient.ErrBlockedAddress) {
			pe.Type = "blocked_address"
		}

		return 0, nil, pe
	}

	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		pe := &ProviderError{Status: resp.StatusCode, Type: "bad_response", Class: ErrUnavailable}
		if errors.Is(err, httpclient.ErrBodyTooLarge) {
			pe.Type = "body_too_large"
		}

		return 0, nil, pe
	}

	return resp.StatusCode, raw, nil
}

func (h *HTTP) wait(ctx context.Context) error {
	d := min(h.retry+rand.N(h.retry), maxRetryWait)

	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func classify(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrRejected
	case status == http.StatusTooManyRequests:
		return ErrRateLimited
	case status >= 500 || status < 200 || (status >= 300 && status < 400):
		return ErrUnavailable
	default:
		return ErrBadRequest
	}
}

// errorType reads the provider's error type or code from an error body
// ({"error":{"type":"..."}}, {"error":{"code":"..."}}), whatever shape it has.
// The message is not read at all.
func errorType(raw []byte) string {
	var e struct {
		Error struct {
			Type string `json:"type"`
			Code any    `json:"code"`
		} `json:"error"`
	}

	if json.Unmarshal(raw, &e) != nil {
		return ""
	}

	if e.Error.Type != "" {
		return e.Error.Type
	}

	if code, ok := e.Error.Code.(string); ok {
		return strings.TrimSpace(code)
	}

	return ""
}

// checkRequest is the smallest request that proves a key works.
func checkRequest() Request {
	return Request{Messages: []Message{{Role: "user", Content: "ping"}}, MaxTokens: 16, Feature: "check"}
}
