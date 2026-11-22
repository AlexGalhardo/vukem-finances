package api

import (
	"errors"
	"net/http"

	vukemapi "github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// actorOf is the actor of a request: the token, its person, the workspace it is
// bound to and the role the person holds there now. Authenticate checked all of
// it before.
func actorOf(p Principal) operations.Actor {
	return operations.NewActor(operations.SurfaceREST, p.User, p.WorkspaceID, p.Role, p.Token.Abilities, p.Token.ID)
}

// restError hands the framework the error a REST client has always received:
// the cause of an operation error, with its status and its problem members.
func restError(err error) error {
	if oe, ok := errors.AsType[*operations.Error](err); ok && oe.Err != nil {
		return oe.Err
	}

	return err
}

// route registers the REST route of an operation of the registry. The route
// only binds the input and calls operations.Invoke: the scope, the role, the
// validation, the idempotency key and the audit are the pipeline's.
func route[In, Out any](a *API, g *router.Group, o *operations.Op[In, Out]) {
	meta := o.Meta()

	row, ok := OperationNamed(meta.Name)
	if !ok {
		panic("api: operation " + meta.Name + " is not in the operations table")
	}

	handler := func(c *web.Ctx, in In) (Out, error) {
		var zero Out

		p, ok := PrincipalFrom(c.Context())
		if !ok {
			return zero, errs.New(http.StatusUnauthorized, "A valid token is required.").WithExtension("code", CodeUnauthorized)
		}

		opts := []operations.CallOption{operations.WithIfMatch(c.Header("If-Match"))}
		if meta.Idempotent {
			opts = append(opts, operations.WithIdempotencyKey(c.Header(vukemapi.IdempotencyHeader)))
		}

		res, err := operations.Invoke(c.Context(), o, actorOf(p), in, opts...)
		if err != nil {
			return zero, restError(err)
		}

		if res.Replayed {
			c.SetHeader("Idempotent-Replayed", "true")
		}

		if res.Existing {
			return zero, c.JSON(http.StatusOK, res.Value)
		}

		return res.Value, nil
	}

	rt := vukemapi.Register(g, row.Method+" "+row.Path, vukemapi.Op{
		Name: row.Name, Summary: row.Summary, Tags: []string{row.Tag}, Security: []string{"bearer"}, Status: row.Status,
	}, handler)

	mws := []router.Middleware{a.Require(row)}
	if row.MaxBody > 0 {
		mws = append(mws, limitBody(row.MaxBody))
	}

	rt.Use(mws...)
}
