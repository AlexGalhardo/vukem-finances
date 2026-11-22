package reports

import (
	"context"
	"slices"
	"strings"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// CategoryRow is what a category took in the period. A parent row counts its
// own transactions plus those of its subcategories (OwnCents is the first
// part) and lists the subcategories in Children. ShareBps is the row's part
// of the report total in basis points. Uncategorized is the row of the
// transactions with no category; it has no UID and no name.
type CategoryRow struct {
	UID           string        `json:"uid"`
	Name          string        `json:"name"`
	Uncategorized bool          `json:"uncategorized"`
	Cents         int64         `json:"cents"`
	OwnCents      int64         `json:"ownCents"`
	ShareBps      int64         `json:"shareBps"`
	Children      []CategoryRow `json:"children"`
}

// CategoryReport is the spending (or income) of the period by category.
type CategoryReport struct {
	Currency   string        `json:"currency"`
	Period     Period        `json:"period"`
	Kind       string        `json:"kind"`
	TotalCents int64         `json:"totalCents"`
	Rows       []CategoryRow `json:"rows"`
}

func sortRows(rows []CategoryRow) {
	slices.SortFunc(rows, func(a, b CategoryRow) int {
		if a.Cents != b.Cents {
			if a.Cents > b.Cents {
				return -1
			}

			return 1
		}

		return strings.Compare(a.Name+"\x00"+a.UID, b.Name+"\x00"+b.UID)
	})
}

// ByCategory totals the transactions of kind (expense when empty) by
// category, parents rolled up with their children beneath.
func (s *Service) ByCategory(ctx context.Context, p Period, kind string) (CategoryReport, error) {
	if kind != models.KindIncome {
		kind = models.KindExpense
	}

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return CategoryReport{}, err
	}

	out := CategoryReport{Currency: ws.Currency, Period: p, Kind: kind, Rows: []CategoryRow{}}

	groups, err := s.ordinary(ctx, p).Where("kind", kind).
		GroupBy("category_id", "amount_cents").
		SelectAs("category_id", "category_id").
		SelectAs("amount_cents", "amount_cents").
		SelectExpr(orm.Count("*"), "n").
		Scan[group](ctx)
	if err != nil {
		return out, err
	}

	// A live transaction may still name a deleted category: trashed ones are read too.
	cats, err := s.categories(ctx)
	if err != nil {
		return out, err
	}

	own := map[int64]int64{}

	var uncategorized int64

	for _, g := range groups {
		cents := mul(g.Amount, g.N)
		out.TotalCents = add(out.TotalCents, cents)

		if g.CategoryID.Valid {
			own[g.CategoryID.V] = add(own[g.CategoryID.V], cents)
		} else {
			uncategorized = add(uncategorized, cents)
		}
	}

	byID := make(map[int64]models.Category, len(cats))
	for _, c := range cats {
		byID[c.ID] = c
	}

	rows := map[int64]*CategoryRow{}

	var top []int64

	for id, cents := range own {
		c, ok := byID[id]
		if !ok {
			continue
		}

		root := id
		if c.ParentID.Valid {
			if _, known := byID[c.ParentID.V]; known {
				root = c.ParentID.V
			}
		}

		r := rows[root]
		if r == nil {
			rc := byID[root]
			r = &CategoryRow{UID: rc.UID, Name: rc.Name, Children: []CategoryRow{}}
			rows[root] = r
			top = append(top, root)
		}

		r.Cents = add(r.Cents, cents)

		if root == id {
			r.OwnCents = add(r.OwnCents, cents)
			continue
		}

		r.Children = append(r.Children, CategoryRow{
			UID: c.UID, Name: c.Name, Cents: cents, OwnCents: cents, ShareBps: bps(cents, out.TotalCents), Children: []CategoryRow{},
		})
	}

	for _, id := range top {
		r := rows[id]
		r.ShareBps = bps(r.Cents, out.TotalCents)
		sortRows(r.Children)
		out.Rows = append(out.Rows, *r)
	}

	if uncategorized > 0 {
		out.Rows = append(out.Rows, CategoryRow{
			Uncategorized: true, Cents: uncategorized, OwnCents: uncategorized,
			ShareBps: bps(uncategorized, out.TotalCents), Children: []CategoryRow{},
		})
	}

	sortRows(out.Rows)

	return out, nil
}

