package ai

import "regexp"

// redactRules run in order: the more specific shapes first, so a CNPJ is not
// first cut up by the long-digit rule. Each match becomes a short label, never
// a partial value.
var redactRules = []struct { //nolint:gochecknoglobals // read-only table
	re    *regexp.Regexp
	label string
}{
	// API keys (sk-, sk-ant-) and bearer tokens pasted into a description.
	{regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}`), "[key]"},
	{regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{16,}`), "[key]"},
	{regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}`), "[email]"},
	// IBAN: country, two check digits, then groups of up to four characters.
	{regexp.MustCompile(`\b[A-Z]{2}\d{2}(?: ?[A-Z0-9]{4}){2,7}(?: ?[A-Z0-9]{1,3})?\b`), "[iban]"},
	// CNPJ 12.345.678/0001-95 and CPF 123.456.789-09, formatted.
	{regexp.MustCompile(`\b\d{2}\.\d{3}\.\d{3}/\d{4}-\d{2}\b`), "[number]"},
	{regexp.MustCompile(`\b\d{3}\.\d{3}\.\d{3}-\d{2}\b`), "[number]"},
	// Card numbers written in groups: 4111 1111 1111 1111, 4111-1111-1111-1111.
	{regexp.MustCompile(`\b\d{4}(?: \d{4}){2} \d{1,7}\b`), "[number]"},
	{regexp.MustCompile(`\b\d{4}(?:-\d{4}){2}-\d{1,7}\b`), "[number]"},
	// Phones: international (+55 11 91234-5678, 0044 20 7946 0958), with an
	// area code in parentheses, or in the usual 3-3-4 and 4/5-4 groups.
	{regexp.MustCompile(`\+\d{1,3}[\d ().-]{6,18}\d|\b00\d{1,3} [\d ().-]{5,17}\d`), "[phone]"},
	{regexp.MustCompile(`\(\d{2,4}\)\s?\d[\d -]{5,12}\d`), "[phone]"},
	{regexp.MustCompile(`\b\d{3}[ .-]\d{3,4}[ .-]\d{4}\b`), "[phone]"},
	{regexp.MustCompile(`\b\d{4,5}-\d{4}\b`), "[phone]"},
	// Any other run of nine or more digits: cards, tax ids, account numbers.
	{regexp.MustCompile(`\d{9,}`), "[number]"},
}

// Redact removes what must not leave the server before text goes to a
// provider: email addresses, phone numbers, runs of nine or more digits (cards,
// tax ids such as CPF and CNPJ, account numbers), IBAN-like strings and key-like
// tokens. Amounts such as 12.50 or R$ 1.234,56 and ISO dates are left alone.
// It is not a promise of anonymity: features send as little as they can and
// pass every free-text field through it.
func Redact(text string) string {
	for _, r := range redactRules {
		text = r.re.ReplaceAllString(text, r.label)
	}

	return text
}
