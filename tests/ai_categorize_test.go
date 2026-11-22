package tests

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/queue"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/ai/fake"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/jobs"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// fakeQueueApp is an app whose jobs are recorded, not run.
func fakeQueueApp(t *testing.T) (*vukemtest.App, *queue.Fake) {
	t.Helper()

	q := queue.NewFake()
	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:        map[string]string{"ADMIN_EMAILS": "boss@example.com", "APP_EDITION": "saas"},
		AppOptions: []vukem.Option{vukem.WithQueueDriver(q)},
	})

	return app, q
}

// aiReady saves a key for w, turns AI on with the features and points the
// shared client at a fake provider: nothing here reaches a real one.
func aiReady(t *testing.T, app *vukemtest.App, w who, autoApply bool, features ...string) *fake.Provider {
	t.Helper()

	svc, err := ai.Shared(app.App)
	if err != nil {
		t.Fatal(err)
	}

	provider := &fake.Provider{}
	svc.Client.Factory = provider.Factory()

	ctx := wsCtx(t, w)

	if _, err := svc.Vault.Save(ctx, w.User.ID, ai.SaveInput{Provider: ai.KindAnthropic, Key: ai.NewKey(aiKey)}); err != nil {
		t.Fatal(err)
	}

	on := map[string]bool{}
	for _, f := range features {
		on[f] = true
	}

	if _, err := svc.Settings.Update(ctx, w.User.ID, ai.UpdateInput{
		Enabled: true, Consent: true, MonthlyCallLimit: 1000, Features: on, AutoApply: autoApply,
	}); err != nil {
		t.Fatal(err)
	}

	return provider
}

// answerEvery makes provider answer, for every transaction it is sent, with the uid and confidence.
func answerEvery(p *fake.Provider, uid string, conf float64) {
	p.Reply = func(req ai.Request) (ai.Response, error) {
		var doc struct {
			Transactions []struct{ ID string } `json:"transactions"`
		}

		_ = json.Unmarshal([]byte(req.Messages[0].Content), &doc)

		items := make([]string, 0, len(doc.Transactions))
		for _, tx := range doc.Transactions {
			b, _ := json.Marshal(map[string]any{"id": tx.ID, "category_uid": uid, "confidence": conf})
			items = append(items, string(b))
		}

		return ai.Response{Text: `{"items":[` + strings.Join(items, ",") + `]}`, Model: "fake-1", InputTokens: 1, OutputTokens: 1}, nil
	}
}

