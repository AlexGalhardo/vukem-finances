package ai

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Sanitize makes free text fit to leave the server, in this order: normalize
// (control and Unicode space characters become one space, invisible format
// characters such as zero-width spaces are dropped, runs of spaces collapse),
// redact, then clip to limit characters. Redacting before normalizing would let
// a card number or IBAN split by double spaces, NBSP, tabs or zero-width
// characters slip past the digit-group rules. Every feature that sends or keeps
// text from a transaction, a category name or a question goes through it.
func Sanitize(s string, limit int) string {
	s = strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		switch {
		case unicode.Is(unicode.Cf, r):
			return -1
		case unicode.IsControl(r), unicode.IsSpace(r):
			return ' '
		}

		return r
	}, s)), " ")

	s = Redact(s)
	if utf8.RuneCountInString(s) > limit {
		s = string([]rune(s)[:limit])
	}

	return s
}
