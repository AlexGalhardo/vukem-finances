package routes

import (
	"net/http"
	"strings"
	"sync"

	"github.com/AlexGalhardo/vukem"
	vukemapi "github.com/AlexGalhardo/vukem/api"
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/openapi"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// API registers the public JSON API under /api/v1 (docs/api.md). Only a bearer
// token authenticates it: the session and CSRF middleware skip the prefix (see
// Web), so a cookie is ignored. Each operation is declared in api.Operations.
func API(r *router.Router, app *vukem.App) error {
	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	ws, err := workspaces.New(app)
	if err != nil {
		return err
	}

	fin, err := finance.New(app, ws)
	if err != nil {
		return err
	}

	limiters, err := app.RateLimiters()
	if err != nil {
		return err
	}

	store, err := app.Cache()
	if err != nil {
		return err
	}

	a := &api.API{Users: acc.Users, Tokens: acc.Tokens, WS: ws, Fin: fin, Now: app.Now}

	// Every error of the API carries a stable "code" next to the problem+json members.
	r.Services().ProblemDecorator = func(c *web.Ctx, p *web.Problem) {
		if !strings.HasPrefix(c.Request().URL.Path, api.BasePath+"/") {
			return
		}

		if _, ok := p.Extensions["code"]; ok {
			return
		}

		if code := api.CodeOf(p.Status); code != "" {
			if p.Extensions == nil {
				p.Extensions = map[string]any{}
			}

			p.Extensions["code"] = code
		}
	}

	failed := limiters("api.failed", config.APIFailedLookupLimit, config.APIWindow)
	reads := ratelimit.Middleware(limiters("api.read", config.APIReadLimit, config.APIWindow), onMethods(api.TokenKey, true))
	writes := ratelimit.Middleware(limiters("api.write", config.APIWriteLimit, config.APIWindow), onMethods(api.TokenKey, false))

	idem := vukemapi.Idempotency(store.Store(), vukemapi.IdempotencyOptions{
		Scope: func(r *http.Request) string {
			if p, ok := api.PrincipalFrom(r.Context()); ok {
				return "api-token:" + p.Token.ID
			}

			return ""
		},
	})

	// The document is public and has no reference page: the static docs site renders it (D-021).
	var (
		docOnce sync.Once
		docJSON []byte
		docErr  error
	)

	r.Get(api.BasePath+"/openapi.json", func(c *web.Ctx) error {
		docOnce.Do(func() {
			var doc *openapi.Document
			if doc, docErr = api.BuildDocument(r.Routes()); docErr == nil {
				docJSON, docErr = api.RenderDocument(doc)
			}
		})

		if docErr != nil {
			return docErr
		}

		c.Writer().Header().Set("Content-Type", "application/json; charset=utf-8")
		c.Writer().Header().Set("Cache-Control", "public, max-age=300")
		c.Writer().WriteHeader(http.StatusOK)
		_, err := c.Writer().Write(docJSON)

		return err
	}).Name("api.openapi").Doc(openapi.Skip{})

	// A path or a method the API does not have is a JSON 404, not the HTML page of the web app.
	r.Handle(api.BasePath+"/{rest...}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.Header.Set("Accept", "application/problem+json")
		web.Fail(w, req, errs.New(http.StatusNotFound, "Not found.").WithExtension("code", api.CodeNotFound))
	}))

	r.Group(api.BasePath, func(g *router.Group) {
		g.Use(auth.ProblemJSON, a.Authenticate(failed), reads, writes)
		a.Routes(g, idem)
	}).Name("api.").ErrorHandler(jsonErrors)

	return nil
}

// jsonErrors makes every error of the API problem+json, whatever the Accept
// header of the client says (a browser, curl without a header).
func jsonErrors(next web.ErrorHandler) web.ErrorHandler {
	return func(c *web.Ctx, err error) {
		c.Request().Header.Set("Accept", "application/problem+json")
		next(c, err)
	}
}
