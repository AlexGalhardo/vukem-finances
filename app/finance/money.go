// Package finance holds the personal finance rules: accounts, categories,
// transactions, budgets, the dashboard numbers and the CSV import. Money is
// int64 cents in the ISO 4217 currency of the workspace, never a float.
// Every query of a domain table goes through workspaces.Scoped.
package finance

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// ErrAmount is the answer to text that is not an amount.
var ErrAmount = errors.New("finance: not a valid amount")

// ParseAmount reads a decimal amount ("12.34", "12,34", "1,234.56",
// "1.234,56", "-$5", "(5.00)") into cents. A single separator is the decimal
// one; with two kinds the last one is, and the other must group thousands.
// Text that only looks like a number ("1.234.56", "1.5e3") is an error, because
// reading it would change the amount. More than two decimals is an error rather
// than a rounding. Amounts beyond +-models.MaxCents (2^53 - 1 cents, what any
// client can hold exactly) are refused.
func ParseAmount(s string) (int64, error) {
	s = strings.TrimSpace(s)
	neg := strings.HasPrefix(s, "-") || (strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")"))

	m := amountShape.FindStringSubmatch(strings.Join(strings.Fields(s), ""))
	if m == nil {
		return 0, ErrAmount
	}

	whole, frac, ok := splitAmount(m[1])
	if !ok || len(frac) > 2 || whole == "" && frac == "" {
		return 0, ErrAmount
	}

	frac += strings.Repeat("0", 2-len(frac))

	if whole == "" {
		whole = "0"
	}

	units, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || units > models.MaxCents/100 {
		return 0, ErrAmount
	}

	cents, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, ErrAmount
	}

	total := units*100 + cents
	if total > models.MaxCents {
		return 0, ErrAmount
	}

	if neg {
		total = -total
	}

	return total, nil
}

// amountShape is one run of digits and separators, with a sign or currency
// around it and no second number.
var amountShape = regexp.MustCompile(`^[^\d.,]*([\d.,]+)[^\d.,]*$`)

// splitAmount returns the digits before and after the decimal separator.
func splitAmount(num string) (whole, frac string, ok bool) {
	dec := strings.LastIndexAny(num, ".,")
	if dec < 0 {
		return num, "", true
	}

	// "1.234.567" has no decimal part: the separator repeats.
	if strings.Count(num, string(num[dec])) > 1 {
		whole, ok = ungroup(num)
		return whole, "", ok
	}

	whole, frac = num[:dec], num[dec+1:]
	if !strings.ContainsAny(whole, ".,") {
		return whole, frac, true
	}

	whole, ok = ungroup(whole)

	return whole, frac, ok
}

// ungroup removes one kind of thousands separator from groups of three digits.
func ungroup(s string) (string, bool) {
	sep := "."
	if strings.Contains(s, ",") {
		if strings.Contains(s, ".") {
			return "", false
		}

		sep = ","
	}

	parts := strings.Split(s, sep)
	if len(parts[0]) < 1 || len(parts[0]) > 3 {
		return "", false
	}

	for _, p := range parts[1:] {
		if len(p) != 3 {
			return "", false
		}
	}

	return strings.Join(parts, ""), true
}

var symbols = map[string]string{"USD": "$", "EUR": "€", "GBP": "£", "BRL": "R$", "JPY": "¥"}

// FormatAmount renders cents for emails: "$1,234.50", "-R$5.00". Currencies
// without a known symbol are prefixed with their code.
func FormatAmount(cents int64, currency string) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}

	symbol, ok := symbols[currency]
	if !ok {
		symbol = currency + " "
	}

	whole := strconv.FormatInt(cents/100, 10)

	var grouped strings.Builder

	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			grouped.WriteByte(',')
		}

		grouped.WriteRune(r)
	}

	frac := strconv.FormatInt(cents%100, 10)
	if len(frac) < 2 {
		frac = "0" + frac
	}

	return sign + symbol + grouped.String() + "." + frac
}
