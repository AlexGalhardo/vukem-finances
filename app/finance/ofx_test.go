package finance

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var _ StatementParser = OFXParser{}

func fixture(t *testing.T, name string) []byte {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "ofx", name))
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func parseOFX(t *testing.T, name string) []StatementRow {
	t.Helper()

	rows, err := OFXParser{}.Parse(fixture(t, name))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}

	return rows
}

func TestOFXReadsSGMLAndXML(t *testing.T) {
	t.Parallel()

	for name, want := range map[string][]StatementRow{
		"sgml.ofx": {
			{Line: 33, Date: "2026-03-01", Description: "Acme Payroll", Cents: 250000, FITID: "2026030101"},
			{Line: 41, Date: "2026-03-02", Description: "Rent March", Cents: -90000, FITID: "2026030201"},
			{Line: 48, Date: "2026-03-03", Description: "Corner & Market", Cents: -8240, FITID: "2026030301"},
		},
		"xml.ofx": {
			{Line: 15, Date: "2026-03-05", Description: "Refund été", Cents: 123456, FITID: "X-1"},
			{Line: 22, Date: "2026-03-06", Description: "Café <Central>", Cents: -1250, FITID: "X-2"},
		},
	} {
		got := parseOFX(t, name)
		if len(got) != len(want) {
			t.Fatalf("%s: %d rows, want %d: %+v", name, len(got), len(want), got)
		}

		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s row %d = %+v, want %+v", name, i, got[i], want[i])
			}
		}
	}
}

func TestOFXDetectsISO88591(t *testing.T) {
	t.Parallel()

	rows := parseOFX(t, "latin1.ofx")
	if len(rows) != 3 || rows[0].Description != "Salário €" || rows[2].Description != "Café São João" {
		t.Fatalf("rows = %+v, want the accents and the euro sign decoded", rows)
	}
}

func TestOFXFITIDsTravelWithTheRows(t *testing.T) {
	t.Parallel()

	a, b := parseOFX(t, "xml.ofx"), parseOFX(t, "fitid-reissued.ofx")
	if a[0].FITID == "" || a[0].FITID != b[0].FITID || a[0].Description == b[0].Description {
		t.Errorf("rows %+v and %+v: want the same FITID and different descriptions", a[0], b[0])
	}
}

func TestOFXRefusesHostileFilesQuickly(t *testing.T) {
	t.Parallel()

	var hits atomic.Int64

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	t.Cleanup(srv.Close)

	remote := []byte(`<?xml version="1.0"?><!DOCTYPE OFX SYSTEM "` + srv.URL + `/x.dtd"><OFX><STMTTRN><DTPOSTED>20260301</DTPOSTED><TRNAMT>-1</TRNAMT></STMTTRN></OFX>`)

	for name, content := range map[string][]byte{
		"external entity":    fixture(t, "hostile-external-entity.ofx"),
		"billion laughs":     fixture(t, "hostile-billion-laughs.ofx"),
		"remote DTD":         remote,
		"CDATA":              []byte(`<OFX><STMTTRN><DTPOSTED>20260301</DTPOSTED><TRNAMT>-1</TRNAMT><NAME><![CDATA[x]]></NAME></STMTTRN></OFX>`),
		"a comment":          []byte(`<OFX><!-- x --><STMTTRN><DTPOSTED>20260301</DTPOSTED><TRNAMT>-1</TRNAMT></STMTTRN></OFX>`),
		"padded declaration": []byte(strings.Repeat(" ", 1<<20) + "<!ENTITY a \"b\">"),
	} {
		start := time.Now()
		rows, err := OFXParser{}.Parse(content)

		// The race detector slows this code several times over; CI runs the
		// time limits in a step of their own, without it.
		if took := time.Since(start); !raceEnabled && took > 100*time.Millisecond {
			t.Errorf("%s: refused after %s, want under 100ms", name, took)
		}

		if !errors.Is(err, errOFXHostile) || rows != nil {
			t.Errorf("%s: rows %v, err %v; want the hostile-file refusal", name, rows, err)
		}
	}

	if n := hits.Load(); n != 0 {
		t.Errorf("the parser reached the network %d times", n)
	}
}

