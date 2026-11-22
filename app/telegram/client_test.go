package telegram_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/config"

	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/tests/fakes"
)

func fakeClient(t *testing.T) (*telegram.Client, *fakes.Telegram) {
	t.Helper()

	f := fakes.NewTelegram(t)

	return telegram.NewTestClient(config.NewSecret(fakes.TelegramToken), f.URL), f
}

func TestClient(t *testing.T) {
	t.Parallel()

	c, f := fakeClient(t)
	ctx := t.Context()

	me, err := c.GetMe(ctx)
	if err != nil || me.Username != "fake_finance_bot" {
		t.Fatalf("getMe: %+v %v", me, err)
	}

	kb := telegram.Keyboard{{{Text: "Yes", CallbackData: "y"}, {Text: "No", CallbackData: "n"}}}
	if err := c.SendMessage(ctx, 42, "<b>hi</b>", kb); err != nil {
		t.Fatal(err)
	}

	call, ok := f.LastSent(42)
	if !ok {
		t.Fatal("nothing sent")
	}

	if call.Params["parse_mode"] != "HTML" {
		t.Errorf("parse_mode = %v, want HTML", call.Params["parse_mode"])
	}

	if got := fakes.Buttons(call); len(got) != 2 || got[0] != "y" {
		t.Errorf("buttons = %v", got)
	}

	f.Enqueue(fakes.TextUpdate(5, 42, "one"))
	f.Enqueue(fakes.TextUpdate(6, 42, "two"))

	ups, err := c.GetUpdates(ctx, 6)
	if err != nil || len(ups) != 1 || ups[0].UpdateID != 6 || ups[0].Message.Text != "two" {
		t.Fatalf("getUpdates(offset 6): %+v %v", ups, err)
	}

	if err := c.AnswerCallbackQuery(ctx, "cb", "done"); err != nil {
		t.Fatal(err)
	}

	if err := c.SetWebhook(ctx, "https://x.test/hook", "a-secret-value-123456"); err != nil {
		t.Fatal(err)
	}

	if u, s := f.Webhook(); u != "https://x.test/hook" || s != "a-secret-value-123456" {
		t.Errorf("webhook = %q %q", u, s)
	}

	f.FailSend(true)

	var apiErr *telegram.APIError
	if err := c.SendMessage(ctx, 42, "x", nil); !errors.As(err, &apiErr) || apiErr.Status != 403 {
		t.Errorf("blocked bot: %v", err)
	}
}

func TestClientTruncatesLongText(t *testing.T) {
	t.Parallel()

	c, f := fakeClient(t)

	if err := c.SendMessage(t.Context(), 1, strings.Repeat("é", telegram.MaxText+50), nil); err != nil {
		t.Fatal(err)
	}

	call, _ := f.LastSent(1)
	if n := len([]rune(call.Params["text"].(string))); n != telegram.MaxText {
		t.Errorf("sent %d characters, want %d", n, telegram.MaxText)
	}
}

func TestEscapeHTML(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{
		"plain":             "plain",
		"<b>bold</b>":       "&lt;b&gt;bold&lt;/b&gt;",
		"a & b":             "a &amp; b",
		"&lt;":              "&amp;lt;",
		"*not* _markdown_":  "*not* _markdown_",
		`<a href="x">y</a>`: `&lt;a href="x"&gt;y&lt;/a&gt;`,
	} {
		if got := telegram.EscapeHTML(in); got != want {
			t.Errorf("EscapeHTML(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTokenNeverInErrors(t *testing.T) {
	t.Parallel()

	const token = "777:very-secret-bot-token"

	// A server that echoes the request path (token included) in its error text.
	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": 400, "description": "bad request " + r.URL.Path})
	}))
	defer echo.Close()

	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()

	var logs bytes.Buffer

	log := slog.New(slog.NewTextHandler(&logs, nil))

	for name, base := range map[string]string{"echoing server": echo.URL, "connection refused": closedURL} {
		c := telegram.NewTestClient(config.NewSecret(token), base)
		ctx := context.Background()

		errs := []error{
			c.SendMessage(ctx, 1, "x", nil),
			c.AnswerCallbackQuery(ctx, "id", ""),
			c.SetWebhook(ctx, "https://x.test", "s"),
		}

		_, getErr := c.GetMe(ctx)
		_, updErr := c.GetUpdates(ctx, 0)
		errs = append(errs, getErr, updErr)

		for _, err := range errs {
			if err == nil {
				t.Fatalf("%s: expected an error", name)
			}

			log.Error("call failed", "err", err)

			for _, text := range []string{err.Error(), fmt.Sprintf("%v %+v %#v", err, err, err)} {
				if strings.Contains(text, token) || strings.Contains(text, "very-secret") {
					t.Errorf("%s: the token is in an error: %s", name, text)
				}
			}
		}

		for _, text := range []string{fmt.Sprint(c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c)} {
			if strings.Contains(text, "very-secret") {
				t.Errorf("%s: the token is in the printed client", name)
			}
		}
	}

	if strings.Contains(logs.String(), "very-secret") {
		t.Errorf("the token is in the log: %s", logs.String())
	}
}
