package tests

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// searchFixture is a Pro workspace with a way to add transactions to search.
func searchFixture(t *testing.T) (who, *finance.Service, int64, func(desc, amount, date string, tags ...string) models.Transaction) {
	t.Helper()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	acc := firstAccount(t, app, ana)

	add := func(desc, amount, date string, tags ...string) models.Transaction {
		t.Helper()

		tx, err := fin.CreateTransaction(ctx, finance.TxInput{
			AccountID: acc, Kind: "expense", Amount: amount, Description: desc, Date: date, Tags: tags,
		})
		if err != nil {
			t.Fatalf("add %q: %v", desc, err)
		}

		return tx
	}

	return ana, fin, acc, add
}

func descriptions(rows []finance.TxRow) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Description
	}

	slices.Sort(out)

	return out
}

func sameSet(got, want []string) bool {
	return len(got) == len(want) && (len(got) == 0 || slices.Equal(got, want))
}

func TestSearchEscapesWildcards(t *testing.T) {
	t.Parallel()

	ana, fin, _, add := searchFixture(t)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

	add("Sale 50% off", "1.00", "2026-03-01")
	add("Sale 500 items", "1.00", "2026-03-02")
	add("under_score", "1.00", "2026-03-03")
	add("underXscore", "1.00", "2026-03-04")
	add(`back\slash`, "1.00", "2026-03-05")
	add("Coffee", "1.00", "2026-03-06")

	for q, want := range map[string][]string{
		"50%":       {"Sale 50% off"},
		"%":         {"Sale 50% off"},
		"under_s":   {"under_score"},
		`k\s`:       {`back\slash`},
		"COFFEE":    {"Coffee"},
		"coff":      {"Coffee"},
		"_":         {"under_score"},
		"no match":  {},
		"sale 50%":  {"Sale 50% off"},
		"  coffee ": {"Coffee"},
	} {
		rows, err := fin.Transactions(ctx, finance.TxFilter{Q: q})
		if err != nil {
			t.Fatalf("search %q: %v", q, err)
		}

		if got := descriptions(rows); !sameSet(got, want) {
			t.Errorf("search %q = %v, want %v", q, got, want)
		}
	}
}

func TestSecurity_SearchTakesSQLMetacharactersAsText(t *testing.T) {
	t.Parallel()

	ana, fin, _, add := searchFixture(t)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

	add("Coffee", "1.00", "2026-03-01")
	add(`it's "quoted"`, "1.00", "2026-03-02")

	for _, q := range []string{"' OR 1=1 --", `"; DROP TABLE transactions; --`, "%' OR '%'='", strings.Repeat("%", 500)} {
		rows, err := fin.Transactions(ctx, finance.TxFilter{Q: q})
		if err != nil {
			t.Errorf("search %q: %v", q, err)
		}

		if len(rows) > 0 {
			t.Errorf("search %q matched %v", q, descriptions(rows))
		}
	}

	if rows, err := fin.Transactions(ctx, finance.TxFilter{Q: `it's "q`}); err != nil || len(rows) != 1 {
		t.Errorf("a quote in the search = %d rows, %v; want the literal match", len(rows), err)
	}

	if rows, _ := fin.Transactions(ctx, finance.TxFilter{}); len(rows) != 2 {
		t.Errorf("%d rows left after the searches, want 2", len(rows))
	}
}

