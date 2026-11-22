// Package oauth wires the Vukem OAuth 2.1 authorization server to the
// application (epic 33): the store with its workspace binding, the consent
// decision and the connected-apps list. It holds no protocol: the code, the
// tokens, PKCE, redirect matching and reuse detection are the framework's.
package oauth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/oauth"
)

// Subject is who a grant acts as: the person and the workspace they chose on
// the consent screen. It is the `sub` of the access token, so the workspace
// rides in a claim the server signed; the grant row binds it again.
type Subject struct {
	UserID      int64
	WorkspaceID int64
}

// String is the form the token carries: "<user>:<workspace>".
func (s Subject) String() string {
	return strconv.FormatInt(s.UserID, 10) + ":" + strconv.FormatInt(s.WorkspaceID, 10)
}

// ParseSubject reads what String wrote. Anything else is refused.
func ParseSubject(v string) (Subject, bool) {
	u, w, ok := strings.Cut(v, ":")
	if !ok {
		return Subject{}, false
	}

	user, err1 := strconv.ParseInt(u, 10, 64)
	ws, err2 := strconv.ParseInt(w, 10, 64)

	if err1 != nil || err2 != nil || user <= 0 || ws <= 0 {
		return Subject{}, false
	}

	return Subject{UserID: user, WorkspaceID: ws}, true
}

// touchEvery is how often the last use of a grant is written.
const touchEvery = time.Minute

// GrantStore is the framework store plus the workspace binding. A grant counts
// only while its mcp_grants row exists and agrees with the subject of the
// grant, so a grant the binding was never written for, or one whose subject
// was altered, is not found.
type GrantStore struct {
	*oauth.SQLStore

	ex  database.Executor
	now func() time.Time
}

// NewGrantStore returns the store over ex.
func NewGrantStore(ex database.Executor, now func() time.Time) *GrantStore {
	if now == nil {
		now = time.Now
	}

	return &GrantStore{SQLStore: oauth.NewSQLStore(ex), ex: ex, now: now}
}

func (s *GrantStore) p(n int) string { return s.ex.Dialect().Placeholder(n) }

// ClientName is what the connected-apps page calls a client: the host of its
// metadata document URL, or the id of a registered one.
func ClientName(clientID string) string {
	if u, err := url.Parse(clientID); err == nil && u.Scheme == "https" && u.Host != "" {
		return u.Host
	}

	return clientID
}

// CreateGrant stores the grant and binds it to the workspace of its subject.
func (s *GrantStore) CreateGrant(ctx context.Context, g oauth.Grant) error {
	sub, ok := ParseSubject(g.Subject)
	if !ok {
		return fmt.Errorf("oauth: grant subject %q is not a person and a workspace", g.Subject)
	}

	if err := s.SQLStore.CreateGrant(ctx, g); err != nil {
		return err
	}

	now := s.now().UTC()

	_, err := s.ex.ExecContext(ctx,
		"INSERT INTO mcp_grants (workspace_id, user_id, grant_id, client_id, client_name, created_at, updated_at) VALUES ("+
			s.p(1)+", "+s.p(2)+", "+s.p(3)+", "+s.p(4)+", "+s.p(5)+", "+s.p(6)+", "+s.p(7)+")",
		sub.WorkspaceID, sub.UserID, g.ID, g.ClientID, ClientName(g.ClientID), now, now)
	if err != nil {
		// Without its binding the grant is unusable: revoke it rather than leave it.
		_ = s.SQLStore.RevokeGrant(ctx, g.ID)

		return fmt.Errorf("oauth: bind grant to its workspace: %w", err)
	}

	return nil
}

// Grant returns the grant when its binding agrees with its subject. The last
// use is written at most once a minute.
func (s *GrantStore) Grant(ctx context.Context, id string) (oauth.Grant, error) {
	g, err := s.SQLStore.Grant(ctx, id)
	if err != nil {
		return oauth.Grant{}, err
	}

	sub, ok := ParseSubject(g.Subject)
	if !ok {
		return oauth.Grant{}, oauth.ErrNotFound
	}

	var (
		workspace, user int64
		last            sql.NullTime
	)

	err = s.ex.QueryRowContext(ctx,
		"SELECT workspace_id, user_id, last_used_at FROM mcp_grants WHERE grant_id = "+s.p(1), id).
		Scan(&workspace, &user, &last)
	if errors.Is(err, sql.ErrNoRows) {
		return oauth.Grant{}, oauth.ErrNotFound
	}

	if err != nil {
		return oauth.Grant{}, fmt.Errorf("oauth: load grant binding: %w", err)
	}

	if workspace != sub.WorkspaceID || user != sub.UserID {
		return oauth.Grant{}, oauth.ErrNotFound
	}

	if now := s.now().UTC(); !g.Revoked && (!last.Valid || now.Sub(last.Time) >= touchEvery) {
		_, _ = s.ex.ExecContext(ctx, "UPDATE mcp_grants SET last_used_at = "+s.p(1)+" WHERE grant_id = "+s.p(2), now, id)
	}

	return g, nil
}
