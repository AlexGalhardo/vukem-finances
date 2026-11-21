package account

import (
	"net/http"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

const recoveryCodesFlash = "two_factor.recovery_codes"

// TwoFactorHandlers serves the two-factor settings page and the login
// challenge (vukem install:auth --2fa).
type TwoFactorHandlers struct {
	a      *accounts.Accounts
	issuer string
	// failed counts the wrong codes of an account, from every IP.
	failed ratelimit.Peeker
}

// NewTwoFactor returns the handlers; issuer names the app in
// authenticator apps. failed is the per-account budget of wrong codes at the
// login challenge (config.TwoFactorAccountLimit per window).
func NewTwoFactor(a *accounts.Accounts, issuer string, failed ratelimit.Peeker) *TwoFactorHandlers {
	return &TwoFactorHandlers{a: a, issuer: issuer, failed: failed}
}

// TwoFactorThrottleKey limits code attempts per user and IP: the signed-in
// user on the settings page, the pending one at the login challenge.
func TwoFactorThrottleKey(r *http.Request) string {
	if r.Method != http.MethodPost {
		return ""
	}

	id := accounts.PendingTwoFactorID(r)
	if u, err := auth.UserFrom[models.User](r.Context(), "web"); err == nil {
		id = u.AuthID()
	}

	return "2fa:" + id + ":" + ratelimit.ByIP(r)
}

// startTwoFactor sends users with two-factor authentication to the code
// step instead of logging them in (called by Login).
func (h *Handlers) startTwoFactor(c *web.Ctx, u models.User, remember bool) (bool, error) {
	pending, err := h.a.StartTwoFactorChallenge(c, u, remember)
	if err != nil || !pending {
		return false, err
	}

	return true, c.Redirect("/two-factor-challenge")
}

// flashedCodes returns the recovery codes flashed by Confirm or
// RegenerateRecoveryCodes (shown once).
func flashedCodes(c *web.Ctx) []string {
	var codes []string

	switch v := c.Session().Get(recoveryCodesFlash).(type) {
	case []string:
		codes = v
	case []any:
		for _, s := range v {
			if code, ok := s.(string); ok {
				codes = append(codes, code)
			}
		}
	}

	return codes
}
