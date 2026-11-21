package models

import (
	"errors"
	"math"
)

// MaxCents is the largest money amount, in cents, any synced field may hold:
// the JavaScript safe integer (2^53 - 1), so a phone keeps it exactly.
const MaxCents int64 = 1<<53 - 1

// ErrAmountRange is the answer to an amount beyond +-MaxCents.
var ErrAmountRange = errors.New("models: amount is beyond 9007199254740991 cents")

// CheckCents refuses an amount outside +-MaxCents. Every write path that
// does not go through finance.ParseAmount (imports, sync) calls it, and the
// models call it in BeforeCreate.
func CheckCents(cents int64) error {
	if cents > MaxCents || cents < -MaxCents || cents == math.MinInt64 {
		return ErrAmountRange
	}

	return nil
}