func TestOFXBoundsWhatItReads(t *testing.T) {
	t.Parallel()

	const head = `<OFX><STMTTRN><DTPOSTED>20260301</DTPOSTED><TRNAMT>-1.00</TRNAMT><NAME>`

	t.Run("2 MiB of one tag", func(t *testing.T) {
		big := head + strings.Repeat("A", MaxImportBytes-len(head)-30) + `</NAME></STMTTRN></OFX>`

		var (
			rows []StatementRow
			err  error
			best time.Duration
		)

		// Load on the machine only ever adds time, so the fastest of three parses is the parser's own cost.
		for i := range 3 {
			start := time.Now()
			rows, err = OFXParser{}.Parse([]byte(big))

			if took := time.Since(start); i == 0 || took < best {
				best = took
			}
		}

		if !raceEnabled && best > 100*time.Millisecond {
			t.Errorf("took %s, want under 100ms", best)
		}

		if err != nil || len(rows) != 1 || rows[0].Problem != "field too long" {
			t.Errorf("rows %+v, err %v; want one row reported as too long", rows, err)
		}
	})

	t.Run("over the size cap", func(t *testing.T) {
		_, err := OFXParser{}.Parse([]byte(head + strings.Repeat("A", MaxImportBytes)))
		if !errors.Is(err, errImportTooBig) {
			t.Errorf("err = %v, want too big", err)
		}
	})

	t.Run("deep nesting", func(t *testing.T) {
		deep := "<OFX>" + strings.Repeat("<A>", 600000)

		start := time.Now()
		_, err := OFXParser{}.Parse([]byte(deep))

		if time.Since(start) > time.Second {
			t.Errorf("took %s on 600000 nested tags", time.Since(start))
		}

		if !errors.Is(err, errImportOFX) {
			t.Errorf("err = %v, want no transactions found", err)
		}
	})

	t.Run("a tag that never ends", func(t *testing.T) {
		if _, err := (OFXParser{}).Parse([]byte("<OFX><STMTTRN>" + strings.Repeat("x", 1000))); err != nil && !errors.Is(err, errImportOFX) {
			t.Errorf("err = %v", err)
		}

		if _, err := (OFXParser{}).Parse([]byte("<OFX><" + strings.Repeat("x", 1000) + ">")); !errors.Is(err, errOFXMalformed) {
			t.Errorf("err = %v, want malformed", err)
		}
	})

	t.Run("20001 rows", func(t *testing.T) {
		row := "<STMTTRN><DTPOSTED>20260301<TRNAMT>-1.00<NAME>x</STMTTRN>\n"
		content := "<OFX>" + strings.Repeat(row, maxImportRows+1)

		if len(content) > MaxImportBytes {
			t.Fatalf("the fixture is %d bytes, over the file cap: shrink the row", len(content))
		}

		if _, err := (OFXParser{}).Parse([]byte(content)); !errors.Is(err, errImportRows) {
			t.Errorf("err = %v, want the row limit", err)
		}

		if rows, err := (OFXParser{}).Parse([]byte("<OFX>" + strings.Repeat(row, maxImportRows))); err != nil || len(rows) != maxImportRows {
			t.Errorf("20000 rows = %d, %v; want all", len(rows), err)
		}
	})
}

func TestOFXRefusesWhatIsNotOFX(t *testing.T) {
	t.Parallel()

	for name, content := range map[string]string{
		"empty":       "",
		"csv":         "Date,Description,Amount\n2026-03-01,x,1.00\n",
		"no rows":     "<OFX><BANKMSGSRSV1></BANKMSGSRSV1></OFX>",
		"binary junk": "\x00\x01\x02<\xff>",
	} {
		if _, err := (OFXParser{}).Parse([]byte(content)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestOFXAmountsAndDatesAreStrict(t *testing.T) {
	t.Parallel()

	row := func(date, amount string) StatementRow {
		rows, err := OFXParser{}.Parse([]byte(fmt.Sprintf("<OFX><STMTTRN><DTPOSTED>%s<TRNAMT>%s<NAME>x</STMTTRN>", date, amount)))
		if err != nil || len(rows) != 1 {
			t.Fatalf("%s %s: %v %v", date, amount, rows, err)
		}

		return rows[0]
	}

	for amount, want := range map[string]int64{
		"-1234.56": -123456, "1234.56": 123456, "+5": 500, ".5": 50, "0.05": 5, "12": 1200, "-12.5": -1250, "1,5": 150,
		"90071992547409.91": 9007199254740991, "1.234,56": 123456, "1,234.56": 123456, "1.234.567,89": 123456789,
	} {
		if r := row("20260301", amount); r.Problem != "" || r.Cents != want {
			t.Errorf("amount %q = %d (%q), want %d", amount, r.Cents, r.Problem, want)
		}
	}

	for _, amount := range []string{
		"1.5e3", "1e2", "12.345", "1.2345,56", "--5", "5-", "abc", "", "0", "0.00", "1 2", "$5", "(5.00)",
		"90071992547409.92", "99999999999999999999",
	} {
		if r := row("20260301", amount); r.Problem != "bad amount" {
			t.Errorf("amount %q = %+v, want it refused", amount, r)
		}
	}

	for date, want := range map[string]string{
		"20260301": "2026-03-01", "20260301120000": "2026-03-01", "20260301120000.123[-3:BRT]": "2026-03-01",
		"20260301[0:GMT]": "2026-03-01", "20240229": "2024-02-29",
	} {
		if r := row(date, "1.00"); r.Problem != "" || r.Date != want {
			t.Errorf("date %q = %q (%q), want %q", date, r.Date, r.Problem, want)
		}
	}

	for _, date := range []string{"", "2026", "20261301", "20260230", "20250229", "2026-03-01", "20260301T1200", "abcdefgh"} {
		if r := row(date, "1.00"); r.Problem != "bad date" {
			t.Errorf("date %q = %+v, want it refused", date, r)
		}
	}
}
