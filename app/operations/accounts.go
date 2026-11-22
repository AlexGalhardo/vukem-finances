package operations

import (
	"context"
	"strings"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

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

type accountOps struct {
	List    *Op[NoInput, List[AccountOut]]
	Show    *Op[UIDIn, AccountOut]
	Create  *Op[AccountCreateIn, AccountOut]
	Update  *Op[AccountUpdateIn, AccountOut]
	Archive *Op[UIDIn, AccountOut]
}

func (c *Catalog) defineAccounts() {
	c.Accounts = accountOps{
		List: Define(c.Registry, Operation[NoInput, List[AccountOut]]{
			Name: "accounts.list", Summary: "List the accounts with their balances", Scope: ScopeAccountsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, _ NoInput) (List[AccountOut], error) { return c.accountsList(ctx) },
		}),
		Show: Define(c.Registry, Operation[UIDIn, AccountOut]{
			Name: "accounts.show", Summary: "Read an account", Scope: ScopeAccountsRead, Kind: KindRead,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (AccountOut, error) { return c.accountsShow(ctx, in) },
		}),
		Create: Define(c.Registry, Operation[AccountCreateIn, AccountOut]{
			Name: "accounts.create", Summary: "Create an account", Scope: ScopeAccountsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in AccountCreateIn) (AccountOut, error) {
				return c.accountsCreate(ctx, in)
			},
		}),
		Update: Define(c.Registry, Operation[AccountUpdateIn, AccountOut]{
			Name: "accounts.update", Summary: "Change the name, kind and opening balance of an account", Scope: ScopeAccountsWrite, Kind: KindWrite, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in AccountUpdateIn) (AccountOut, error) {
				return c.accountsUpdate(ctx, in)
			},
		}),
		Archive: Define(c.Registry, Operation[UIDIn, AccountOut]{
			Name: "accounts.archive", Summary: "Archive an account", Scope: ScopeAccountsWrite, Kind: KindDestructive, Idempotent: true,
			Run: func(ctx context.Context, _ Actor, in UIDIn) (AccountOut, error) { return c.accountsArchive(ctx, in) },
		}),
	}
}

func (c *Catalog) accountViews(ctx context.Context) (map[int64]int64, string, error) {
	ws, err := c.WS.Current(ctx)
	if err != nil {
		return nil, "", err
	}

	balances, err := c.Fin.AccountBalances(ctx, "")
	if err != nil {
		return nil, "", err
	}

	out := make(map[int64]int64, len(balances))
	for _, b := range balances {
		out[b.AccountID] = b.BalanceCents
	}

	return out, ws.Currency, nil
}

func (c *Catalog) accountView(ctx context.Context, x models.Account) (AccountOut, error) {
	balances, currency, err := c.accountViews(ctx)
	if err != nil {
		return AccountOut{}, err
	}

	return accountOut(x, balances[x.ID], currency), nil
}

func (c *Catalog) accountsList(ctx context.Context) (List[AccountOut], error) {
	all, err := c.Fin.Accounts(ctx)
	if err != nil {
		return List[AccountOut]{}, err
	}

	balances, currency, err := c.accountViews(ctx)
	if err != nil {
		return List[AccountOut]{}, err
	}

	data := make([]AccountOut, 0, len(all))
	for _, x := range all {
		data = append(data, accountOut(x, balances[x.ID], currency))
	}

	return List[AccountOut]{Data: data}, nil
}

func (c *Catalog) accountsShow(ctx context.Context, in UIDIn) (AccountOut, error) {
	x, err := c.accountByUID(ctx, in.UID)
	if err != nil {
		return AccountOut{}, err
	}

	return c.accountView(ctx, x)
}

func (c *Catalog) accountsCreate(ctx context.Context, in AccountCreateIn) (AccountOut, error) {
	if err := c.checkUID(in.UID); err != nil {
		return AccountOut{}, err
	}

	if old, found, err := existing[models.Account](ctx, c, in.UID); err != nil || found {
		if err != nil {
			return AccountOut{}, err
		}

		if old.DeletedAt.Valid {
			return AccountOut{}, errUIDDeleted
		}

		if old.Name != strings.TrimSpace(in.Name) || old.Kind != in.Kind || old.OpeningCents != in.OpeningCents {
			return AccountOut{}, errUIDDiffers
		}

		MarkExisting(ctx)

		return c.accountView(ctx, old)
	}

	x, err := c.Fin.CreateAccountWithUID(ctx, in.UID, in.Name, in.Kind, centsText(in.OpeningCents))
	if err != nil {
		return AccountOut{}, err
	}

	return c.accountView(ctx, x)
}

func (c *Catalog) accountsUpdate(ctx context.Context, in AccountUpdateIn) (AccountOut, error) {
	x, err := c.accountByUID(ctx, in.UID)
	if err != nil {
		return AccountOut{}, err
	}

	cur, err := c.accountView(ctx, x)
	if err != nil {
		return AccountOut{}, err
	}

	if err := checkVersion(ctx, x.Version, cur); err != nil {
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

	saved, err := c.Fin.UpdateAccount(ctx, x.ID, name, kind, centsText(opening))
	if err != nil {
		return AccountOut{}, err
	}

	return c.accountView(ctx, saved)
}

func (c *Catalog) accountsArchive(ctx context.Context, in UIDIn) (AccountOut, error) {
	x, err := c.accountByUID(ctx, in.UID)
	if err != nil {
		return AccountOut{}, err
	}

	cur, err := c.accountView(ctx, x)
	if err != nil {
		return AccountOut{}, err
	}

	if err := checkVersion(ctx, x.Version, cur); err != nil {
		return AccountOut{}, err
	}

	saved, err := c.Fin.ArchiveAccount(ctx, x.ID)
	if err != nil {
		return AccountOut{}, err
	}

	return c.accountView(ctx, saved)
}
