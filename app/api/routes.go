package api

import (
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// MeIn is the input of me.show: none.
type MeIn struct{}

func (a *API) me(c *web.Ctx, _ MeIn) (MeOut, error) {
	ctx := c.Context()

	p, _ := PrincipalFrom(ctx)

	ws, err := a.WS.Current(ctx)
	if err != nil {
		return MeOut{}, err
	}

	out := MeOut{
		TokenName: p.Token.Name, Scopes: p.Token.Abilities, Role: p.Role,
		Workspace: WorkspaceOut{ID: ws.ID, Name: ws.Name, Currency: ws.Currency, Timezone: ws.Timezone},
	}

	if p.Token.ExpiresAt != nil {
		s := stamp(*p.Token.ExpiresAt)
		out.ExpiresAt = &s
	}

	return out, nil
}

// UIDIn is the input of an operation on one row.
type UIDIn = operations.UIDIn

// Empty is the output of an operation that answers 204.
type Empty = operations.Empty

// Routes registers the operations of the API on g.
func (a *API) Routes(g *router.Group, idem router.Middleware) {
	register(a, g, idem, "me.show", a.me)

	route(a, g, a.Ops.Accounts.List)
	route(a, g, a.Ops.Accounts.Show)
	route(a, g, a.Ops.Accounts.Create)
	route(a, g, a.Ops.Accounts.Update)
	route(a, g, a.Ops.Accounts.Archive)

	route(a, g, a.Ops.Categories.List)
	route(a, g, a.Ops.Categories.Show)
	route(a, g, a.Ops.Categories.Create)
	route(a, g, a.Ops.Categories.Update)
	route(a, g, a.Ops.Categories.Archive)

	route(a, g, a.Ops.Tags.List)
	route(a, g, a.Ops.Tags.Create)
	route(a, g, a.Ops.Tags.Update)
	route(a, g, a.Ops.Tags.Delete)

	routeWith(a, g, a.Ops.Transactions.List, pageLinks)
	route(a, g, a.Ops.Transactions.Show)
	route(a, g, a.Ops.Transactions.Create)
	route(a, g, a.Ops.Transactions.Update)
	route(a, g, a.Ops.Transactions.Delete)

	route(a, g, a.Ops.Transactions.Transfer)
	route(a, g, a.Ops.Transactions.Untransfer)

	register(a, g, idem, "budgets.list", a.budgetsList)
	register(a, g, idem, "budgets.set", a.budgetsSet)
	register(a, g, idem, "budgets.delete", a.budgetsDelete)

	register(a, g, idem, "summary.show", a.summaryShow)

	a.syncRoutes(g, idem)
}
