package api

import (
	"net/http"

	vukemapi "github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"
)

// register adds the typed handler of a REST-only operation (the token and the
// sync protocol) to g, behind the scope check of its row in the operations
// table and, for a write, the Idempotency-Key replay. An operation of the
// registry is added with route and gets both from the pipeline.
func register[In, Out any](a *API, g *router.Group, idem router.Middleware, name string, fn func(*web.Ctx, In) (Out, error)) {
	op, ok := OperationNamed(name)
	if !ok {
		panic("api: operation " + name + " is not in the operations table")
	}

	rt := vukemapi.Register(g, op.Method+" "+op.Path, vukemapi.Op{
		Name: op.Name, Summary: op.Summary, Tags: []string{op.Tag}, Security: []string{"bearer"}, Status: op.Status,
	}, fn)

	mws := []router.Middleware{a.Require(op)}
	if op.MaxBody > 0 {
		mws = append(mws, limitBody(op.MaxBody))
	}

	if !op.Read() && idem != nil {
		mws = append(mws, idem)
	}

	rt.Use(mws...)
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

// lookups are the uids of the accounts and categories rows point at.
type lookups struct {
	accounts   map[int64]string
	categories map[int64]string
	currency   string
}
