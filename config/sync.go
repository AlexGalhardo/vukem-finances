package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	vconfig "github.com/AlexGalhardo/vukem/config"
)

// Tombstone retention bounds, in days. The floor is the 30-day restore window
// and the lifetime of sync_conflicts: a shorter retention would purge a row
// that can still be restored.
const (
	DefaultTombstoneRetentionDays = 90
	MinTombstoneRetentionDays     = 30
)

// TombstoneRetention reads TOMBSTONE_RETENTION_DAYS from src: how long a
// deleted row and its log entries are kept before compaction purges them
// (docs/specs/sync-data-model.md). Unset means 90 days; a value that is not a
// whole number of at least 30 is an error.
func TombstoneRetention(src vconfig.Lookuper) (time.Duration, error) {
	v, _ := src.Lookup("TOMBSTONE_RETENTION_DAYS")

	v = strings.TrimSpace(v)
	if v == "" {
		return DefaultTombstoneRetentionDays * 24 * time.Hour, nil
	}

	days, err := strconv.Atoi(v)
	if err != nil || days < MinTombstoneRetentionDays {
		return 0, fmt.Errorf("TOMBSTONE_RETENTION_DAYS=%q is invalid: use a whole number of days, %d or more",
			v, MinTombstoneRetentionDays)
	}

	return time.Duration(days) * 24 * time.Hour, nil
}
