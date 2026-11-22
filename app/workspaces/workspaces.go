// Package workspaces is the multi-tenant core of the finance app: workspaces,
// memberships, invites, plan limits and the role policies. Every workspace
// owns its accounts, transactions and budgets (the workspace_id column),
// and every query of them goes through Scoped (see tenant.go).
package workspaces

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"weak"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/authz"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/mail"
	"github.com/AlexGalhardo/vukem/orm"

	appmail "github.com/AlexGalhardo/vukem-finances/app/mail"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// DefaultCurrency is the currency of a new workspace (ISO 4217).
const DefaultCurrency = "USD"

var (
	errBadRole     = errs.New(http.StatusUnprocessableEntity, "The role must be admin, member or viewer.")
	errBadCurrency = errs.New(http.StatusUnprocessableEntity, "The currency must be an ISO 4217 code, such as USD.")
	errBadTimezone = errs.New(http.StatusUnprocessableEntity, "The timezone must be an IANA name, such as America/Sao_Paulo.")
	errOwner       = errs.New(http.StatusConflict, "The owner cannot be removed.")
	errNoWorkspace = errs.New(http.StatusForbidden, "This account belongs to no workspace: ask its owner for an invitation.")
)

// Service holds the workspace rules.
type Service struct {
	DB     *database.DB
	Mailer *mail.Mailer
	Log    *slog.Logger
	Gate   *authz.Gate[models.User]
	// Edition is the shape the app runs in (APP_EDITION).
	Edition config.Edition
	// URL is the public base URL used in emails (APP_URL).
	URL string
	// InviteTTL is how long an invite link works.
	InviteTTL time.Duration

	now func() time.Time
}

// New builds the service from the app and makes the app's mailer skip the
// addresses in email_suppressions (bounces and complaints reported by the
// Resend webhook). Routes and jobs both call it, so a queue worker
// honours the list too.
func New(app *vukem.App) (*Service, error) {
	db, err := app.DB()
	if err != nil {
		return nil, err
	}

	mailer, err := app.Mailer()
	if err != nil {
		return nil, err
	}

	honourSuppressions(mailer, db)

	edition, err := config.LoadEdition(app.Source)
	if err != nil {
		return nil, err
	}

	return &Service{
		DB: db, Mailer: mailer, Log: app.Log, Gate: newGate(), Edition: edition,
		URL: strings.TrimSuffix(app.Config.URL, "/"), InviteTTL: 7 * 24 * time.Hour, now: app.Now,
	}, nil
}

var suppressionInstalled sync.Map // weak.Pointer[mail.Mailer] -> *sync.Once

// honourSuppressions installs the check once per mailer. Jobs build a Service
// on every run, concurrently, and Mailer.SetSuppression is not safe to call
// while the mailer is in use. The key is weak so a finished app can be collected.
func honourSuppressions(mailer *mail.Mailer, db *database.DB) {
	once, _ := suppressionInstalled.LoadOrStore(weak.Make(mailer), &sync.Once{})

	once.(*sync.Once).Do(func() {
		mailer.SetSuppression(func(ctx context.Context, addr string) (bool, error) {
			return orm.QueryOf[models.EmailSuppression](db).Where("address", addr).Exists(ctx)
		})
	})
}

// Notify queues mailable. An address on the suppression list (it bounced or
// complained) is skipped quietly: the action that caused the mail still succeeds.
func (s *Service) Notify(ctx context.Context, m mail.Mailable) error {
	if err := s.Mailer.SendLater(ctx, m); err != nil && !errors.Is(err, mail.ErrAllSuppressed) {
		return err
	}

	return nil
}

type roleKey struct{}

// Role is the role of the signed-in user in the workspace of ctx ("" when
// unknown): the tenant middleware puts it there.
func Role(ctx context.Context) string {
	role, _ := ctx.Value(roleKey{}).(string)
	return role
}

// roleOf reports the role of the user in the workspace; both are ids.
func (s *Service) roleOf(ctx context.Context, userID, workspaceID string) (string, bool, error) {
	u, err1 := strconv.ParseInt(userID, 10, 64)

	w, err2 := strconv.ParseInt(workspaceID, 10, 64)
	if err1 != nil || err2 != nil {
		return "", false, nil //nolint:nilerr // an id that is not a number names no workspace
	}

	m, err := orm.QueryOf[models.Membership](s.DB).Where("workspace_id", w).Where("user_id", u).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return "", false, nil
	}

	return m.Role, err == nil, err
}

