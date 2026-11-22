package operations

import (
	"context"
	"strings"

	vukemapi "github.com/AlexGalhardo/vukem/api"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// TransactionListIn is the query of transactions.list.
type TransactionListIn struct {
	From     string `json:"from"      validate:"max=10" doc:"First date, YYYY-MM-DD" example:"2026-11-01"`
	To       string `json:"to"        validate:"max=10" doc:"Last date, YYYY-MM-DD" example:"2026-11-30"`
	Account  string `json:"account"   validate:"max=26" doc:"Only this account (uid)"`
	Category string `json:"category"  validate:"max=26" doc:"Only this category and its subcategories (uid)"`
	Tag      string `json:"tag"       validate:"max=26" doc:"Only transactions with this tag (uid)"`
	Kind     string `json:"kind"      validate:"max=20" doc:"income or expense"`
	Q        string `json:"q"         validate:"max=200" doc:"Text of the description, case-insensitive"`
	MinCents *int64 `json:"min_cents" doc:"Smallest absolute amount, in cents"`
	MaxCents *int64 `json:"max_cents" doc:"Largest absolute amount, in cents"`
	Page     int    `json:"page"      validate:"min=0"  doc:"Page number, from 1; a page holds 50 rows" example:"1"`
}

// TransactionCreateIn is the body of transactions.create.
type TransactionCreateIn struct {
	UID         string   `json:"uid"          validate:"max=26"  doc:"Optional uid chosen by the client (a ULID); repeating the call returns the row already made"`
	AccountUID  string   `json:"account_uid"  validate:"required,max=26"`
	CategoryUID string   `json:"category_uid" validate:"max=26"  doc:"A category of the same kind; leave out for none"`
	Kind        string   `json:"kind"         validate:"required,max=20" doc:"income or expense" example:"expense"`
	AmountCents int64    `json:"amount_cents" doc:"A positive whole number of cents; the kind gives the sign" example:"1250"`
	Date        string   `json:"date"         validate:"required,max=10" example:"2026-11-21"`
	Description string   `json:"description"  validate:"max=200" example:"Lunch"`
	Tags        []string `json:"tags"         validate:"max=40" doc:"Tag names, created when new; at most 10 per transaction"`
}

// TransactionUpdateIn is the body of transactions.update; a field left out stays
// as it is. category_uid "" removes the category, and tags replaces the tags
// (an empty list removes them all).
type TransactionUpdateIn struct {
	UID         string    `json:"uid"          validate:"required,max=26"`
	AccountUID  *string   `json:"account_uid"  validate:"max=26"`
	CategoryUID *string   `json:"category_uid" validate:"max=26"`
	Kind        *string   `json:"kind"         validate:"max=20"`
	AmountCents *int64    `json:"amount_cents"`
	Date        *string   `json:"date"         validate:"max=10"`
	Description *string   `json:"description"  validate:"max=200"`
	Tags        *[]string `json:"tags"         validate:"max=40"`
}

// TransferCreateIn is the body of transfers.create.
type TransferCreateIn struct {
	FromAccountUID string `json:"from_account_uid" validate:"required,max=26"`
	ToAccountUID   string `json:"to_account_uid"   validate:"required,max=26"`
	AmountCents    int64  `json:"amount_cents"     doc:"A positive whole number of cents" example:"50000"`
	Date           string `json:"date"             validate:"required,max=10" example:"2026-11-21"`
	Description    string `json:"description"      validate:"max=200" example:"Move to savings"`
}

type transactionOps struct {
	List       *Op[TransactionListIn, List[TransactionOut]]
	Show       *Op[UIDIn, TransactionOut]
	Create     *Op[TransactionCreateIn, TransactionOut]
	Update     *Op[TransactionUpdateIn, TransactionOut]
	Delete     *Op[UIDIn, Empty]
	Transfer   *Op[TransferCreateIn, TransferOut]
	Untransfer *Op[UIDIn, Empty]
}

func (c *Catalog) defineTransactions() {
	c.Transactions = transactionOps{
		List: Define(c.Registry, Operation[TransactionListIn, List[TransactionOut]]{
			Name: "transactions.list", Summary: "List transactions, newest first, with filters", Scope: ScopeTransactionsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in TransactionListIn) (List[TransactionOut], error) {
				return c.transactionsList(ctx, in)
			},
		}),
		Show: Define(c.Registry, Operation[UIDIn, TransactionOut]{
			Name: "transactions.show", Summary: "Read a transaction", Scope: ScopeTransactionsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (TransactionOut, error) {
				return c.transactionsShow(ctx, in)
			},
		}),
		Create: Define(c.Registry, Operation[TransactionCreateIn, TransactionOut]{
			Name: "transactions.create", Summary: "Record a transaction", Scope: ScopeTransactionsWrite, Kind: KindWrite, Idempotent: true,
			Run: c.transactionsCreate,
		}),
		Update: Define(c.Registry, Operation[TransactionUpdateIn, TransactionOut]{
			Name: "transactions.update", Summary: "Change a transaction", Scope: ScopeTransactionsWrite, Kind: KindWrite, Idempotent: true,
			Run: c.transactionsUpdate,
		}),
		Delete: Define(c.Registry, Operation[UIDIn, Empty]{
			Name: "transactions.delete", Summary: "Delete a transaction", Scope: ScopeTransactionsWrite, Kind: KindDestructive,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (Empty, error) { return c.transactionsDelete(ctx, in) },
		}),
		Transfer: Define(c.Registry, Operation[TransferCreateIn, TransferOut]{
			Name: "transfers.create", Summary: "Move money between two accounts", Scope: ScopeTransfersWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in TransferCreateIn) (TransferOut, error) {
				return c.transfersCreate(ctx, in)
			},
		}),
		Untransfer: Define(c.Registry, Operation[UIDIn, Empty]{
			Name: "transfers.delete", Summary: "Delete a transfer and both of its legs", Scope: ScopeTransfersWrite, Kind: KindDestructive,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (Empty, error) { return c.transfersDelete(ctx, in) },
		}),
	}
}

