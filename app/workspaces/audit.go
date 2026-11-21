package workspaces

import (
	"context"
	"fmt"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// Actions written to audit_events.
const (
	AuditRoleChanged      = "member.role_changed"
	AuditOwnershipMoved   = "workspace.ownership_transferred"
	AuditMemberRemoved    = "member.removed"
	AuditMemberLeft       = "member.left"
	AuditInviteCreated    = "invite.created"
	AuditSettingsChanged  = "workspace.settings_changed"
	auditEventsPageLength = 50
)

// record writes one audit event of the workspace of ctx, inside ex (the
// transaction of the change itself when there is one, so the two stand or
// fall together).
func (s *Service) record(ctx context.Context, ex database.Executor, actor int64, action, subject string) error {
	return orm.Create(ctx, ex, &models.AuditEvent{ActorUserID: actor, Action: action, Subject: subject})
}

// AuditEntry is an audit event with the name of its actor.
type AuditEntry struct {
	ID        int64  `json:"id"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Subject   string `json:"subject"`
	CreatedAt string `json:"createdAt"`
}

// AuditEvents lists the latest audit events of the workspace of ctx, newest
// first. Only owners and admins may see them: the caller checks members.manage.
func (s *Service) AuditEvents(ctx context.Context) ([]AuditEntry, error) {
	events, err := Scoped[models.AuditEvent](ctx, s.DB).OrderBy("id", orm.Desc).Limit(auditEventsPageLength).Get(ctx)
	if err != nil || len(events) == 0 {
		return []AuditEntry{}, err
	}

	ids := make([]any, 0, len(events))
	for _, e := range events {
		ids = append(ids, e.ActorUserID)
	}

	users, err := orm.QueryOf[models.User](s.DB).WhereIn("id", ids...).Get(ctx)
	if err != nil {
		return nil, err
	}

	names := make(map[int64]string, len(users))
	for _, u := range users {
		names[u.ID] = u.Name
	}

	out := make([]AuditEntry, len(events))

	for i, e := range events {
		actor := names[e.ActorUserID]
		if actor == "" {
			actor = fmt.Sprintf("user %d", e.ActorUserID)
		}

		out[i] = AuditEntry{
			ID: e.ID, Actor: actor, Action: e.Action, Subject: e.Subject,
			CreatedAt: e.CreatedAt.Format("2006-01-02 15:04"),
		}
	}

	return out, nil
}