// Ensure returns the first workspace of u. In saas it creates a personal one
// (and sends the welcome email) when u has none. The personal edition has
// one implicit workspace, made by the setup: a user without a membership has
// no business there and gets a 403, never a second workspace.
func (s *Service) Ensure(ctx context.Context, u models.User) (models.Workspace, error) {
	m, err := orm.QueryOf[models.Membership](s.DB).Where("user_id", u.ID).OrderBy("id", orm.Asc).First(ctx)
	if err == nil {
		return orm.QueryOf[models.Workspace](s.DB).Find(ctx, m.WorkspaceID)
	}

	if !errors.Is(err, orm.ErrNotFound) {
		return models.Workspace{}, err
	}

	if s.Edition.Personal() {
		return models.Workspace{}, errNoWorkspace
	}

	ws, err := s.Create(ctx, u, u.Name+"'s workspace", DefaultCurrency)
	if err != nil {
		return ws, err
	}

	return ws, s.Notify(ctx, appmail.Welcome{
		To: u.Email, Name: u.Name, Workspace: ws.Name, URL: s.URL + "/dashboard",
	})
}

// Create makes a workspace owned by u, with a first account and the default
// categories so the first transaction takes seconds.
func (s *Service) Create(ctx context.Context, u models.User, name, currency string) (models.Workspace, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !validCurrency(currency) {
		return models.Workspace{}, errBadCurrency
	}

	ws := models.Workspace{Name: strings.TrimSpace(name), Currency: currency, Timezone: DefaultTimezone}

	err := s.DB.Tx(ctx, func(tx *database.Tx) error { return createIn(ctx, tx, u, &ws) })

	return ws, err
}

// createIn stores ws with u as its owner and the starting data, inside tx.
func createIn(ctx context.Context, tx *database.Tx, u models.User, ws *models.Workspace) error {
	if err := orm.Create(ctx, tx, ws); err != nil {
		return err
	}

	if err := orm.Create(ctx, tx, &models.Membership{
		WorkspaceID: ws.ID, UserID: u.ID, Role: models.RoleOwner,
	}); err != nil {
		return err
	}

	return seed(WithTenant(ctx, ws.ID), tx)
}

// setupMu serializes CreateOwner: the "no user yet" check and the insert
// must not interleave between two requests.
var setupMu sync.Mutex

// CreateOwner makes the first user of a personal installation, with email
// already verified (the setup secret vouches for the person at the console),
// and the implicit workspace they own. It answers 404 once any user exists,
// so setup works exactly once even with concurrent requests.
func (s *Service) CreateOwner(
	ctx context.Context, u models.User, workspace, currency string,
) (models.User, models.Workspace, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !validCurrency(currency) {
		return u, models.Workspace{}, errBadCurrency
	}

	setupMu.Lock()
	defer setupMu.Unlock()

	ws := models.Workspace{Name: strings.TrimSpace(workspace), Currency: currency, Timezone: DefaultTimezone}

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		n, err := orm.QueryOf[models.User](tx).Count(ctx)
		if err != nil {
			return err
		}

		if n > 0 {
			return errs.ErrNotFound
		}

		u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = s.now().UTC(), true
		if err := orm.Create(ctx, tx, &u); err != nil {
			return err
		}

		return createIn(ctx, tx, u, &ws)
	})

	return u, ws, err
}

// SetupOpen reports whether the installation has no user yet, the only time
// the setup page exists.
func (s *Service) SetupOpen(ctx context.Context) (bool, error) {
	n, err := orm.QueryOf[models.User](s.DB).Count(ctx)

	return n == 0, err
}

func seed(ctx context.Context, tx *database.Tx) error {
	if err := orm.Create(ctx, tx, &models.Account{Name: "Main account", Kind: "checking"}); err != nil {
		return err
	}

	for _, c := range []models.Category{
		{Name: "Salary", Kind: models.KindIncome, Color: "green", Icon: "briefcase"},
		{Name: "Other income", Kind: models.KindIncome, Color: "teal", Icon: "coins"},
		{Name: "Housing", Kind: models.KindExpense, Color: "blue", Icon: "home"},
		{Name: "Groceries", Kind: models.KindExpense, Color: "olive", Icon: "cart"},
		{Name: "Transport", Kind: models.KindExpense, Color: "slate", Icon: "car"},
		{Name: "Leisure", Kind: models.KindExpense, Color: "violet", Icon: "film"},
		{Name: "Health", Kind: models.KindExpense, Color: "pink", Icon: "heart"},
	} {
		if err := orm.Create(ctx, tx, &c); err != nil {
			return err
		}
	}

	return nil
}

