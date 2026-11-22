package privacy

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/export"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Export document identity. The personal export wraps the workspace exports of
// docs/export-format.md, one per workspace the person owns.
const (
	ExportFormat  = "vukem-finances-personal-export"
	ExportVersion = 1
)

// The types of the document. Every field is named on purpose, like the
// workspace export: a column added to a table reaches no export until it is
// added here, so a hash, a secret or a token can never ride along by accident.

type exportProfile struct {
	ID               int64    `json:"id"`
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	EmailVerifiedAt  *string  `json:"email_verified_at"`
	CreatedAt        string   `json:"created_at"`
	TwoFactorEnabled bool     `json:"two_factor_enabled"`
	SignInProviders  []string `json:"sign_in_providers"`
}

type exportMembership struct {
	WorkspaceID   int64  `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
	Role          string `json:"role"`
	JoinedAt      string `json:"joined_at"`
}

type exportToken struct {
	Name        string  `json:"name"`
	WorkspaceID string  `json:"workspace_id"`
	Abilities   string  `json:"abilities"`
	CreatedAt   *string `json:"created_at"`
	LastUsedAt  *string `json:"last_used_at"`
	ExpiresAt   *string `json:"expires_at"`
	RevokedAt   *string `json:"revoked_at"`
}

type exportDevice struct {
	WorkspaceID int64   `json:"workspace_id"`
	Name        string  `json:"name"`
	Platform    string  `json:"platform"`
	CreatedAt   string  `json:"created_at"`
	LastSeenAt  *string `json:"last_seen_at"`
	RevokedAt   *string `json:"revoked_at"`
}

type exportGrant struct {
	WorkspaceID int64   `json:"workspace_id"`
	ClientName  string  `json:"client_name"`
	CreatedAt   string  `json:"created_at"`
	LastUsedAt  *string `json:"last_used_at"`
}

type exportSessions struct {
	RememberedBrowsers int            `json:"remembered_browsers"`
	APITokens          []exportToken  `json:"api_tokens"`
	SyncDevices        []exportDevice `json:"sync_devices"`
	ConnectedApps      []exportGrant  `json:"connected_apps"`
	TelegramLinked     bool           `json:"telegram_linked"`
}

func nullStr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}

	return &n.String
}

// Export streams the personal export of u to w: the profile, the memberships,
// the metadata of the credentials (never their values) and the full data of
// every workspace u owns. Password hashes, two-factor secrets and recovery
// codes, token hashes and the sealed AI keys are never read.
func (s *Service) Export(ctx context.Context, w io.Writer, u models.User) error {
	profile, err := s.exportProfile(ctx, u)
	if err != nil {
		return err
	}

	memberships, owned, err := s.exportMemberships(ctx, u.ID)
	if err != nil {
		return err
	}

	sessions, err := s.exportSessions(ctx, u.ID)
	if err != nil {
		return err
	}

	bw := bufio.NewWriter(w)
	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false)

	var werr error

	put := func(key string, v any) {
		if werr != nil {
			return
		}

		b, _ := json.Marshal(key)
		if _, werr = bw.Write(append(b, ':')); werr == nil {
			werr = enc.Encode(v)
		}

		if werr == nil {
			_, werr = bw.WriteString(",")
		}
	}

	if _, err := bw.WriteString("{"); err != nil {
		return err
	}

	put("format", ExportFormat)
	put("version", ExportVersion)
	put("exported_at", s.now().UTC().Format(time.RFC3339))
	put("profile", profile)
	put("memberships", memberships)
	put("sessions", sessions)

	if werr != nil {
		return werr
	}

	if _, err := bw.WriteString(`"workspaces":[`); err != nil {
		return err
	}

	for i, id := range owned {
		if i > 0 {
			if _, err := bw.WriteString(","); err != nil {
				return err
			}
		}

		if err := bw.Flush(); err != nil {
			return err
		}

		if err := export.WriteJSON(workspaces.WithTenant(ctx, id), w, s.Fin, s.now()); err != nil {
			return fmt.Errorf("exporting workspace: %w", err)
		}
	}

	if _, err := bw.WriteString("]}"); err != nil {
		return err
	}

	return bw.Flush()
}

func (s *Service) exportProfile(ctx context.Context, u models.User) (exportProfile, error) {
	p := exportProfile{
		ID: u.ID, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		SignInProviders: []string{},
	}

	if u.EmailVerifiedAt.Valid {
		v := u.EmailVerifiedAt.V.UTC().Format(time.RFC3339)
		p.EmailVerifiedAt = &v
	}

	ph := s.DB.Dialect().Placeholder

	var confirmed sql.NullString
	if err := s.DB.QueryRowContext(ctx, "SELECT two_factor_confirmed_at FROM users WHERE id = "+ph(1), u.ID).
		Scan(&confirmed); err != nil {
		return p, err
	}

	p.TwoFactorEnabled = confirmed.Valid

	rows, err := s.DB.QueryContext(ctx, "SELECT provider FROM social_accounts WHERE user_id = "+ph(1)+" ORDER BY provider", u.ID)
	if err != nil {
		return p, err
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		if err := rows.Scan(&provider); err != nil {
			return p, err
		}

		p.SignInProviders = append(p.SignInProviders, provider)
	}

	return p, rows.Err()
}

// exportMemberships lists the workspaces of the person and, apart, the ids of
// the ones they own: only those leave with the export, because the data of a
// shared workspace belongs to its owner, who exports it.
func (s *Service) exportMemberships(ctx context.Context, userID int64) ([]exportMembership, []int64, error) {
	ms, err := orm.QueryOf[models.Membership](s.DB).Where("user_id", userID).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return nil, nil, err
	}

	out := make([]exportMembership, 0, len(ms))

	var owned []int64

	for _, m := range ms {
		w, err := orm.QueryOf[models.Workspace](s.DB).Find(ctx, m.WorkspaceID)
		if err != nil {
			return nil, nil, err
		}

		out = append(out, exportMembership{
			WorkspaceID: w.ID, WorkspaceName: w.Name, Role: m.Role, JoinedAt: m.CreatedAt.UTC().Format(time.RFC3339),
		})

		if m.Role == models.RoleOwner {
			owned = append(owned, w.ID)
		}
	}

	return out, owned, nil
}

func (s *Service) exportSessions(ctx context.Context, userID int64) (exportSessions, error) {
	ph := s.DB.Dialect().Placeholder
	out := exportSessions{APITokens: []exportToken{}, SyncDevices: []exportDevice{}, ConnectedApps: []exportGrant{}}

	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM remember_tokens WHERE user_id = "+ph(1), userID).
		Scan(&out.RememberedBrowsers); err != nil {
		return out, err
	}

	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) > 0 FROM telegram_links WHERE user_id = "+ph(1)+
		" AND revoked_at IS NULL", userID).Scan(&out.TelegramLinked); err != nil {
		return out, err
	}

	rows, err := s.DB.QueryContext(ctx, "SELECT name, resource, abilities, created_at, last_used_at, expires_at, revoked_at "+
		"FROM access_tokens WHERE user_id = "+ph(1)+" ORDER BY created_at", strconv.FormatInt(userID, 10))
	if err != nil {
		return out, err
	}

	for rows.Next() {
		var t exportToken

		var resource, created, used, expires, revoked sql.NullString
		if err := rows.Scan(&t.Name, &resource, &t.Abilities, &created, &used, &expires, &revoked); err != nil {
			rows.Close()
			return out, err
		}

		t.WorkspaceID, t.CreatedAt, t.LastUsedAt, t.ExpiresAt, t.RevokedAt =
			resource.String, nullStr(created), nullStr(used), nullStr(expires), nullStr(revoked)
		out.APITokens = append(out.APITokens, t)
	}

	if err := rows.Close(); err != nil {
		return out, err
	}

	devices, err := s.DB.QueryContext(ctx, "SELECT workspace_id, name, platform, created_at, last_seen_at, revoked_at "+
		"FROM sync_devices WHERE user_id = "+ph(1)+" ORDER BY id", userID)
	if err != nil {
		return out, err
	}

	for devices.Next() {
		var d exportDevice

		var seen, revoked sql.NullString
		if err := devices.Scan(&d.WorkspaceID, &d.Name, &d.Platform, &d.CreatedAt, &seen, &revoked); err != nil {
			devices.Close()
			return out, err
		}

		d.LastSeenAt, d.RevokedAt = nullStr(seen), nullStr(revoked)
		out.SyncDevices = append(out.SyncDevices, d)
	}

	if err := devices.Close(); err != nil {
		return out, err
	}

	grants, err := s.DB.QueryContext(ctx, "SELECT workspace_id, client_name, created_at, last_used_at "+
		"FROM mcp_grants WHERE user_id = "+ph(1)+" ORDER BY id", userID)
	if err != nil {
		return out, err
	}

	for grants.Next() {
		var g exportGrant

		var used sql.NullString
		if err := grants.Scan(&g.WorkspaceID, &g.ClientName, &g.CreatedAt, &used); err != nil {
			grants.Close()
			return out, err
		}

		g.LastUsedAt = nullStr(used)
		out.ConnectedApps = append(out.ConnectedApps, g)
	}

	return out, grants.Close()
}
