package changes_test

import (
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/ulid"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
)

func TestValidUID(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	gen := ulid.NewGenerator()
	good := gen.At(now)

	cases := []struct {
		name string
		uid  string
		ok   bool
	}{
		{"now", good, true},
		{"old", gen.At(now.AddDate(-3, 0, 0)), true},
		{"just inside 24 hours", gen.At(now.Add(23 * time.Hour)), true},
		{"future", gen.At(now.Add(25 * time.Hour)), false},
		{"lower case", strings.ToLower(good), false},
		{"25 characters", good[:25], false},
		{"27 characters", good + "0", false},
		{"bad alphabet", "U" + good[1:], false},
		{"empty", "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			err := changes.ValidUID(c.uid, now)
			if (err == nil) != c.ok {
				t.Fatalf("ValidUID(%q) = %v, want ok=%v", c.uid, err, c.ok)
			}
		})
	}
}
