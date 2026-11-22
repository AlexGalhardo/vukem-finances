package models

import (
	"context"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// Roles of an assistant message.
const (
	AssistantRoleUser      = "user"
	AssistantRoleAssistant = "assistant"
)

// AssistantConversation is a thread of questions of one person in one
// workspace. It is private to UserID: no query returns it to anyone else.
type AssistantConversation struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	UserID      int64  `json:"user_id"`
	Title       string `json:"title"`
}

// Table maps AssistantConversation to assistant_conversations.
func (AssistantConversation) Table() string { return "assistant_conversations" }

// AssistantMessage is one turn. Data is the JSON of the figures tables and the
// queries that backed an answer.
type AssistantMessage struct {
	orm.Model

	WorkspaceID    int64  `json:"workspace_id"`
	ConversationID int64  `json:"conversation_id"`
	UserID         int64  `json:"user_id"`
	Role           string `json:"role"`
	Content        string `json:"content"`
	Data           string `json:"data"`
}

// Table maps AssistantMessage to assistant_messages.
func (AssistantMessage) Table() string { return "assistant_messages" }

// SetTenant implements tenancy.Scoped.
func (m *AssistantConversation) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *AssistantConversation) BeforeCreate(ctx context.Context) error {
	return tenancy.Assign(ctx, m)
}

// SetTenant implements tenancy.Scoped.
func (m *AssistantMessage) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *AssistantMessage) BeforeCreate(ctx context.Context) error {
	if m.Data == "" {
		m.Data = "{}"
	}

	return tenancy.Assign(ctx, m)
}
