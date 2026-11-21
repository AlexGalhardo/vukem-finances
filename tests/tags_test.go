package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func TestTags(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	makePro(t, app, ana)

	fin := softService(t, app)
	ctx := workspaces.WithTenant(t.Context(), ana.Workspace.ID)
	bobCtx := workspaces.WithTenant(t.Context(), bob.Workspace.ID)
	acc := firstAccount(t, app, ana)

	newTx := func(tags []string) (models.Transaction, error) {
		return fin.CreateTransaction(ctx, finance.TxInput{
			AccountID: acc, Kind: "expense", Amount: "5.00", Description: "x", Date: "2026-03-01", Tags: tags,
		})
	}

	uidsOf := func(tx models.Transaction) []string {
		t.Helper()

		var uids []string
		if err := json.Unmarshal([]byte(tx.TagUIDs), &uids); err != nil {
			t.Fatalf("tag_uids = %q: %v", tx.TagUIDs, err)
		}

		return uids
	}

	t.Run("created inline, matched without regard to case, one uid each", func(t *testing.T) {
		tx, err := newTx([]string{"Trip", "trip", " Food "})
		if err != nil {
			t.Fatal(err)
		}

		tags, err := fin.Tags(ctx)
		if err != nil || len(tags) != 2 {
			t.Fatalf("tags = %+v, %v; want Food and Trip once each", tags, err)
		}

		want := []string{tags[0].UID, tags[1].UID}
		slices.Sort(want)

		if got := uidsOf(tx); !slices.Equal(got, want) {
			t.Fatalf("tag_uids = %v, want %v", got, want)
		}

		// A second transaction reuses the same tag, whatever the case.
		if _, err := newTx([]string{"TRIP"}); err != nil {
			t.Fatal(err)
		}

		if tags, _ := fin.Tags(ctx); len(tags) != 2 {
			t.Errorf("%d tags after reusing one, want 2", len(tags))
		}

		// The change log says the transaction's tag_uids changed, and a snapshot sends it decoded.
		snap, err := changes.Snapshot(ctx, fin.DB, ana.Workspace.ID)
		if err != nil {
			t.Fatal(err)
		}

		found := false

		for _, row := range snap.Rows["transaction"] {
			if row["uid"] == tx.UID {
				found = true

				if arr, ok := row["tag_uids"].([]any); !ok || len(arr) != 2 {
					t.Errorf("snapshot tag_uids = %#v, want an array of 2", row["tag_uids"])
				}
			}
		}

		if !found || len(snap.Rows["tag"]) != 2 {
			t.Errorf("snapshot has the transaction %v and %d tags; want true and 2", found, len(snap.Rows["tag"]))
		}
	})

	t.Run("names are unique per workspace without regard to case", func(t *testing.T) {
		if _, err := fin.CreateTag(ctx, "TRIP", ""); statusOf(err) != http.StatusConflict {
			t.Errorf("a second Trip = %v, want 409", err)
		}

		// Another workspace is another namespace.
		if _, err := fin.CreateTag(bobCtx, "Trip", "blue"); err != nil {
			t.Errorf("the same name in another workspace: %v", err)
		}

		for _, bad := range []string{"", "   ", "line\nbreak", string(make([]byte, 41))} {
			if _, err := fin.CreateTag(ctx, bad, ""); statusOf(err) != http.StatusUnprocessableEntity {
				t.Errorf("tag %q = %v, want 422", bad, err)
			}
		}

		if _, err := fin.CreateTag(ctx, "Colored", "#fff"); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("a hex color = %v, want 422", err)
		}
	})

	t.Run("at most 10 tags on a transaction, and a failed save leaves nothing", func(t *testing.T) {
		before := count[models.Transaction](t, app, ana)
		tagsBefore, _ := fin.Tags(ctx)

		var many []string
		for i := range 11 {
			many = append(many, fmt.Sprintf("many-%d", i))
		}

		if _, err := newTx(many); statusOf(err) != http.StatusUnprocessableEntity {
			t.Fatalf("11 tags = %v, want 422", err)
		}

		if _, err := newTx([]string{"fine", "bad\ntag"}); statusOf(err) != http.StatusUnprocessableEntity {
			t.Fatalf("a bad name = %v, want 422", err)
		}

		tagsAfter, _ := fin.Tags(ctx)
		if after := count[models.Transaction](t, app, ana); after != before || len(tagsAfter) != len(tagsBefore) {
			t.Errorf("transactions %d -> %d, tags %d -> %d; a refused save must change nothing", before, after, len(tagsBefore), len(tagsAfter))
		}

		if _, err := newTx(many[:10]); err != nil {
			t.Errorf("exactly 10 tags: %v", err)
		}
	})

	t.Run("attach and detach", func(t *testing.T) {
		tx, err := newTx(nil)
		if err != nil {
			t.Fatal(err)
		}

		tag, err := fin.CreateTag(ctx, "Solo", "teal")
		if err != nil {
			t.Fatal(err)
		}

		for range 2 {
			if err := fin.AttachTag(ctx, tx.ID, tag.ID); err != nil {
				t.Fatal(err)
			}
		}

		row := txByID(t, app, ana, tx.ID)
		if uids := uidsOf(row); len(uids) != 1 || uids[0] != tag.UID {
			t.Fatalf("tag_uids = %v, want [%s]", uids, tag.UID)
		}

		if err := fin.DetachTag(ctx, tx.ID, tag.ID); err != nil {
			t.Fatal(err)
		}

		if uids := uidsOf(txByID(t, app, ana, tx.ID)); len(uids) != 0 {
			t.Errorf("tag_uids = %v after detach, want none", uids)
		}

		// A tag of another workspace can never be attached, by id or by name.
		foreign, err := fin.CreateTag(bobCtx, "Bob only", "")
		if err != nil {
			t.Fatal(err)
		}

		if err := fin.AttachTag(ctx, tx.ID, foreign.ID); err == nil {
			t.Error("attached a tag of another workspace")
		}

		if err := fin.AttachTag(bobCtx, tx.ID, foreign.ID); err == nil {
			t.Error("attached to a transaction of another workspace")
		}

		if tags, _ := fin.Tags(ctx); slices.ContainsFunc(tags, func(g models.Tag) bool { return g.Name == "Bob only" }) {
			t.Error("a tag of another workspace is listed")
		}

		// A transfer leg carries no tags.
		other, err := fin.CreateAccount(ctx, "Second", "savings", "0")
		if err != nil {
			t.Fatal(err)
		}

		tr, err := fin.CreateTransfer(ctx, finance.TransferInput{FromAccountID: acc, ToAccountID: other.ID, Amount: "3.00", Date: "2026-03-02"})
		if err != nil {
			t.Fatal(err)
		}

		if err := fin.AttachTag(ctx, tr.From.ID, tag.ID); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("tag on a transfer leg = %v, want 422", err)
		}
	})

	t.Run("a rename keeps the tag on its transactions, a delete takes it off", func(t *testing.T) {
		tx, err := newTx([]string{"Keep", "Drop"})
		if err != nil {
			t.Fatal(err)
		}

		tags, _ := fin.Tags(ctx)

		var drop models.Tag

		for _, g := range tags {
			if g.Name == "Drop" {
				drop = g
			}
		}

		renamed, err := fin.UpdateTag(ctx, drop.ID, "Dropped", "red")
		if err != nil || renamed.UID != drop.UID || renamed.Name != "Dropped" {
			t.Fatalf("rename = %+v, %v", renamed, err)
		}

		if len(uidsOf(txByID(t, app, ana, tx.ID))) != 2 {
			t.Fatal("a rename must keep the tag on the transaction")
		}

		if _, err := fin.UpdateTag(ctx, drop.ID, "keep", ""); statusOf(err) != http.StatusConflict {
			t.Errorf("rename onto another tag's name = %v, want 409", err)
		}

		if err := fin.DeleteTag(ctx, drop.ID); err != nil {
			t.Fatal(err)
		}

		if uids := uidsOf(txByID(t, app, ana, tx.ID)); len(uids) != 1 || uids[0] == drop.UID {
			t.Errorf("tag_uids = %v after the delete, want only Keep", uids)
		}

		// The name is free again.
		if _, err := fin.CreateTag(ctx, "Dropped", ""); err != nil {
			t.Errorf("reusing a deleted name: %v", err)
		}
	})

	t.Run("at most 200 tags per workspace", func(t *testing.T) {
		carl := signUp(t, app, "Carl", "carl@example.com")
		cctx := workspaces.WithTenant(t.Context(), carl.Workspace.ID)

		for i := range models.MaxTagsPerWorkspace {
			if _, err := fin.CreateTag(cctx, fmt.Sprintf("t%d", i), ""); err != nil {
				t.Fatalf("tag %d: %v", i, err)
			}
		}

		if _, err := fin.CreateTag(cctx, "one too many", ""); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("the 201st tag = %v, want 422", err)
		}

		cacc := firstAccount(t, app, carl)
		if _, err := fin.CreateTransaction(cctx, finance.TxInput{
			AccountID: cacc, Kind: "expense", Amount: "1.00", Date: "2026-03-01", Tags: []string{"inline too many"},
		}); statusOf(err) != http.StatusUnprocessableEntity {
			t.Errorf("an inline 201st tag = %v, want 422", err)
		}
	})
}

func txByID(t *testing.T, app *vukemtest.App, w who, id int64) models.Transaction {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	row, err := workspaces.Scoped[models.Transaction](workspaces.WithTenant(t.Context(), w.Workspace.ID), db).Where("id", id).First(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return row
}
