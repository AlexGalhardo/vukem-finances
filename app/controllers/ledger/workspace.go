package ledger

import (
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func view(w models.Workspace) pages.WorkspaceView {
	return pages.WorkspaceView{ID: w.ID, Name: w.Name, Currency: w.Currency, Timezone: w.Timezone}
}

// Workspace shows the settings, the members and the invites.
func (h *Handlers) Workspace(c *web.Ctx) error {
	ws, err := h.WS.Current(c.Context())
	if err != nil {
		return err
	}

	members, err := h.WS.Members(c.Context())
	if err != nil {
		return err
	}

	invites, err := h.WS.Invites(c.Context())
	if err != nil {
		return err
	}

	lim, err := h.WS.Limits(c.Context())
	if err != nil {
		return err
	}

	canManage := h.can(c, workspaces.AbilityManageMembers) == nil

	// The audit log is for the people who manage the workspace only.
	audit := []workspaces.AuditEntry{}

	if canManage {
		if audit, err = h.WS.AuditEvents(c.Context()); err != nil {
			return err
		}
	}

	return pages.Workspace.Render(c, pages.WorkspaceProps{
		WorkspaceDetails: view(ws), Members: members, Invites: invites, Limits: lim,
		CanManage: canManage, Audit: audit,
	})
}

// UpdateWorkspace renames the workspace and sets its currency.
func (h *Handlers) UpdateWorkspace(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageMembers); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.WorkspaceSchema)
	if err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if err := h.WS.Rename(c.Context(), u, in.Name, in.Currency, in.Timezone); err != nil {
		return back(c, "/workspace", err)
	}

	return done(c, "/workspace", "Workspace saved.")
}

// CreateWorkspace makes another workspace owned by the user and switches to it.
func (h *Handlers) CreateWorkspace(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.WorkspaceSchema)
	if err != nil {
		return err
	}

	ws, err := h.WS.Create(c.Context(), u, in.Name, in.Currency)
	if err != nil {
		return back(c, "/workspace", err)
	}

	if err := workspaces.Switch(c.Request(), ws.ID); err != nil {
		return err
	}

	return done(c, "/dashboard", "Workspace created.")
}

// SwitchWorkspace makes another workspace of the user the current one. The
// next request checks the membership, so a workspace that is not theirs is a 404.
func (h *Handlers) SwitchWorkspace(c *web.Ctx) error {
	wid, err := c.ParamInt("workspace")
	if err != nil {
		return err
	}

	if err := workspaces.Switch(c.Request(), wid); err != nil {
		return err
	}

	return c.Redirect("/dashboard")
}

// Invite emails an invitation.
func (h *Handlers) Invite(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageMembers); err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.InviteSchema)
	if err != nil {
		return err
	}

	link, err := h.WS.Invite(c.Context(), u, in.Email, in.Role)
	if err != nil {
		return back(c, "/workspace", err)
	}

	// A personal installation usually has no mail server, so nobody gets the
	// email: show the owner the link once. It is a secret, flashed to the
	// next page only.
	if h.WS.Edition.Personal() {
		inertia.Flash(c, "inviteLink", link)
	}

	return done(c, "/workspace", "Invitation sent.")
}

// RemoveMember takes a member out of the workspace.
func (h *Handlers) RemoveMember(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageMembers); err != nil {
		return err
	}

	uid, err := c.ParamInt("user")
	if err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if err := h.WS.RemoveMember(c.Context(), u, uid); err != nil {
		return back(c, "/workspace", err)
	}

	return done(c, "/workspace", "Member removed.")
}

// ChangeRole sets the role of a member.
func (h *Handlers) ChangeRole(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityManageMembers); err != nil {
		return err
	}

	uid, err := c.ParamInt("user")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.MemberRoleSchema)
	if err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if err := h.WS.ChangeRole(c.Context(), u, uid, in.Role); err != nil {
		return back(c, "/workspace", err)
	}

	return done(c, "/workspace", "Role changed.")
}

// TransferOwnership hands the workspace to another member; the owner becomes an admin.
func (h *Handlers) TransferOwnership(c *web.Ctx) error {
	if err := h.can(c, workspaces.AbilityTransfer); err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.TransferSchema)
	if err != nil {
		return err
	}

	if err := h.WS.TransferOwnership(c.Context(), u, id(in.UserID)); err != nil {
		return back(c, "/workspace", err)
	}

	return done(c, "/workspace", "Ownership transferred: you are now an admin.")
}

// Leave takes the signed-in user out of the workspace; the next request
// picks another workspace of theirs.
func (h *Handlers) Leave(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if err := h.WS.Leave(c.Context(), u); err != nil {
		return back(c, "/workspace", err)
	}

	return done(c, "/dashboard", "You left the workspace.")
}

// ShowInvite asks the signed-in user to confirm the invitation.
func (*Handlers) ShowInvite(c *web.Ctx) error {
	return pages.InviteAccept.Render(c, pages.InviteAcceptProps{Token: c.Param("token")})
}

// AcceptInvite joins the workspace of the invitation and switches to it.
func (h *Handlers) AcceptInvite(c *web.Ctx) error {
	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	ws, err := h.WS.Accept(c.Context(), u, c.Param("token"))
	if err != nil {
		return err
	}

	if err := workspaces.Switch(c.Request(), ws.ID); err != nil {
		return err
	}

	return done(c, "/dashboard", "Welcome to "+ws.Name+".")
}
