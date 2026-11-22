package tests

import (
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// Fuzz targets of the text parsers (epic 24 task 8). Each one states an
// invariant, not just "does not panic". The seeds are in the f.Add calls and in
// testdata/fuzz/<Target>; a crash input is committed there as a regression seed.
// The list of targets is `go run ./scripts/qa/fuzz-targets`, run by
// .github/workflows/fuzz.yml.

func FuzzParseAmount(f *testing.F) {
	for _, s := range []string{
		"12.34", "12,34", "1,234.56", "1.234,56", "-$5", "(5.00)", "0", "0.00", ".5", "5.", "1e3", "1.234.56", "9007199254740991",
		"90071992547409.91", "90071992547409.92", "-0.01", "  7  ", "R$ 1.000,00", "١٢٣", "+5", "--5", "0x10", "1_000", "", "NaN", "Inf",
	} {
		f.Add(s, int64(0))
	}

	f.Add("", int64(models.MaxCents))
	f.Add("", int64(-models.MaxCents))
	f.Add("", int64(1))

	f.Fuzz(func(t *testing.T, s string, n int64) {
		cents, err := finance.ParseAmount(s)
		if err == nil {
			if cents > models.MaxCents || cents < -models.MaxCents {
				t.Fatalf("ParseAmount(%q) = %d, beyond MaxCents", s, cents)
			}

			if back, err := finance.ParseAmount(finance.FormatDecimalCents(cents)); err != nil || back != cents {
				t.Fatalf("%q -> %d -> %q -> %d, %v: money does not round trip", s, cents, finance.FormatDecimalCents(cents), back, err)
			}

			_ = finance.FormatAmount(cents, "USD")
		}

		if n >= -models.MaxCents && n <= models.MaxCents {
			text := finance.FormatDecimalCents(n)
			if back, err := finance.ParseAmount(text); err != nil || back != n {
				t.Fatalf("%d -> %q -> %d, %v: money does not round trip", n, text, back, err)
			}
		}
	})
}

// checkRows asserts what every statement parser promises about a row it
// called good.
func checkRows(t *testing.T, rows []finance.StatementRow) {
	t.Helper()

	if len(rows) > 20000 {
		t.Fatalf("%d rows: the parser cut or exceeded the bound", len(rows))
	}

	for _, r := range rows {
		if r.Problem != "" {
			continue
		}

		if _, err := time.Parse("2006-01-02", r.Date); err != nil {
			t.Fatalf("line %d: date %q is not an ISO date", r.Line, r.Date)
		}

		if r.Cents > models.MaxCents || r.Cents < -models.MaxCents {
			t.Fatalf("line %d: %d cents is beyond MaxCents", r.Line, r.Cents)
		}

		if !utf8.ValidString(r.Description) || !utf8.ValidString(r.FITID) || !utf8.ValidString(r.Category) {
			t.Fatalf("line %d: invalid UTF-8 in a text field", r.Line)
		}
	}
}

func FuzzCSVParser(f *testing.F) {
	f.Add([]byte("Date,Description,Amount\n2026-03-01,Coffee,-3.50\n"), "Date", "Description", "Amount", "", "2006-01-02", false)
	f.Add([]byte("Date;Description;Amount;Category\n01/03/2026;\"a;b\";\"1.234,56\";Food\n"), "Date", "Description", "Amount", "Category", "02/01/2006", true)
	f.Add([]byte("Date,Description,Amount\n2026-03-01,\"=HYPERLINK(\"\"x\"\")\",1e3\n"), "Date", "Description", "Amount", "", "", false)
	f.Add([]byte("\xEF\xBB\xBFDate,Description,Amount\r\n2026-13-45,x,1\r\n"), "Date", "Description", "Amount", "", "", false)
	f.Add([]byte(""), "", "", "", "", "", false)
	f.Add([]byte("a\x00b,c\n1,2\n"), "a\x00b", "c", "c", "", "", false)

	f.Fuzz(func(t *testing.T, content []byte, date, description, amount, category, dateFormat string, invert bool) {
		rows, err := finance.CSVParser{Mapping: finance.Mapping{
			Date: date, Description: description, Amount: amount, Category: category, DateFormat: dateFormat, Invert: invert,
		}}.Parse(content)
		if err != nil {
			return
		}

		checkRows(t, rows)
	})
}

func FuzzOFXParser(f *testing.F) {
	f.Add([]byte("OFXHEADER:100\nDATA:OFXSGML\n<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS><BANKTRANLIST><STMTTRN><TRNTYPE>DEBIT<DTPOSTED>20260301<TRNAMT>-3.50<FITID>1<NAME>Coffee</STMTTRN></BANKTRANLIST></STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>"))
	f.Add([]byte("<?xml version=\"1.0\"?><OFX><STMTTRN><DTPOSTED>20260301120000[-3:BRT]</DTPOSTED><TRNAMT>1.234,56</TRNAMT><FITID>a&amp;b</FITID><MEMO>&#x41;&#66;&lt;</MEMO></STMTTRN></OFX>"))
	f.Add([]byte("<!DOCTYPE x [<!ENTITY a \"aaaa\"><!ENTITY b \"&a;&a;&a;&a;\">]><OFX><STMTTRN><MEMO>&b;</MEMO></STMTTRN></OFX>"))
	f.Add([]byte("<OFX><STMTTRN><DTPOSTED>99999999<TRNAMT>1e999<NAME><![CDATA[x]]></STMTTRN></OFX>"))
	f.Add([]byte(""))

	f.Fuzz(func(t *testing.T, content []byte) {
		rows, err := finance.OFXParser{}.Parse(content)
		if err != nil {
			return
		}

		checkRows(t, rows)

		// Entities are never expanded into more than the file held: a bomb would show here.
		total := 0
		for _, r := range rows {
			total += len(r.Description) + len(r.FITID) + len(r.Category)
		}

		if total > 4*len(content)+64 {
			t.Fatalf("%d bytes of text out of %d bytes of file", total, len(content))
		}
	})
}

func FuzzNormalizeDescription(f *testing.F) {
	for _, s := range []string{
		"STARBUCKS #1234 SP 12/11", "Starbucks 99", "PIX RECEBIDO JOÃO DA SILVA 12345678901234567890", "", "   ", "ÀÉÎÕÜ çÇ ñÑ",
		"a b c d e f g h", "1234567890123456789012345", "İstanbul ǅ ß", "\x00\x01‮", strings.Repeat("ab ", 5000),
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		key := finance.NormalizeDescription(s)

		if key != finance.NormalizeDescription(s) {
			t.Fatal("the key is not deterministic")
		}

		if !utf8.ValidString(key) && utf8.ValidString(s) {
			t.Fatalf("valid UTF-8 %q gave an invalid key %q", s, key)
		}

		if key != strings.TrimSpace(key) || strings.Contains(key, "  ") {
			t.Fatalf("key %q has loose whitespace", key)
		}

		if n := len(strings.Fields(key)); n > 4 {
			t.Fatalf("key %q has %d words, the limit is 4", key, n)
		}

		for _, r := range key {
			if unicode.IsControl(r) {
				t.Fatalf("key %q holds a control character", key)
			}
		}

		if len(key) > 4*len(s)+16 {
			t.Fatalf("a key of %d bytes out of %d", len(key), len(s))
		}
	})
}

