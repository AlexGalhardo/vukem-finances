package oauth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/oauth"
)

// ErrNoGrant is returned for a grant that does not exist for this person in this
// workspace, whoever it belongs to: the caller answers 404.
var ErrNoGrant = errors.New("oauth: no such grant")

// Connected is a grant of the connected-apps page.
type Connected struct {
	ID         string
	Client     string
	Scopes     []string
	CreatedAt  time.Time
	LastUsedAt sql.NullTime
	Revoked    bool
}

// Connected lists the grants the person made in the workspace, newest first.
func (s *GrantStore) Connected(ctx context.Context, userID, workspaceID int64) ([]Connected, error) {
	rows, err := s.ex.QueryContext(ctx,
		"SELECT m.grant_id, m.client_name, g.scopes, m.created_at, m.last_used_at, g.revoked_at "+
			"FROM mcp_grants m JOIN oauth_grants g ON g.id = m.grant_id "+
			"WHERE m.workspace_id = "+s.p(1)+" AND m.user_id = "+s.p(2)+" ORDER BY m.id DESC",
		workspaceID, userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var out []Connected

	for rows.Next() {
		var (
			c       Connected
			scopes  string
			revoked sql.NullTime
		)

		if err := rows.Scan(&c.ID, &c.Client, &scopes, &c.CreatedAt, &c.LastUsedAt, &revoked); err != nil {
			return nil, err
		}

		c.Scopes, c.Revoked = strings.Fields(scopes), revoked.Valid
		out = append(out, c)
	}

	return out, rows.Err()
}

// Revoke ends a grant of the person in the workspace: its refresh tokens stop
// and its access tokens fail on the next call. Another person's grant, or one of
// another workspace, is no grant at all.
func (s *GrantStore) Revoke(ctx context.Context, userID, workspaceID int64, grantID string) error {
	var found string

	err := s.ex.QueryRowContext(ctx,
		"SELECT grant_id FROM mcp_grants WHERE grant_id = "+s.p(1)+" AND workspace_id = "+s.p(2)+" AND user_id = "+s.p(3),
		grantID, workspaceID, userID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNoGrant
	}

	if err != nil {
		return err
	}

	return s.SQLStore.RevokeGrant(ctx, found)
}

// RevokeMember ends every grant a person holds in a workspace, for when they
// leave or are removed: they must not work again if the person is invited back.
func (s *GrantStore) RevokeMember(ctx context.Context, userID, workspaceID int64) error {
	_, err := s.ex.ExecContext(ctx,
		"UPDATE oauth_grants SET revoked_at = "+s.p(1)+" WHERE revoked_at IS NULL AND id IN "+
			"(SELECT grant_id FROM mcp_grants WHERE workspace_id = "+s.p(2)+" AND user_id = "+s.p(3)+")",
		s.now().UTC(), workspaceID, userID)

	return err
}

var _ oauth.Store = (*GrantStore)(nil)
