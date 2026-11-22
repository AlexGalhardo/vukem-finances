package routes

import (
	"errors"
	"net/http"
	"sync"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth"
	vmcp "github.com/AlexGalhardo/vukem/mcp"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
	appmcp "github.com/AlexGalhardo/vukem-finances/app/mcp"
	appoauth "github.com/AlexGalhardo/vukem-finances/app/oauth"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// MCPBypass are the patterns of the remote MCP endpoint and of the token
// endpoints of its authorization server, served without a session and without
// the CSRF check: a bearer token or a client authenticates them, and no cookie
// does. The authorization endpoint is a browser route and is not here.
var MCPBypass = []string{"/mcp", "POST /token", "POST /revoke", "/.well-known/"} //nolint:gochecknoglobals // a constant table

// mcpStack is the remote endpoint and its authorization server.
type mcpStack struct {
	oauth   *appoauth.Service
	handler *vmcp.HTTPHandler
	ip      ratelimit.Limiter
}

var (
	mcpMu     sync.Mutex                   //nolint:gochecknoglobals // one stack per booted app
	mcpStacks = map[*vukem.App]*mcpStack{} //nolint:gochecknoglobals // see mcpMu
)

// mcpOf returns the remote stack of the app, or nil when it is off: the
// personal edition has no remote endpoint, and one whose public URL is not
// https cannot run an authorization server.
func mcpOf(app *vukem.App) (*mcpStack, error) {
	mcpMu.Lock()
	defer mcpMu.Unlock()

	if s, built := mcpStacks[app]; built {
		return s, nil
	}

	s, err := buildMCP(app)
	if err != nil {
		return nil, err
	}

	mcpStacks[app] = s

	return s, nil
}

func buildMCP(app *vukem.App) (*mcpStack, error) {
	ws, err := workspaces.New(app)
	if err != nil {
		return nil, err
	}

	if !ws.Edition.SaaS() {
		return nil, nil
	}

	cfg, err := config.LoadMCP(app.Source)
	if err != nil {
		return nil, err
	}

	cat, _, err := appmcp.CatalogOf(app)
	if err != nil {
		return nil, err
	}

	limiters, err := app.RateLimiters()
	if err != nil {
		return nil, err
	}

	oauthLimit := limiters("mcp.oauth", config.MCPOAuthLimit, config.MCPWindow)

	// The authorization server asks before it works. A limiter that fails refuses.
	allow := func(r *http.Request, endpoint string) bool {
		res, err := oauthLimit.Allow(r.Context(), endpoint+":"+ratelimit.ByIP(r))

		return err == nil && res.Allowed
	}

	svc, err := appoauth.New(app, ws, appoauth.Config{
		Scopes: appmcp.RemoteScopes(cat, cfg), Registered: cfg.OAuthClients, Allow: allow,
	})
	if errors.Is(err, appoauth.ErrIssuerNotHTTPS) {
		app.Log.Warn("remote MCP is off", "reason", err.Error())

		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	h, err := appmcp.NewRemote(appmcp.RemoteOptions{
		Catalog: cat, WS: ws, OAuth: svc, Config: cfg, Version: config.Version, Log: app.Log,
		Reads:  limiters("mcp.read", config.MCPReadLimit, config.MCPWindow),
		Writes: limiters("mcp.write", config.MCPWriteLimit, config.MCPWindow),
	})
	if err != nil {
		return nil, err
	}

	return &mcpStack{oauth: svc, handler: h, ip: limiters("mcp.ip", config.MCPIPLimit, config.MCPWindow)}, nil
}

// MCP registers the remote MCP endpoint, its metadata and the authorization
// server (docs/mcp.md). All of it is saas only: in the personal edition none of
// these paths exist (404), and MCP stays on stdio.
func MCP(r *router.Router, app *vukem.App) error {
	stack, err := mcpOf(app)
	if err != nil || stack == nil {
		return err
	}

	acc, err := accounts.New(app)
	if err != nil {
		return err
	}

	ws, err := workspaces.New(app)
	if err != nil {
		return err
	}

	h := stack.handler

	r.Handle("/mcp", appmcp.Allow(stack.ip)(h)).Name("mcp.endpoint")
	r.Handle("GET "+h.MetadataPath(), h.MetadataHandler()).Name("mcp.resource-metadata")
	r.Handle("GET /.well-known/oauth-protected-resource", h.MetadataHandler()).Name("mcp.resource-metadata.root")
	r.Handle("GET /.well-known/oauth-authorization-server", stack.oauth.Server.MetadataHandler()).Name("oauth.metadata")
	r.Handle("GET /.well-known/jwks.json", stack.oauth.Server.JWKSHandler()).Name("oauth.jwks")
	r.Handle("POST /token", stack.oauth.Server.TokenHandler()).Name("oauth.token")
	r.Handle("POST /revoke", stack.oauth.Server.RevokeHandler()).Name("oauth.revoke")

	// The consent screen: the person signs in, is in a workspace they belong to,
	// and the framework decides everything the screen does not.
	required, verified := auth.Required(acc.Web), auth.Verified(acc.Web)

	r.Handle("GET /authorize", stack.oauth.Server.AuthorizeHandler()).Name("oauth.authorize").
		Use(required, verified, ws.Middleware())
	r.Handle("POST /authorize", stack.oauth.Server.AuthorizeHandler()).Name("oauth.authorize.decide").
		Use(required, verified, ws.Middleware())
	r.Handle("POST /oauth/workspace", http.HandlerFunc(appoauth.PickWorkspace)).Name("oauth.workspace").
		Use(required, verified)

	return nil
}
