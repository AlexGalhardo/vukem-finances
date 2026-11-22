package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	vukemapi "github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

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

// register adds the typed handler of the operation named name to g, behind the
// scope check of its row in the operations table (and, for a write, the
// Idempotency-Key replay: after the scope check, so a replay never skips it).
func register[In, Out any](g *router.Group, idem router.Middleware, name string, fn func(*web.Ctx, In) (Out, error)) {
	op, ok := OperationNamed(name)
	if !ok {
		panic("api: operation " + name + " is not in the operations table")
	}

	rt := vukemapi.Register(g, op.Method+" "+op.Path, vukemapi.Op{
		Name: op.Name, Summary: op.Summary, Tags: []string{op.Tag}, Security: []string{"bearer"}, Status: op.Status,
	}, fn)

	mws := []router.Middleware{Require(op)}
	if op.MaxBody > 0 {
		mws = append(mws, limitBody(op.MaxBody))
	}

	if !op.Read() && idem != nil {
		mws = append(mws, idem)
	}

	rt.Use(mws...)
}

// checkUID validates a uid a client chose.
func (a *API) checkUID(uid string) error {
	if uid == "" {
		return nil
	}

	if err := changes.ValidUID(uid, a.Now()); err != nil {
		return errBadUID
	}

	return nil
}

// lookups loads the uids that rows of the workspace point at.
func (a *API) lookups(ctx context.Context) (lookups, error) {
	ws, err := a.WS.Current(ctx)
	if err != nil {
		return lookups{}, err
	}

	accounts, err := a.Fin.Accounts(ctx)
	if err != nil {
		return lookups{}, err
	}

	cats, err := a.Fin.Categories(ctx)
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

// ifMatch reads the If-Match header: -1 when absent.
func ifMatch(c *web.Ctx) (int64, error) {
	raw := strings.Trim(strings.TrimSpace(c.Header("If-Match")), `"`)
	if raw == "" {
		return -1, nil
	}

	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return 0, errIfMatch
	}

	return v, nil
}

// checkVersion answers 409 with the current row when If-Match names another
// version. The check precedes the write: a writer that slips in between is
// the one case a conditional UPDATE inside the service would catch (docs/api.md).
func checkVersion(c *web.Ctx, current int64, row any) error {
	want, err := ifMatch(c)
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

func (a *API) accountByUID(ctx context.Context, uid string) (models.Account, error) {
	x, err := workspaces.Scoped[models.Account](ctx, a.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

func (a *API) categoryByUID(ctx context.Context, uid string) (models.Category, error) {
	x, err := workspaces.Scoped[models.Category](ctx, a.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

func (a *API) tagByUID(ctx context.Context, uid string) (models.Tag, error) {
	x, err := workspaces.Scoped[models.Tag](ctx, a.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

// requireScopeForNewTags stops a token without tags:write from creating tags
// through a transaction: the service creates a name it does not know. Naming
// tags that already exist only needs the scope of the transaction.
func (a *API) requireScopeForNewTags(ctx context.Context, names []string) error {
	p, ok := PrincipalFrom(ctx)
	if !ok || slices.Contains(p.Token.Abilities, ScopeTagsWrite) {
		return nil
	}

	for _, name := range names {
		found, err := workspaces.Scoped[models.Tag](ctx, a.WS.DB).
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

func (a *API) transactionByUID(ctx context.Context, uid string) (models.Transaction, error) {
	x, err := workspaces.Scoped[models.Transaction](ctx, a.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

func (a *API) budgetByUID(ctx context.Context, uid string) (models.Budget, error) {
	x, err := workspaces.Scoped[models.Budget](ctx, a.WS.DB).Where("uid", uid).First(ctx)

	return x, lookupErr(err)
}

// existing looks up a row a client wants to create under a uid it chose, in
// the workspace of ctx and among the deleted rows too. Found live: the caller
// compares it and replays. Found deleted: 409.
func existing[T any](ctx context.Context, a *API, uid string) (row T, found bool, err error) {
	if uid == "" {
		return row, false, nil
	}

	row, err = workspaces.Scoped[T](ctx, a.WS.DB).WithTrashed().Where("uid", uid).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return row, false, nil
	}

	return row, err == nil, err
}

// limitBody answers 413 for a body over max bytes, by its declared length and,
// for one that lies or has none, while it is read.
func limitBody(limit int64) router.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limit {
				fail(w, r, http.StatusRequestEntityTooLarge, CodeBadRequest, "The body is larger than the limit of the operation.")
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}
