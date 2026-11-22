// Package fakes holds fake servers for the tests: nothing here reaches a real
// network and no test needs a real token.
package fakes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TelegramToken is the bot token the fake accepts.
const TelegramToken = "123456:fake-token-for-tests-only"

// Call is one request the fake received.
type Call struct {
	Method string
	Params map[string]any
}

// Telegram stands in for api.telegram.org: getMe, getUpdates with offsets,
// sendMessage, editMessageReplyMarkup, answerCallbackQuery, setWebhook and
// deleteWebhook. It records every call.
type Telegram struct {
	URL string

	t   *testing.T
	srv *httptest.Server

	mu            sync.Mutex
	updates       []map[string]any
	calls         []Call
	nextMessage   int
	failSend      bool
	webhookURL    string
	webhookSecret string
	pollWait      time.Duration
}

// NewTelegram starts the fake and stops it with the test.
func NewTelegram(t *testing.T) *Telegram {
	t.Helper()

	f := &Telegram{t: t, nextMessage: 100, pollWait: 20 * time.Millisecond}
	f.srv = httptest.NewServer(http.HandlerFunc(f.serve))
	f.URL = f.srv.URL

	t.Cleanup(f.srv.Close)

	return f
}

func (f *Telegram) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	prefix := "/bot" + TelegramToken + "/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": 401, "description": "Unauthorized"})

		return
	}

	method := strings.TrimPrefix(r.URL.Path, prefix)
	params := map[string]any{}

	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&params)
	}

	f.mu.Lock()
	f.calls = append(f.calls, Call{Method: method, Params: params})
	f.mu.Unlock()

	reply := func(result any) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result})
	}

	switch method {
	case "getMe":
		reply(map[string]any{"id": 123456, "is_bot": true, "first_name": "Fake", "username": "fake_finance_bot"})
	case "getUpdates":
		reply(f.pollUpdates(params))
	case "sendMessage":
		f.mu.Lock()
		fail := f.failSend
		f.nextMessage++
		id := f.nextMessage
		f.mu.Unlock()

		if fail {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok": false, "error_code": 403, "description": "Forbidden: bot was blocked by the user",
			})

			return
		}

		reply(map[string]any{"message_id": id, "chat": map[string]any{"id": params["chat_id"]}, "text": params["text"]})
	case "editMessageReplyMarkup", "answerCallbackQuery", "deleteWebhook":
		reply(true)
	case "setWebhook":
		f.mu.Lock()
		f.webhookURL, _ = params["url"].(string)
		f.webhookSecret, _ = params["secret_token"].(string)
		f.mu.Unlock()
		reply(true)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": 404, "description": "Not Found"})
	}
}

// pollUpdates answers getUpdates: updates below offset are confirmed (dropped),
// the rest are returned, up to limit. With none, it waits a moment, like a
// long poll that times out.
func (f *Telegram) pollUpdates(params map[string]any) []map[string]any {
	offset, _ := params["offset"].(float64)
	limit := 100

	if l, ok := params["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	for range 2 {
		f.mu.Lock()

		kept := f.updates[:0]
		for _, u := range f.updates {
			if id, _ := u["update_id"].(int); float64(id) >= offset {
				kept = append(kept, u)
			}
		}

		f.updates = kept
		out := append([]map[string]any(nil), kept...)
		wait := f.pollWait
		f.mu.Unlock()

		if len(out) > 0 {
			if len(out) > limit {
				out = out[:limit]
			}

			return out
		}

		time.Sleep(wait)
	}

	return []map[string]any{}
}

// Enqueue adds an update the next getUpdates returns. update must carry an int
// update_id.
func (f *Telegram) Enqueue(update map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.updates = append(f.updates, update)
}

// TextUpdate is a private-chat text message from user chatID.
func TextUpdate(updateID int, chatID int64, text string) map[string]any {
	return ChatUpdate(updateID, chatID, "private", text)
}

// ChatUpdate is a text message in a chat of the given type.
func ChatUpdate(updateID int, chatID int64, chatType, text string) map[string]any {
	return map[string]any{
		"update_id": updateID,
		"message": map[string]any{
			"message_id": updateID, "date": time.Now().Unix(),
			"from": map[string]any{"id": chatID, "is_bot": false, "first_name": "Test"},
			"chat": map[string]any{"id": chatID, "type": chatType},
			"text": text,
		},
	}
}

// CallbackUpdate is a button tap in a private chat.
func CallbackUpdate(updateID int, chatID int64, data string) map[string]any {
	return map[string]any{
		"update_id": updateID,
		"callback_query": map[string]any{
			"id":   fmt.Sprintf("cb-%d", updateID),
			"from": map[string]any{"id": chatID, "is_bot": false, "first_name": "Test"},
			"message": map[string]any{
				"message_id": updateID, "date": time.Now().Unix(),
				"chat": map[string]any{"id": chatID, "type": "private"},
			},
			"data": data,
		},
	}
}

// FailSend makes sendMessage answer 403, like a user who blocked the bot.
func (f *Telegram) FailSend(on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.failSend = on
}

// Calls returns the recorded calls of method ("" for all).
func (f *Telegram) Calls(method string) []Call {
	f.mu.Lock()
	defer f.mu.Unlock()

	var out []Call

	for _, c := range f.calls {
		if method == "" || c.Method == method {
			out = append(out, c)
		}
	}

	return out
}

// Sent returns the text of every sendMessage to chat, oldest first.
func (f *Telegram) Sent(chat int64) []string {
	var out []string

	for _, c := range f.Calls("sendMessage") {
		if id, _ := c.Params["chat_id"].(float64); int64(id) == chat {
			text, _ := c.Params["text"].(string)
			out = append(out, text)
		}
	}

	return out
}

// LastSent is the newest sendMessage call to chat; ok is false when none.
func (f *Telegram) LastSent(chat int64) (Call, bool) {
	var last Call

	found := false

	for _, c := range f.Calls("sendMessage") {
		if id, _ := c.Params["chat_id"].(float64); int64(id) == chat {
			last, found = c, true
		}
	}

	return last, found
}

// Buttons lists the callback_data of the inline keyboard of a sendMessage call.
func Buttons(c Call) []string {
	markup, _ := c.Params["reply_markup"].(map[string]any)
	rows, _ := markup["inline_keyboard"].([]any)

	var out []string

	for _, row := range rows {
		cells, _ := row.([]any)
		for _, cell := range cells {
			b, _ := cell.(map[string]any)
			if d, ok := b["callback_data"].(string); ok {
				out = append(out, d)
			}
		}
	}

	return out
}

// Webhook returns what the last setWebhook registered.
func (f *Telegram) Webhook() (url, secret string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.webhookURL, f.webhookSecret
}

// Reset forgets the recorded calls.
func (f *Telegram) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = nil
}
