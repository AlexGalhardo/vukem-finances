package finance

import (
	"strings"
	"testing"
)

func TestSearchEscapesWildcardsHelper(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{
		"50%":     `50\%`,
		"a_b":     `a\_b`,
		`c:\dir`:  `c:\\dir`,
		`%_\`:     `\%\_\\`,
		"plain":   "plain",
		"":        "",
		"100% _x": `100\% \_x`,
	} {
		if got := likeEscape(in); got != want {
			t.Errorf("likeEscape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSearchClipsLongText(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("é", 300)
	if got := clip(long, maxSearchLength); len([]rune(got)) != maxSearchLength {
		t.Errorf("clip kept %d characters, want %d", len([]rune(got)), maxSearchLength)
	}

	if got := clip("short", maxSearchLength); got != "short" {
		t.Errorf("clip changed a short text: %q", got)
	}
}
