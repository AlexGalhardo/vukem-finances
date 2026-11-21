package tests

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// deleteForm sends the values in the query string: Go reads no form body of a DELETE.
func deleteForm(c *vukemtest.Client, target string, form url.Values) *vukemtest.Response {
	if len(form) > 0 {
		target += "?" + form.Encode()
	}

	return c.Do(http.MethodDelete, target, strings.NewReader(""), "")
}

func TestCategoriesHTTP(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	vera := viewerOf(t, app, ana)
	max := newVerifiedUser(t, app, "Max", "max@example.com")
	join(t, app, ana, max, models.RoleMember)
	member := who{max, ana.Workspace, browser(t, app, max, ana.Workspace.ID)}
	acc := firstAccount(t, app, ana)

	// The page lists the categories, the tags and the palette.
	ana.Client.Inertia("").Get("/categories").AssertOK().AssertInertia("Categories").
		HasProp("colors", len(models.CategoryColors)).HasProp("icons", len(models.CategoryIcons))

	// Create a parent and a child with a look; the child takes the kind of its parent.
	ana.Client.PostForm("/categories", url.Values{"name": {"Food"}, "kind": {"expense"}, "color": {"olive"}, "icon": {"utensils"}}).
		AssertRedirect("/categories")

	food := firstOf[models.Category](t, app, ana, "Food")
	if food.Color != "olive" || food.Icon != "utensils" {
		t.Fatalf("food = %+v, want its color and icon", food)
	}

	ana.Client.PostForm("/categories", url.Values{"name": {"Pizza"}, "parentId": {idString(food.ID)}}).AssertRedirect("/categories")

	pizza := firstOf[models.Category](t, app, ana, "Pizza")
	if !pizza.ParentID.Valid || pizza.ParentID.V != food.ID || pizza.Kind != "expense" {
		t.Fatalf("pizza = %+v, want a child of Food", pizza)
	}

	// The rules answer as flashes, nothing changes.
	ana.Client.PostForm("/categories", url.Values{"name": {"Slice"}, "parentId": {idString(pizza.ID)}}).AssertRedirect("/categories")
	ana.Client.PostForm("/categories", url.Values{"name": {"Bad"}, "kind": {"expense"}, "color": {"#fff"}}).AssertRedirect("/categories")
	vukemtest.AssertDatabaseMissing(t, app, "categories", map[string]any{"name": "Slice"})
	vukemtest.AssertDatabaseMissing(t, app, "categories", map[string]any{"name": "Bad"})

	// Update, archive, unarchive.
	ana.Client.PatchJSON("/categories/"+idString(pizza.ID), map[string]any{"name": "Pizzas", "color": "red", "parentId": idString(food.ID)}).
		AssertRedirect("/categories")
	vukemtest.AssertDatabaseHas(t, app, "categories", map[string]any{"id": pizza.ID, "name": "Pizzas", "color": "red"})

	ana.Client.PostJSON("/categories/"+idString(food.ID)+"/archive", nil).AssertRedirect("/categories")
	vukemtest.AssertDatabaseCount(t, app, "categories", rowCount(t, app, "categories"))

	if got := firstOf[models.Category](t, app, ana, "Food"); !got.ArchivedAt.Valid {
		t.Error("Food is not archived")
	}

	ana.Client.PostJSON("/categories/"+idString(food.ID)+"/unarchive", nil).AssertRedirect("/categories")

	// A used category cannot be deleted without a target; with one, its transactions move.
	groceries := categoryID(t, app, ana, "Groceries")
	addTransaction(t, ana, acc, groceries, "expense", "4.00", "2026-03-01").AssertRedirect("/transactions")

	deleteForm(ana.Client, "/categories/"+idString(groceries), url.Values{}).AssertRedirect("/categories")
	vukemtest.AssertDatabaseHas(t, app, "categories", map[string]any{"id": groceries, "deleted_at": nil})

	deleteForm(ana.Client, "/categories/"+idString(groceries), url.Values{"reassignTo": {idString(food.ID)}}).AssertRedirect("/categories")
	vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{"category_id": food.ID, "amount_cents": 400})

	// The viewer reads and cannot write; the member records but does not manage.
	vera.Client.Get("/categories").AssertOK()
	member.Client.Get("/categories").AssertOK()

	before := rowCount(t, app, "categories")

	for name, c := range map[string]who{"viewer": vera, "member": member} {
		for _, call := range []func() *vukemtest.Response{
			func() *vukemtest.Response {
				return c.Client.PostForm("/categories", url.Values{"name": {"Sneaky"}, "kind": {"expense"}})
			},
			func() *vukemtest.Response {
				return c.Client.PatchJSON("/categories/"+idString(food.ID), map[string]any{"name": "Sneaky"})
			},
			func() *vukemtest.Response { return c.Client.PostJSON("/categories/"+idString(food.ID)+"/archive", nil) },
			func() *vukemtest.Response {
				return deleteForm(c.Client, "/categories/"+idString(food.ID), url.Values{})
			},
		} {
			if res := call(); res.Status() != http.StatusForbidden {
				t.Errorf("%s: status %d, want 403", name, res.Status())
			}
		}
	}

	if got := rowCount(t, app, "categories"); got != before {
		t.Errorf("categories went from %d to %d under refused calls", before, got)
	}

	if got := firstOf[models.Category](t, app, ana, "Food"); got.Name != "Food" || got.ArchivedAt.Valid {
		t.Errorf("food = %+v, refused calls changed it", got)
	}
}

