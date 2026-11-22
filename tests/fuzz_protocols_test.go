package tests

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/telegram"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// Fuzz targets of the protocols that carry untrusted structured input: the
// Telegram entry and update, and the sync merge (epic 24 task 8). The HTTP
// protocols (sync push, MCP, OAuth) are in fuzz_http_test.go.

func FuzzTelegramEntry(f *testing.F) {
	for _, s := range []string{
		"12.50 lunch", "+3000 salary", "- 5 coffee yesterday", "R$ 1.234,56 mercado 12/11", "$9 taxi 2026-02-30", "0", "-", "+",
		"1e3 x", "9999999999999999999 x", "5 ontem hoje", "5 x 31/04/2026", "\x00 5", strings.Repeat("9", 400), "5 " + strings.Repeat("é", 600),
	} {
		f.Add(s)
	}

	today := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)

	f.Fuzz(func(t *testing.T, text string) {
		e, err := telegram.ParseEntry(text, today)
		if err != nil {
			return
		}

		if utf8.RuneCountInString(text) > config.TelegramInputMax {
			t.Fatalf("an entry of %d characters was accepted, the limit is %d", utf8.RuneCountInString(text), config.TelegramInputMax)
		}

		if e.Cents <= 0 || e.Cents > models.MaxCents {
			t.Fatalf("%q: %d cents", text, e.Cents)
		}

		if e.Kind != models.KindExpense && e.Kind != models.KindIncome {
			t.Fatalf("%q: kind %q", text, e.Kind)
		}

		if _, err := time.Parse("2006-01-02", e.Date); err != nil {
			t.Fatalf("%q: date %q", text, e.Date)
		}

		for _, r := range e.Description {
			if unicode.IsControl(r) {
				t.Fatalf("%q: the description holds a control character", text)
			}
		}

		if back, err := finance.ParseAmount(finance.FormatDecimalCents(e.Cents)); err != nil || back != e.Cents {
			t.Fatalf("%d cents do not round trip", e.Cents)
		}
	})
}

func FuzzTelegramUpdate(f *testing.F) {
	f.Add([]byte(`{"update_id":1,"message":{"message_id":2,"from":{"id":3},"chat":{"id":4,"type":"private"},"text":"12.50 lunch"}}`))
	f.Add([]byte(`{"update_id":1,"callback_query":{"id":"x","from":{"id":3},"data":"a:b:c"}}`))
	f.Add([]byte(`{"update_id":-1}`))
	f.Add([]byte(`{"update_id":99999999999999999999}`))
	f.Add([]byte(`{"update_id":1} {"update_id":2}`))
	f.Add([]byte(`{"update_id":1,"message":null,"callback_query":null}`))
	f.Add([]byte(`{"update_id":1,"message":{"text":"<b>&"}}`))
	f.Add([]byte(strings.Repeat("[", 5000)))
	f.Add([]byte(``))

	today := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)

	f.Fuzz(func(t *testing.T, body []byte) {
		// The decode of the webhook: one JSON document, a positive update id.
		var u telegram.Update

		dec := json.NewDecoder(bytes.NewReader(body))
		if err := dec.Decode(&u); err != nil || dec.More() || u.UpdateID <= 0 {
			return
		}

		if u.Message != nil {
			_, _ = telegram.ParseEntry(u.Message.Text, today)
		}

		// Whatever a person typed is escaped before it goes back into a message.
		text := ""
		if u.Message != nil {
			text = u.Message.Text
		}

		if escaped := telegram.EscapeHTML(text); strings.ContainsAny(escaped, "<>") {
			t.Fatalf("EscapeHTML(%q) = %q still holds markup", text, escaped)
		}
	})
}

func FuzzChangesMerge(f *testing.F) {
	f.Add(true, false, false, false, false, int64(0), `{"name":"A","version":1}`, `{"name":"B"}`, "2026-11-22", uint8(0))
	f.Add(false, false, true, false, false, int64(0), `{}`, `{"name":"new"}`, "", uint8(0))
	f.Add(true, true, false, false, true, int64(3), `{"name":"A"}`, `{}`, "", uint8(1))
	f.Add(true, false, false, true, false, int64(1), `{"generated_through":"2026-01-01"}`, `{"generated_through":"2999-01-01"}`, "2026-11-22", uint8(2))
	f.Add(true, false, false, false, false, int64(-1), `{"amount_cents":1}`, `{"amount_cents":1.0,"x":null}`, "bad", uint8(255))
	f.Add(true, true, true, true, true, int64(1<<62), `null`, `[1]`, "", uint8(1))

	f.Fuzz(func(t *testing.T, exists, deleted, create, del, restore bool, base int64, currentJSON, fieldsJSON, today string, parent uint8) {
		var current, fields map[string]any

		if json.Unmarshal([]byte(currentJSON), &current) != nil || json.Unmarshal([]byte(fieldsJSON), &fields) != nil {
			return
		}

		row := changes.Row{Exists: exists, Deleted: deleted, Version: 1, Fields: current}
		in := changes.Mutation{
			Create: create, Delete: del, Restore: restore, BaseSeq: base, Device: "d", Fields: fields, Today: today,
			Parents: map[string]changes.ParentState{"account_id": changes.ParentState(parent % 3)},
		}
		since := []changes.Change{{Seq: 2, Op: "update", Fields: []string{"name"}, Device: "other"}}

		before, _ := json.Marshal([]any{row, in, since})

		res := changes.Merge(row, in, since)

		if after, _ := json.Marshal([]any{row, in, since}); !bytes.Equal(before, after) {
			t.Fatal("Merge changed its input")
		}

		if again := changes.Merge(row, in, since); !reflect.DeepEqual(res, again) {
			t.Fatalf("Merge is not deterministic: %+v and %+v", res, again)
		}

		switch res.Outcome {
		case changes.OutcomeApplied, changes.OutcomeNoop, changes.OutcomeDropped, changes.OutcomeRejected:
		default:
			t.Fatalf("unknown outcome %q", res.Outcome)
		}

		if res.Outcome == changes.OutcomeRejected && res.Code == "" {
			t.Fatal("a rejection with no code")
		}

		if res.Outcome != changes.OutcomeApplied && len(res.Apply) != 0 {
			t.Fatalf("a %s result writes %v", res.Outcome, res.Apply)
		}

		// Nothing is written that the mutation did not name.
		for column := range res.Apply {
			if _, named := in.Fields[column]; !named && column != "deleted_at" {
				t.Fatalf("Merge writes %q, which the mutation never named", column)
			}
		}
	})
}
