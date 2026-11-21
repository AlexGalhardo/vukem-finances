package config

import "time"

// The public API (docs/api.md). Limits are per token, counted apart for reads
// (GET) and writes; a failed token lookup counts against the client address.
const (
	// APIReadLimit is the reads of one token per APIWindow.
	APIReadLimit = 120
	// APIWriteLimit is the writes of one token per APIWindow.
	APIWriteLimit = 60
	APIWindow     = time.Minute
	// APIFailedLookupLimit is the failed token lookups of one address per APIWindow.
	APIFailedLookupLimit = 60

	// TokenMaxLive is the live tokens one person holds in one workspace.
	TokenMaxLive = 20
	// TokenMaxDays is the longest life of a token.
	TokenMaxDays = 365
)

// TokenExpiryDays are the lives a token can be given, in days.
var TokenExpiryDays = []int{30, 90, TokenMaxDays}
