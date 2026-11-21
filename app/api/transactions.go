package api

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	vukemapi "github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/web"

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

func (a *API) transactionView(c *web.Ctx, x models.Transaction) (TransactionOut, error) {
	ctx := c.Context()

	rows, err := a.Fin.Rows(ctx, []models.Transaction{x})
	if err != nil {
		return TransactionOut{}, err
	}

	l, err := a.lookups(ctx)
	if err != nil {
		return TransactionOut{}, err
	}

	return l.transaction(rows[0]), nil
}

// accountID and the like turn the uid of a filter or a body into the id the
// services take; a uid that names nothing here is a 404.
func (a *API) accountID(c *web.Ctx, uid string) (int64, error) {
	x, err := a.accountByUID(c.Context(), uid)

	return x.ID, err
}

func (a *API) categoryID(c *web.Ctx, uid string) (int64, error) {
	if uid == "" {
		return 0, nil
	}

	x, err := a.categoryByUID(c.Context(), uid)

	return x.ID, err
}

func centsFilter(v *int64) string {
	if v == nil {
		return ""
	}

	return centsText(*v)
}

func (a *API) transactionsList(c *web.Ctx, in TransactionListIn) (List[TransactionOut], error) {
	ctx := c.Context()
	f := finance.TxFilter{
		From: in.From, To: in.To, Kind: in.Kind, Q: in.Q, Min: centsFilter(in.MinCents), Max: centsFilter(in.MaxCents), Page: in.Page,
	}

	var err error

	if in.Account != "" {
		if f.AccountID, err = a.accountID(c, in.Account); err != nil {
			return List[TransactionOut]{}, err
		}
	}

	if f.CategoryID, err = a.categoryID(c, in.Category); err != nil {
		return List[TransactionOut]{}, err
	}

	if in.Tag != "" {
		tag, err := a.tagByUID(ctx, in.Tag)
		if err != nil {
			return List[TransactionOut]{}, err
		}

		f.TagIDs = []int64{tag.ID}
	}

	page, err := a.Fin.TransactionPage(ctx, f)
	if err != nil {
		return List[TransactionOut]{}, err
	}

	l, err := a.lookups(ctx)
	if err != nil {
		return List[TransactionOut]{}, err
	}

	data := make([]TransactionOut, 0, len(page.Rows))
	for _, r := range page.Rows {
		data = append(data, l.transaction(r))
	}

	last := max(int((page.Total+int64(page.PageSize)-1)/int64(page.PageSize)), 1)
	links := &vukemapi.Links{Self: pageLink(c, page.Page)}

	if page.Page < last {
		next := pageLink(c, page.Page+1)
		links.Next = &next
	}

	if page.Page > 1 {
		prev := pageLink(c, min(page.Page-1, last))
		links.Prev = &prev
	}

	total := page.Total

	return List[TransactionOut]{
		Data:  data,
		Meta:  &vukemapi.Meta{Page: page.Page, PerPage: page.PageSize, Total: &total, LastPage: last},
		Links: links,
	}, nil
}

// pageLink is the URL of page n of the request: its path and query with page set.
func pageLink(c *web.Ctx, n int) string {
	u := *c.Request().URL
	q := u.Query()
	q.Set("page", strconv.Itoa(n))

	return (&url.URL{Path: u.Path, RawQuery: q.Encode()}).RequestURI()
}

func (a *API) transactionsShow(c *web.Ctx, in UIDIn) (TransactionOut, error) {
	x, err := a.transactionByUID(c.Context(), in.UID)
	if err != nil {
		return TransactionOut{}, err
	}

	return a.transactionView(c, x)
}