func (c *Catalog) transactionView(ctx context.Context, x models.Transaction) (TransactionOut, error) {
	rows, err := c.Fin.Rows(ctx, []models.Transaction{x})
	if err != nil {
		return TransactionOut{}, err
	}

	l, err := c.lookups(ctx)
	if err != nil {
		return TransactionOut{}, err
	}

	return l.transaction(rows[0]), nil
}

// accountID and the like turn the uid of a filter or a body into the id the
// services take; a uid that names nothing here is a 404.
func (c *Catalog) accountID(ctx context.Context, uid string) (int64, error) {
	x, err := c.accountByUID(ctx, uid)

	return x.ID, err
}

func (c *Catalog) categoryID(ctx context.Context, uid string) (int64, error) {
	if uid == "" {
		return 0, nil
	}

	x, err := c.categoryByUID(ctx, uid)

	return x.ID, err
}

func centsFilter(v *int64) string {
	if v == nil {
		return ""
	}

	return centsText(*v)
}

// transactionsList answers a page with its meta. The links are URLs of the
// surface, so the adapter builds them from the meta.
func (c *Catalog) transactionsList(ctx context.Context, in TransactionListIn) (List[TransactionOut], error) {
	f := finance.TxFilter{
		From: in.From, To: in.To, Kind: in.Kind, Q: in.Q, Min: centsFilter(in.MinCents), Max: centsFilter(in.MaxCents), Page: in.Page,
	}

	var err error

	if in.Account != "" {
		if f.AccountID, err = c.accountID(ctx, in.Account); err != nil {
			return List[TransactionOut]{}, err
		}
	}

	if f.CategoryID, err = c.categoryID(ctx, in.Category); err != nil {
		return List[TransactionOut]{}, err
	}

	if in.Tag != "" {
		tag, err := c.tagByUID(ctx, in.Tag)
		if err != nil {
			return List[TransactionOut]{}, err
		}

		f.TagIDs = []int64{tag.ID}
	}

	page, err := c.Fin.TransactionPage(ctx, f)
	if err != nil {
		return List[TransactionOut]{}, err
	}

	l, err := c.lookups(ctx)
	if err != nil {
		return List[TransactionOut]{}, err
	}

	data := make([]TransactionOut, 0, len(page.Rows))
	for _, r := range page.Rows {
		data = append(data, l.transaction(r))
	}

	last := max(int((page.Total+int64(page.PageSize)-1)/int64(page.PageSize)), 1)
	total := page.Total

	return List[TransactionOut]{
		Data: data,
		Meta: &vukemapi.Meta{Page: page.Page, PerPage: page.PageSize, Total: &total, LastPage: last},
	}, nil
}

func (c *Catalog) transactionsShow(ctx context.Context, in UIDIn) (TransactionOut, error) {
	x, err := c.transactionByUID(ctx, in.UID)
	if err != nil {
		return TransactionOut{}, err
	}

	return c.transactionView(ctx, x)
}

