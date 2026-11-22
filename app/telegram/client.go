// Package telegram is the Telegram bot of the application: a small client of
// the Bot API (net/http only, no library), the update handling, the account
// linking and the commands (docs/telegram.md).
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/config"
	"github.com/AlexGalhardo/vukem/httpclient"
)

// DefaultBase is the Bot API host. Only the tests point the client elsewhere.
const DefaultBase = "https://api.telegram.org"

const (
	callTimeout = 10 * time.Second
	pollTimeout = 35 * time.Second
	// pollSeconds is the long-poll wait sent as getUpdates timeout.
	pollSeconds = 30
	maxResponse = 1 << 20
	// MaxText is Telegram's limit for a message, in characters.
	MaxText = 4096
	redact  = "[redacted]"
)

// APIError is a refusal of the Bot API. It names the method, the status and
// Telegram's description, never the token.
type APIError struct {
	Method      string
	Status      int
	Description string
}

// Error implements error.
func (e *APIError) Error() string {
	return fmt.Sprintf("telegram: %s: %d %s", e.Method, e.Status, e.Description)
}

// Client calls the Bot API.
type Client struct {
	token config.Secret
	base  string
	short *http.Client
	long  *http.Client
}

// NewClient builds the client that talks to the real Bot API through the safe
// HTTP client (public addresses only).
func NewClient(token config.Secret) *Client {
	return &Client{
		token: token, base: DefaultBase,
		short: httpclient.Safe(httpclient.Options{Timeout: callTimeout, MaxBody: maxResponse}),
		long:  httpclient.Safe(httpclient.Options{Timeout: pollTimeout, MaxBody: maxResponse}),
	}
}

// NewTestClient builds a client for a fake server at base. It uses plain HTTP
// clients (the safe client refuses loopback), so it is for tests and for
// development only: the app builds it only outside production.
func NewTestClient(token config.Secret, base string) *Client {
	return &Client{
		token: token, base: strings.TrimSuffix(base, "/"),
		short: &http.Client{Timeout: callTimeout}, long: &http.Client{Timeout: pollTimeout},
	}
}

// String and GoString keep the token out of anything that prints the client.
func (*Client) String() string { return "telegram.Client{token: " + redact + "}" }

// GoString implements fmt.GoStringer.
func (c *Client) GoString() string { return c.String() }

// User is a Telegram user.
type User struct {
	ID       int64  `json:"id"`
	IsBot    bool   `json:"is_bot"`
	Username string `json:"username"`
}

// Chat is a Telegram chat. Only Type "private" is served.
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// Message is a Telegram message, reduced to what the bot reads.
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

// CallbackQuery is a tap on an inline button.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

// Update is one item of getUpdates or one webhook body.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

// Button is an inline keyboard button that sends CallbackData back (at most 64 bytes).
type Button struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// Keyboard is an inline keyboard: rows of buttons.
type Keyboard [][]Button

// EscapeHTML escapes the three characters Telegram's HTML mode reads. Every
// string that comes from a person or from the database goes through it before
// it is put into a message.
func EscapeHTML(s string) string { return htmlEscaper.Replace(s) }

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

type envelope struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
}

// scrub removes the token from text. The token sits in the request path, and
// the standard library repeats the URL in transport errors.
func (c *Client) scrub(text string) string {
	if tok := c.token.Reveal(); tok != "" {
		text = strings.ReplaceAll(text, tok, redact)
		text = strings.ReplaceAll(text, url.PathEscape(tok), redact)
	}

	return text
}

func (c *Client) call(ctx context.Context, hc *http.Client, method string, params, out any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("telegram: %s: encode: %w", method, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/bot"+c.token.Reveal()+"/"+method, bytes.NewReader(body))
	if err != nil {
		return errors.New("telegram: " + method + ": " + c.scrub(err.Error()))
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := hc.Do(req)
	if err != nil {
		// A *url.Error carries the URL, token included: keep only the cause.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}

		return errors.New("telegram: " + method + ": " + c.scrub(err.Error()))
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponse))
	if err != nil {
		return errors.New("telegram: " + method + ": read: " + c.scrub(err.Error()))
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return &APIError{Method: method, Status: res.StatusCode, Description: "unreadable response"}
	}

	if !env.OK {
		status := env.ErrorCode
		if status == 0 {
			status = res.StatusCode
		}

		desc := c.scrub(env.Description)
		if utf8.RuneCountInString(desc) > 200 {
			desc = string([]rune(desc)[:200])
		}

		return &APIError{Method: method, Status: status, Description: desc}
	}

	if out == nil {
		return nil
	}

	if err := json.Unmarshal(env.Result, out); err != nil {
		return fmt.Errorf("telegram: %s: decode result: %w", method, err)
	}

	return nil
}

// GetMe checks the token and returns the bot's own user.
func (c *Client) GetMe(ctx context.Context) (User, error) {
	var u User

	return u, c.call(ctx, c.short, "getMe", struct{}{}, &u)
}

// GetUpdates long-polls for updates from offset on, asking only for messages
// and button taps. Telegram forgets the updates below offset.
func (c *Client) GetUpdates(ctx context.Context, offset int64) ([]Update, error) {
	var out []Update

	err := c.call(ctx, c.long, "getUpdates", map[string]any{
		"offset": offset, "timeout": pollSeconds, "limit": 100,
		"allowed_updates": []string{"message", "callback_query"},
	}, &out)

	return out, err
}

// SetWebhook registers url, with secret echoed back in X-Telegram-Bot-Api-Secret-Token.
func (c *Client) SetWebhook(ctx context.Context, hookURL, secret string) error {
	return c.call(ctx, c.short, "setWebhook", map[string]any{
		"url": hookURL, "secret_token": secret, "max_connections": 10,
		"allowed_updates": []string{"message", "callback_query"},
	}, nil)
}

// DeleteWebhook removes the webhook, which polling needs.
func (c *Client) DeleteWebhook(ctx context.Context) error {
	return c.call(ctx, c.short, "deleteWebhook", map[string]any{"drop_pending_updates": false}, nil)
}

// SendMessage sends text as HTML to chatID, with an optional inline keyboard.
// The caller escapes every user or database string with EscapeHTML first.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, kb Keyboard) error {
	if utf8.RuneCountInString(text) > MaxText {
		text = string([]rune(text)[:MaxText-1]) + "…"
	}

	params := map[string]any{
		"chat_id": chatID, "text": text, "parse_mode": "HTML",
		"link_preview_options": map[string]any{"is_disabled": true},
	}
	if len(kb) > 0 {
		params["reply_markup"] = map[string]any{"inline_keyboard": kb}
	}

	return c.call(ctx, c.short, "sendMessage", params, nil)
}

// EditMessageReplyMarkup removes the keyboard of a message, so a spent button
// cannot be tapped again.
func (c *Client) EditMessageReplyMarkup(ctx context.Context, chatID, messageID int64) error {
	return c.call(ctx, c.short, "editMessageReplyMarkup", map[string]any{
		"chat_id": chatID, "message_id": messageID, "reply_markup": map[string]any{"inline_keyboard": [][]Button{}},
	}, nil)
}

// AnswerCallbackQuery stops the spinner of a tapped button, with an optional short notice.
func (c *Client) AnswerCallbackQuery(ctx context.Context, id, text string) error {
	params := map[string]any{"callback_query_id": id}
	if text != "" {
		params["text"] = text
	}

	return c.call(ctx, c.short, "answerCallbackQuery", params, nil)
}
