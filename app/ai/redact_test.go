package ai_test

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
)

func TestRedact(t *testing.T) {
	t.Parallel()

	cases := []struct{ name, in, want string }{
		// emails
		{"email", "pay ana@example.com now", "pay [email] now"},
		{"email with plus and subdomain", "ana.silva+bank@mail.example.co.uk", "[email]"},
		{"two emails", "a@b.io, c@d.org", "[email], [email]"},
		// phones
		{"br mobile international", "call +55 11 91234-5678 today", "call [phone] today"},
		{"br with parens", "tel (11) 91234-5678", "tel [phone]"},
		{"br local", "zap 91234-5678", "zap [phone]"},
		{"us dashes", "555-123-4567", "[phone]"},
		{"us parens", "+1 (555) 123-4567", "[phone]"},
		{"uk international", "+44 20 7946 0958", "[phone]"},
		{"double zero prefix", "0044 20 7946 0958", "[phone]"},
		{"dots", "555.123.4567", "[phone]"},
		// Brazilian ids
		{"cpf", "CPF 123.456.789-09 titular", "CPF [number] titular"},
		{"cnpj", "CNPJ 12.345.678/0001-95", "CNPJ [number]"},
		{"cpf unformatted", "12345678909", "[number]"},
		{"cnpj unformatted", "12345678000195", "[number]"},
		// cards and accounts
		{"card spaces", "card 4111 1111 1111 1111 used", "card [number] used"},
		{"card dashes", "4111-1111-1111-1111", "[number]"},
		{"card plain", "4111111111111111", "[number]"},
		{"account number", "acct 000123456789", "acct [number]"},
		{"nine digits", "123456789", "[number]"},
		// IBAN
		{"iban gb spaced", "to GB82 WEST 1234 5698 7654 32 ok", "to [iban] ok"},
		{"iban de compact", "DE89370400440532013000", "[iban]"},
		{"iban br", "BR1800360305000010009795493C1", "[iban]"},
		// keys
		{"openai key", "key sk-proj-abcdefghijklmnop1234", "key [key]"},
		{"anthropic key", "sk-ant-api03-abcdefghijklmnopqrstuv", "[key]"},
		{"bearer", "Authorization: Bearer abcdefghijklmnop12345678", "Authorization: [key]"},
		// left untouched
		{"plain amount", "paid 12.50 for lunch", "paid 12.50 for lunch"},
		{"amount thousands br", "R$ 1.234,56", "R$ 1.234,56"},
		{"amount thousands us", "$1,234,567.89", "$1,234,567.89"},
		{"amount eight digits", "1234567.89", "1234567.89"},
		{"iso date", "2026-11-21", "2026-11-21"},
		{"two dates", "2026-11-21 2026-11-22", "2026-11-21 2026-11-22"},
		{"word with sk", "task-force budget", "task-force budget"},
		{"category", "Groceries / Mercado", "Groceries / Mercado"},
		{"empty", "", ""},
		{"percent", "rate 4.5% on 1000", "rate 4.5% on 1000"},
	}

	if len(cases) < 25 {
		t.Fatalf("only %d cases", len(cases))
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := ai.Redact(tc.in); got != tc.want {
				t.Errorf("Redact(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
