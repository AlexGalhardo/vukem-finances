package workspaces

import (
	"context"
	"slices"
	"strconv"

	"github.com/AlexGalhardo/vukem/database"
)

// MemberRole is the role of userID in workspaceID, and whether the user is a
// member at all. The API calls it on every request, so a demotion or a removal
// counts at once.
func (s *Service) MemberRole(ctx context.Context, userID, workspaceID int64) (string, bool, error) {
	return s.roleOf(ctx, strconv.FormatInt(userID, 10), strconv.FormatInt(workspaceID, 10))
}

// WithRole returns ctx acting with role, for the API, where no session
// middleware has put it there.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey{}, role)
}

// RoleCan reports whether role holds ability: the role side of the ceiling that
// an API token's scope is intersected with.
func RoleCan(role, ability string) bool { return slices.Contains(grants[ability], role) }

// revokeTokens revokes the API tokens and OAuth grants userID made for the workspace of ctx, so
// they do not work again if the person is invited back. A token also fails the
// membership check while the person is out; this closes the later window.
func (s *Service) revokeTokens(ctx context.Context, tx *database.Tx, userID int64) error {
	ws, ok := TenantID(ctx)
	if !ok {
		return nil
	}

	p := tx.Dialect().Placeholder
	_, err := tx.ExecContext(ctx,
		"UPDATE access_tokens SET revoked_at = "+p(1)+" WHERE user_id = "+p(2)+" AND resource = "+p(3)+" AND revoked_at IS NULL",
		s.now().UTC(), strconv.FormatInt(userID, 10), strconv.FormatInt(ws, 10))
	if err != nil {
		return err
	}

	// The OAuth grants of remote MCP (epic 33) end the same way. Each call also
	// reads the membership, so they fail while the person is out; this closes the
	// later window.
	_, err = tx.ExecContext(ctx,
		"UPDATE oauth_grants SET revoked_at = "+p(1)+" WHERE revoked_at IS NULL AND id IN "+
			"(SELECT grant_id FROM mcp_grants WHERE workspace_id = "+p(2)+" AND user_id = "+p(3)+")",
		s.now().UTC(), ws, userID)

	return err
}
