package privacy_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func statusOf(err error) int {
	if he, ok := errors.AsType[*errs.HTTPError](err); ok {
		return he.Status
	}

	return 0
}

func (w *world) reload(t *testing.T) models.Workspace {
	t.Helper()

	db, err := w.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	ws, err := orm.QueryOf[models.Workspace](db).Find(t.Context(), w.space.ID)
	if err != nil {
		t.Fatal(err)
	}

	return ws
}

func TestWorkspaceDeletion(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		role       string
		wantStatus int
	}{
		{models.RoleOwner, 0},
		{models.RoleAdmin, http.StatusForbidden},
		{models.RoleMember, http.StatusForbidden},
		{models.RoleViewer, http.StatusForbidden},
	} {
		t.Run(tc.role, func(t *testing.T) {
			t.Parallel()

			app := newApp(t)
			w := newWorld(t, app, "owner-"+tc.role+"@example.com")
			actor := w.owner

			if tc.role != models.RoleOwner {
				actor = newUser(t, app, "Other", tc.role+"@example.com")
				w.joinAs(t, actor.ID, tc.role)
			}

			ctx := workspaces.WithTenant(t.Context(), w.space.ID)

			err := w.svc.RequestWorkspaceDeletion(ctx, actor, w.space.Name)
			if got := statusOf(err); got != tc.wantStatus {
				t.Fatalf("request as %s = %v (status %d), want status %d", tc.role, err, got, tc.wantStatus)
			}

			ws := w.reload(t)
			if tc.wantStatus != 0 {
				if ws.PendingDeletion() || ws.PurgeAt.Valid {
					t.Fatalf("a %s changed the workspace: %+v", tc.role, ws)
				}

				// Nor can they cancel one the owner asked for.
				if err := w.svc.RequestWorkspaceDeletion(ctx, w.owner, w.space.Name); err != nil {
					t.Fatal(err)
				}

				if got := statusOf(w.svc.CancelWorkspaceDeletion(ctx, actor)); got != http.StatusForbidden {
					t.Fatalf("cancel as %s: status %d, want 403", tc.role, got)
				}

				if !w.reload(t).PendingDeletion() {
					t.Fatalf("a %s cancelled the deletion", tc.role)
				}

				return
			}

			if !ws.PendingDeletion() || !ws.PurgeAt.Valid {
				t.Fatalf("the workspace is not pending: %+v", ws)
			}

			if d := ws.PurgeAt.V.Sub(ws.DeletionRequestedAt.V); d < 29*24*60*60*1e9 || d > 31*24*60*60*1e9 {
				t.Errorf("grace period = %v, want 30 days", d)
			}

			if statusOf(w.svc.RequestWorkspaceDeletion(ctx, actor, w.space.Name)) != http.StatusConflict {
				t.Error("a second request must be refused")
			}

			if err := w.svc.CancelWorkspaceDeletion(ctx, actor); err != nil {
				t.Fatal(err)
			}

			if ws := w.reload(t); ws.PendingDeletion() || ws.PurgeAt.Valid {
				t.Fatalf("cancel left the workspace pending: %+v", ws)
			}

			if statusOf(w.svc.CancelWorkspaceDeletion(ctx, actor)) != http.StatusConflict {
				t.Error("cancelling an active workspace must be refused")
			}

			events, err := w.svc.Audit(t.Context(), w.svc.Ref("workspace", w.space.ID))
			if err != nil || len(events) != 2 || events[0].Event != "workspace.deletion_requested" ||
				events[1].Event != "workspace.deletion_cancelled" {
				t.Errorf("audit = %+v, %v", events, err)
			}
		})
	}
}

func TestWorkspaceDeletionNeedsTheExactName(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")
	ctx := workspaces.WithTenant(t.Context(), w.space.ID)

	for _, name := range []string{"", "home of ana@example.com", w.space.Name + " ", "Other"} {
		if got := statusOf(w.svc.RequestWorkspaceDeletion(ctx, w.owner, name)); got != http.StatusUnprocessableEntity {
			t.Errorf("name %q: status %d, want 422", name, got)
		}
	}

	if w.reload(t).PendingDeletion() {
		t.Fatal("a wrong name scheduled the deletion")
	}
}

func TestWorkspaceDeletionIsPerWorkspace(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	a := newWorld(t, app, "ana@example.com")
	b := newWorld(t, app, "bob@example.com")

	// Ana owns a workspace, not Bob's: Bob's workspace answers 404 (not a member).
	ctx := workspaces.WithTenant(t.Context(), b.space.ID)
	if got := statusOf(a.svc.RequestWorkspaceDeletion(ctx, a.owner, b.space.Name)); got != http.StatusNotFound {
		t.Fatalf("a stranger got status %d, want 404", got)
	}

	if b.reload(t).PendingDeletion() {
		t.Fatal("a stranger scheduled the deletion of another workspace")
	}
}

func TestWorkspaceDeletionTellsTheMembersAndSurvivesASuppressedAddress(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")
	bob := newUser(t, app, "Bob", "bob@example.com")
	w.joinAs(t, bob.ID, models.RoleMember)

	carl := newUser(t, app, "Carl", "carl@example.com")
	w.joinAs(t, carl.ID, models.RoleViewer)

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	// A bounced address is skipped quietly: the deletion still stands.
	if err := orm.Create(t.Context(), db, &models.EmailSuppression{Address: "carl@example.com", Reason: "bounce"}); err != nil {
		t.Fatal(err)
	}

	ctx := workspaces.WithTenant(t.Context(), w.space.ID)
	if err := w.svc.RequestWorkspaceDeletion(ctx, w.owner, w.space.Name); err != nil {
		t.Fatal(err)
	}

	if !w.reload(t).PendingDeletion() {
		t.Fatal("the workspace is not pending")
	}

	for _, to := range []string{"ana@example.com", "bob@example.com"} {
		sent := app.Mails().SentTo(to)
		if len(sent) != 1 || sent[0].Subject != w.space.Name+" will be deleted" {
			t.Errorf("mail to %s = %+v", to, sent)
		}
	}

	if n := len(app.Mails().SentTo("carl@example.com")); n != 0 {
		t.Errorf("a suppressed address got %d mails", n)
	}
}
