package privacy_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/privacy"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Seeded secret values. Tests scan an export for them.
const (
	secretHash     = "hash-of-the-password-xyz"
	secretTwoFA    = "JBSWY3DPEHPK3PXP-secret"
	secretRecovery = "recovery-code-abc-123"
	secretAIKey    = "sealed-ai-key-ciphertext-789"
)

type world struct {
	app   *vukemtest.App
	svc   *privacy.Service
	ws    *workspaces.Service
	fin   *finance.Service
	owner models.User
	space models.Workspace
	token string
}

// newWorld makes an owner, a workspace with a transaction, and credentials of
// every kind the export and the purge must handle.
func newWorld(t *testing.T, app *vukemtest.App, email string) *world {
	t.Helper()

	ws := workspaceSvc(t, app)

	fin, err := finance.New(app.App, ws)
	if err != nil {
		t.Fatal(err)
	}

	svc := newService(t, app)
	svc.Fin = fin

	owner := newUser(t, app, "Owner "+email, email)

	space, err := ws.Create(t.Context(), owner, "Home of "+email, "USD")
	if err != nil {
		t.Fatal(err)
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.ExecContext(t.Context(),
		"UPDATE users SET password = ?, two_factor_secret = ?, two_factor_recovery_codes = ?, two_factor_confirmed_at = ? WHERE id = ?",
		secretHash, secretTwoFA, secretRecovery, time.Now().UTC(), owner.ID); err != nil {
		t.Fatal(err)
	}

	ctx := workspaces.WithTenant(t.Context(), space.ID)

	acct, err := workspaces.Scoped[models.Account](ctx, db).First(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := fin.CreateTransaction(ctx, finance.TxInput{
		AccountID: acct.ID, Kind: models.KindExpense, Amount: "12.34", Description: "Coffee", Date: "2026-11-02",
	}); err != nil {
		t.Fatal(err)
	}

	if err := orm.Create(ctx, db, &models.AICredential{
		Provider: "openai", Label: "main", KeyCiphertext: secretAIKey, KeyHint: "...1234", CreatedBy: owner.ID,
	}); err != nil {
		t.Fatal(err)
	}

	acc, err := accounts.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	plain, _, err := acc.Tokens.CreateWith(t.Context(), strconv.FormatInt(owner.ID, 10), "ci", []string{"accounts:read"},
		time.Hour, auth.CreateOptions{Resource: strconv.FormatInt(space.ID, 10)})
	if err != nil {
		t.Fatal(err)
	}

	return &world{app: app, svc: svc, ws: ws, fin: fin, owner: owner, space: space, token: plain}
}

// joinAs makes userID a member of the workspace of w.
func (w *world) joinAs(t *testing.T, userID int64, role string) {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	m := models.Membership{WorkspaceID: w.space.ID, UserID: userID, Role: role}
	if err := orm.Create(t.Context(), db, &m); err != nil {
		t.Fatal(err)
	}
}
