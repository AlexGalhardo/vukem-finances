package operations

import (
	"time"

	vukemapi "github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// The wire format. Money is an integer *_cents in the workspace currency, a
// date is YYYY-MM-DD, a timestamp is RFC 3339 in UTC and a row is named by its
// uid; the numeric ids of the database never leave the server.

// Stamp is a timestamp as the wire has it: RFC 3339 in UTC.
func Stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// StampNull is Stamp of a nullable time, nil when null.
func StampNull(t orm.NullTime) *string {
	if !t.Valid {
		return nil
	}

	s := Stamp(t.V)

	return &s
}

// NoInput is the input of an operation that takes none.
type NoInput struct{}

// Empty is the output of an operation that answers 204.
type Empty struct{}

// List is the answer of a list that comes whole, and of a page of a paged list
// (with meta and links).
type List[R any] = vukemapi.Envelope[R]

// AccountOut is an account with its balance as of today.
type AccountOut struct {
	UID          string  `json:"uid"           example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Version      int64   `json:"version"       example:"1"`
	Name         string  `json:"name"          example:"Checking"`
	Kind         string  `json:"kind"          example:"checking"`
	OpeningCents int64   `json:"opening_cents" example:"100000"`
	BalanceCents int64   `json:"balance_cents" example:"123450"`
	Currency     string  `json:"currency"      example:"USD"`
	Archived     bool    `json:"archived"`
	ArchivedAt   *string `json:"archived_at"`
	Position     int     `json:"position"      example:"1"`
	CreatedAt    string  `json:"created_at"    example:"2026-11-21T12:00:00Z"`
	UpdatedAt    string  `json:"updated_at"    example:"2026-11-21T12:00:00Z"`
}

func accountOut(a models.Account, balance int64, currency string) AccountOut {
	return AccountOut{
		UID: a.UID, Version: a.Version, Name: a.Name, Kind: a.Kind, OpeningCents: a.OpeningCents,
		BalanceCents: balance, Currency: currency, Archived: a.ArchivedAt.Valid, ArchivedAt: StampNull(a.ArchivedAt),
		Position: a.Position, CreatedAt: Stamp(a.CreatedAt), UpdatedAt: Stamp(a.UpdatedAt),
	}
}

// CategoryOut is a category.
type CategoryOut struct {
	UID       string  `json:"uid"        example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Version   int64   `json:"version"    example:"1"`
	Name      string  `json:"name"       example:"Groceries"`
	Kind      string  `json:"kind"       example:"expense"`
	ParentUID *string `json:"parent_uid"`
	Color     string  `json:"color"      example:"green"`
	Icon      string  `json:"icon"       example:"cart"`
	Archived  bool    `json:"archived"`
	CreatedAt string  `json:"created_at" example:"2026-11-21T12:00:00Z"`
	UpdatedAt string  `json:"updated_at" example:"2026-11-21T12:00:00Z"`
}

func categoryOut(c models.Category, uids map[int64]string) CategoryOut {
	out := CategoryOut{
		UID: c.UID, Version: c.Version, Name: c.Name, Kind: c.Kind, Color: c.Color, Icon: c.Icon,
		Archived: c.ArchivedAt.Valid, CreatedAt: Stamp(c.CreatedAt), UpdatedAt: Stamp(c.UpdatedAt),
	}

	if c.ParentID.Valid {
		if uid, ok := uids[c.ParentID.V]; ok {
			out.ParentUID = &uid
		}
	}

	return out
}

// TagOut is a tag.
type TagOut struct {
	UID       string `json:"uid"        example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Version   int64  `json:"version"    example:"1"`
	Name      string `json:"name"       example:"work"`
	Color     string `json:"color"      example:"blue"`
	CreatedAt string `json:"created_at" example:"2026-11-21T12:00:00Z"`
	UpdatedAt string `json:"updated_at" example:"2026-11-21T12:00:00Z"`
}

func tagOut(t models.Tag) TagOut {
	return TagOut{
		UID: t.UID, Version: t.Version, Name: t.Name, Color: t.Color,
		CreatedAt: Stamp(t.CreatedAt), UpdatedAt: Stamp(t.UpdatedAt),
	}
}

// TagRefOut is a tag on a transaction.
type TagRefOut struct {
	UID   string `json:"uid"   example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Name  string `json:"name"  example:"work"`
	Color string `json:"color" example:"blue"`
}

// TransactionOut is a transaction. A transfer leg carries the transfer_uid
// both legs share.
type TransactionOut struct {
	UID         string      `json:"uid"          example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Version     int64       `json:"version"      example:"1"`
	AccountUID  string      `json:"account_uid"  example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	CategoryUID *string     `json:"category_uid"`
	Kind        string      `json:"kind"         example:"expense"`
	AmountCents int64       `json:"amount_cents" example:"1250"`
	Currency    string      `json:"currency"     example:"USD"`
	Description string      `json:"description"  example:"Lunch"`
	Date        string      `json:"date"         example:"2026-11-21"`
	Tags        []TagRefOut `json:"tags"`
	TransferUID *string     `json:"transfer_uid"`
	CreatedAt   string      `json:"created_at"   example:"2026-11-21T12:00:00Z"`
	UpdatedAt   string      `json:"updated_at"   example:"2026-11-21T12:00:00Z"`
}

// lookups are the uids of the accounts and categories rows point at.
type lookups struct {
	accounts   map[int64]string
	categories map[int64]string
	currency   string
}

func (l lookups) transaction(r finance.TxRow) TransactionOut {
	t := r.Transaction
	out := TransactionOut{
		UID: t.UID, Version: t.Version, AccountUID: l.accounts[t.AccountID], Kind: t.Kind, AmountCents: t.AmountCents,
		Currency: l.currency, Description: t.Description, Date: t.OccurredOn, Tags: make([]TagRefOut, 0, len(r.Tags)),
		CreatedAt: Stamp(t.CreatedAt), UpdatedAt: Stamp(t.UpdatedAt),
	}

	if t.CategoryID.Valid {
		if uid, ok := l.categories[t.CategoryID.V]; ok {
			out.CategoryUID = &uid
		}
	}

	if t.TransferUID.Valid {
		uid := t.TransferUID.V
		out.TransferUID = &uid
	}

	for _, tg := range r.Tags {
		out.Tags = append(out.Tags, TagRefOut{UID: tg.UID, Name: tg.Name, Color: tg.Color})
	}

	return out
}

// TransferOut is a transfer: the expense leg on the source account and the
// income leg on the destination.
type TransferOut struct {
	UID  string         `json:"uid" example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	From TransactionOut `json:"from"`
	To   TransactionOut `json:"to"`
}

// BudgetOut is a budget with what the month spent against it.
type BudgetOut struct {
	UID          string `json:"uid"           example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Version      int64  `json:"version"       example:"1"`
	CategoryUID  string `json:"category_uid"  example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	CategoryName string `json:"category_name" example:"Groceries"`
	Month        string `json:"month"         example:"2026-11"`
	LimitCents   int64  `json:"limit_cents"   example:"50000"`
	SpentCents   int64  `json:"spent_cents"   example:"12000"`
	Percent      int    `json:"percent"       example:"24"`
	Currency     string `json:"currency"      example:"USD"`
}

// SummaryAccountOut is the balance of one account in the summary.
type SummaryAccountOut struct {
	UID          string `json:"uid"           example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Name         string `json:"name"          example:"Checking"`
	BalanceCents int64  `json:"balance_cents" example:"123450"`
	Archived     bool   `json:"archived"`
}

// SummaryOut is a month of the workspace. Transfers are neither income nor
// expense; the balances are as of today in the workspace timezone.
type SummaryOut struct {
	Month         string              `json:"month"           example:"2026-11"`
	Currency      string              `json:"currency"        example:"USD"`
	IncomeCents   int64               `json:"income_cents"    example:"500000"`
	ExpenseCents  int64               `json:"expense_cents"   example:"320000"`
	NetCents      int64               `json:"net_cents"       example:"180000"`
	NetWorthCents int64               `json:"net_worth_cents" example:"1234500"`
	Accounts      []SummaryAccountOut `json:"accounts"`
}
