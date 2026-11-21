package ledger

import (
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// TokenHandlers serves the Tokens page: the API tokens of the signed-in person
// for the current workspace. Every member may make tokens; what a token can do
// is limited by the role at each request (docs/api.md).
type TokenHandlers struct {
	h      *Handlers
	Tokens *api.TokenService
	now    func() time.Time
}

// NewTokens returns the token handlers.
func NewTokens(h *Handlers, tokens *api.TokenService, now func() time.Time) *TokenHandlers {
	return &TokenHandlers{h: h, Tokens: tokens, now: now}
}

func (t *TokenHandlers) props(c *web.Ctx, created *pages.CreatedToken) (pages.TokensProps, error) {
	ctx := c.Context()

	u, err := auth.UserFrom[models.User](ctx, "web")
	if err != nil {
		return pages.TokensProps{}, err
	}

	ws, _ := workspaces.TenantID(ctx)

	list, err := t.Tokens.List(ctx, u, ws)
	if err != nil {
		return pages.TokensProps{}, err
	}

	now := t.now().UTC()
	views := make([]pages.TokenView, 0, len(list))

	for _, tok := range list {
		v := pages.TokenView{ID: tok.ID, Name: tok.Name, Scopes: tok.Abilities, Status: "live"}
		if tok.LastUsedAt != nil {
			v.LastUsedAt = tok.LastUsedAt.UTC().Format(time.RFC3339)
		}

		if tok.ExpiresAt != nil {
			v.ExpiresAt = tok.ExpiresAt.UTC().Format(time.RFC3339)
		}

		switch {
		case tok.RevokedAt != nil:
			v.RevokedAt, v.Status = tok.RevokedAt.UTC().Format(time.RFC3339), "revoked"
		case !api.Live(tok, now):
			v.Status = "expired"
		}

		views = append(views, v)
	}

	names := make([]string, 0, len(api.Scopes()))
	for _, s := range api.Scopes() {
		names = append(names, s.Name)
	}

	return pages.TokensProps{
		Tokens: views, Scopes: names, ExpiryDays: config.TokenExpiryDays, Max: config.TokenMaxLive, Created: created,
	}, nil
}

// Show lists the tokens.
func (t *TokenHandlers) Show(c *web.Ctx) error {
	p, err := t.props(c, nil)
	if err != nil {
		return err
	}

	return pages.Tokens.Render(c, p)
}

// Store makes a token and answers with the page that shows its secret once:
// no redirect and no flash, so the secret never goes through the session.
func (t *TokenHandlers) Store(c *web.Ctx) error {
	ctx := c.Context()

	u, err := auth.UserFrom[models.User](ctx, "web")
	if err != nil {
		return err
	}

	in, err := web.Parse(c, requests.TokenSchema)
	if err != nil {
		return err
	}

	days, _ := strconv.Atoi(in.Days)
	ws, _ := workspaces.TenantID(ctx)

	plain, tok, err := t.Tokens.Create(ctx, u, ws, api.TokenInput{Name: in.Name, Scopes: in.Scopes, Days: days})
	if err != nil {
		return back(c, "/tokens", err)
	}

	p, err := t.props(c, &pages.CreatedToken{Name: tok.Name, Secret: plain})
	if err != nil {
		return err
	}

	c.Writer().Header().Set("Cache-Control", "no-store")

	return pages.Tokens.Render(c, p)
}

// Destroy revokes a token of the person in this workspace.
func (t *TokenHandlers) Destroy(c *web.Ctx) error {
	ctx := c.Context()

	u, err := auth.UserFrom[models.User](ctx, "web")
	if err != nil {
		return err
	}

	ws, _ := workspaces.TenantID(ctx)

	if err := t.Tokens.Revoke(ctx, u, ws, c.Param("token")); err != nil {
		return err
	}

	return done(c, "/tokens", "Token revoked.")
}
