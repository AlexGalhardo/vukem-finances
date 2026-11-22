package finance

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNormalizeDescriptionCutsAtACleanBoundary(t *testing.T) {
	t.Parallel()

	for name, in := range map[string]string{
		"the limit falls on a space":  strings.Repeat("a", ruleKeyMax-1) + " " + strings.Repeat("b", 20),
		"the limit falls in a letter": strings.Repeat("a", ruleKeyMax-1) + "ж" + strings.Repeat("b", 20),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			key := NormalizeDescription(in)

			if len(key) > ruleKeyMax {
				t.Fatalf("key has %d bytes, the limit is %d", len(key), ruleKeyMax)
			}

			if !utf8.ValidString(key) {
				t.Fatalf("key %q is not valid UTF-8", key)
			}

			if key != strings.TrimSpace(key) {
				t.Fatalf("key %q has loose whitespace", key)
			}
		})
	}
}