// Mine lists the workspaces u belongs to, for the switcher.
func (s *Service) Mine(ctx context.Context, u models.User) ([]models.Workspace, error) {
	ms, err := orm.QueryOf[models.Membership](s.DB).Where("user_id", u.ID).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil || len(ms) == 0 {
		return nil, err
	}

	ids := make([]any, len(ms))
	for i, m := range ms {
		ids[i] = m.WorkspaceID
	}

	return orm.QueryOf[models.Workspace](s.DB).WhereIn("id", ids...).OrderBy("id", orm.Asc).Get(ctx)
}

// Current loads the workspace of ctx.
func (s *Service) Current(ctx context.Context) (models.Workspace, error) {
	id, ok := TenantID(ctx)
	if !ok {
		return models.Workspace{}, errs.ErrNotFound
	}

	return orm.QueryOf[models.Workspace](s.DB).Find(ctx, id)
}

// Member is a user of the current workspace with their role.
type Member struct {
	UserID int64  `json:"userId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// Members lists the members of the workspace of ctx.
func (s *Service) Members(ctx context.Context) ([]Member, error) {
	ms, err := Scoped[models.Membership](ctx, s.DB).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil || len(ms) == 0 {
		return nil, err
	}

	ids := make([]any, len(ms))
	for i, m := range ms {
		ids[i] = m.UserID
	}

	users, err := orm.QueryOf[models.User](s.DB).WhereIn("id", ids...).Get(ctx)
	if err != nil {
		return nil, err
	}

	byID := make(map[int64]*models.User, len(users))
	for i := range users {
		byID[users[i].ID] = &users[i]
	}

	out := make([]Member, 0, len(ms))

	for _, m := range ms {
		if u := byID[m.UserID]; u != nil {
			out = append(out, Member{UserID: m.UserID, Name: u.Name, Email: u.Email, Role: m.Role})
		}
	}

	return out, nil
}

// Invites lists the pending invites of the workspace of ctx.
func (s *Service) Invites(ctx context.Context) ([]models.Invite, error) {
	return Scoped[models.Invite](ctx, s.DB).WhereNull("accepted_at").OrderBy("id", orm.Asc).Get(ctx)
}

// InviteLink is the secret link of a new invite and when it stops working.
// It is shown once to whoever created the invite: only a hash is stored.
type InviteLink struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Invite emails a link that joins the workspace of ctx with role, and
// returns the link. Members and pending invites count against the plan's
// member limit.
func (s *Service) Invite(ctx context.Context, inviter models.User, email, role string) (InviteLink, error) {
	if !slices.Contains([]string{models.RoleAdmin, models.RoleMember, models.RoleViewer}, role) {
		return InviteLink{}, errBadRole
	}

	email = strings.ToLower(strings.TrimSpace(email))

	lim, err := s.Limits(ctx)
	if err != nil {
		return InviteLink{}, err
	}

	// A new invite to the same address replaces the old one.
	pendingFor := Scoped[models.Invite](ctx, s.DB).Where("email", email).WhereNull("accepted_at")
	if _, err := pendingFor.Delete(ctx); err != nil {
		return InviteLink{}, err
	}

	members, err := Scoped[models.Membership](ctx, s.DB).Count(ctx)
	if err != nil {
		return InviteLink{}, err
	}

	pending, err := Scoped[models.Invite](ctx, s.DB).WhereNull("accepted_at").Count(ctx)
	if err != nil {
		return InviteLink{}, err
	}

	if err := lim.CheckMembers(int(members + pending)); err != nil {
		return InviteLink{}, err
	}

	token := rand.Text() + rand.Text()
	expires := orm.NullTime{}
	expires.V, expires.Valid = s.now().UTC().Add(s.InviteTTL), true

	inv := models.Invite{Email: email, Role: role, TokenHash: hashToken(token), ExpiresAt: expires}
	if err := orm.Create(ctx, s.DB, &inv); err != nil {
		return InviteLink{}, err
	}

	if err := s.record(ctx, s.DB, inviter.ID, AuditInviteCreated, email+" as "+role); err != nil {
		return InviteLink{}, err
	}

	ws, err := s.Current(ctx)
	if err != nil {
		return InviteLink{}, err
	}

	link := InviteLink{URL: s.URL + "/invites/" + token, ExpiresAt: expires.V}

	return link, s.Notify(ctx, appmail.WorkspaceInvite{
		To: email, Inviter: inviter.Name, Workspace: ws.Name, Role: role, URL: link.URL,
	})
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Accept adds u to the workspace of the invite behind token. The invite is
// for one address: anyone else (and any unknown, used or expired token)
// gets a 404, so tokens cannot be probed.
func (s *Service) Accept(ctx context.Context, u models.User, token string) (models.Workspace, error) {
	// No tenant yet: the invite is how the user finds the workspace.
	inv, err := orm.QueryOf[models.Invite](
		s.DB,
	).Where("token_hash", hashToken(token)).
		WhereNull("accepted_at").
		First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return models.Workspace{}, errs.ErrNotFound
	}

	if err != nil {
		return models.Workspace{}, err
	}

	expired := !inv.ExpiresAt.Valid || s.now().After(inv.ExpiresAt.V)
	if !strings.EqualFold(inv.Email, u.Email) || !u.EmailVerified() || expired {
		return models.Workspace{}, errs.ErrNotFound
	}

	err = s.DB.Tx(ctx, func(tx *database.Tx) error {
		taken, err := orm.QueryOf[models.Membership](
			tx,
		).Where("workspace_id", inv.WorkspaceID).
			Where("user_id", u.ID).
			Exists(ctx)
		if err != nil {
			return err
		}

		if !taken {
			m := models.Membership{WorkspaceID: inv.WorkspaceID, UserID: u.ID, Role: inv.Role}
			if err := orm.Create(ctx, tx, &m); err != nil {
				return err
			}
		}

		_, err = orm.QueryOf[models.Invite](tx).Where("id", inv.ID).Update(ctx, orm.Set{"accepted_at": s.now().UTC()})

		return err
	})
	if err != nil {
		return models.Workspace{}, err
	}

	return orm.QueryOf[models.Workspace](s.DB).Find(ctx, inv.WorkspaceID)
}

// RemoveMember takes userID out of the workspace of ctx; the owner stays.
func (s *Service) RemoveMember(ctx context.Context, actor models.User, userID int64) error {
	return s.DB.Tx(ctx, func(tx *database.Tx) error {
		if err := lockWorkspace(ctx, tx); err != nil {
			return err
		}

		m, err := Scoped[models.Membership](ctx, tx).Where("user_id", userID).First(ctx)
		if err != nil {
			return err
		}

		if m.Role == models.RoleOwner {
			return errOwner
		}

		if _, err = Scoped[models.Membership](ctx, tx).Where("id", m.ID).Delete(ctx); err != nil {
			return err
		}

		if err := s.revokeTokens(ctx, tx, userID); err != nil {
			return err
		}

		return s.record(ctx, tx, actor.ID, AuditMemberRemoved, fmt.Sprintf("user:%d", userID))
	})
}

// Rename changes the name, currency and timezone of the workspace of ctx; an
// empty timezone leaves it as it is.
func (s *Service) Rename(ctx context.Context, actor models.User, name, currency, timezone string) error {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !validCurrency(currency) {
		return errBadCurrency
	}

	timezone = strings.TrimSpace(timezone)
	if _, err := LoadZone(timezone); err != nil {
		return err
	}

	id, ok := TenantID(ctx)
	if !ok {
		return errs.ErrNotFound
	}

	set := orm.Set{"name": strings.TrimSpace(name), "currency": currency}
	if timezone != "" {
		set["timezone"] = timezone
	}

	if _, err := orm.QueryOf[models.Workspace](s.DB).Where("id", id).Update(ctx, set); err != nil {
		return err
	}

	return s.record(ctx, s.DB, actor.ID, AuditSettingsChanged, fmt.Sprintf("%s %s %s", set["name"], currency, timezone))
}

// OwnerEmails lists the addresses of the owners of a workspace: the people
// billing and summary emails go to.
func (s *Service) OwnerEmails(ctx context.Context, workspaceID int64) ([]string, error) {
	ms, err := orm.QueryOf[models.Membership](s.DB).Where("workspace_id", workspaceID).
		Where("role", models.RoleOwner).Get(ctx)
	if err != nil || len(ms) == 0 {
		return nil, err
	}

	ids := make([]any, len(ms))
	for i, m := range ms {
		ids[i] = m.UserID
	}

	users, err := orm.QueryOf[models.User](s.DB).WhereIn("id", ids...).Get(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]string, len(users))
	for i := range users {
		out[i] = users[i].Email
	}

	return out, nil
}
