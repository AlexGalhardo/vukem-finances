package lang

import (
	"regexp"
	"slices"
	"testing"
)

var verb = regexp.MustCompile(`%[sd]`)

func TestEveryLanguageHasEveryKey(t *testing.T) {
	t.Parallel()

	want := Keys(EN)

	for _, code := range Codes() {
		if got := Keys(code); !slices.Equal(got, want) {
			t.Errorf("%s keys differ from en: %v vs %v", code, got, want)
		}

		for _, k := range want {
			a, b := verb.FindAllString(texts[EN][k], -1), verb.FindAllString(texts[code][k], -1)
			if !slices.Equal(a, b) {
				t.Errorf("%s/%s: format verbs %v differ from en %v", code, k, b, a)
			}
		}
	}
}

func TestUnknownFallsBackToEnglish(t *testing.T) {
	t.Parallel()

	if got := T("fr", Cancelled); got != "Cancelled." {
		t.Errorf("T(fr) = %q", got)
	}

	if got := T(EN, "no_such_key"); got != "no_such_key" {
		t.Errorf("unknown key = %q", got)
	}
}
