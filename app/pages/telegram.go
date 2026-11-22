package pages

import "github.com/AlexGalhardo/vukem/inertia"

// Telegram is the Telegram settings page: link status, the connect button, the
// linked chats and the alert switch.
var Telegram = inertia.Define[TelegramProps]("Settings/Telegram")

// TelegramLinkView is a linked chat as the page may show it. The chat id is
// never sent, only its last four digits to tell chats apart.
type TelegramLinkView struct {
	ID         int64  `json:"id"`
	ChatHint   string `json:"chatHint"`
	Language   string `json:"language"`
	LinkedAt   string `json:"linkedAt"`
	LastSeenAt string `json:"lastSeenAt"`
}

// TelegramCode is the one-time answer of Connect: the deep link that carries
// the single-use code. It is set only on the response that makes it.
type TelegramCode struct {
	Link string `json:"link"`
	// Code is the same code, to paste after /start when the link cannot be opened.
	Code      string `json:"code"`
	ExpiresAt string `json:"expiresAt"`
	// Seconds is how long the code works from the moment of the response.
	Seconds int `json:"seconds"`
}

// TelegramProps are the props of Telegram.
type TelegramProps struct {
	Username string             `json:"username"`
	Links    []TelegramLinkView `json:"links"`
	// AlertsOn is the person's switch for the alerts of this workspace.
	AlertsOn bool          `json:"alertsOn"`
	MaxLinks int           `json:"maxLinks"`
	Created  *TelegramCode `json:"created"`
}
