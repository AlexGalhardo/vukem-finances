package tests

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func TestAuditEvents(t *testing.T) {
	t.Parallel()

	tm := newTeam(t)
	makePro(t, tm.app, tm.owner)

	// A refused action writes nothing.
	setRole(tm.member.Client, tm.viewer.User, models.RoleAdmin).AssertForbidden()
	vukemtest.AssertDatabaseCount(t, tm.app, "audit_events", 0)

	steps := []struct {
		action string
		actor  who
		do     func() *vukemtest.Response
	}{
		{workspaces.AuditSettingsChanged, tm.admin, func() *vukemtest.Response {
			return putForm(tm.admin.Client, "/workspace", url.Values{"name": {"Renamed"}, "currency": {"EUR"}})
		}},
		{workspaces.AuditInviteCreated, tm.admin, func() *vukemtest.Response {
			return tm.admin.Client.PostForm("/workspace/invites", url.Values{"email": {"new@example.com"}, "role": {"viewer"}})
		}},
		{workspaces.AuditRoleChanged, tm.admin, func() *vukemtest.Response {
			return setRole(tm.admin.Client, tm.viewer.User, models.RoleMember)
		}},
		{workspaces.AuditMemberRemoved, tm.admin, func() *vukemtest.Response {
			return tm.admin.Client.Delete("/workspace/members/" + idString(tm.viewer.User.ID))
		}},
		{workspaces.AuditMemberLeft, tm.member, func() *vukemtest.Response {
			return tm.member.Client.PostForm("/workspace/leave", nil)
		}},
		{workspaces.AuditOwnershipMoved, tm.owner, func() *vukemtest.Response {
			return transfer(tm.owner.Client, tm.admin.User)
		}},
	}

	for i, st := range steps {
		if got := st.do().Status(); got != http.StatusSeeOther {
			t.Fatalf("%s answered %d", st.action, got)
		}

		vukemtest.AssertDatabaseCount(t, tm.app, "audit_events", int64(i+1))
		vukemtest.AssertDatabaseHas(t, tm.app, "audit_events", map[string]any{
			"workspace_id": tm.owner.Workspace.ID, "actor_user_id": st.actor.User.ID, "action": st.action,
		})
	}

	// Owners and admins read the log on the workspace page; the others get none of it.
	tm.admin.Client.Inertia("").Get("/workspace").AssertInertia("Workspace").HasProp("audit", len(steps))
	tm.owner.Client.Inertia("").Get("/workspace").AssertInertia("Workspace").HasProp("audit", len(steps))

	// The log of one workspace never shows in another.
	bob := signUp(t, tm.app, "Bob", "bob@example.com")
	bob.Client.Inertia("").Get("/workspace").AssertInertia("Workspace").HasProp("audit", 0)

	db, err := tm.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	n, err := workspaces.Scoped[models.AuditEvent](workspaces.WithTenant(t.Context(), bob.Workspace.ID), db).Count(t.Context())
	if err != nil || n != 0 {
		t.Errorf("another workspace sees %d events, %v", n, err)
	}
}

func TestAuditLogIsHiddenFromMembersAndViewers(t *testing.T) {
	t.Parallel()

	tm := newTeam(t)
	setRole(tm.owner.Client, tm.viewer.User, models.RoleMember).AssertStatus(http.StatusSeeOther)

	tm.member.Client.Inertia("").Get("/workspace").AssertInertia("Workspace").HasProp("audit", 0)
	tm.viewer.Client.Inertia("").Get("/workspace").AssertInertia("Workspace").HasProp("audit", 0)
	tm.owner.Client.Inertia("").Get("/workspace").AssertInertia("Workspace").HasProp("audit", 1)
}
