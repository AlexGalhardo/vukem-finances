package ledger

import (
	"errors"
	"net/http"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/pages"
	"github.com/AlexGalhardo/vukem-finances/app/requests"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// AIHandlers serve the AI settings page: the key, the switches and the usage.
// Everything that changes them is the owner's (ability ai.manage), checked on
// the server in each handler. No response, page or flash ever carries the key.
type AIHandlers struct {
	h *Handlers
	s *ai.Service
}

// NewAI returns the AI handlers.
func NewAI(h *Handlers, s *ai.Service) *AIHandlers { return &AIHandlers{h: h, s: s} }

const aiPath = "/ai"

// Show renders the settings, the month's usage and, for the owner, the saved
// key as its hint.
func (a *AIHandlers) Show(c *web.Ctx) error {
	ctx := c.Context()

	settings, err := a.s.Settings.Get(ctx)
	if err != nil {
		return err
	}

	usage, err := a.s.Client.Usage(ctx)
	if err != nil {
		return err
	}

	props := pages.AIProps{
		Settings: settings, Usage: usage, Providers: ai.Kinds(), PrivateHosts: a.s.PrivateHosts,
		CanManage: a.h.can(c, workspaces.AbilityManageAI) == nil,
	}

	if props.CanManage {
		cred, ok, err := a.s.Vault.Active(ctx)
		if err != nil {
			return err
		}

		if ok {
			props.Credential = &pages.AICredentialView{
				ID: cred.ID, Provider: cred.Provider, Label: cred.Label, KeyHint: cred.KeyHint, BaseURL: cred.BaseURL,
				Model: cred.DefaultModel, CreatedAt: cred.CreatedAt.UTC().Format("2006-01-02 15:04"),
			}
			if cred.LastUsedAt.Valid {
				props.Credential.LastUsedAt = cred.LastUsedAt.V.UTC().Format("2006-01-02 15:04")
			}
		}
	}

	return pages.AI.Render(c, props)
}

// SaveKey stores a new key (and revokes the previous one).
func (a *AIHandlers) SaveKey(c *web.Ctx) error {
	if err := a.h.can(c, workspaces.AbilityManageAI); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.AIKeySchema)
	if err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	_, err = a.s.Vault.Save(c.Context(), u.ID, ai.SaveInput{
		Provider: in.Provider, Label: in.Label, Key: ai.NewKey(in.APISecret), BaseURL: in.BaseURL, Model: in.Model,
	})
	if err != nil {
		return back(c, aiPath, err)
	}

	return done(c, aiPath, "Key saved.")
}

// DeleteKey revokes a key of this workspace (404 for any other id).
func (a *AIHandlers) DeleteKey(c *web.Ctx) error {
	if err := a.h.can(c, workspaces.AbilityManageAI); err != nil {
		return err
	}

	id, err := c.ParamInt("credential")
	if err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	if err := a.s.Vault.Revoke(c.Context(), u.ID, id); err != nil {
		return back(c, aiPath, err)
	}

	return done(c, aiPath, "Key deleted.")
}

// Update saves the switches, the features and the monthly limit.
func (a *AIHandlers) Update(c *web.Ctx) error {
	if err := a.h.can(c, workspaces.AbilityManageAI); err != nil {
		return err
	}

	in, err := web.Parse(c, requests.AISettingsSchema)
	if err != nil {
		return err
	}

	u, err := auth.User[models.User](c, "web")
	if err != nil {
		return err
	}

	_, err = a.s.Settings.Update(c.Context(), u.ID, ai.UpdateInput{
		Enabled: in.Enabled, Consent: in.Consent, MonthlyCallLimit: int(id(in.Limit)), AutoApply: in.AutoApply,
		Features: map[string]bool{ai.FeatureCategorize: in.Categorize, ai.FeatureAssistant: in.Assistant},
	})
	if err != nil {
		return back(c, aiPath, err)
	}

	return done(c, aiPath, "AI settings saved.")
}

// Check asks the provider whether the saved key works. The answer is one
// generic sentence: the provider's own text is never passed on.
func (a *AIHandlers) Check(c *web.Ctx) error {
	if err := a.h.can(c, workspaces.AbilityManageAI); err != nil {
		return err
	}

	ws, _ := workspaces.TenantID(c.Context())
	err := a.s.Client.Check(c.Context(), ws)
	message, ok := "The key works.", err == nil

	switch {
	case err == nil:
	case errors.Is(err, ai.ErrRejected):
		message = "The provider rejected the key."
	case errors.Is(err, ai.ErrNoCredential):
		message = "Save a key first."
	case errors.Is(err, ai.ErrUnreachable):
		message = "No connection to the AI provider: this feature needs the internet."
	case errors.Is(err, ai.ErrTooManyCalls), errors.Is(err, ai.ErrRateLimited):
		message = "Too many requests: try again in a minute."
	default:
		var he *errs.HTTPError
		if errors.As(err, &he) {
			message = he.Message
		} else {
			message = "The provider is unavailable."
		}
	}

	if c.WantsJSON() {
		return c.JSON(http.StatusOK, map[string]any{"ok": ok, "message": message})
	}

	kind := "success"
	if !ok {
		kind = "error"
	}

	inertia.Flash(c, kind, message)

	return c.Redirect(aiPath)
}
