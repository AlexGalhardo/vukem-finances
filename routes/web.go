package routes

import (
	"encoding/json"
	"net/http"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/inertia"
	"github.com/AlexGalhardo/vukem/middleware"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/controllers"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
	"github.com/AlexGalhardo/vukem-finances/resources"
)

// Web registers the browser routes. vukem route:list shows them.
func Web(r *router.Router, app *vukem.App) error {
	// The API authenticates with a bearer token only: no session is read, so a cookie is
	// ignored, and no CSRF check applies (nothing the browser sends by itself authenticates).
	apiPrefix := api.BasePath + "/"
	// The Telegram webhook is not a browser route either: the secret header is its credential.
	hookPrefix := config.WebhookPath + "/"
	// The remote MCP endpoint and the token endpoints of its authorization server
	// are bearer or client-credential routes too (epic 33): no cookie reaches them.
	// The authorization endpoint is a browser route and keeps both.
	mcpRoutes := append([]string{apiPrefix, hookPrefix}, MCPBypass...)
	if err := vukem.DefaultMiddlewareWith(app, r, vukem.MiddlewareOptions{
		SessionBypass: mcpRoutes, CSRFBypass: mcpRoutes,
	}); err != nil {
		return err
	}

	r.UseServer(middleware.Gzip()) // proxies in front of the app may not compress
	r.UseServer(middleware.StaticFS(resources.Public(), middleware.StaticConfig{Prefix: "/"}))
	edition, err := config.LoadEdition(app.Source)
	if err != nil {
		return err
	}

	r.Use(app.Inertia.Middleware, shareProps(app, edition))

	// The personal edition has no landing page.
	if edition.Personal() {
		acc, err := accounts.New(app)
		if err != nil {
			return err
		}

		ws, err := workspaces.New(app)
		if err != nil {
			return err
		}

		r.Get("/", controllers.PersonalHome(acc, ws)).Name("home")
	} else {
		r.Get("/", controllers.Home).Name("home")
	}

	r.Post("/locale", setLocale).Name("locale")

	return nil
}

// setLocale stores en, pt-BR or es on the session. Anything else stays English.
func setLocale(c *web.Ctx) error {
	s := c.Session()
	if s == nil {
		return c.Back("/")
	}

	s.Put("locale", financeLocale(c))

	return c.Back("/")
}

func financeLocale(c *web.Ctx) string {
	lang := c.Request().FormValue("language")
	if lang == "" {
		var body struct {
			Language string `json:"language"`
		}
		_ = json.NewDecoder(c.Request().Body).Decode(&body)
		lang = body.Language
	}

	switch lang {
	case "en", "pt-BR", "es":
		return lang
	default:
		return "en"
	}
}

// shareProps adds the props every page receives (usePage().props).
func shareProps(app *vukem.App, edition config.Edition) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			inertia.Share(r.Context(), "appName", app.Config.Name)
			inertia.Share(r.Context(), "edition", edition.String())
			inertia.Share(r.Context(), "locale", sharedLocale(r))
			inertia.Share(r.Context(), "googleEnabled", accounts.GoogleEnabled(app))

			// A personal install is the owner's own machine. In saas the version is for
			// signed-in people only (see shareAuth), so a scanner cannot read it.
			if edition.Personal() {
				inertia.Share(r.Context(), "release", config.DisplayVersion())
			}

			next.ServeHTTP(w, r)
		})
	}
}

func sharedLocale(r *http.Request) string {
	s := web.SessionFrom(r.Context())
	if s == nil {
		return "en"
	}

	lang, _ := s.Get("locale").(string)
	switch lang {
	case "en", "pt-BR", "es":
		return lang
	default:
		return "en"
	}
}
