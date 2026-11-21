package models

import (
	"context"
	"time"

	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/tenancy"
)

// Statuses of a PIX payment.
const (
	PixPending  = "pending"
	PixPaid     = "paid"
	PixExpired  = "expired"
	PixRefunded = "refunded"
	PixDisputed = "disputed"
)

// Reasons access can be revoked.
const (
	RevokedRefund      = "refund"
	RevokedDispute     = "dispute"
	RevokedDisputeLost = "dispute_lost"
	RevokedAdmin       = "admin"
)

// PixPayment is one PIX charge of a workspace. It is server-only data (no uid,
// never synced): PublicRef is the id used in URLs and logs, ProviderRef the
// id at AbacatePay. AmountCents comes from the server's price, never from the
// request. Records follow decision D-022 (5 years, no legal review).
type PixPayment struct {
	orm.Model

	PublicRef   string       `json:"public_ref"`
	WorkspaceID int64        `json:"workspace_id"`
	ProviderRef string       `json:"-"`
	Status      string       `json:"status"`
	AmountCents int64        `json:"amount_cents"`
	Currency    string       `json:"currency"`
	Months      int          `json:"months"`
	BRCode      string       `json:"-" db:"br_code"`
	QRPNGBase64 string       `json:"-" db:"qr_png_base64"`
	ExpiresAt   time.Time    `json:"expires_at"`
	PaidAt      orm.NullTime `json:"paid_at"`
	CreatedBy   int64        `json:"created_by"`
}

// Table maps PixPayment to pix_payments.
func (PixPayment) Table() string { return "pix_payments" }

// SetTenant implements tenancy.Scoped.
func (m *PixPayment) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant and the public reference.
func (m *PixPayment) BeforeCreate(ctx context.Context) error {
	if err := CheckCents(m.AmountCents); err != nil {
		return err
	}

	m.PublicRef = stampUID(m.PublicRef)

	return tenancy.Assign(ctx, m)
}

// PixEntitlement is the PIX access of a workspace: ValidUntil is always
// AddMonths(AnchorAt, PaidMonths), so stacked payments do not drift.
type PixEntitlement struct {
	orm.Model

	WorkspaceID   int64        `json:"workspace_id"`
	AnchorAt      time.Time    `json:"anchor_at"`
	PaidMonths    int          `json:"paid_months"`
	ValidUntil    time.Time    `json:"valid_until"`
	RevokedAt     orm.NullTime `json:"revoked_at"`
	RevokedReason string       `json:"revoked_reason"`
}

// Table maps PixEntitlement to pix_entitlements.
func (PixEntitlement) Table() string { return "pix_entitlements" }

// SetTenant implements tenancy.Scoped.
func (m *PixEntitlement) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *PixEntitlement) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }

// Granting reports whether the entitlement gives Pro at now.
func (m PixEntitlement) Granting(now time.Time) bool {
	return !m.RevokedAt.Valid && m.ValidUntil.After(now)
}

// BillingEvent is a line of the billing audit trail: provider ids only, no
// card data and no personal data.
type BillingEvent struct {
	orm.Model

	WorkspaceID int64  `json:"workspace_id"`
	Source      string `json:"source"`
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	Summary     string `json:"summary"`
}

// Table maps BillingEvent to billing_events.
func (BillingEvent) Table() string { return "billing_events" }

// SetTenant implements tenancy.Scoped.
func (m *BillingEvent) SetTenant(id tenancy.ID) { m.WorkspaceID = tenantID(id) }

// BeforeCreate stamps the tenant.
func (m *BillingEvent) BeforeCreate(ctx context.Context) error { return tenancy.Assign(ctx, m) }
