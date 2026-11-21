package config

import "time"

// Rate limits of the workspace routes. The 429 carries Retry-After. The
// Stripe and Resend webhooks are not limited by IP (their senders share
// addresses): they authenticate with a signature instead.
const (
	// ReadLimit is the reads (GET) of one user per ReadWindow.
	ReadLimit  = 120
	ReadWindow = time.Minute
	// WriteLimit is the writes (POST, PUT, PATCH, DELETE) of one user per WriteWindow.
	WriteLimit  = 60
	WriteWindow = time.Minute
	// ImportLimit is the CSV uploads and confirmations of one workspace per ImportWindow.
	ImportLimit  = 5
	ImportWindow = 10 * time.Minute
	// InviteLimit is the invitations of one workspace per InviteWindow.
	InviteLimit  = 10
	InviteWindow = time.Hour
)

// Two-factor codes: ten failed codes per account in TwoFactorAccountWindow,
// whatever the IPs they come from, end the pending login. The per-IP limit of
// the challenge route stays as the first line.
const (
	TwoFactorAccountLimit  = 10
	TwoFactorAccountWindow = 15 * time.Minute
)