// AccountRow is the income and expense of one account in the period.
type AccountRow struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Sum
}

// AccountReport is the period by account; only accounts with movement are listed.
type AccountReport struct {
	Currency string       `json:"currency"`
	Period   Period       `json:"period"`
	Rows     []AccountRow `json:"rows"`
}

// ByAccount sums income and expense per account (archived ones included).
func (s *Service) ByAccount(ctx context.Context, p Period) (AccountReport, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return AccountReport{}, err
	}

	out := AccountReport{Currency: ws.Currency, Period: p, Rows: []AccountRow{}}

	groups, err := s.ordinary(ctx, p).
		GroupBy("account_id", "kind", "amount_cents").
		SelectAs("account_id", "account_id").
		SelectAs("kind", "kind").
		SelectAs("amount_cents", "amount_cents").
		SelectExpr(orm.Count("*"), "n").
		Scan[group](ctx)
	if err != nil {
		return out, err
	}

	accounts, err := s.accounts(ctx)
	if err != nil {
		return out, err
	}

	byID := make(map[int64]*AccountRow, len(accounts))
	order := []int64{}

	for _, g := range groups {
		a, ok := accounts[g.AccountID]
		if !ok {
			continue
		}

		r := byID[g.AccountID]
		if r == nil {
			r = &AccountRow{UID: a.UID, Name: a.Name, Kind: a.Kind}
			byID[g.AccountID] = r
			order = append(order, g.AccountID)
		}

		r.put(g)
	}

	for _, id := range order {
		out.Rows = append(out.Rows, *byID[id])
	}

	slices.SortFunc(out.Rows, func(a, b AccountRow) int {
		if a.ExpenseCents != b.ExpenseCents {
			if a.ExpenseCents > b.ExpenseCents {
				return -1
			}

			return 1
		}

		return strings.Compare(a.Name+"\x00"+a.UID, b.Name+"\x00"+b.UID)
	})

	return out, nil
}

// TagRow is the income and expense of the transactions that carry a tag.
type TagRow struct {
	UID  string `json:"uid"`
	Name string `json:"name"`
	Sum
}

// TagReport is the period by tag. A transaction with several tags counts
// once under each of them, so the rows can add up to more than the totals.
type TagReport struct {
	Currency string   `json:"currency"`
	Period   Period   `json:"period"`
	Rows     []TagRow `json:"rows"`
}

// ByTag sums income and expense per tag.
func (s *Service) ByTag(ctx context.Context, p Period) (TagReport, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return TagReport{}, err
	}

	out := TagReport{Currency: ws.Currency, Period: p, Rows: []TagRow{}}

	// The join drops soft-deleted transactions through its own ON clause only for the
	// joined model, so the transaction's deleted_at is the scope of the base query.
	groups, err := s.ordinary(ctx, p).
		InnerJoin[models.TransactionTag]("transaction_tags.transaction_id", "=", "transactions.id").
		Where("transaction_tags.workspace_id", wsID(ctx)).
		GroupBy("transaction_tags.tag_id", "transactions.kind", "transactions.amount_cents").
		SelectAs("transaction_tags.tag_id", "tag_id").
		SelectAs("transactions.kind", "kind").
		SelectAs("transactions.amount_cents", "amount_cents").
		SelectExpr(orm.Count("*"), "n").
		Scan[group](ctx)
	if err != nil {
		return out, err
	}

	tags, err := s.tags(ctx)
	if err != nil {
		return out, err
	}

	byID := map[int64]*TagRow{}
	order := []int64{}

	for _, g := range groups {
		t, ok := tags[g.TagID]
		if !ok {
			continue
		}

		r := byID[g.TagID]
		if r == nil {
			r = &TagRow{UID: t.UID, Name: t.Name}
			byID[g.TagID] = r
			order = append(order, g.TagID)
		}

		r.put(g)
	}

	for _, id := range order {
		out.Rows = append(out.Rows, *byID[id])
	}

	slices.SortFunc(out.Rows, func(a, b TagRow) int {
		if a.ExpenseCents != b.ExpenseCents {
			if a.ExpenseCents > b.ExpenseCents {
				return -1
			}

			return 1
		}

		return strings.Compare(a.Name+"\x00"+a.UID, b.Name+"\x00"+b.UID)
	})

	return out, nil
}