func TestTagsHTTP(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	vera := viewerOf(t, app, ana)
	max := newVerifiedUser(t, app, "Max", "max@example.com")
	join(t, app, ana, max, models.RoleMember)
	member := who{max, ana.Workspace, browser(t, app, max, ana.Workspace.ID)}
	acc := firstAccount(t, app, ana)

	// Tags are made inline by the transaction form (JSON, as the dialog sends it).
	ana.Client.PostJSON("/transactions", map[string]any{
		"accountId": idString(acc), "kind": "expense", "amount": "1.234,56", "description": "Hotel", "date": "2026-03-01",
		"tags": []string{"Trip", "trip", "Work"},
	}).AssertStatus(http.StatusCreated)

	vukemtest.AssertDatabaseCount(t, app, "tags", 2)
	vukemtest.AssertDatabaseCount(t, app, "transaction_tags", 2)
	vukemtest.AssertDatabaseHas(t, app, "transactions", map[string]any{"description": "Hotel", "amount_cents": 123456})

	// A member records with tags too; a viewer cannot.
	member.Client.PostJSON("/transactions", map[string]any{
		"accountId": idString(acc), "kind": "expense", "amount": "5", "description": "Taxi", "date": "2026-03-02", "tags": []string{"Trip"},
	}).AssertStatus(http.StatusCreated)
	vera.Client.PostJSON("/transactions", map[string]any{
		"accountId": idString(acc), "kind": "expense", "amount": "5", "description": "Nope", "date": "2026-03-02", "tags": []string{"Sneak"},
	}).AssertForbidden()
	vukemtest.AssertDatabaseCount(t, app, "tags", 2)

	// Eleven tags are refused whole.
	var eleven []string
	for i := range 11 {
		eleven = append(eleven, "t"+strconv.Itoa(i))
	}

	ana.Client.PostJSON("/transactions", map[string]any{
		"accountId": idString(acc), "kind": "expense", "amount": "5", "description": "Too many", "date": "2026-03-02", "tags": eleven,
	}).AssertStatus(http.StatusUnprocessableEntity)
	vukemtest.AssertDatabaseMissing(t, app, "transactions", map[string]any{"description": "Too many"})
	vukemtest.AssertDatabaseCount(t, app, "tags", 2)

	// Managing the tags themselves is for owners and admins.
	ana.Client.PostJSON("/tags", map[string]any{"name": "Solo", "color": "teal"}).AssertStatus(http.StatusCreated)
	ana.Client.PostJSON("/tags", map[string]any{"name": "SOLO"}).AssertStatus(http.StatusConflict)

	for name, c := range map[string]who{"viewer": vera, "member": member} {
		c.Client.PostJSON("/tags", map[string]any{"name": "Sneaky"}).AssertForbidden()
		c.Client.PatchJSON("/tags/1", map[string]any{"name": "Sneaky"}).AssertForbidden()
		c.Client.Delete("/tags/1").AssertForbidden()

		_ = name
	}

	vukemtest.AssertDatabaseMissing(t, app, "tags", map[string]any{"name": "Sneaky"})

	// Attach and detach on an existing transaction: the member may, the viewer may not.
	solo := firstOf[models.Tag](t, app, ana, "Solo")
	taxi := firstOf[models.Transaction](t, app, ana, "")

	for _, c := range []who{ana, member} {
		c.Client.PostJSON("/transactions/"+idString(taxi.ID)+"/tags/"+idString(solo.ID), nil).AssertOK()
		c.Client.Delete("/transactions/" + idString(taxi.ID) + "/tags/" + idString(solo.ID)).AssertStatus(http.StatusSeeOther)
	}

	vera.Client.PostJSON("/transactions/"+idString(taxi.ID)+"/tags/"+idString(solo.ID), nil).AssertForbidden()
	vukemtest.AssertDatabaseCount(t, app, "transaction_tags", 3)

	// The list filters by the query string, with pt-BR and en amounts.
	trip := firstOf[models.Tag](t, app, ana, "Trip")
	list := ana.Client.Inertia("")

	list.Get("/transactions?tag="+idString(trip.ID)).AssertInertia("Transactions").HasProp("transactions", 2)
	list.Get("/transactions?tag="+idString(trip.ID)+"&min=1.000,00").AssertInertia("Transactions").HasProp("transactions", 1)
	list.Get("/transactions?tag="+idString(trip.ID)+"&min=1,000.00&max=2000").AssertInertia("Transactions").HasProp("transactions", 1)
	list.Get("/transactions?q=hot").AssertInertia("Transactions").HasProp("transactions", 1)
	list.Get("/transactions?q=%25").AssertInertia("Transactions").HasProp("transactions", 0)
	list.Get("/transactions?account="+idString(acc)).AssertInertia("Transactions").HasProp("transactions", 2)
	list.Get("/transactions?min=abc").AssertRedirect("/transactions")

	// Deleting a tag takes it off the transactions.
	ana.Client.Delete("/tags/" + idString(trip.ID)).AssertRedirect("/categories")
	list.Get("/transactions?tag="+idString(trip.ID)).AssertInertia("Transactions").HasProp("transactions", 0)
	vukemtest.AssertDatabaseCount(t, app, "transaction_tags", 1)
}
