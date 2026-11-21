package ai_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
)

// localHTTP reaches the loopback fake servers of the tests, and retries fast.
func localHTTP() *ai.HTTP {
	return ai.NewHTTP(ai.HTTPOptions{
		Allow: []netip.Prefix{netip.MustParsePrefix("127.0.0.0/8")}, Retry: time.Millisecond,
	})
}

const anthropicOK = `{"id":"msg_1","type":"message","role":"assistant","model":"claude-test",
"content":[{"type":"text","text":"{\"category\":\"Groceries\"}"}],"stop_reason":"end_turn",
"usage":{"input_tokens":42,"output_tokens":17}}`

func TestAnthropic(t *testing.T) {
	t.Parallel()

	newProvider := func(h http.HandlerFunc) (*ai.Anthropic, *httptest.Server) {
		srv := httptest.NewServer(h)
		t.Cleanup(srv.Close)

		p := ai.NewAnthropic(ai.NewKey(testKey), "claude-test", localHTTP())
		p.Endpoint = srv.URL + "/v1/messages"

		return p, srv
	}

	t.Run("request and response shapes", func(t *testing.T) {
		t.Parallel()

		var got map[string]any

		var headers http.Header

		p, _ := newProvider(func(w http.ResponseWriter, r *http.Request) {
			headers = r.Header.Clone()
			if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}

			_ = json.NewDecoder(r.Body).Decode(&got)
			_, _ = io.WriteString(w, anthropicOK)
		})

		resp, err := p.Complete(t.Context(), ai.Request{
			System: "You categorize.", Messages: []ai.Message{{Role: "user", Content: "Mercado"}}, MaxTokens: 100,
			JSONSchema: []byte(`{"type":"object"}`),
		})
		if err != nil {
			t.Fatal(err)
		}

		if resp.Text != `{"category":"Groceries"}` || resp.InputTokens != 42 || resp.OutputTokens != 17 ||
			resp.Model != "claude-test" || resp.Provider != ai.KindAnthropic {
			t.Errorf("response = %+v", resp)
		}

		if headers.Get("x-api-key") != testKey || headers.Get("anthropic-version") != "2023-06-01" ||
			headers.Get("Content-Type") != "application/json" {
			t.Errorf("headers = %v", headers)
		}

		if got["model"] != "claude-test" || got["system"] != "You categorize." || got["max_tokens"] != float64(100) {
			t.Errorf("body = %v", got)
		}

		format := got["output_config"].(map[string]any)["format"].(map[string]any)
		if format["type"] != "json_schema" {
			t.Errorf("output_config = %v", got["output_config"])
		}
	})

	t.Run("401 is a rejected key and the echoed key stays out of the error", func(t *testing.T) {
		t.Parallel()

		p, _ := newProvider(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key `+testKey+`"}}`)
		})

		_, err := p.Complete(t.Context(), ai.Request{Messages: []ai.Message{{Role: "user", Content: "Mercado"}}})
		if !errors.Is(err, ai.ErrRejected) {
			t.Fatalf("err = %v", err)
		}

		if strings.Contains(err.Error(), "0123456789") {
			t.Errorf("the error carries the key: %v", err)
		}

		pe, ok := errors.AsType[*ai.ProviderError](err)
		if !ok || pe.Status != 401 || pe.Type != "authentication_error" {
			t.Errorf("provider error = %+v", pe)
		}
	})

	t.Run("429 then 200 is retried once", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32

		p, _ := newProvider(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			_, _ = io.WriteString(w, anthropicOK)
		})

		resp, err := p.Complete(t.Context(), ai.Request{Messages: []ai.Message{{Role: "user", Content: "x"}}})
		if err != nil || resp.OutputTokens != 17 || calls.Load() != 2 {
			t.Fatalf("resp = %+v, err = %v, calls = %d", resp, err, calls.Load())
		}
	})

	t.Run("429 twice and 500 twice stop after one retry", func(t *testing.T) {
		t.Parallel()

		for status, want := range map[int]error{429: ai.ErrRateLimited, 500: ai.ErrUnavailable, 529: ai.ErrUnavailable} {
			var calls atomic.Int32

			p, _ := newProvider(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
			})

			_, err := p.Complete(t.Context(), ai.Request{Messages: []ai.Message{{Role: "user", Content: "x"}}})
			if !errors.Is(err, want) || calls.Load() != 2 {
				t.Errorf("status %d: err = %v, calls = %d", status, err, calls.Load())
			}
		}
	})

	t.Run("a 400 is never retried", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32

		p, _ := newProvider(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusBadRequest)
		})

		_, err := p.Complete(t.Context(), ai.Request{Messages: []ai.Message{{Role: "user", Content: "x"}}})
		if !errors.Is(err, ai.ErrBadRequest) || calls.Load() != 1 {
			t.Errorf("err = %v, calls = %d", err, calls.Load())
		}
	})

	t.Run("a 2 MiB body is refused", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32

		p, _ := newProvider(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			_, _ = io.WriteString(w, `{"content":[{"type":"text","text":"`+strings.Repeat("a", 2<<20)+`"}]}`)
		})

		_, err := p.Complete(t.Context(), ai.Request{Messages: []ai.Message{{Role: "user", Content: "x"}}})
		if !errors.Is(err, ai.ErrUnavailable) {
			t.Fatalf("err = %v", err)
		}

		if pe, _ := errors.AsType[*ai.ProviderError](err); pe == nil || pe.Type != "body_too_large" {
			t.Errorf("provider error = %+v", pe)
		}

		if calls.Load() != 1 {
			t.Errorf("a refused body was retried: %d calls", calls.Load())
		}
	})

	t.Run("a redirect is not followed and the key goes nowhere else", func(t *testing.T) {
		t.Parallel()

		var elsewhere atomic.Int32

		other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { elsewhere.Add(1) }))
		t.Cleanup(other.Close)

		p, _ := newProvider(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, other.URL+"/steal", http.StatusTemporaryRedirect)
		})

		_, err := p.Complete(t.Context(), ai.Request{Messages: []ai.Message{{Role: "user", Content: "x"}}})
		if !errors.Is(err, ai.ErrUnavailable) {
			t.Fatalf("err = %v", err)
		}

		if elsewhere.Load() != 0 {
			t.Fatal("the redirect was followed")
		}
	})

	t.Run("Check makes a minimal request", func(t *testing.T) {
		t.Parallel()

		var got map[string]any

		p, _ := newProvider(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&got)
			_, _ = io.WriteString(w, anthropicOK)
		})

		if err := p.Check(t.Context()); err != nil || got["max_tokens"] != float64(16) {
			t.Errorf("Check = %v, body = %v", err, got)
		}
	})

	t.Run("a blocked address is refused without an allow list", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Error("the safe client reached a loopback server")
		}))
		t.Cleanup(srv.Close)

		p := ai.NewAnthropic(ai.NewKey(testKey), "", ai.NewHTTP(ai.HTTPOptions{Retry: time.Millisecond}))
		p.Endpoint = srv.URL

		_, err := p.Complete(t.Context(), ai.Request{Messages: []ai.Message{{Role: "user", Content: "x"}}})
		if pe, ok := errors.AsType[*ai.ProviderError](err); !ok || pe.Type != "blocked_address" {
			t.Errorf("err = %v", err)
		}
	})
}
