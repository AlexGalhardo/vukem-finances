package ai_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
)

const chatOK = `{"id":"chatcmpl-1","object":"chat.completion","model":"gpt-test",
"choices":[{"index":0,"message":{"role":"assistant","content":"{\"category\":\"Groceries\"}"},"finish_reason":"stop"}],
"usage":{"prompt_tokens":30,"completion_tokens":9,"total_tokens":39}}`

func chatRequest() ai.Request {
	return ai.Request{
		System: "You categorize.", Messages: []ai.Message{{Role: "user", Content: "Mercado"}}, MaxTokens: 80,
		JSONSchema: []byte(`{"type":"object"}`), Temperature: 0.2,
	}
}

func TestOpenAI(t *testing.T) {
	t.Parallel()

	newProvider := func(h http.HandlerFunc) *ai.Chat {
		srv := httptest.NewServer(h)
		t.Cleanup(srv.Close)

		p := ai.NewOpenAI(ai.NewKey(testKey), "gpt-test", localHTTP())
		p.Endpoint = srv.URL + "/v1/chat/completions"

		return p
	}

	t.Run("request and response shapes", func(t *testing.T) {
		t.Parallel()

		var got map[string]any

		var auth string

		p := newProvider(func(w http.ResponseWriter, r *http.Request) {
			auth = r.Header.Get("Authorization")
			_ = json.NewDecoder(r.Body).Decode(&got)
			_, _ = io.WriteString(w, chatOK)
		})

		resp, err := p.Complete(t.Context(), chatRequest())
		if err != nil {
			t.Fatal(err)
		}

		if resp.Text != `{"category":"Groceries"}` || resp.InputTokens != 30 || resp.OutputTokens != 9 ||
			resp.Model != "gpt-test" || resp.Provider != ai.KindOpenAI {
			t.Errorf("response = %+v", resp)
		}

		if auth != "Bearer "+testKey {
			t.Errorf("Authorization = %q", auth)
		}

		msgs := got["messages"].([]any)
		if len(msgs) != 2 || msgs[0].(map[string]any)["role"] != "system" {
			t.Errorf("messages = %v", msgs)
		}

		if got["max_completion_tokens"] != float64(80) || got["max_tokens"] != nil || got["temperature"] != 0.2 {
			t.Errorf("body = %v", got)
		}

		rf := got["response_format"].(map[string]any)
		if rf["type"] != "json_schema" {
			t.Errorf("response_format = %v", rf)
		}
	})

	t.Run("401, 429 then 200, 400 and a 2 MiB body", func(t *testing.T) {
		t.Parallel()

		var calls atomic.Int32

		p := newProvider(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			_, _ = io.WriteString(w, chatOK)
		})

		if _, err := p.Complete(t.Context(), chatRequest()); err != nil || calls.Load() != 2 {
			t.Fatalf("retry: err = %v, calls = %d", err, calls.Load())
		}

		for status, want := range map[int]error{401: ai.ErrRejected, 403: ai.ErrRejected, 400: ai.ErrBadRequest, 503: ai.ErrUnavailable} {
			q := newProvider(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"error":{"message":"bad key `+testKey+`","code":"invalid_api_key"}}`)
			})

			_, err := q.Complete(t.Context(), chatRequest())
			if !errors.Is(err, want) || strings.Contains(err.Error(), "0123456789") {
				t.Errorf("status %d: err = %v", status, err)
			}
		}

		big := newProvider(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"`+strings.Repeat("a", 2<<20)+`"}}]}`)
		})

		if _, err := big.Complete(t.Context(), chatRequest()); !errors.Is(err, ai.ErrUnavailable) {
			t.Errorf("2 MiB body: err = %v", err)
		}
	})

	t.Run("a provider nobody answers for is unreachable, and still unavailable", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close()

		p := ai.NewOpenAI(ai.NewKey(testKey), "gpt-test", localHTTP())
		p.Endpoint = url + "/v1/chat/completions"

		_, err := p.Complete(t.Context(), chatRequest())
		if !errors.Is(err, ai.ErrUnreachable) || !errors.Is(err, ai.ErrUnavailable) {
			t.Errorf("err = %v, want ErrUnreachable and ErrUnavailable", err)
		}

		if strings.Contains(err.Error(), url) {
			t.Errorf("the error names the address: %v", err)
		}
	})

	t.Run("an answer with no choices is an error", func(t *testing.T) {
		t.Parallel()

		p := newProvider(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"choices":[]}`) })

		if _, err := p.Complete(t.Context(), chatRequest()); !errors.Is(err, ai.ErrUnavailable) {
			t.Errorf("err = %v", err)
		}
	})
}

func TestCompatible(t *testing.T) {
	t.Parallel()

	var path string

	var body map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, chatOK)
	}))
	t.Cleanup(srv.Close)

	p := ai.NewCompatible(ai.NewKey(testKey), srv.URL+"/v1/", "local-model", localHTTP())

	resp, err := p.Complete(t.Context(), chatRequest())
	if err != nil || resp.Provider != ai.KindOpenAICompatible {
		t.Fatalf("resp = %+v, err = %v", resp, err)
	}

	if path != "/v1/chat/completions" || body["max_tokens"] != float64(80) || body["max_completion_tokens"] != nil ||
		body["model"] != "local-model" {
		t.Errorf("path = %q, body = %v", path, body)
	}

	t.Run("the factory refuses a private host in saas and allows it in personal", func(t *testing.T) {
		t.Parallel()

		cfg := ai.ProviderConfig{Kind: ai.KindOpenAICompatible, Key: ai.NewKey(testKey), BaseURL: srv.URL + "/v1", Model: "m"}

		if _, err := ai.NewProvider(false)(cfg); err == nil {
			t.Error("saas accepted a loopback base URL")
		}

		prov, err := ai.NewProvider(true)(cfg)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := prov.Complete(t.Context(), chatRequest()); err != nil {
			t.Errorf("personal with the private-host switch could not reach the local server: %v", err)
		}
	})
}

func TestBaseURL(t *testing.T) {
	t.Parallel()

	const compat = ai.KindOpenAICompatible

	cases := []struct {
		name    string
		kind    string
		url     string
		private bool
		ok      bool
	}{
		{"public https", compat, "https://api.example.com/v1", false, true},
		{"public https with port", compat, "https://llm.example.com:8443/v1", false, true},
		{"http refused in saas", compat, "http://api.example.com/v1", false, false},
		{"http to a public name refused in personal", compat, "http://api.example.com/v1", true, false},
		{"loopback ip refused in saas", compat, "https://127.0.0.1/v1", false, false},
		{"localhost refused in saas", compat, "https://localhost/v1", false, false},
		{"metadata ip refused in saas", compat, "https://169.254.169.254/latest", false, false},
		{"metadata ip refused in personal", compat, "https://169.254.169.254/latest", true, false},
		{"http metadata refused in personal", compat, "http://169.254.169.254/latest", true, false},
		{"private ip refused in saas", compat, "https://10.0.0.5/v1", false, false},
		{"private 192.168 refused in saas", compat, "https://192.168.1.10/v1", false, false},
		{"localhost with a trailing dot refused in saas", compat, "https://localhost./v1", false, false},
		{"a decimal ip refused in saas", compat, "https://2130706433/v1", false, false},
		{"a hex ip refused in saas", compat, "https://0x7f.0.0.1/v1", false, false},
		{"an octal ip refused in saas", compat, "https://017700000001/v1", false, false},
		{"a short ip refused in saas", compat, "https://127.1/v1", false, false},
		{"a decimal ip refused in personal with the switch", compat, "http://2130706433:11434/v1", true, false},
		{"localhost with a trailing dot allowed in personal with the switch", compat, "http://localhost.:11434/v1", true, true},
		{"ipv6 loopback refused in saas", compat, "https://[::1]/v1", false, false},
		{"ipv4-mapped loopback refused in saas", compat, "https://[::ffff:127.0.0.1]/v1", false, false},
		{"http loopback refused in saas", compat, "http://127.0.0.1:11434/v1", false, false},
		{"http loopback allowed in personal with the switch", compat, "http://127.0.0.1:11434/v1", true, true},
		{"http localhost allowed in personal with the switch", compat, "http://localhost:11434/v1", true, true},
		{"http lan ip allowed in personal with the switch", compat, "http://192.168.1.10:8080/v1", true, true},
		{"https private allowed in personal with the switch", compat, "https://10.0.0.5/v1", true, true},
		{"user info refused", compat, "https://user:pw@api.example.com/v1", false, false},
		{"query refused", compat, "https://api.example.com/v1?key=1", false, false},
		{"other scheme refused", compat, "ftp://api.example.com/v1", false, false},
		{"no scheme refused", compat, "api.example.com/v1", false, false},
		{"empty refused for compatible", compat, "", false, false},
		{"empty fine for anthropic", ai.KindAnthropic, "", false, true},
		{"a base url on openai refused", ai.KindOpenAI, "https://api.example.com/v1", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ai.CheckBaseURL(tc.kind, tc.url, tc.private)
			if (err == nil) != tc.ok {
				t.Errorf("CheckBaseURL(%q, private=%v) = %v, want ok=%v", tc.url, tc.private, err, tc.ok)
			}
		})
	}

	t.Run("a name that resolves to loopback is refused at connect time", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Error("the request reached a loopback server")
		}))
		t.Cleanup(srv.Close)

		// localhost passes a name check only where the switch is on, so build the
		// provider directly: this is the DNS-resolution case the safe client covers.
		port := srv.URL[strings.LastIndex(srv.URL, ":"):]
		p := ai.NewCompatible(ai.NewKey(testKey), "https://localhost"+port+"/v1", "m", ai.NewHTTP(ai.HTTPOptions{Retry: time.Millisecond}))

		_, err := p.Complete(t.Context(), chatRequest())
		if pe, ok := errors.AsType[*ai.ProviderError](err); !ok || pe.Type != "blocked_address" {
			t.Errorf("err = %v", err)
		}

		// What the client sees is what a connection that failed shows: it must not tell
		// an address the server refuses from one that did not answer (an internal network oracle).
		dead := httptest.NewServer(http.NotFoundHandler())
		deadURL := dead.URL
		dead.Close()

		_, failed := ai.NewCompatible(ai.NewKey(testKey), deadURL+"/v1", "m", ai.NewHTTP(ai.HTTPOptions{
			Retry: time.Millisecond, AllowPrivate: true,
		})).Complete(t.Context(), chatRequest())

		if !errors.Is(err, ai.ErrUnreachable) || !errors.Is(failed, ai.ErrUnreachable) || err.Error() != failed.Error() {
			t.Errorf("blocked = %v, failed = %v; want the same class and message", err, failed)
		}
	})

	t.Run("a redirect to the metadata address is not followed", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://169.254.169.254/latest/meta-data", http.StatusFound)
		}))
		t.Cleanup(srv.Close)

		p := ai.NewCompatible(ai.NewKey(testKey), srv.URL+"/v1", "m", localHTTP())

		_, err := p.Complete(t.Context(), chatRequest())

		pe, ok := errors.AsType[*ai.ProviderError](err)
		if !ok || pe.Status != http.StatusFound || !errors.Is(err, ai.ErrUnavailable) {
			t.Errorf("err = %v (%+v)", err, pe)
		}
	})
}