func (a *API) transactionsCreate(c *web.Ctx, in TransactionCreateIn) (TransactionOut, error) {
	ctx := c.Context()

	if err := a.checkUID(in.UID); err != nil {
		return TransactionOut{}, err
	}

	account, err := a.accountID(c, in.AccountUID)
	if err != nil {
		return TransactionOut{}, err
	}

	category, err := a.categoryID(c, in.CategoryUID)
	if err != nil {
		return TransactionOut{}, err
	}

	if old, found, err := existing[models.Transaction](ctx, a, in.UID); err != nil || found {
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

		v, err := a.transactionView(c, old)
		if err != nil {
			return TransactionOut{}, err
		}

		return TransactionOut{}, c.JSON(http.StatusOK, v)
	}

	if err := a.requireScopeForNewTags(ctx, in.Tags); err != nil {
		return TransactionOut{}, err
	}

	x, err := a.Fin.CreateTransaction(ctx, finance.TxInput{
		UID: in.UID, AccountID: account, CategoryID: category, Kind: in.Kind, Amount: centsText(in.AmountCents),
		Description: in.Description, Date: in.Date, Tags: in.Tags,
	})
	if err != nil {
		// Two calls with one uid at once: the unique index let one in, the other is the replay.
		if a.WS.DB.UniqueViolation(err) {
			if old, found, lerr := existing[models.Transaction](ctx, a, in.UID); lerr == nil && found && !old.DeletedAt.Valid {
				v, verr := a.transactionView(c, old)
				if verr != nil {
					return TransactionOut{}, verr
				}

				return TransactionOut{}, c.JSON(http.StatusOK, v)
			}
		}

		return TransactionOut{}, err
	}

	return a.transactionView(c, x)
}

func (a *API) transactionsUpdate(c *web.Ctx, in TransactionUpdateIn) (TransactionOut, error) {
	ctx := c.Context()

	x, err := a.transactionByUID(ctx, in.UID)
	if err != nil {
		return TransactionOut{}, err
	}

	cur, err := a.transactionView(c, x)
	if err != nil {
		return TransactionOut{}, err
	}

	if err := checkVersion(c, x.Version, cur); err != nil {
		return TransactionOut{}, err
	}

	next := finance.TxInput{
		AccountID: x.AccountID, CategoryID: x.CategoryID.V, Kind: x.Kind, Amount: centsText(x.AmountCents),
		Description: x.Description, Date: x.OccurredOn,
	}

	if in.AccountUID != nil {
		if next.AccountID, err = a.accountID(c, *in.AccountUID); err != nil {
			return TransactionOut{}, err
		}
	}

	if in.CategoryUID != nil {
		if next.CategoryID, err = a.categoryID(c, *in.CategoryUID); err != nil {
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

		if err := a.requireScopeForNewTags(ctx, next.Tags); err != nil {
			return TransactionOut{}, err
		}
	}

	saved, err := a.Fin.UpdateTransaction(ctx, x.ID, next)
	if err != nil {
		return TransactionOut{}, err
	}

	return a.transactionView(c, saved)
}

func (a *API) transactionsDelete(c *web.Ctx, in UIDIn) (Empty, error) {
	ctx := c.Context()

	x, err := a.transactionByUID(ctx, in.UID)
	if err != nil {
		return Empty{}, err
	}

	cur, err := a.transactionView(c, x)
	if err != nil {
		return Empty{}, err
	}

	if err := checkVersion(c, x.Version, cur); err != nil {
		return Empty{}, err
	}

	return Empty{}, a.Fin.DeleteTransaction(ctx, x.ID)
}

func (a *API) transfersCreate(c *web.Ctx, in TransferCreateIn) (TransferOut, error) {
	ctx := c.Context()

	from, err := a.accountID(c, in.FromAccountUID)
	if err != nil {
		return TransferOut{}, err
	}

	to, err := a.accountID(c, in.ToAccountUID)
	if err != nil {
		return TransferOut{}, err
	}

	t, err := a.Fin.CreateTransfer(ctx, finance.TransferInput{
		FromAccountID: from, ToAccountID: to, Amount: centsText(in.AmountCents), Description: in.Description, Date: in.Date,
	})
	if err != nil {
		return TransferOut{}, err
	}

	rows, err := a.Fin.Rows(ctx, []models.Transaction{t.From, t.To})
	if err != nil {
		return TransferOut{}, err
	}

	l, err := a.lookups(ctx)
	if err != nil {
		return TransferOut{}, err
	}

	return TransferOut{UID: t.UID, From: l.transaction(rows[0]), To: l.transaction(rows[1])}, nil
}

func (a *API) transfersDelete(c *web.Ctx, in UIDIn) (Empty, error) {
	ctx := c.Context()

	leg, err := workspaces.Scoped[models.Transaction](ctx, a.WS.DB).Where("transfer_uid", in.UID).First(ctx)
	if err != nil {
		return Empty{}, lookupErr(err)
	}

	return Empty{}, a.Fin.DeleteTransfer(ctx, leg.ID)
}
