package tests

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"
)

// The SSRF attack suite (epic 24 task 6). The only outbound URL a person
// controls in this product is the base URL of an OpenAI-compatible AI server
// (the client metadata fetch of remote MCP is attacked in
// TestSecurity_MCP_ClientMetadata). DNS rebinding is closed at connect time by
// the safe client (httpclient.Allowed after resolution), which the unit tests of
// app/ai prove; a DNS answer cannot be faked from here.

var secSSRFBases = []string{
	"http://example.com/v1",
	"http://169.254.169.254/latest/meta-data",
	"https://169.254.169.254/latest/meta-data",
	"https://127.0.0.1/v1",
	"https://127.0.0.1:8443/v1",
	"https://localhost/v1",
	"https://foo.localhost/v1",
	"https://10.0.0.5/v1",
	"https://172.16.0.1/v1",
	"https://192.168.1.10/v1",
	"https://100.64.0.1/v1",
	"https://0.0.0.0/v1",
	"https://[::1]/v1",
	"https://[::]/v1",
	"https://[::ffff:127.0.0.1]/v1",
	"https://[fe80::1]/v1",
	"https://[fd00::1]/v1",
	"https://user:" + "word" + "@api.example.com/v1",
	"https://api.example.com/v1?key=1",
	"https://api.example.com/v1#frag",
	"file:///etc/passwd",
	"ftp://api.example.com/v1",
	"gopher://api.example.com/v1",
	"javascript:alert(1)",
	"//api.example.com/v1",
	"api.example.com/v1",
	"https://",
	"",
}

func TestSecurity_SSRF_AIBaseURLInSaaS(t *testing.T) {
	t.Parallel()

	app := newSaaSApp(t, map[string]string{"AI_ALLOW_PRIVATE_HOSTS": "true"})
	ana := signUp(t, app, "Ana", "ana@example.com")

	for _, base := range secSSRFBases {
		t.Run(base, func(t *testing.T) {
			form := aiKeyForm("openai_compatible")
			form.Set("baseUrl", base)

			res := ana.Client.PostForm("/ai/key", form)
			if res.Status() == http.StatusTooManyRequests {
				t.Fatal("rate limited: the case proves nothing")
			}

			if got := aiCredentials(t, app, ana); len(got) != 0 {
				// Each case starts with none, so one acceptance fails one case and not the rest.
				stackless(t, app, "DELETE FROM ai_credentials")
				t.Fatalf("%q was accepted", base)
			}

			if strings.Contains(fmt.Sprint(res.Session())+res.Body(), aiMarker) {
				t.Error("the refusal echoes the key")
			}
		})
	}
}

// The early check of the base URL is a courtesy; the safe client refuses to
// connect to a private address after it resolved the name, and this is the proof
// that matters: nothing reaches a listener on this machine, however the name is
// spelled.
func TestSecurity_SSRF_SaaSNeverConnectsToThisMachine(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = ln.Close() })

	var accepted atomic.Int32

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}

			accepted.Add(1)
			_ = c.Close()
		}
	}()

	_, port, _ := net.SplitHostPort(ln.Addr().String())

	app := newSaaSApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")

	for _, host := range []string{"localhost.", "2130706433", "0x7f.0.0.1", "017700000001", "127.1", "localhost"} {
		t.Run(host, func(t *testing.T) {
			form := aiKeyForm("openai_compatible")
			form.Set("baseUrl", "https://"+host+":"+port+"/v1")
			ana.Client.PostForm("/ai/key", form)

			ana.Client.PostForm("/ai/check", url.Values{})
			stackless(t, app, "DELETE FROM ai_credentials")

			if n := accepted.Load(); n != 0 {
				t.Fatalf("a connection reached this machine through %q", host)
			}
		})
	}
}

func TestSecurity_SSRF_AIBaseURLInPersonal(t *testing.T) {
	t.Parallel()

	fake := newAIFakeServer(t, http.StatusOK)

	t.Run("without the switch a local server is refused", func(t *testing.T) {
		app := newPersonalApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")

		for _, base := range []string{fake.URL + "/v1", "http://127.0.0.1:11434/v1", "http://localhost:11434/v1", "https://10.0.0.5/v1"} {
			form := aiKeyForm("openai_compatible")
			form.Set("baseUrl", base)
			ana.Client.PostForm("/ai/key", form)

			if got := aiCredentials(t, app, ana); len(got) != 0 {
				t.Fatalf("%s was accepted without the switch: %+v", base, got)
			}
		}

		if fake.hits.Load() != 0 {
			t.Fatalf("the server was called %d times", fake.hits.Load())
		}
	})

	t.Run("with the switch only a local host may use http, and nothing public or link-local", func(t *testing.T) {
		app := newPersonalApp(t, map[string]string{"AI_ALLOW_PRIVATE_HOSTS": "true"})
		ana := signUp(t, app, "Ana", "ana@example.com")

		for _, base := range []string{
			"http://example.com/v1",
			"http://127.0.0.1.evil.example/v1",
			"http://localhost.evil.example/v1",
			"http://localhost@evil.example/v1",
			"http://169.254.169.254/latest/meta-data",
			"https://169.254.169.254/latest/meta-data",
			"http://[fe80::1]/v1",
			"ftp://127.0.0.1/v1",
		} {
			form := aiKeyForm("openai_compatible")
			form.Set("baseUrl", base)
			ana.Client.PostForm("/ai/key", form)

			if got := aiCredentials(t, app, ana); len(got) != 0 {
				t.Errorf("%s was accepted: %+v", base, got)

				for _, c := range got {
					_ = c
				}

				break
			}
		}
	})

	t.Run("a redirect from the provider is never followed", func(t *testing.T) {
		app := newPersonalApp(t, map[string]string{"AI_ALLOW_PRIVATE_HOSTS": "true"})
		ana := signUp(t, app, "Ana", "ana@example.com")

		var internal atomic.Int32

		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			internal.Add(1)
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(target.Close)

		var hops atomic.Int32

		redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hops.Add(1)
			http.Redirect(w, r, target.URL+"/secret", http.StatusTemporaryRedirect)
		}))
		t.Cleanup(redirector.Close)

		form := aiKeyForm("openai_compatible")
		form.Set("baseUrl", redirector.URL+"/v1")
		ana.Client.PostForm("/ai/key", form).AssertRedirect("/ai")

		ana.Client.PostForm("/ai/check", url.Values{})

		if hops.Load() == 0 {
			t.Fatal("the redirecting server was never called: the case proves nothing")
		}

		if internal.Load() != 0 {
			t.Fatalf("the redirect was followed %d times", internal.Load())
		}
	})
}

func stackless(t *testing.T, app *vukemtest.App, query string) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.SQL().ExecContext(t.Context(), query); err != nil {
		t.Fatal(err)
	}
}
