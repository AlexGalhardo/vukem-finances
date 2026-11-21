package finance

import (
	"context"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// QuickAddAccount is an account a picker may offer: archived ones are left out.
type QuickAddAccount struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// QuickAddCategory is a category a picker may offer.
type QuickAddCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// QuickAddOptions is what the new transaction and new transfer dialogs need,
// on any page: the pickers, today in the workspace timezone and the currency.
type QuickAddOptions struct {
	Accounts   []QuickAddAccount  `json:"accounts"`
	Categories []QuickAddCategory `json:"categories"`
	Today      string             `json:"today"`
	Currency   string             `json:"currency"`
}

// QuickAddOptions lists the live, not archived accounts and the categories.
func (s *Service) QuickAddOptions(ctx context.Context) (QuickAddOptions, error) {
	ws, err := s.WS.Current(ctx)
	if err != nil {
		return QuickAddOptions{}, err
	}

	accounts, err := workspaces.Scoped[models.Account](ctx, s.DB).WhereNull("archived_at").
		OrderBy("position", orm.Asc).OrderBy("id", orm.Asc).Get(ctx)
	if err != nil {
		return QuickAddOptions{}, err
	}

	categories, err := s.ActiveCategories(ctx)
	if err != nil {
		return QuickAddOptions{}, err
	}

	out := QuickAddOptions{
		Accounts: make([]QuickAddAccount, len(accounts)), Categories: make([]QuickAddCategory, len(categories)),
		Today: s.nowIn(ws).Format(isoDate), Currency: ws.Currency,
	}

	for i, a := range accounts {
		out.Accounts[i] = QuickAddAccount{ID: a.ID, Name: a.Name, Kind: a.Kind}
	}

	for i, c := range categories {
		out.Categories[i] = QuickAddCategory{ID: c.ID, Name: c.Name, Kind: c.Kind}
	}

	return out, nil
}