func (c *Catalog) transactionsCreate(ctx context.Context, actor Actor, in TransactionCreateIn) (TransactionOut, error) {
	if err := c.checkUID(in.UID); err != nil {
		return TransactionOut{}, err
	}

	account, err := c.accountID(ctx, in.AccountUID)
	if err != nil {
		return TransactionOut{}, err
	}

	category, err := c.categoryID(ctx, in.CategoryUID)
	if err != nil {
		return TransactionOut{}, err
	}

	if old, found, err := existing[models.Transaction](ctx, c, in.UID); err != nil || found {
		if err != nil {
			return TransactionOut{}, err
		}

		if old.DeletedAt.Valid {
			return TransactionOut{}, errUIDDeleted
		}

		if old.AccountID != account || old.Kind != in.Kind || old.AmountCents != in.AmountCents || old.OccurredOn != in.Date ||
			old.Description != strings.TrimSpace(in.Description) || old.CategoryID.V != category {
			return TransactionOut{}, errUIDDiffers
		}

		MarkExisting(ctx)

		return c.transactionView(ctx, old)
	}

	if err := c.requireScopeForNewTags(ctx, actor, in.Tags); err != nil {
		return TransactionOut{}, err
	}

	x, err := c.Fin.CreateTransaction(ctx, finance.TxInput{
		UID: in.UID, AccountID: account, CategoryID: category, Kind: in.Kind, Amount: centsText(in.AmountCents),
		Description: in.Description, Date: in.Date, Tags: in.Tags,
	})
	if err != nil {
		// Two calls with one uid at once: the unique index let one in, the other is the replay.
		if c.WS.DB.UniqueViolation(err) {
			if old, found, lerr := existing[models.Transaction](ctx, c, in.UID); lerr == nil && found && !old.DeletedAt.Valid {
				MarkExisting(ctx)

				return c.transactionView(ctx, old)
			}
		}

		return TransactionOut{}, err
	}

	return c.transactionView(ctx, x)
}

func (c *Catalog) transactionsUpdate(ctx context.Context, actor Actor, in TransactionUpdateIn) (TransactionOut, error) {
	x, err := c.transactionByUID(ctx, in.UID)
	if err != nil {
		return TransactionOut{}, err
	}

	cur, err := c.transactionView(ctx, x)
	if err != nil {
		return TransactionOut{}, err
	}

	if err := checkVersion(ctx, x.Version, cur); err != nil {
		return TransactionOut{}, err
	}

	next := finance.TxInput{
		AccountID: x.AccountID, CategoryID: x.CategoryID.V, Kind: x.Kind, Amount: centsText(x.AmountCents),
		Description: x.Description, Date: x.OccurredOn,
	}

	if in.AccountUID != nil {
		if next.AccountID, err = c.accountID(ctx, *in.AccountUID); err != nil {
			return TransactionOut{}, err
		}
	}

	if in.CategoryUID != nil {
		if next.CategoryID, err = c.categoryID(ctx, *in.CategoryUID); err != nil {
			return TransactionOut{}, err
		}
	}

	if in.Kind != nil {
		next.Kind = *in.Kind
	}

	if in.AmountCents != nil {
		next.Amount = centsText(*in.AmountCents)
	}

	if in.Date != nil {
		next.Date = *in.Date
	}

	if in.Description != nil {
		next.Description = *in.Description
	}

	if in.Tags != nil {
		next.Tags = *in.Tags
		if next.Tags == nil {
			next.Tags = []string{}
		}

		if err := c.requireScopeForNewTags(ctx, actor, next.Tags); err != nil {
			return TransactionOut{}, err
		}
	}

	saved, err := c.Fin.UpdateTransaction(ctx, x.ID, next)
	if err != nil {
		return TransactionOut{}, err
	}

	return c.transactionView(ctx, saved)
}

func (c *Catalog) transactionsDelete(ctx context.Context, in UIDIn) (Empty, error) {
	x, err := c.transactionByUID(ctx, in.UID)
	if err != nil {
		return Empty{}, err
	}

	cur, err := c.transactionView(ctx, x)
	if err != nil {
		return Empty{}, err
	}

	if err := checkVersion(ctx, x.Version, cur); err != nil {
		return Empty{}, err
	}

	return Empty{}, c.Fin.DeleteTransaction(ctx, x.ID)
}

func (c *Catalog) transfersCreate(ctx context.Context, in TransferCreateIn) (TransferOut, error) {
	from, err := c.accountID(ctx, in.FromAccountUID)
	if err != nil {
		return TransferOut{}, err
	}

	to, err := c.accountID(ctx, in.ToAccountUID)
	if err != nil {
		return TransferOut{}, err
	}

	t, err := c.Fin.CreateTransfer(ctx, finance.TransferInput{
		FromAccountID: from, ToAccountID: to, Amount: centsText(in.AmountCents), Description: in.Description, Date: in.Date,
	})
	if err != nil {
		return TransferOut{}, err
	}

	rows, err := c.Fin.Rows(ctx, []models.Transaction{t.From, t.To})
	if err != nil {
		return TransferOut{}, err
	}

	l, err := c.lookups(ctx)
	if err != nil {
		return TransferOut{}, err
	}

	return TransferOut{UID: t.UID, From: l.transaction(rows[0]), To: l.transaction(rows[1])}, nil
}

func (c *Catalog) transfersDelete(ctx context.Context, in UIDIn) (Empty, error) {
	leg, err := workspaces.Scoped[models.Transaction](ctx, c.WS.DB).Where("transfer_uid", in.UID).First(ctx)
	if err != nil {
		return Empty{}, lookupErr(err)
	}

	return Empty{}, c.Fin.DeleteTransfer(ctx, leg.ID)
}
