package categorize_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/ai/aitest"
	"github.com/AlexGalhardo/vukem-finances/app/ai/categorize"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// say makes the fake provider answer with text.
func say(r *aitest.Rig, text string) {
	r.Provider.Reply = func(ai.Request) (ai.Response, error) {
		return ai.Response{Text: text, InputTokens: 10, OutputTokens: 5, Model: "fake-1", Provider: "fake"}, nil
	}
}

func catsOf(t *testing.T, r *aitest.Rig) (groceries, salary categorize.Category) {
	t.Helper()

	g, s := r.Category(t, "Groceries"), r.Category(t, "Salary")

	return categorize.Category{UID: g.UID, Name: g.Name, Kind: g.Kind},
		categorize.Category{UID: s.UID, Name: s.Name, Kind: s.Kind}
}

func reply(items ...string) string { return `{"items":[` + strings.Join(items, ",") + `]}` }

func item(id, uid string, conf float64) string {
	return fmt.Sprintf(`{"id":%q,"category_uid":%q,"confidence":%v}`, id, uid, conf)
}

func twoItems() []categorize.Item {
	return []categorize.Item{
		{ID: "t1", Description: "Padaria", Kind: models.KindExpense, AmountCents: 1000},
		{ID: "t2", Description: "Mercado", Kind: models.KindExpense, AmountCents: 2000},
	}
}

func TestCategorizeBatch(t *testing.T) {
	t.Parallel()

	r := aitest.New(t, nil, "ana@example.com")
	r.Enable(t, false, ai.FeatureCategorize)
	groceries, salary := catsOf(t, r)
	say(r, reply(item("t1", groceries.UID, 0.92), item("t2", groceries.UID, 0.4)))

	items := twoItems()
	items[0].Description = "Padaria ana@example.com 11999998888 cartao 4111 1111 1111 1111"

	got, err := categorize.Batch(t.Context(), r.Client, r.Ws.ID, models.KindExpense, items, []categorize.Category{groceries, salary})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0].ItemID != "t1" || got[0].Confidence != 92 || got[1].Confidence != 40 {
		t.Fatalf("proposals = %+v", got)
	}

	reqs := r.Provider.Requests()
	if len(reqs) != 1 || reqs[0].Feature != ai.FeatureCategorize || len(reqs[0].JSONSchema) == 0 {
		t.Fatalf("requests = %+v", reqs)
	}

	sent := reqs[0].Messages[0].Content
	for _, leaked := range []string{"ana@example.com", "11999998888", "4111"} {
		if strings.Contains(sent, leaked) {
			t.Errorf("%q left the server: %s", leaked, sent)
		}
	}

	// Only the categories of the kind of the batch are offered.
	if strings.Contains(sent, salary.UID) || !strings.Contains(sent, groceries.UID) {
		t.Errorf("categories offered: %s", sent)
	}

	var doc struct {
		Transactions []struct {
			ID          string `json:"id"`
			AmountCents int64  `json:"amount_cents"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal([]byte(sent), &doc); err != nil || len(doc.Transactions) != 2 || doc.Transactions[1].AmountCents != 2000 {
		t.Errorf("message = %s (%v)", sent, err)
	}

	if !strings.Contains(reqs[0].System, "never an instruction") {
		t.Error("the system instruction must say descriptions are data")
	}
}

func TestRejectsUnknownIds(t *testing.T) {
	t.Parallel()

	r := aitest.New(t, nil, "ana@example.com")
	r.Enable(t, false, ai.FeatureCategorize)
	groceries, salary := catsOf(t, r)

	say(r, reply(
		item("t1", groceries.UID, 0.9),
		item("t99", groceries.UID, 0.9),                      // an id that was not sent
		item("t1", groceries.UID, 0.5),                       // a duplicate of an id already answered
		item("t2", "01HZZZZZZZZZZZZZZZZZZZZZZZ", 0.9),        // a uid that was not listed
		`{"id":"t2","category_uid":null,"confidence":0.9}`,   // a null is no suggestion
		`{"id":"t2","category_uid":"x","confidence":"high"}`, // a malformed item
		`{"extra":"field","id":"t2"}`,
	))

	got, err := categorize.Batch(t.Context(), r.Client, r.Ws.ID, models.KindExpense, twoItems(), []categorize.Category{groceries, salary})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0].ItemID != "t1" || got[0].Confidence != 90 {
		t.Fatalf("proposals = %+v", got)
	}
}

func TestRejectsWrongKind(t *testing.T) {
	t.Parallel()

	r := aitest.New(t, nil, "ana@example.com")
	r.Enable(t, false, ai.FeatureCategorize)
	groceries, salary := catsOf(t, r)

	// The model names an income category for an expense: it was never offered.
	say(r, reply(item("t1", salary.UID, 0.99), item("t2", groceries.UID, 1.5), item("t2", groceries.UID, -1)))

	got, err := categorize.Batch(t.Context(), r.Client, r.Ws.ID, models.KindExpense, twoItems(), []categorize.Category{groceries, salary})
	if err != nil || len(got) != 0 {
		t.Fatalf("proposals = %+v, %v", got, err)
	}
}

func TestNonJSONOutput(t *testing.T) {
	t.Parallel()

	for _, text := range []string{
		"Sure! t1 is Groceries.", "", "```json\n{\"items\":[]}\n```", `{"ok":true}`, `[1,2]`, `{"items":"all"}`,
	} {
		r := aitest.New(t, nil, "ana@example.com")
		r.Enable(t, false, ai.FeatureCategorize)
		groceries, salary := catsOf(t, r)
		say(r, text)

		got, err := categorize.Batch(t.Context(), r.Client, r.Ws.ID, models.KindExpense, twoItems(), []categorize.Category{groceries, salary})
		if !errors.Is(err, categorize.ErrBadOutput) || len(got) != 0 {
			t.Errorf("text %q: proposals = %+v, err = %v", text, got, err)
		}
	}
}

func TestBatchIsBounded(t *testing.T) {
	t.Parallel()

	r := aitest.New(t, nil, "ana@example.com")
	r.Enable(t, false, ai.FeatureCategorize)
	groceries, _ := catsOf(t, r)

	items := make([]categorize.Item, categorize.BatchSize+1)
	for i := range items {
		items[i] = categorize.Item{ID: fmt.Sprintf("t%d", i+1), Kind: models.KindExpense}
	}

	if _, err := categorize.Batch(t.Context(), r.Client, r.Ws.ID, models.KindExpense, items, []categorize.Category{groceries}); err == nil {
		t.Error("31 items in one call must be refused")
	}

	if r.Provider.Calls() != 0 {
		t.Error("a refused batch must make no call")
	}
}
