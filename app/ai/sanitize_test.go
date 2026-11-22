package ai_test

import (
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
)

func TestSanitizeRedactsSplitNumbers(t *testing.T) {
	t.Parallel()

	for name, in := range map[string]string{
		"double spaces":  "card 4111  1111  1111  1111 end",
		"nbsp":           "card 4111 1111 1111 1111 end",
		"tabs":           "card 4111\t1111\t1111\t1111 end",
		"newlines":       "card 4111\n1111\r\n1111\n1111 end",
		"control chars":  "card 4111\x001111\x001111\x001111 end",
		"zero width":     "card 4111​1111​1111​1111 end",
		"iban spaces":    "to GB82  WEST  1234  5698  7654  32 ok",
		"iban nbsp":      "to GB82 WEST 1234 5698 7654 32 ok",
		"iban tabs":      "to GB82\tWEST\t1234\t5698\t7654\t32 ok",
		"ideographic sp": "card 4111　1111　1111　1111 end",
	} {
		out := ai.Sanitize(in, 200)
		if strings.ContainsAny(out, "0123456789") && !strings.Contains(out, "[") {
			t.Errorf("%s: %q", name, out)
		}

		for _, leaked := range []string{"4111", "1111", "WEST", "5698", "7654"} {
			if strings.Contains(out, leaked) {
				t.Errorf("%s: %q leaked in %q", name, leaked, out)
			}
		}
	}
}

func TestSanitizeNormalizesAndClips(t *testing.T) {
	t.Parallel()

	if got := ai.Sanitize("  a \t b\n\nc d  ", 50); got != "a b c d" {
		t.Errorf("got %q", got)
	}

	if got := ai.Sanitize(strings.Repeat("ab ", 100), 10); len([]rune(got)) != 10 {
		t.Errorf("clipped to %d", len([]rune(got)))
	}

	if got := ai.Sanitize("price 12.50 on 2026-11-10", 50); got != "price 12.50 on 2026-11-10" {
		t.Errorf("amounts and dates must stay: %q", got)
	}
}
