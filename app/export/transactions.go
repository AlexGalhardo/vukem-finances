package export

import (
	"context"
	"encoding/json"
	"iter"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Transactions yields the transactions of the workspace of ctx that a filter
// matches, as CSV rows. The names of accounts, categories and tags are read
// first (a few small tables, trashed ones too, since a live transaction may
// still name one); the transactions themselves stream in chunks.
func Transactions(ctx context.Context, fin *finance.Service, f finance.TxFilter) (iter.Seq2[TransactionRow, error], error) {
	accounts, err := workspaces.Scoped[models.Account](ctx, fin.DB).WithTrashed().Get(ctx)
	if err != nil {
		return nil, err
	}

	cats, err := workspaces.Scoped[models.Category](ctx, fin.DB).WithTrashed().Get(ctx)
	if err != nil {
		return nil, err
	}

	tags, err := workspaces.Scoped[models.Tag](ctx, fin.DB).WithTrashed().Get(ctx)
	if err != nil {
		return nil, err
	}

	an := make(map[int64]string, len(accounts))
	for _, a := range accounts {
		an[a.ID] = a.Name
	}

	cn := make(map[int64]string, len(cats))
	for _, c := range cats {
		cn[c.ID] = c.Name
	}

	tn := make(map[string]string, len(tags))
	for _, t := range tags {
		tn[t.UID] = t.Name
	}

	return func(yield func(TransactionRow, error) bool) {
		for t, err := range fin.EachTransaction(ctx, f) {
			if err != nil {
				yield(TransactionRow{}, err)

				return
			}

			if !yield(rowOf(t, an, cn, tn), nil) {
				return
			}
		}
	}, nil
}

func rowOf(t models.Transaction, accounts, categories map[int64]string, tags map[string]string) TransactionRow {
	r := TransactionRow{
		UID: t.UID, Date: t.OccurredOn, Account: accounts[t.AccountID], Kind: t.Kind,
		AmountCents: t.AmountCents, Description: t.Description,
	}

	if t.Kind != models.KindIncome {
		r.AmountCents = -t.AmountCents
	}

	if t.CategoryID.Valid {
		r.Category = categories[t.CategoryID.V]
	}

	if t.TransferUID.Valid {
		r.TransferUID = t.TransferUID.V
	}

	var uids []string
	if json.Unmarshal([]byte(t.TagUIDs), &uids) == nil {
		for _, u := range uids {
			if name, ok := tags[u]; ok {
				r.Tags = append(r.Tags, name)
			}
		}
	}

	return r
}