func suggestionRows(t *testing.T, app *vukemtest.App, w who) []models.Suggestion {
	t.Helper()

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	rows, err := workspaces.Scoped[models.Suggestion](wsCtx(t, w), db).OrderBy("id", orm.Asc).Get(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	return rows
}

func transactionByUID(t *testing.T, app *vukemtest.App, w who, uid string) models.Transaction {
	t.Helper()

	return firstOfWhere[models.Transaction](t, app, w, "uid", uid)
}

func TestCategorizeJob(t *testing.T) {
	t.Parallel()

	app, fq := fakeQueueApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	bob := signUp(t, app, "Bob", "bob@example.com")
	provider := aiReady(t, app, ana, false, ai.FeatureCategorize)
	groceries := firstOf[models.Category](t, app, ana, "Groceries")
	answerEvery(provider, groceries.UID, 0.8)

	// A transaction created without a category asks for a run; one with a category does not.
	addDescribed(t, ana, firstAccount(t, app, ana), 0, "Padaria Alfa", "10.00")
	addDescribed(t, ana, firstAccount(t, app, ana), groceries.ID, "Mercado Beta", "11.00")
	addDescribed(t, bob, firstAccount(t, app, bob), 0, "Posto Gama", "12.00")

	asked := queue.Dispatched[finance.AICategorize](fq)
	if len(asked) != 2 {
		t.Fatalf("jobs asked = %+v, want one per uncategorized creation", asked)
	}

	for _, job := range asked {
		if job.WorkspaceID != ana.Workspace.ID && job.WorkspaceID != bob.Workspace.ID {
			t.Errorf("job for an unknown workspace: %+v", job)
		}
	}

	// Run Ana's job: one suggestion for her own transaction, none for Bob's, nothing applied.
	for _, job := range asked {
		if job.WorkspaceID == ana.Workspace.ID {
			if err := jobs.HandleAICategorize(app.App)(t.Context(), job); err != nil {
				t.Fatal(err)
			}
		}
	}

	got := suggestionRows(t, app, ana)
	if len(got) != 1 || got[0].CategoryUID != groceries.UID || got[0].Status != models.SuggestionPending || got[0].Source != models.SuggestionAI {
		t.Fatalf("suggestions = %+v", got)
	}

	if tx := transactionByUID(t, app, ana, got[0].TransactionUID); tx.CategoryID.Valid {
		t.Error("a suggestion must not change the transaction")
	}

	if len(suggestionRows(t, app, bob)) != 0 || provider.Calls() != 1 {
		t.Errorf("bob's workspace was touched or the batch was split: %d calls", provider.Calls())
	}

	// Running it again makes no second call and no second suggestion.
	if err := jobs.HandleAICategorize(app.App)(t.Context(), asked[0]); err != nil {
		t.Fatal(err)
	}

	if len(suggestionRows(t, app, ana)) != 1 || provider.Calls() != 1 {
		t.Error("a second run repeated work")
	}
}

func TestCategorizeJobRunsAfterAnImport(t *testing.T) {
	t.Parallel()

	app, fq := fakeQueueApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	makePro(t, app, ana)

	acc := firstAccount(t, app, ana)
	csv := "Date,Description,Amount\n2026-11-02,Padaria Sao Joao,-12.50\n2026-11-03,Cinema,-30.00\n"
	runImport(t, app, ana, acc, "s.csv", []byte(csv), csvMapping)

	asked := queue.Dispatched[finance.AICategorize](fq)

	var imported *finance.AICategorize

	for i := range asked {
		if asked[i].ImportID > 0 {
			imported = &asked[i]
		}
	}

	if imported == nil || imported.WorkspaceID != ana.Workspace.ID {
		t.Fatalf("no categorization asked for the import: %+v", asked)
	}

	// AI is off: the run uses the memory only and makes no call.
	if err := jobs.HandleAICategorize(app.App)(t.Context(), *imported); err != nil {
		t.Fatal(err)
	}

	if len(suggestionRows(t, app, ana)) != 0 {
		t.Error("with no memory and AI off there is nothing to suggest")
	}
}

func TestAutoApplyOffByDefault(t *testing.T) {
	t.Parallel()

	app, _ := fakeQueueApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	provider := aiReady(t, app, ana, false, ai.FeatureCategorize)
	groceries := firstOf[models.Category](t, app, ana, "Groceries")
	answerEvery(provider, groceries.UID, 0.99)

	if set := aiSettings(t, app, ana); set.AutoApplySuggestions {
		t.Fatal("auto-apply must be off until the owner turns it on")
	}

	addTransaction(t, ana, firstAccount(t, app, ana), 0, "expense", "10.00", today())

	if err := jobs.HandleAICategorize(app.App)(t.Context(), jobs.AICategorize{WorkspaceID: ana.Workspace.ID}); err != nil {
		t.Fatal(err)
	}

	for _, tx := range transactionsOf(t, app, ana) {
		if tx.CategoryID.Valid {
			t.Errorf("transaction %d was categorized without a review", tx.ID)
		}
	}

	if got := suggestionRows(t, app, ana); len(got) != 1 || got[0].Status != models.SuggestionPending {
		t.Errorf("suggestions = %+v", got)
	}
}

func TestUndoAutoApplied(t *testing.T) {
	t.Parallel()

	app, _ := fakeQueueApp(t)
	ana := signUp(t, app, "Ana", "ana@example.com")
	provider := aiReady(t, app, ana, true, ai.FeatureCategorize)
	groceries := firstOf[models.Category](t, app, ana, "Groceries")
	answerEvery(provider, groceries.UID, 0.95)

	addTransaction(t, ana, firstAccount(t, app, ana), 0, "expense", "10.00", today())

	if err := jobs.HandleAICategorize(app.App)(t.Context(), jobs.AICategorize{WorkspaceID: ana.Workspace.ID}); err != nil {
		t.Fatal(err)
	}

	rows := suggestionRows(t, app, ana)
	if len(rows) != 1 || rows[0].Status != models.SuggestionAccepted || !rows[0].AppliedAt.Valid {
		t.Fatalf("suggestions = %+v", rows)
	}

	if tx := transactionByUID(t, app, ana, rows[0].TransactionUID); !tx.CategoryID.Valid || tx.CategoryID.V != groceries.ID {
		t.Fatalf("the confident suggestion was not applied: %+v", tx.CategoryID)
	}

	ana.Client.PostForm("/suggestions/"+idStr(rows[0].ID)+"/undo", nil).AssertRedirect("/suggestions")

	if tx := transactionByUID(t, app, ana, rows[0].TransactionUID); tx.CategoryID.Valid {
		t.Error("undo must take the category back")
	}
}
