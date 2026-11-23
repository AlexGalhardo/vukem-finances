package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

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
	// ImportLimit is the statement uploads, previews and confirmations of one
	// workspace per ImportWindow: one import takes three of them.
	ImportLimit  = 15
	ImportWindow = 10 * time.Minute
	// InviteLimit is the invitations of one workspace per InviteWindow.
	InviteLimit  = 10
	InviteWindow = time.Hour
	// ExportLimit is the downloads (CSV and JSON together) of one workspace per
	// ExportWindow: an export streams a whole table, so it is the costly read.
	ExportLimit  = 10
	ExportWindow = time.Hour
	// PixPollLimit is the status polls of the PIX dialog of one user per
	// PixPollWindow: the dialog polls every 3 seconds (20 a minute).
	PixPollLimit  = 30
	PixPollWindow = time.Minute
	// PixStartLimit is the PIX charges one workspace may create per PixStartWindow.
	PixStartLimit  = 10
	PixStartWindow = 10 * time.Minute
)

// WorkspaceReadLimit reads WORKSPACE_READ_LIMIT from src: the reads of one user
// per ReadWindow. Unset means ReadLimit; anything but a positive whole number is
// an error. The end-to-end suite raises it, because one signed-in user walks
// every page of the app far faster than a person can.
func WorkspaceReadLimit(src vconfig.Lookuper) (int, error) {
	return positiveLimit(src, "WORKSPACE_READ_LIMIT", ReadLimit)
}

// AICallLimit is the AI calls of one workspace per minute.
const AICallLimit = 20

// AICallsPerMinute reads AI_CALLS_PER_MINUTE from src. Unset means AICallLimit;
// anything but a positive whole number is an error. The end-to-end suite raises
// it, because all its AI specs share one workspace.
func AICallsPerMinute(src vconfig.Lookuper) (int, error) {
	return positiveLimit(src, "AI_CALLS_PER_MINUTE", AICallLimit)
}

func positiveLimit(src vconfig.Lookuper, key string, fallback int) (int, error) {
	v, _ := src.Lookup(key)

	v = strings.TrimSpace(v)
	if v == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%s=%q is invalid: use a positive whole number", key, v)
	}

	return n, nil
}

// Two-factor codes: ten failed codes per account in TwoFactorAccountWindow,
// whatever the IPs they come from, end the pending login. The per-IP limit of
// the challenge route stays as the first line.
const (
	TwoFactorAccountLimit  = 10
	TwoFactorAccountWindow = 15 * time.Minute
)
