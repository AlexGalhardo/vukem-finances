package config

import (
	"fmt"
	"strconv"
	"strings"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

// DefaultPaymentRecordRetentionYears is how long the minimal payment record of a
// deleted workspace is kept (decision D-022: 5 years, no legal review).
const DefaultPaymentRecordRetentionYears = 5

// PaymentRecordRetentionYears reads PAYMENT_RECORD_RETENTION_YEARS from src.
// Unset means 5; a value that is not a whole number between 1 and 30 is an error.
func PaymentRecordRetentionYears(src vconfig.Lookuper) (int, error) {
	v, _ := src.Lookup("PAYMENT_RECORD_RETENTION_YEARS")

	v = strings.TrimSpace(v)
	if v == "" {
		return DefaultPaymentRecordRetentionYears, nil
	}

	years, err := strconv.Atoi(v)
	if err != nil || years < 1 || years > 30 {
		return 0, fmt.Errorf("PAYMENT_RECORD_RETENTION_YEARS=%q is invalid: use a whole number of years from 1 to 30", v)
	}

	return years, nil
}
