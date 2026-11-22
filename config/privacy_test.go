package config_test

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/config"
)

func TestPaymentRecordRetentionYears(t *testing.T) {
	t.Parallel()

	for value, want := range map[string]int{"": 5, " 7 ": 7, "1": 1, "30": 30} {
		got, err := config.PaymentRecordRetentionYears(mapSource{"PAYMENT_RECORD_RETENTION_YEARS": value})
		if err != nil || got != want {
			t.Errorf("%q = %d, %v; want %d", value, got, err, want)
		}
	}

	for _, bad := range []string{"0", "31", "x", "-1", "2.5"} {
		if _, err := config.PaymentRecordRetentionYears(mapSource{"PAYMENT_RECORD_RETENTION_YEARS": bad}); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}
