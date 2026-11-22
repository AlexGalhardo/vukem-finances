package assistant

import (
	"context"
	"encoding/json"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// listCap bounds the conversations a person's list shows.
const listCap = 50

// ConversationView is a conversation in the list.
type ConversationView struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updatedAt"`
}

// MessageView is one turn. Reply is set on an assistant message.
type MessageView struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Reply   *Reply `json:"reply"`
}

// cutoff is the moment before which a conversation no longer exists for its
// owner, whether or not the daily job has removed it yet.
func (s *Service) cutoff() time.Time { return s.now().UTC().Add(-Retention) }

// mine is the conversation id of the person userID in the workspace of ctx. A
// conversation of anyone else, of another workspace or past retention is not
// found (a 404): no role, an owner and an admin included, reads another
// person's conversation.
func (s *Service) mine(ctx context.Context, ex database.Executor, userID, id int64) (models.AssistantConversation, error) {
	return workspaces.Scoped[models.AssistantConversation](ctx, ex).Where("id", id).Where("user_id", userID).
		Where("updated_at", ">=", s.cutoff()).First(ctx)
}

// Conversations lists the person's conversations, newest first.
func (s *Service) Conversations(ctx context.Context, userID int64) ([]ConversationView, error) {
	rows, err := workspaces.Scoped[models.AssistantConversation](ctx, s.DB).Where("user_id", userID).
		Where("updated_at", ">=", s.cutoff()).OrderBy("updated_at", orm.Desc).OrderBy("id", orm.Desc).
		Limit(listCap).Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]ConversationView, len(rows))
	for i, c := range rows {
		out[i] = ConversationView{ID: c.ID, Title: c.Title, UpdatedAt: c.UpdatedAt.UTC().Format("2006-01-02 15:04")}
	}

	return out, nil
}

// Messages returns the turns of one of the person's conversations, oldest first.
func (s *Service) Messages(ctx context.Context, userID, id int64) ([]MessageView, error) {
	if _, err := s.mine(ctx, s.DB, userID, id); err != nil {
		return nil, err
	}

	rows, err := workspaces.Scoped[models.AssistantMessage](ctx, s.DB).Where("conversation_id", id).
		Where("user_id", userID).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]MessageView, len(rows))

	for i, m := range rows {
		out[i] = MessageView{Role: m.Role, Content: m.Content}

		if m.Role == models.AssistantRoleAssistant {
			var r Reply
			if json.Unmarshal([]byte(m.Data), &r) == nil {
				r.Text, r.ConversationID = m.Content, id
				out[i].Reply = &r
			}
		}
	}

	return out, nil
}

// Delete removes one of the person's conversations with its messages.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if _, err := s.mine(ctx, tx, userID, id); err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.AssistantMessage](ctx, tx).Where("conversation_id", id).Delete(ctx); err != nil {
			return err
		}

		_, err := workspaces.Scoped[models.AssistantConversation](ctx, tx).Where("id", id).Delete(ctx)

		return err
	})
}

// DeleteAll removes every conversation of the person in the workspace, in one
// click. It returns how many conversations went.
func (s *Service) DeleteAll(ctx context.Context, userID int64) (int64, error) {
	var n int64

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		if _, err := workspaces.Scoped[models.AssistantMessage](ctx, tx).Where("user_id", userID).Delete(ctx); err != nil {
			return err
		}

		var err error

		n, err = workspaces.Scoped[models.AssistantConversation](ctx, tx).Where("user_id", userID).Delete(ctx)

		return err
	})

	return n, err
}

// Prune removes the conversations nobody has touched for Retention, in every
// workspace (the daily job). It returns how many went.
func Prune(ctx context.Context, db *database.DB, now time.Time) (int64, error) {
	cut := now.UTC().Add(-Retention)

	if _, err := db.ExecContext(ctx, `DELETE FROM assistant_messages WHERE conversation_id IN
		(SELECT id FROM assistant_conversations WHERE updated_at < ?)`, cut); err != nil {
		return 0, err
	}

	res, err := db.ExecContext(ctx, `DELETE FROM assistant_conversations WHERE updated_at < ?`, cut)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}