func TestTransactionFilters(t *testing.T) {
	t.Parallel()

	ana, fin, acc, add := searchFixture(t)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

	add("Hotel", "1.234,56", "2026-03-01", "Trip", "Work")
	add("Train", "80,00", "2026-03-02", "Trip")
	add("Lunch", "12.50", "2026-03-03")

	tags, _ := fin.Tags(ctx)
	ids := map[string]int64{}

	for _, g := range tags {
		ids[g.Name] = g.ID
	}

	cases := map[string]struct {
		f    finance.TxFilter
		want []string
	}{
		"one tag":          {finance.TxFilter{TagIDs: []int64{ids["Trip"]}}, []string{"Hotel", "Train"}},
		"every tag asked":  {finance.TxFilter{TagIDs: []int64{ids["Trip"], ids["Work"]}}, []string{"Hotel"}},
		"a pt-BR minimum":  {finance.TxFilter{Min: "1.000,00"}, []string{"Hotel"}},
		"an en maximum":    {finance.TxFilter{Max: "80.00"}, []string{"Lunch", "Train"}},
		"a window":         {finance.TxFilter{Min: "12,50", Max: "80"}, []string{"Lunch", "Train"}},
		"text and tag":     {finance.TxFilter{Q: "tr", TagIDs: []int64{ids["Trip"]}}, []string{"Train"}},
		"from a date":      {finance.TxFilter{AccountID: acc, From: "2026-03-02"}, []string{"Lunch", "Train"}},
		"a foreign tag id": {finance.TxFilter{TagIDs: []int64{999999}}, []string{}},
		"a malformed date": {finance.TxFilter{From: "yesterday"}, []string{"Hotel", "Lunch", "Train"}},
		"the kind":         {finance.TxFilter{Kind: "expense"}, []string{"Hotel", "Lunch", "Train"}},
	}

	for name, c := range cases {
		rows, err := fin.Transactions(ctx, c.f)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}

		if got := descriptions(rows); !sameSet(got, c.want) {
			t.Errorf("%s = %v, want %v", name, got, c.want)
		}
	}

	for _, bad := range []string{"abc", "-5.00", "99999999999999999"} {
		if _, err := fin.Transactions(ctx, finance.TxFilter{Min: bad}); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("min %q = %v, want 422", bad, err)
		}
	}

	// A parent category shows what is filed under its subcategories.
	food, _ := fin.AddCategory(ctx, finance.CategoryInput{Name: "Food", Kind: "expense"})
	pizza, _ := fin.AddCategory(ctx, finance.CategoryInput{Name: "Pizza", ParentID: food.ID})

	if _, err := fin.CreateTransaction(ctx, finance.TxInput{
		AccountID: acc, CategoryID: pizza.ID, Kind: "expense", Amount: "9.00", Description: "Slice", Date: "2026-03-04",
	}); err != nil {
		t.Fatal(err)
	}

	for _, id := range []int64{food.ID, pizza.ID} {
		rows, err := fin.Transactions(ctx, finance.TxFilter{CategoryID: id})
		if err != nil || len(rows) != 1 || rows[0].Description != "Slice" {
			t.Errorf("category %d = %v, %v; want Slice", id, descriptions(rows), err)
		}
	}
}

func TestPagination(t *testing.T) {
	t.Parallel()

	ana, fin, _, add := searchFixture(t)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

	for i := range 120 {
		add("row "+strconv.Itoa(1000+i), "1.00", fmt.Sprintf("2026-%02d-%02d", 1+i/28, 1+i%28))
	}

	seen := map[int64]bool{}

	var newest string

	for page, wantRows := range map[int]int{1: 50, 2: 50, 3: 20, 4: 0} {
		p, err := fin.TransactionPage(ctx, finance.TxFilter{Page: page})
		if err != nil {
			t.Fatal(err)
		}

		if len(p.Rows) != wantRows || p.Total != 120 || p.PageSize != 50 || p.Page != page {
			t.Errorf("page %d = %d rows, total %d, size %d; want %d rows of 120", page, len(p.Rows), p.Total, p.PageSize, wantRows)
		}

		for _, r := range p.Rows {
			if seen[r.ID] {
				t.Errorf("row %d is on two pages", r.ID)
			}

			seen[r.ID] = true
		}

		if page == 1 {
			newest = p.Rows[0].OccurredOn
		}
	}

	if len(seen) != 120 {
		t.Errorf("%d distinct rows over the pages, want 120", len(seen))
	}

	if newest < "2026-05" {
		t.Errorf("the first row is %s, want the newest", newest)
	}

	// The total follows the filter, and a page below 1 is the first.
	p, err := fin.TransactionPage(ctx, finance.TxFilter{Q: "row 100", Page: -3})
	if err != nil || p.Total != 10 || p.Page != 1 || len(p.Rows) != 10 {
		t.Errorf("filtered page = %+v, %v; want 10 rows on page 1", p, err)
	}
}
