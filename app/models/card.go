package models

import (
	"context"
	"database/sql"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// AccountCreditCard is the kind of the account behind a credit card.
const AccountCreditCard = "credit_card"

// CreditCard holds the settings of a credit card; the card itself is an
// account of kind credit_card (its name, its archive state, its balance: the
// debt, negative while money is owed). Only the last four digits are kept:
// no field holds a card number, an expiry or a security code.
type CreditCard struct {
	orm.Model
	orm.SoftDeletes

	WorkspaceID int64  `json:"workspace_id"`
	UID         string `json:"uid"`
	Version     int64  `json:"version" orm:"readonly"`
	AccountID   int64  `json:"account_id"`
	LimitCents  int64  `json:"limit_cents"`
	// ClosingDay and DueDay are days of the month, 1 to 31; a month with
	// fewer days uses its last day.
	ClosingDay int `json:"closing_day"`
	DueDay     int `json:"due_day"`
	// PaymentAccountID is the account statements are paid from by default.
	PaymentAccountID sql.Null[int64] `json:"payment_account_id"`
	Brand            string          `json:"brand"`
	Last4            string          `json:"last4"`
}

// SetTenant implements tenancy.Scoped.
func (m *CreditCard) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant and the uid.
func (m *CreditCard) BeforeCreate(ctx context.Context) error {
	if err := CheckCents(m.LimitCents); err != nil {
		return err
	}

	m.UID = stampUID(m.UID)

	return tenancy.Assign(ctx, m)
}
