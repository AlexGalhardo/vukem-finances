package tests

import (
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func TestEntitled(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	revoked := orm.NullTime{}
	revoked.V, revoked.Valid = now.Add(-time.Hour), true

	tests := []struct {
		name string
		sub  *models.WorkspaceSubscription
		pix  *models.PixEntitlement
		want bool
	}{
		{name: "nothing", want: false},
		{name: "active stripe", sub: &models.WorkspaceSubscription{Status: "active"}, want: true},
		{name: "trialing stripe", sub: &models.WorkspaceSubscription{Status: "trialing"}, want: true},
		{name: "past due stripe", sub: &models.WorkspaceSubscription{Status: "past_due"}, want: false},
		{
			name: "revoked but still active at Stripe",
			sub:  &models.WorkspaceSubscription{Status: "active", AccessRevokedAt: revoked, RevokedReason: models.RevokedRefund},
			want: false,
		},
		{
			name: "valid pix",
			pix:  &models.PixEntitlement{AnchorAt: now, PaidMonths: 1, ValidUntil: now.Add(24 * time.Hour)},
			want: true,
		},
		{
			name: "expired pix",
			pix:  &models.PixEntitlement{AnchorAt: now, PaidMonths: 1, ValidUntil: now.Add(-time.Second)},
			want: false,
		},
		{
			name: "revoked pix",
			pix: &models.PixEntitlement{
				AnchorAt: now, PaidMonths: 1, ValidUntil: now.Add(24 * time.Hour), RevokedAt: revoked,
			},
			want: false,
		},
		{
			name: "revoked stripe and valid pix",
			sub:  &models.WorkspaceSubscription{Status: "active", AccessRevokedAt: revoked},
			pix:  &models.PixEntitlement{AnchorAt: now, PaidMonths: 1, ValidUntil: now.Add(24 * time.Hour)},
			want: true,
		},
	}

	app := newApp(t)
	svc := service(t, app)
	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	for i, tc := range tests {
		ana := signUp(t, app, "Ana", "ana"+string(rune('a'+i))+"@example.com")
		ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)

		if tc.sub != nil {
			tc.sub.WorkspaceID, tc.sub.StripeSubscriptionID = ana.Workspace.ID, "sub_"+tc.name
			if err := orm.Create(t.Context(), db, tc.sub); err != nil {
				t.Fatal(err)
			}
		}

		if tc.pix != nil {
			if err := orm.Create(ctx, db, tc.pix); err != nil {
				t.Fatal(err)
			}
		}

		got, err := svc.Entitled(t.Context(), ana.Workspace.ID, now)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}

		if got != tc.want {
			t.Errorf("%s: Entitled = %v, want %v", tc.name, got, tc.want)
		}
	}
}
