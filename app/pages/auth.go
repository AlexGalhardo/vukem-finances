package pages

import "github.com/AlexGalhardo/vukem/inertia"

// The typed pages of vukem install:auth.
var (
	Login          = inertia.Define[struct{}]("Auth/Login")
	Register       = inertia.Define[struct{}]("Auth/Register")
	ForgotPassword = inertia.Define[struct{}]("Auth/ForgotPassword")
	CheckEmail     = inertia.Define[struct{}]("Auth/CheckEmail")
	VerifyEmail    = inertia.Define[struct{}]("Auth/VerifyEmail")
	Dashboard      = inertia.Define[DashboardProps]("Dashboard")
	ResetPassword  = inertia.Define[ResetPasswordProps]("Auth/ResetPassword")
	Setup          = inertia.Define[SetupProps]("Setup")
)

// SetupProps are the props of Setup, the first-owner form of the personal
// edition.
type SetupProps struct {
	Currency string `json:"currency"`
	// CodeRequired is false in the desktop app, where the gate stands in for
	// the setup code.
	CodeRequired bool `json:"codeRequired"`
}

// ResetPasswordProps are the props of Auth/ResetPassword.
type ResetPasswordProps struct {
	Token string `json:"token"`
	Email string `json:"email"`
}
