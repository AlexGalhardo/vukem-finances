package privacy_test

import (
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func (w *world) withAccess(t *testing.T) {
	t.Helper()

	acc, err := accounts.New(w.app.App)
	if err != nil {
		t.Fatal(err)
	}

	w.svc.Access = acc
}

func (w *world) user(t *testing.T, id int64) models.User {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u, err := orm.QueryOf[models.User](db).Find(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}

	return u
}

func (w *world) count(t *testing.T, query string, args ...any) int {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.QueryRowContext(t.Context(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

func TestAccountDeletionRefusesTheOnlyOwnerOfASharedWorkspace(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")
	w.withAccess(t)

	bob := newUser(t, app, "Bob", "bob@example.com")
	w.joinAs(t, bob.ID, models.RoleMember)

	before := w.user(t, w.owner.ID)

	err := w.svc.RequestAccountDeletion(t.Context(), w.owner)
	if statusOf(err) != http.StatusConflict {
		t.Fatalf("err = %v (status %d), want 409", err, statusOf(err))
	}

	// Nothing changed: not the account, not the workspace, not a credential.
	after := w.user(t, w.owner.ID)
	if after.PendingDeletion() || after.SessionVersion != before.SessionVersion {
		t.Fatalf("a refused request changed the account: %+v", after)
	}

	if w.reload(t).PendingDeletion() {
		t.Fatal("a refused request put the workspace in its grace period")
	}

	if n := w.count(t, "SELECT COUNT(*) FROM access_tokens WHERE revoked_at IS NULL"); n != 1 {
		t.Fatalf("%d live tokens after a refusal, want 1", n)
	}

	if n := w.count(t, "SELECT COUNT(*) FROM privacy_audit"); n != 0 {
		t.Fatalf("%d audit rows after a refusal", n)
	}

	// Transfer the ownership: now they may go, and the workspace stays for the others.
	ctx := workspaces.WithTenant(t.Context(), w.space.ID)
	if err := w.ws.TransferOwnership(ctx, w.owner, bob.ID); err != nil {
		t.Fatal(err)
	}

	if err := w.svc.RequestAccountDeletion(t.Context(), w.owner); err != nil {
		t.Fatal(err)
	}

	if w.reload(t).PendingDeletion() {
		t.Fatal("a workspace other people use went with the account")
	}

	if !w.user(t, w.owner.ID).PendingDeletion() {
		t.Fatal("the account is not pending")
	}
}

func TestAccountDeletionTakesTheWorkspacesOfWhichThePersonIsTheOnlyMember(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")
	w.withAccess(t)

	before := w.user(t, w.owner.ID)

	if err := w.svc.RequestAccountDeletion(t.Context(), w.owner); err != nil {
		t.Fatal(err)
	}

	u := w.user(t, w.owner.ID)
	ws := w.reload(t)

	if !u.PendingDeletion() || !ws.PendingDeletion() || !u.PurgeAt.V.Equal(ws.PurgeAt.V) {
		t.Fatalf("account and workspace must enter the grace period together: %+v %+v", u, ws)
	}

	// Every credential ended at once.
	if u.SessionVersion == before.SessionVersion {
		t.Error("the session version was not bumped: sessions still work")
	}

	if n := w.count(t, "SELECT COUNT(*) FROM access_tokens WHERE revoked_at IS NULL"); n != 0 {
		t.Errorf("%d tokens still work", n)
	}

	if n := w.count(t, "SELECT COUNT(*) FROM remember_tokens"); n != 0 {
		t.Errorf("%d remember-me tokens remain", n)
	}

	events, err := w.svc.Audit(t.Context(), w.svc.Ref("user", w.owner.ID))
	if err != nil || len(events) != 1 || events[0].Event != "account.deletion_requested" {
		t.Errorf("audit = %+v, %v", events, err)
	}

	if sent := app.Mails().SentTo("ana@example.com"); len(sent) != 1 || sent[0].Subject != "Your account will be deleted" {
		t.Errorf("mails = %+v", sent)
	}

	if statusOf(w.svc.RequestAccountDeletion(t.Context(), w.owner)) != http.StatusConflict {
		t.Error("a second request must be refused")
	}
}

func TestAccountDeletionCancelBringsBackWhatWentWithIt(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")
	w.withAccess(t)

	// A second workspace the person asked to delete on its own, before the account.
	ws2, err := w.ws.Create(t.Context(), w.owner, "Side project", "USD")
	if err != nil {
		t.Fatal(err)
	}

	if err := w.svc.RequestWorkspaceDeletion(workspaces.WithTenant(t.Context(), ws2.ID), w.owner, "Side project"); err != nil {
		t.Fatal(err)
	}

	if err := w.svc.RequestAccountDeletion(t.Context(), w.owner); err != nil {
		t.Fatal(err)
	}

	if err := w.svc.CancelAccountDeletion(t.Context(), w.owner); err != nil {
		t.Fatal(err)
	}

	if w.user(t, w.owner.ID).PendingDeletion() || w.reload(t).PendingDeletion() {
		t.Fatal("cancel left the account or its workspace pending")
	}

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	side, err := orm.QueryOf[models.Workspace](db).Find(t.Context(), ws2.ID)
	if err != nil || !side.PendingDeletion() {
		t.Fatalf("the workspace deleted on its own must keep waiting: %+v %v", side, err)
	}

	if statusOf(w.svc.CancelAccountDeletion(t.Context(), w.owner)) != http.StatusConflict {
		t.Error("cancelling an active account must be refused")
	}
}

func TestAccountDeletionIgnoresAWorkspaceAlreadyWaiting(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")
	w.withAccess(t)

	bob := newUser(t, app, "Bob", "bob@example.com")
	w.joinAs(t, bob.ID, models.RoleMember)

	// The owner already asked to delete the shared workspace: it does not block the account.
	ctx := workspaces.WithTenant(t.Context(), w.space.ID)
	if err := w.svc.RequestWorkspaceDeletion(ctx, w.owner, w.space.Name); err != nil {
		t.Fatal(err)
	}

	if err := w.svc.RequestAccountDeletion(t.Context(), w.owner); err != nil {
		t.Fatalf("a workspace already waiting blocked the account: %v", err)
	}
}
