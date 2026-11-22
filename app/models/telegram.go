package models

import (
	"database/sql"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
)

// The Telegram tables are server state with no workspace_id (see the
// migration): a link belongs to a person, and the workspace is resolved from
// the live membership at every message. They are not TenantModels.

// TelegramLink binds one private chat to one person. ChatID is unique: a
// revoked row is reused when the chat links again. CurrentWorkspaceID is where
// the bot works now; it is only a preference, the role is read from the
// membership at every message.
type TelegramLink struct {
	orm.Model

	UserID             int64            `json:"user_id"`
	ChatID             int64            `json:"chat_id"`
	CurrentWorkspaceID sql.Null[int64]  `json:"current_workspace_id"`
	DefaultAccountUID  sql.Null[string] `json:"default_account_uid"`
	Language           string           `json:"language"`
	RevokedAt          orm.NullTime     `json:"revoked_at"`
	LastSeenAt         orm.NullTime     `json:"last_seen_at"`
}

// Table maps TelegramLink to telegram_links.
func (TelegramLink) Table() string { return "telegram_links" }

// TelegramLinkCode is a single-use code that proves the signed-in owner of an
// account is the one linking a chat. Only the SHA-256 of the code is stored.
type TelegramLinkCode struct {
	orm.Model

	CodeHash       string       `json:"-"`
	UserID         int64        `json:"user_id"`
	ForWorkspaceID int64        `json:"for_workspace_id"`
	ExpiresAt      time.Time    `json:"expires_at"`
	UsedAt         orm.NullTime `json:"used_at"`
}

// Table maps TelegramLinkCode to telegram_link_codes.
func (TelegramLinkCode) Table() string { return "telegram_link_codes" }

// TelegramPending is a draft entry waiting for a tap. After the tap it keeps
// the uid of the transaction it made, for the 60 second undo.
type TelegramPending struct {
	orm.Model

	Ref            string           `json:"ref"`
	ChatID         int64            `json:"chat_id"`
	UserID         int64            `json:"user_id"`
	ForWorkspaceID int64            `json:"for_workspace_id"`
	Payload        string           `json:"-"`
	ExpiresAt      time.Time        `json:"expires_at"`
	ConsumedAt     orm.NullTime     `json:"consumed_at"`
	TransactionUID sql.Null[string] `json:"transaction_uid"`
	UndoneAt       orm.NullTime     `json:"undone_at"`
}

// Table maps TelegramPending to telegram_pending.
func (TelegramPending) Table() string { return "telegram_pending" }

// TelegramUpdate records a processed update id: the idempotency key.
type TelegramUpdate struct {
	orm.Model

	UpdateID int64 `json:"update_id"`
}

// Table maps TelegramUpdate to telegram_updates.
func (TelegramUpdate) Table() string { return "telegram_updates" }

// TelegramState is the polling offset and the lease of the polling worker.
type TelegramState struct {
	orm.Model

	Name       string       `json:"name"`
	NextOffset int64        `json:"next_offset"`
	LeaseOwner string       `json:"lease_owner"`
	LeaseUntil orm.NullTime `json:"lease_until"`
}

// Table maps TelegramState to telegram_state.
func (TelegramState) Table() string { return "telegram_state" }

// TelegramCounter is one durable counter of the bot rate limits.
type TelegramCounter struct {
	orm.Model

	Bucket    string    `json:"bucket"`
	Count     int64     `json:"count"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Table maps TelegramCounter to telegram_counters.
func (TelegramCounter) Table() string { return "telegram_counters" }
