package operations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/changes"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

var (
	errBadUID     = errs.New(http.StatusUnprocessableEntity, "The uid must be a ULID: 26 upper-case characters, at most a day ahead of the server clock.")
	errUIDDeleted = errs.New(http.StatusConflict, "That uid belongs to a row that was deleted.").WithExtension("code", CodeConflict)
	errUIDDiffers = errs.New(http.StatusConflict, "That uid belongs to a different row.").WithExtension("code", CodeConflict)
	errIfMatch    = errs.New(http.StatusBadRequest, "If-Match must be the version of the row, a whole number.")
)

// checkUID validates a uid a client chose.
func (c *Catalog) checkUID(uid string) error {
	if uid == "" {
		return nil
	}

	if err := changes.ValidUID(uid, c.cfg.Now()); err != nil {
		return errBadUID
	}

	return nil
}

// lookups loads the uids that rows of the workspace point at.
func (c *Catalog) lookups(ctx context.Context) (lookups, error) {
	ws, err := c.WS.Current(ctx)
	if err != nil {
		return lookups{}, err
	}

	accounts, err := c.Fin.Accounts(ctx)
	if err != nil {
		return lookups{}, err
	}

	cats, err := c.Fin.Categories(ctx)
	if err != nil {
		return lookups{}, err
	}

	l := lookups{accounts: map[int64]string{}, categories: map[int64]string{}, currency: ws.Currency}
	for _, x := range accounts {
		l.accounts[x.ID] = x.UID
	}

	for _, x := range cats {
		l.categories[x.ID] = x.UID
	}

	return l, nil
}

// centsText writes cents the way the finance services read an amount.
func centsText(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}

	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// precondition reads the version the caller named with WithIfMatch: -1 when
// it named none. The raw value is parsed here, after the row is found, so a
// uid of another workspace is a 404 before a bad header is a 400.
func precondition(ctx context.Context) (int64, error) {
	st, ok := ctx.Value(callKey{}).(*call)
	if !ok {
		return -1, nil
	}

	raw := strings.Trim(strings.TrimSpace(st.ifMatch), `"`)
	if raw == "" {
		return -1, nil
	}

	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return 0, errIfMatch
	}

	return v, nil
}

// checkVersion answers 409 with the current row when the caller named another
// version. The check precedes the write: a writer that slips in between is the
// one case a conditional UPDATE inside the service would catch.
func checkVersion(ctx context.Context, current int64, row any) error {
	want, err := precondition(ctx)
	if err != nil || want < 0 || want == current {
		return err
	}

	return errs.New(http.StatusConflict, "The row changed: its version is not the one in If-Match.").
		WithExtension("code", CodeVersionConflict).WithExtension("current", row)
}

// lookupErr is the 404 of a uid that names nothing in the workspace.
func lookupErr(err error) error {
	if errors.Is(err, orm.ErrNotFound) {
		return errs.ErrNotFound
	}

	return err
}

func (c *Catalog) accountByUID(ctx context.Context, uid string) (models.Account, error) {
	x, err := workspaces.Scoped[models.Account](ctx, c.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

func (c *Catalog) categoryByUID(ctx context.Context, uid string) (models.Category, error) {
	x, err := workspaces.Scoped[models.Category](ctx, c.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

func (c *Catalog) tagByUID(ctx context.Context, uid string) (models.Tag, error) {
	x, err := workspaces.Scoped[models.Tag](ctx, c.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

func (c *Catalog) transactionByUID(ctx context.Context, uid string) (models.Transaction, error) {
	x, err := workspaces.Scoped[models.Transaction](ctx, c.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

func (c *Catalog) budgetByUID(ctx context.Context, uid string) (models.Budget, error) {
	x, err := workspaces.Scoped[models.Budget](ctx, c.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

// requireScopeForNewTags stops a token without tags:write from creating tags
// through a transaction: the service creates a name it does not know. Naming
// tags that already exist only needs the scope of the transaction.
func (c *Catalog) requireScopeForNewTags(ctx context.Context, a Actor, names []string) error {
	if a.HasScope(ScopeTagsWrite) {
		return nil
	}

	for _, name := range names {
		found, err := workspaces.Scoped[models.Tag](ctx, c.WS.DB).
			WhereRaw("lower(name) = lower(?)", strings.TrimSpace(name)).Exists(ctx)
		if err != nil {
			return err
		}

		if !found {
			return errs.New(http.StatusForbidden, "Creating a tag needs the scope "+ScopeTagsWrite+".").
				WithExtension("code", CodeMissingScope)
		}
	}

	return nil
}

// existing looks up a row a client wants to create under a uid it chose, in
// the workspace of ctx and among the deleted rows too. Found live: the caller
// compares it and replays. Found deleted: 409.
func existing[T any](ctx context.Context, c *Catalog, uid string) (row T, found bool, err error) {
	if uid == "" {
		return row, false, nil
	}

	row, err = workspaces.Scoped[T](ctx, c.WS.DB).WithTrashed().Where("uid", uid).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return row, false, nil
	}

	return row, err == nil, err
}
