package api

import (
	"net/http"
	"strings"

	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
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
type UIDIn struct {
	UID string `json:"uid" validate:"required,max=26" doc:"The uid of the row" example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
}

// AccountCreateIn is the body of accounts.create.
type AccountCreateIn struct {
	UID          string `json:"uid"           validate:"max=26"          doc:"Optional uid chosen by the client (a ULID); repeating the call returns the row already made"`
	Name         string `json:"name"          validate:"required,max=80" example:"Checking"`
	Kind         string `json:"kind"          validate:"required,max=20" doc:"checking, savings, cash, investment or wallet" example:"checking"`
	OpeningCents int64  `json:"opening_cents" doc:"Opening balance in cents" example:"100000"`
}

// AccountUpdateIn is the body of accounts.update; a field left out stays as it is.
type AccountUpdateIn struct {
	UID          string  `json:"uid"           validate:"required,max=26"`
	Name         *string `json:"name"          validate:"max=80"`
	Kind         *string `json:"kind"          validate:"max=20"`
	OpeningCents *int64  `json:"opening_cents"`
}

func (a *API) accountViews(c *web.Ctx) (map[int64]int64, string, error) {
	ctx := c.Context()

	ws, err := a.WS.Current(ctx)
	if err != nil {
		return nil, "", err
	}

	balances, err := a.Fin.AccountBalances(ctx, "")
	if err != nil {
		return nil, "", err
	}

	out := make(map[int64]int64, len(balances))
	for _, b := range balances {
		out[b.AccountID] = b.BalanceCents
	}

	return out, ws.Currency, nil
}

func (a *API) accountView(c *web.Ctx, x models.Account) (AccountOut, error) {
	balances, currency, err := a.accountViews(c)
	if err != nil {
		return AccountOut{}, err
	}

	return accountOut(x, balances[x.ID], currency), nil
}

func (a *API) accountsList(c *web.Ctx, _ MeIn) (List[AccountOut], error) {
	all, err := a.Fin.Accounts(c.Context())
	if err != nil {
		return List[AccountOut]{}, err
	}

	balances, currency, err := a.accountViews(c)
	if err != nil {
		return List[AccountOut]{}, err
	}

	data := make([]AccountOut, 0, len(all))
	for _, x := range all {
		data = append(data, accountOut(x, balances[x.ID], currency))
	}

	return List[AccountOut]{Data: data}, nil
}

func (a *API) accountsShow(c *web.Ctx, in UIDIn) (AccountOut, error) {
	x, err := a.accountByUID(c.Context(), in.UID)
	if err != nil {
		return AccountOut{}, err
	}

	return a.accountView(c, x)
}

func (a *API) accountsCreate(c *web.Ctx, in AccountCreateIn) (AccountOut, error) {
	ctx := c.Context()

	if err := a.checkUID(in.UID); err != nil {
		return AccountOut{}, err
	}

	if old, found, err := existing[models.Account](ctx, a, in.UID); err != nil || found {
		if err != nil {
			return AccountOut{}, err
		}

		if old.DeletedAt.Valid {
			return AccountOut{}, errUIDDeleted
		}

		if old.Name != strings.TrimSpace(in.Name) || old.Kind != in.Kind || old.OpeningCents != in.OpeningCents {
			return AccountOut{}, errUIDDiffers
		}

		v, err := a.accountView(c, old)
		if err != nil {
			return AccountOut{}, err
		}

		return AccountOut{}, c.JSON(http.StatusOK, v)
	}

	x, err := a.Fin.CreateAccountWithUID(ctx, in.UID, in.Name, in.Kind, centsText(in.OpeningCents))
	if err != nil {
		return AccountOut{}, err
	}

	return a.accountView(c, x)
}

func (a *API) accountsUpdate(c *web.Ctx, in AccountUpdateIn) (AccountOut, error) {
	ctx := c.Context()

	x, err := a.accountByUID(ctx, in.UID)
	if err != nil {
		return AccountOut{}, err
	}

	cur, err := a.accountView(c, x)
	if err != nil {
		return AccountOut{}, err
	}

	if err := checkVersion(c, x.Version, cur); err != nil {
		return AccountOut{}, err
	}

	name, kind, opening := x.Name, x.Kind, x.OpeningCents
	if in.Name != nil {
		name = *in.Name
	}

	if in.Kind != nil {
		kind = *in.Kind
	}

	if in.OpeningCents != nil {
		opening = *in.OpeningCents
	}

	saved, err := a.Fin.UpdateAccount(ctx, x.ID, name, kind, centsText(opening))
	if err != nil {
		return AccountOut{}, err
	}

	return a.accountView(c, saved)
}

func (a *API) accountsArchive(c *web.Ctx, in UIDIn) (AccountOut, error) {
	x, err := a.accountByUID(c.Context(), in.UID)
	if err != nil {
		return AccountOut{}, err
	}

	cur, err := a.accountView(c, x)
	if err != nil {
		return AccountOut{}, err
	}

	if err := checkVersion(c, x.Version, cur); err != nil {
		return AccountOut{}, err
	}

	saved, err := a.Fin.ArchiveAccount(c.Context(), x.ID)
	if err != nil {
		return AccountOut{}, err
	}

	return a.accountView(c, saved)
}

// Routes registers the operations of the API on g.
func (a *API) Routes(g *router.Group, idem router.Middleware) {
	register(g, idem, "me.show", a.me)

	register(g, idem, "accounts.list", a.accountsList)
	register(g, idem, "accounts.show", a.accountsShow)
	register(g, idem, "accounts.create", a.accountsCreate)
	register(g, idem, "accounts.update", a.accountsUpdate)
	register(g, idem, "accounts.archive", a.accountsArchive)

	register(g, idem, "categories.list", a.categoriesList)
	register(g, idem, "categories.show", a.categoriesShow)
	register(g, idem, "categories.create", a.categoriesCreate)
	register(g, idem, "categories.update", a.categoriesUpdate)
	register(g, idem, "categories.archive", a.categoriesArchive)

	register(g, idem, "tags.list", a.tagsList)
	register(g, idem, "tags.create", a.tagsCreate)
	register(g, idem, "tags.update", a.tagsUpdate)
	register(g, idem, "tags.delete", a.tagsDelete)

	register(g, idem, "transactions.list", a.transactionsList)
	register(g, idem, "transactions.show", a.transactionsShow)
	register(g, idem, "transactions.create", a.transactionsCreate)
	register(g, idem, "transactions.update", a.transactionsUpdate)
	register(g, idem, "transactions.delete", a.transactionsDelete)

	register(g, idem, "transfers.create", a.transfersCreate)
	register(g, idem, "transfers.delete", a.transfersDelete)

	register(g, idem, "budgets.list", a.budgetsList)
	register(g, idem, "budgets.set", a.budgetsSet)
	register(g, idem, "budgets.delete", a.budgetsDelete)

	register(g, idem, "summary.show", a.summaryShow)

	a.syncRoutes(g, idem)
}
