package tests

import (
	"fmt"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/config"
)

// twoFactorApp trusts the proxy address of the test requests, so each call can
// come from another client IP through X-Forwarded-For.
func twoFactorApp(t *testing.T) *vukemtest.App {
	t.Helper()

	return authTestApp(t, vukemtest.Options{Env: map[string]string{"TRUSTED_PROXIES": "192.0.2.1"}})
}

func lockMails(app *vukemtest.App, to string) int {
	n := 0

	for _, m := range app.Mails().SentTo(to) {
		if m.Subject == "Too many wrong two-factor codes" {
			n++
		}
	}

	return n
}

func fromIP(c *vukemtest.Client, i int) *vukemtest.Client {
	return c.WithHeader("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i))
}

func TestSecurity_TwoFactorPerAccountLimit(t *testing.T) {
	t.Parallel()

	app := twoFactorApp(t)
	secret, _ := enableTwoFactor(t, app, app.Client(t).Inertia(""), "lock2fa@example.com")

	attacker := app.Client(t).Inertia("")
	passwordStep(fromIP(attacker, 1), "lock2fa@example.com", false).AssertRedirect("/two-factor-challenge")

	// Nine wrong codes, each from a new IP, keep the challenge open.
	for i := 1; i < config.TwoFactorAccountLimit; i++ {
		fromIP(attacker, i).WithHeader("Referer", "/two-factor-challenge").
			PostJSON("/two-factor-challenge", map[string]any{"code": wrongCode(t, secret)}).
			AssertRedirect("/two-factor-challenge")
	}

	if n := lockMails(app, "lock2fa@example.com"); n != 0 {
		t.Fatalf("%d lock mails before the limit", n)
	}

	// The tenth ends the challenge and tells the owner.
	fromIP(attacker, 10).PostJSON("/two-factor-challenge", map[string]any{"code": wrongCode(t, secret)}).
		AssertRedirect("/login")
	fromIP(attacker, 11).Get("/two-factor-challenge").AssertRedirect("/login")

	locked := lockMails(app, "lock2fa@example.com")
	if locked != 1 {
		t.Fatalf("lock mails = %d, want 1", locked)
	}

	// The 11th try from a new IP is refused even with the right code: nothing is pending.
	fromIP(attacker, 12).PostJSON("/two-factor-challenge", map[string]any{"code": totpCode(t, secret, 1)}).
		AssertRedirect("/login")
	fromIP(attacker, 12).Get("/dashboard").AssertRedirect("/login")

	// A new password login does not reset the budget, and the refusals send no more mail.
	again := app.Client(t).Inertia("")
	passwordStep(fromIP(again, 20), "lock2fa@example.com", false).AssertRedirect("/two-factor-challenge")
	fromIP(again, 21).PostJSON("/two-factor-challenge", map[string]any{"code": totpCode(t, secret, 1)}).
		AssertRedirect("/login")
	fromIP(again, 21).Get("/dashboard").AssertRedirect("/login")

	if n := lockMails(app, "lock2fa@example.com"); n != locked {
		t.Errorf("mails grew to %d on refused attempts", n)
	}

	// Another account is not affected.
	other, _ := enableTwoFactor(t, app, app.Client(t).Inertia(""), "free2fa@example.com")
	c := app.Client(t).Inertia("")
	passwordStep(c, "free2fa@example.com", false).AssertRedirect("/two-factor-challenge")
	c.PostJSON("/two-factor-challenge", map[string]any{"code": totpCode(t, other, 1)}).AssertRedirect("/dashboard")
}

func TestSecurity_TwoFactorLimitNeedsPassword(t *testing.T) {
	t.Parallel()

	app := twoFactorApp(t)
	secret, _ := enableTwoFactor(t, app, app.Client(t).Inertia(""), "victim2fa@example.com")

	// A stranger has no pending challenge: nothing is spent, whatever the volume and the IPs.
	stranger := app.Client(t).Inertia("")
	for i := range 3 * config.TwoFactorAccountLimit {
		fromIP(stranger, i+1).PostJSON("/two-factor-challenge", map[string]any{"code": wrongCode(t, secret)}).
			AssertRedirect("/login")
	}

	// The owner still gets in with the password and the right code.
	owner := app.Client(t).Inertia("")
	passwordStep(owner, "victim2fa@example.com", false).AssertRedirect("/two-factor-challenge")
	owner.PostJSON("/two-factor-challenge", map[string]any{"code": totpCode(t, secret, 1)}).AssertRedirect("/dashboard")

	if n := lockMails(app, "victim2fa@example.com"); n != 0 {
		t.Errorf("the owner got %d lock mails from a stranger", n)
	}
}
