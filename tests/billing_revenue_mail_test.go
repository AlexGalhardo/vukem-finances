package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestAdminRevenue(t *testing.T) {
	t.Parallel()

	f, app := pixApp(t, newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC)))
	b := pixBilling(t, app)
	boss := signUp(t, app, "Boss", "boss@example.com")
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	cat := signUp(t, app, "Cat", "cat@example.com")
	dan := signUp(t, app, "Dan", "dan@example.com")

	makePro(t, app, ana) // Stripe active, 900 cents

	// Stripe: revoked by a refund, so it is not revenue.
	makePro(t, app, bob)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := orm.QueryOf[models.WorkspaceSubscription](db).Where("workspace_id", bob.Workspace.ID).
		Update(t.Context(), orm.Set{"access_revoked_at": time.Now().UTC(), "revoked_reason": "refund"}); err != nil {
		t.Fatal(err)
	}

	// PIX: 3 months for 5970 is 1990 a month; Dan's is refunded and counts for nothing.
	paidPix(t, f, b, cat, 3)
	refunded := paidPix(t, f, b, dan, 1)
	f.setStatus(refunded.ProviderRef, "REFUNDED")

	if err := b.HandlePixEvent(t.Context(), refunded.ProviderRef); err != nil {
		t.Fatal(err)
	}

	page := boss.Client.Inertia("").Get("/admin/revenue").AssertInertia("Admin/Revenue")
	page.PropEquals("mrrCents", float64(900))
	page.PropEquals("active", float64(1))
	page.PropEquals("pixMrrCents", float64(1990))
	page.PropEquals("pixActive", float64(1))
	page.PropEquals("recentPix.0.canRefund", false)
	page.PropEquals("recentPix.1.canRefund", true)

	// Only an admin sees it.
	ana.Client.Inertia("").Get("/admin/revenue").AssertNotFound()
}

func TestBillingMails(t *testing.T) {
	t.Parallel()

	t.Run("revoked mail says why", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct{ reason, status, sentence string }{
			{"refund", "", "the payment was refunded"},
			{"dispute", "needs_response", "under dispute"},
			{"dispute_lost", "lost", "ruled against the payment"},
		} {
			f, app, ana := proViaStripe(t)
			f.setCharge("ch_1", 900, 900)

			if tc.reason == "refund" {
				postStripe(t, app, "evt_x", "charge.refunded", map[string]any{"id": "ch_1"}).AssertOK()
			} else {
				f.setCharge("ch_1", 900, 0)
				f.setDispute("dp_1", "ch_1", tc.status)
				disputeEvent(t, app, "evt_x", "charge.dispute.created")
			}

			var html string

			for _, m := range app.Mails().SentTo("ana@example.com") {
				if m.Subject == "Your Pro access has ended" {
					html = m.HTML
				}
			}

			if !strings.Contains(html, tc.sentence) || !strings.Contains(html, "/billing") {
				t.Errorf("%s: mail = %q", tc.reason, html)
			}

			_ = ana
		}
	})

	t.Run("PIX receipt names the amount and the end", func(t *testing.T) {
		t.Parallel()

		f, app := pixApp(t, newPixClock(time.Date(2027, 1, 31, 10, 0, 0, 0, time.UTC)))
		b := pixBilling(t, app)
		ana := signUp(t, app, "Ana", "ana@example.com")
		paidPix(t, f, b, ana, 3)

		var found bool

		for _, m := range app.Mails().SentTo("ana@example.com") {
			if strings.HasPrefix(m.Subject, "Your PIX payment of") {
				found = true

				for _, want := range []string{"59.70", "3 month", "April 30, 2027", "does not renew"} {
					if !strings.Contains(m.Subject+m.HTML, want) {
						t.Errorf("receipt lacks %q: %s", want, m.HTML)
					}
				}
			}
		}

		if !found {
			t.Error("no PIX receipt")
		}
	})

}
