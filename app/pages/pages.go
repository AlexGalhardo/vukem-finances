// Package pages declares the props of every Inertia page. Controllers render
// them with pages.<Page>.Render(c, props), inertia.Config lists them in
// Pages, and vukem codegen:pages writes resources/js/page-props.ts from them,
// so the page components and the Go controllers share one declaration.
package pages

import (
	"time"

	"github.com/AlexGalhardo/vukem/inertia"

	"github.com/AlexGalhardo/vukem-finances/app/alerts"
)

// Shared is the props every page receives (inertia.Config.SharedType).
type Shared struct {
	AppName       string           `json:"appName"`
	Edition       string           `json:"edition"` // personal or saas
	Auth          *SharedAuth      `json:"auth"`
	Workspace     *SharedWorkspace `json:"workspace"` // set on the pages inside a workspace
	Locale        string           `json:"locale"`
	GoogleEnabled bool             `json:"googleEnabled"`
	// Notifications is the unread count and the latest few notifications of
	// the signed-in person, for the bell of the header.
	Notifications *SharedNotifications `json:"notifications"`
}

// SharedNotifications feeds the notification bell.
type SharedNotifications struct {
	Unread int64         `json:"unread"`
	Recent []alerts.View `json:"recent"`
}

// SharedAuth is the signed-in user, or null.
type SharedAuth struct {
	User *SharedUser `json:"user"`
}

// SharedUser is the part of the user every page receives.
type SharedUser struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
}

// Home is the landing page.
var Home = inertia.Define[struct{}]("Home")

// All lists the typed pages for inertia.Config.Pages.
func All() []inertia.PageInfo {
	return []inertia.PageInfo{
		Home.Info(),
		Transactions.Info(),
		Accounts.Info(),
		Cards.Info(),
		CardStatement.Info(),
		Categories.Info(),
		Budgets.Info(),
		Goals.Info(),
		Notifications.Info(),
		Recurrences.Info(),
		Import.Info(),
		Workspace.Info(),
		InviteAccept.Info(),
		InviteRegister.Info(),
		Billing.Info(),
		AI.Info(),
		Tokens.Info(),
		AdminRevenue.Info(),
		Login.Info(),
		Register.Info(),
		ForgotPassword.Info(),
		CheckEmail.Info(),
		VerifyEmail.Info(),
		Dashboard.Info(),
		ResetPassword.Info(),
		Setup.Info(),
		TwoFactor.Info(),
		TwoFactorChallenge.Info(),
		AdminDashboard.Info(),
		AdminResourceList.Info(),
		AdminResourceShow.Info(),
	}
}
