package config_test

import (
	"testing"
	"time"

	appconfig "github.com/AlexGalhardo/vukem-finances/config"
)

func TestTombstoneRetention(t *testing.T) {
	t.Parallel()

	day := 24 * time.Hour

	for _, tc := range []struct {
		value   string
		want    time.Duration
		wantErr bool
	}{
		{"", 90 * day, false},
		{" 45 ", 45 * day, false},
		{"30", 30 * day, false},
		{"29", 0, true},
		{"0", 0, true},
		{"-5", 0, true},
		{"90.5", 0, true},
		{"soon", 0, true},
	} {
		got, err := appconfig.TombstoneRetention(mapSource{"TOMBSTONE_RETENTION_DAYS": tc.value})
		if (err != nil) != tc.wantErr || got != tc.want {
			t.Errorf("TOMBSTONE_RETENTION_DAYS=%q = %v, %v; want %v, error=%v", tc.value, got, err, tc.want, tc.wantErr)
		}
	}
}
