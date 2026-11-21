package models

import (
	"github.com/AlexGalhardo/vukem/orm"
)

// WorkspaceSubscription mirrors a Stripe subscription of a workspace;
// webhooks keep it in sync. UnitAmount is the price in cents per month.
type WorkspaceSubscription struct {
	orm.Model

	WorkspaceID          int64        `json:"workspace_id"`
	StripeSubscriptionID string       `json:"-"`
	Status               string       `json:"status"`
	PriceID              string       `json:"-"`
	UnitAmount           int64        `json:"unit_amount"`
	CurrentPeriodEnd     orm.NullTime `json:"current_period_end"`
	CancelAtPeriodEnd    bool         `json:"cancel_at_period_end"`
}

// Table maps WorkspaceSubscription to workspace_subscriptions.
func (WorkspaceSubscription) Table() string { return "workspace_subscriptions" }

// Valid reports whether the subscription grants the Pro plan: active or in
// its trial. past_due, unpaid, incomplete and canceled do not.
func (s WorkspaceSubscription) Valid() bool { return s.Status == "active" || s.Status == "trialing" }
