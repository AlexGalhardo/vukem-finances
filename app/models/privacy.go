package models

import (
	"time"

	"github.com/AlexGalhardo/vukem/orm"
)

// PrivacyAudit is the minimal record of a deletion event: what happened, when
// and a non-reversible reference to the subject. It holds no financial data and
// no name, so it can outlive the rows it describes.
type PrivacyAudit struct {
	orm.Model

	Event      string `json:"event"`
	SubjectRef string `json:"subject_ref"`
}

// Table maps PrivacyAudit to privacy_audit.
func (PrivacyAudit) Table() string { return "privacy_audit" }

// PaymentRecord is what is kept of a payment after its workspace is purged: the
// amount, the date, the provider reference and a non-reversible reference to the
// subject, until RetainUntil (D-022).
type PaymentRecord struct {
	orm.Model

	SubjectRef  string    `json:"subject_ref"`
	Provider    string    `json:"provider"`
	ProviderRef string    `json:"provider_ref"`
	AmountCents int64     `json:"amount_cents"`
	Currency    string    `json:"currency"`
	PaidAt      time.Time `json:"paid_at"`
	RetainUntil time.Time `json:"retain_until"`
}

// Table maps PaymentRecord to payment_records.
func (PaymentRecord) Table() string { return "payment_records" }

// PurgedToken is the public id of an API token of a purged workspace.
type PurgedToken struct {
	orm.Model

	TokenID string `json:"token_id"`
}

// Table maps PurgedToken to purged_tokens.
func (PurgedToken) Table() string { return "purged_tokens" }
