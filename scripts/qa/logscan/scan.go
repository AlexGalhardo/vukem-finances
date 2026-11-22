// Package main is the log scanner of the quality campaign (epic 24, task 7).
// It reads the captured output of a run (server logs, test output, mail log,
// exports, responses) and reports any bearer token, API or OAuth token, API
// key, APP_KEY, bot token, password, session id or card-like number it finds.
// It never prints a secret: a finding shows the rule, the place and a masked
// preview.
package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Finding is one secret-shaped value in an input.
type Finding struct {
	Source  string
	Line    int
	Rule    string
	Preview string
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", f.Source, f.Line, f.Rule, f.Preview)
}

type rule struct {
	name string
	re   *regexp.Regexp
	// group is the submatch that holds the secret value (0 for the whole match).
	group int
	// valid, when set, rejects a match that only looks like a secret.
	valid func(value string) bool
}

const keyValue = `(?i)["']?\b(%s)["']?\s*[:=]\s*["']?([^\s"',;}&\]]{6,})`

var rules = []rule{
	{name: "bearer token", re: regexp.MustCompile(`(?i)\bbearer\s+([A-Za-z0-9._~+/=-]{16,})`), group: 1, valid: notRedacted},
	{name: "api token", re: regexp.MustCompile(`\bvf_[A-Za-z0-9]{4,}\.[A-Za-z0-9_-]{16,}`)},
	{name: "jwt", re: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`)},
	{name: "payment provider key", re: regexp.MustCompile(`\b(?:sk|rk)_(?:live|test)_[A-Za-z0-9]{16,}|\bwhsec_[A-Za-z0-9]{16,}`)},
	{name: "provider api key", re: regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{20,}|\babc_(?:dev|prod)_[A-Za-z0-9_]{8,}`)},
	{name: "bot token", re: regexp.MustCompile(`\b(?:bot)?\d{6,12}:[A-Za-z0-9_-]{30,}`)},
	{name: "app key", re: regexp.MustCompile(`\bbase64:[A-Za-z0-9+/]{42,}={0,2}`)},
	{
		name:  "credential field",
		re:    regexp.MustCompile(fmt.Sprintf(keyValue, `password|passwd|client_secret|api_?key|apiSecret|access_token|refresh_token|authorization_code`)),
		group: 2, valid: notRedacted,
	},
	{name: "session id", re: regexp.MustCompile(`(?i)\b(?:vukem_session|vf_desktop|session_id)=([A-Za-z0-9._~%+/-]{16,})`), group: 1, valid: notRedacted},
	{name: "card number", re: regexp.MustCompile(`(?:^|[^\w.-])((?:[3-6]\d{3}[ -]?){3}\d{1,7})(?:$|[^\w.-])`), group: 1, valid: cardLike},
}

// redactions are the placeholders the application writes in place of a secret.
var redactions = []string{"[redacted]", "{secret}", "{token}", "***", "<redacted>", "redacted", "null", "none", "true", "false"}

func notRedacted(value string) bool {
	v := strings.ToLower(strings.Trim(value, "[]{}<>*"))
	for _, r := range redactions {
		if v == strings.Trim(r, "[]{}<>*") {
			return false
		}
	}

	return true
}

// cardLike accepts 13 to 19 digits that pass the Luhn check.
func cardLike(value string) bool {
	digits := make([]int, 0, 19)

	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) < 13 || len(digits) > 19 {
		return false
	}

	sum := 0

	for i := range digits {
		d := digits[len(digits)-1-i]
		if i%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
	}

	return sum%10 == 0
}

// Scanner looks for secret-shaped values and for values it was told about.
type Scanner struct {
	known []string
}

// NewScanner returns a scanner that also reports every occurrence of one of
// the known literal secrets (shorter than 8 bytes are ignored: they would
// match by chance).
func NewScanner(known []string) *Scanner {
	s := &Scanner{}

	for _, k := range known {
		if k = strings.TrimSpace(k); len(k) >= 8 {
			s.known = append(s.known, k)
		}
	}

	return s
}

// Scan reads r line by line and returns the findings, in order.
func (s *Scanner) Scan(source string, r io.Reader) ([]Finding, error) {
	var out []Finding

	br := bufio.NewReaderSize(r, 1<<20)

	for n := 1; ; n++ {
		line, err := br.ReadString('\n')
		if line != "" {
			out = append(out, s.scanLine(source, n, line)...)
		}

		if err == io.EOF {
			return out, nil
		}

		if err != nil {
			return out, err
		}
	}
}

func (s *Scanner) scanLine(source string, n int, line string) []Finding {
	var out []Finding

	for _, k := range s.known {
		if strings.Contains(line, k) {
			out = append(out, Finding{source, n, "known secret", mask(k)})
		}
	}

	for _, rl := range rules {
		for _, m := range rl.re.FindAllStringSubmatch(line, -1) {
			value := m[rl.group]
			if rl.valid != nil && !rl.valid(value) {
				continue
			}

			out = append(out, Finding{source, n, rl.name, mask(value)})
		}
	}

	return out
}

// mask keeps two characters and the length: enough to find the line, never
// enough to use the value.
func mask(v string) string {
	if len(v) <= 4 {
		return fmt.Sprintf("%d chars", len(v))
	}

	return fmt.Sprintf("%s... (%d chars)", v[:2], len(v))
}