func FuzzRecurrenceOccurrences(f *testing.F) {
	f.Add("monthly", 1, "2026-01-31", "", 0, "2026-01-01", "2026-12-31", 0)
	f.Add("weekly", 2, "2026-02-28", "2026-06-01", 5, "2026-01-01", "2027-01-01", 3)
	f.Add("daily", 1, "0001-01-01", "", 0, "0001-01-01", "9999-12-31", 100)
	f.Add("yearly", 1, "2024-02-29", "", 0, "2024-01-01", "2040-01-01", 0)
	f.Add("monthly", 1000000, "2026-01-31", "", 0, "2026-01-01", "2026-12-31", 0)
	f.Add("hourly", 0, "x", "y", -1, "", "", -5)

	f.Fuzz(func(t *testing.T, freq string, interval int, start, end string, maxCount int, from, to string, limit int) {
		rule := finance.Rule{Frequency: freq, Interval: interval, StartDate: start, EndDate: end, MaxCount: maxCount}

		dates := rule.Occurrences(from, to, limit)

		if limit > 0 && len(dates) > limit {
			t.Fatalf("%d dates beyond the limit %d", len(dates), limit)
		}

		if maxCount > 0 && len(dates) > maxCount {
			t.Fatalf("%d dates beyond the count %d", len(dates), maxCount)
		}

		var prev string

		for _, d := range dates {
			if _, err := time.Parse("2006-01-02", d); err != nil {
				t.Fatalf("%q is not an ISO date", d)
			}

			if d <= prev {
				t.Fatalf("dates are not strictly ascending: %q after %q", d, prev)
			}

			if d < start || d < from || d > to || (end != "" && d > end) {
				t.Fatalf("%q is outside start %q, window %q..%q, end %q", d, start, from, to, end)
			}

			prev = d
		}

		if again := rule.Occurrences(from, to, limit); strings.Join(again, ",") != strings.Join(dates, ",") {
			t.Fatal("the engine is not deterministic")
		}
	})
}

func FuzzSplitInstallments(f *testing.F) {
	f.Add(int64(10000), 3)
	f.Add(int64(1), 1)
	f.Add(int64(models.MaxCents), 60)
	f.Add(int64(59), 60)
	f.Add(int64(0), 0)
	f.Add(int64(-5), 2)

	f.Fuzz(func(t *testing.T, total int64, parts int) {
		// The service checks 2 to 60 parts and a total of at least one cent per part before it splits.
		if parts < 2 || parts > 60 || total < int64(parts) || total > models.MaxCents {
			return
		}

		out := finance.SplitInstallments(total, parts)
		if len(out) != parts {
			t.Fatalf("%d parts, want %d", len(out), parts)
		}

		var sum int64

		for i, p := range out {
			sum += p

			if p < 1 {
				t.Fatalf("part %d is %d", i, p)
			}

			if i > 0 && p > out[i-1] {
				t.Fatalf("parts grow: %v", out)
			}
		}

		if sum != total {
			t.Fatalf("parts add up to %d, want %d: %v", sum, total, out)
		}

		if out[0]-out[parts-1] > 1 {
			t.Fatalf("parts differ by more than one cent: %v", out)
		}
	})
}
