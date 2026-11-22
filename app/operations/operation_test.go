package operations_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

type echoIn struct {
	Name string `json:"name" validate:"required,max=5"`
}

type echoOut struct {
	Name      string `json:"name"`
	Workspace int64  `json:"workspace"`
}

func echo(reg *operations.Registry, d operations.Operation[echoIn, echoOut]) *operations.Op[echoIn, echoOut] {
	if d.Run == nil {
		d.Run = func(ctx context.Context, a operations.Actor, in echoIn) (echoOut, error) {
			ws, _ := workspaces.TenantID(ctx)

			return echoOut{Name: in.Name, Workspace: ws}, nil
		}
	}

	return operations.Define(reg, d)
}

func user(id int64) models.User {
	var u models.User
	u.ID = id

	return u
}

func actor(role string, scopes ...string) operations.Actor {
	return operations.NewActor(operations.SurfaceREST, user(7), 3, role, scopes, "tok")
}

func TestInvokeRefusals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		edition config.Edition
		op      operations.Operation[echoIn, echoOut]
		actor   operations.Actor
		in      echoIn
		code    string
	}{
		{"allowed", config.EditionSaaS, operations.Operation[echoIn, echoOut]{Name: "t.read", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead},
			actor(models.RoleViewer, operations.ScopeAccountsRead), echoIn{Name: "a"}, ""},
		{"no actor", config.EditionSaaS, operations.Operation[echoIn, echoOut]{Name: "t.read", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead},
			operations.Actor{}, echoIn{Name: "a"}, operations.CodeForbidden},
		{"missing scope", config.EditionSaaS, operations.Operation[echoIn, echoOut]{Name: "t.read", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead},
			actor(models.RoleOwner, operations.ScopeTagsRead), echoIn{Name: "a"}, operations.CodeMissingScope},
		{"role below the ability", config.EditionSaaS, operations.Operation[echoIn, echoOut]{Name: "t.write", Scope: operations.ScopeAccountsWrite, Kind: operations.KindWrite},
			actor(models.RoleMember, operations.ScopeAccountsWrite), echoIn{Name: "a"}, operations.CodeForbidden},
		{"viewer cannot record", config.EditionSaaS, operations.Operation[echoIn, echoOut]{Name: "t.rec", Scope: operations.ScopeTransactionsWrite, Kind: operations.KindWrite},
			actor(models.RoleViewer, operations.ScopeTransactionsWrite), echoIn{Name: "a"}, operations.CodeForbidden},
		{"ability override", config.EditionSaaS, operations.Operation[echoIn, echoOut]{Name: "t.export", Scope: operations.ScopeExportRead, Ability: workspaces.AbilityManageData, Kind: operations.KindRead},
			actor(models.RoleMember, operations.ScopeExportRead), echoIn{Name: "a"}, operations.CodeForbidden},
		{"wrong edition", config.EditionPersonal, operations.Operation[echoIn, echoOut]{Name: "t.saas", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead, Editions: []config.Edition{config.EditionSaaS}},
			actor(models.RoleOwner, operations.ScopeAccountsRead), echoIn{Name: "a"}, operations.CodeNotFound},
		{"right edition", config.EditionSaaS, operations.Operation[echoIn, echoOut]{Name: "t.saas", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead, Editions: []config.Edition{config.EditionSaaS}},
			actor(models.RoleOwner, operations.ScopeAccountsRead), echoIn{Name: "a"}, ""},
		{"invalid input", config.EditionSaaS, operations.Operation[echoIn, echoOut]{Name: "t.read", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead},
			actor(models.RoleOwner, operations.ScopeAccountsRead), echoIn{Name: "too long name"}, operations.CodeValidation},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			reg := operations.NewRegistry(operations.Config{Edition: tc.edition})
			ran := false
			run := func(ctx context.Context, _ operations.Actor, in echoIn) (echoOut, error) {
				ran = true
				ws, _ := workspaces.TenantID(ctx)

				return echoOut{Name: in.Name, Workspace: ws}, nil
			}
			tc.op.Run = run

			res, err := operations.Invoke(t.Context(), echo(reg, tc.op), tc.actor, tc.in)

			if tc.code == "" {
				if err != nil || !ran || res.Value.Workspace != 3 {
					t.Fatalf("err = %v, ran = %v, out = %+v", err, ran, res.Value)
				}

				return
			}

			if ran {
				t.Error("Run was called on a refused call")
			}

			if !operations.IsCode(err, tc.code) {
				t.Errorf("err = %v, want code %s", err, tc.code)
			}
		})
	}
}

func TestErrorsKeepTheirStatusForREST(t *testing.T) {
	t.Parallel()

	reg := operations.NewRegistry(operations.Config{Edition: config.EditionSaaS})
	op := echo(reg, operations.Operation[echoIn, echoOut]{
		Name: "t.fail", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead,
		Run: func(context.Context, operations.Actor, echoIn) (echoOut, error) {
			return echoOut{}, errs.New(http.StatusPaymentRequired, "Upgrade.")
		},
	})

	_, err := operations.Invoke(t.Context(), op, actor(models.RoleOwner, operations.ScopeAccountsRead), echoIn{Name: "a"})
	if !operations.IsCode(err, operations.CodePlanLimit) || errs.Status(err) != http.StatusPaymentRequired {
		t.Fatalf("err = %v, status %d", err, errs.Status(err))
	}

	internal := echo(reg, operations.Operation[echoIn, echoOut]{
		Name: "t.boom", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead,
		Run: func(context.Context, operations.Actor, echoIn) (echoOut, error) {
			return echoOut{}, errors.New("sql: secret table name")
		},
	})

	_, err = operations.Invoke(t.Context(), internal, actor(models.RoleOwner, operations.ScopeAccountsRead), echoIn{Name: "a"})

	var oe *operations.Error
	if !errors.As(err, &oe) || oe.Code != operations.CodeInternal || oe.Message == "" || oe.Message == "sql: secret table name" {
		t.Fatalf("err = %#v, want an internal error with a generic message", err)
	}
}

func TestDefineRejectsMalformedDeclarations(t *testing.T) {
	t.Parallel()

	reg := operations.NewRegistry(operations.Config{})
	ok := func(context.Context, operations.Actor, echoIn) (echoOut, error) { return echoOut{}, nil }

	for name, d := range map[string]operations.Operation[echoIn, echoOut]{
		"no scope":          {Name: "a.b", Kind: operations.KindRead, Run: ok},
		"no kind":           {Name: "a.b", Scope: operations.ScopeAccountsRead, Run: ok},
		"idempotent read":   {Name: "a.b", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead, Idempotent: true, Run: ok},
		"opt out no reason": {Name: "a.b", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead, OptOut: map[operations.Surface]string{operations.SurfaceCLI: ""}, Run: ok},
		"no run":            {Name: "a.b", Scope: operations.ScopeAccountsRead, Kind: operations.KindRead},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: Define did not panic", name)
				}
			}()

			operations.Define(reg, d)
		}()
	}
}
