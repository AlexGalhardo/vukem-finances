package fakes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func post(t *testing.T, f *Telegram, token, method string, body map[string]any) (int, map[string]any) {
	t.Helper()

	raw, _ := json.Marshal(body)

	res, err := http.Post(f.URL+"/bot"+token+"/"+method, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)

	return res.StatusCode, out
}

func TestFakeTelegram(t *testing.T) {
	t.Parallel()

	f := NewTelegram(t)

	if status, _ := post(t, f, "wrong", "getMe", nil); status != http.StatusUnauthorized {
		t.Fatalf("wrong token: %d", status)
	}

	if status, out := post(t, f, TelegramToken, "getMe", nil); status != 200 || out["ok"] != true {
		t.Fatalf("getMe: %d %v", status, out)
	}

	f.Enqueue(TextUpdate(1, 7, "a"))
	f.Enqueue(TextUpdate(2, 7, "b"))
	f.Enqueue(TextUpdate(3, 7, "c"))

	_, out := post(t, f, TelegramToken, "getUpdates", map[string]any{"offset": 2})
	if got := out["result"].([]any); len(got) != 2 {
		t.Fatalf("offset 2 should return updates 2 and 3, got %d", len(got))
	}

	_, out = post(t, f, TelegramToken, "getUpdates", map[string]any{"offset": 4})
	if got := out["result"].([]any); len(got) != 0 {
		t.Fatalf("offset 4 confirms everything, got %d", len(got))
	}

	post(t, f, TelegramToken, "sendMessage", map[string]any{"chat_id": 7, "text": "hi"})

	if got := f.Sent(7); len(got) != 1 || got[0] != "hi" {
		t.Fatalf("sent: %v", got)
	}

	f.FailSend(true)

	if status, _ := post(t, f, TelegramToken, "sendMessage", map[string]any{"chat_id": 7, "text": "x"}); status != http.StatusForbidden {
		t.Fatalf("blocked bot: %d", status)
	}

	post(t, f, TelegramToken, "setWebhook", map[string]any{"url": "https://x.test/h", "secret_token": "s3cr3t"})

	if u, s := f.Webhook(); u != "https://x.test/h" || s != "s3cr3t" {
		t.Fatalf("webhook: %q %q", u, s)
	}
}
