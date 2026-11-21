package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func fillMonth(t *testing.T, app *vukemtest.App, w who, account int64, n int, month string) {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	rows := make([]models.Transaction, n)
	for i := range rows {
		rows[i] = models.Transaction{
			AccountID: account, Kind: "expense", AmountCents: 100, Description: "bulk", OccurredOn: month + "-15",
		}
	}

	if err := orm.CreateMany(workspaces.WithTenant(t.Context(), w.Workspace.ID), db, rows); err != nil {
		t.Fatal(err)
	}
}

// The plan is read inside the transaction that checks the quota, under the
// workspace lock: a plan that changes while a request waits for the lock is the
// plan the request is judged by, in both directions.
func TestQuotaAtomicWhenPlanChanges(t *testing.T) {
	t.Parallel()

	month := time.Now().UTC().Format("2006-01")

	t.Run("a refund while the request waits", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		acc := firstAccount(t, app, ana)
		fillMonth(t, app, ana, acc, 100, month)
		makePro(t, app, ana)

		db, err := app.DB()
		if err != nil {
			t.Fatal(err)
		}

		held, release := make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)

		// A transaction holds the write lock; the plan is revoked inside it.
		go func() {
			done <- db.Tx(t.Context(), func(tx *database.Tx) error {
				if _, err := orm.QueryOf[models.WorkspaceSubscription](tx).Where("workspace_id", ana.Workspace.ID).
					Update(t.Context(), orm.Set{"access_revoked_at": time.Now().UTC(), "revoked_reason": "refund"}); err != nil {
					return err
				}

				close(held)
				<-release

				return nil
			})
		}()

		<-held

		res := make(chan *vukemtest.Response, 1)

		go func() { res <- addTransaction(t, ana, acc, 0, "expense", "1.00", month+"-16") }()

		time.Sleep(300 * time.Millisecond) // the request has started and waits for the lock
		close(release)

		if err := <-done; err != nil {
			t.Fatal(err)
		}

		(<-res).AssertRedirect("/billing")
		vukemtest.AssertDatabaseCount(t, app, "transactions", 100)
	})

	t.Run("a payment while the request waits", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		acc := firstAccount(t, app, ana)
		fillMonth(t, app, ana, acc, 100, month)

		db, err := app.DB()
		if err != nil {
			t.Fatal(err)
		}

		held, release := make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)

		go func() {
			done <- db.Tx(t.Context(), func(tx *database.Tx) error {
				sub := models.WorkspaceSubscription{
					WorkspaceID: ana.Workspace.ID, StripeSubscriptionID: "sub_late", Status: "active", PriceID: "price_pro",
				}
				if err := orm.Create(t.Context(), tx, &sub); err != nil {
					return err
				}

				close(held)
				<-release

				return nil
			})
		}()

		<-held

		res := make(chan *vukemtest.Response, 1)

		go func() { res <- addTransaction(t, ana, acc, 0, "expense", "1.00", month+"-16") }()

		time.Sleep(300 * time.Millisecond)
		close(release)

		if err := <-done; err != nil {
			t.Fatal(err)
		}

		(<-res).AssertRedirect("/transactions")
		vukemtest.AssertDatabaseCount(t, app, "transactions", 101)
	})

	t.Run("concurrent adds never pass the limit", func(t *testing.T) {
		t.Parallel()

		app := newApp(t)
		ana := signUp(t, app, "Ana", "ana@example.com")
		acc := firstAccount(t, app, ana)
		fillMonth(t, app, ana, acc, 90, month)

		var wg sync.WaitGroup

		for range 30 {
			// A client is not safe for concurrent use: one per request.
			wg.Go(func() {
				c := who{ana.User, ana.Workspace, browser(t, app, ana.User, ana.Workspace.ID)}
				addTransaction(t, c, acc, 0, "expense", "1.00", month+"-16")
			})
		}

		wg.Wait()
		vukemtest.AssertDatabaseCount(t, app, "transactions", 100)
	})
}
