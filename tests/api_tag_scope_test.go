package tests

import (
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/api"
)

// A token without tags:write may name existing tags on a transaction, but the
// transaction is not a way to create one.
func TestAPITransactionCannotCreateTagsWithoutTheScope(t *testing.T) {
	t.Parallel()

	app, ana, acc := proWorkspace(t)
	fin := softService(t, app)
	ctx := wsCtx(t, ana)

	if _, err := fin.CreateTag(ctx, "trip", ""); err != nil {
		t.Fatal(err)
	}

	accounts, err := fin.Accounts(ctx)
	if err != nil {
		t.Fatal(err)
	}

	uid := ""

	for _, a := range accounts {
		if a.ID == acc {
			uid = a.UID
		}
	}

	token := makeToken(t, app, ana, 30, api.ScopeTransactionsWrite)
	body := func(tags ...string) map[string]any {
		return map[string]any{
			"account_uid": uid, "kind": "expense", "amount_cents": 500, "date": "2026-03-01",
			"description": "scoped", "tags": tags,
		}
	}

	if got := apiClient(t, app, token).PostJSON("/api/v1/transactions", body("new-tag")).Status(); got != http.StatusForbidden {
		t.Errorf("creating a tag without tags:write = %d, want 403", got)
	}

	created := apiClient(t, app, token).PostJSON("/api/v1/transactions", body("TRIP"))
	if created.Status() != http.StatusCreated {
		t.Fatalf("naming an existing tag = %d: %s", created.Status(), created.Body())
	}

	path := "/api/v1/transactions/" + uidOf(t, created, "uid")
	patch := map[string]any{"tags": []string{"another-new-tag"}}

	if got := apiClient(t, app, token).PatchJSON(path, patch).Status(); got != http.StatusForbidden {
		t.Errorf("adding a new tag on update without tags:write = %d, want 403", got)
	}

	tags, err := fin.Tags(ctx)
	if err != nil || len(tags) != 1 {
		t.Errorf("tags = %d, %v; want only the existing one", len(tags), err)
	}
}
