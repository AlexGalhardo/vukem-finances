package oauth

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/auth/jwt"
	"github.com/AlexGalhardo/vukem/oauth"

	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// AccessTokenTTL is how long an access token lives (D-018); the refresh token
// gets a new one.
const AccessTokenTTL = 15 * time.Minute

// MCPPath is the path of the remote endpoint; the audience of every token is
// the canonical URL of it.
const MCPPath = "/mcp"

// ErrIssuerNotHTTPS is returned when the public URL of the app is not https:
// the authorization server refuses to run on it, and remote MCP stays off.
var ErrIssuerNotHTTPS = errors.New("APP_URL must be an https URL for the remote MCP endpoint")

// Config is what the application decides; the rest comes from the app.
type Config struct {
	// Scopes are the scopes a client may ask for.
	Scopes []string
	// Registered are the clients known ahead of time; any other client id must
	// be the address of a Client ID Metadata Document, fetched with the SSRF
	// defenses of the framework.
	Registered []config.OAuthClient
	// Clients replaces both, for tests.
	Clients oauth.ClientResolver
	// Allow limits the rate of the endpoints per client; nil allows all.
	Allow func(r *http.Request, endpoint string) bool
	// Now is the clock; nil means the app's.
	Now func() time.Time
}

// Service is the authorization server of the application, wired.
type Service struct {
	Server   *oauth.Server
	Store    *GrantStore
	WS       *workspaces.Service
	Issuer   string
	Resource string
	Log      *slog.Logger
}

// Canonical returns the issuer and the canonical resource URL of an app whose
// public URL is base.
func Canonical(base string) (issuer, resource string, err error) {
	issuer = strings.TrimSuffix(strings.TrimSpace(base), "/")

	u, perr := url.Parse(issuer)
	if perr != nil || u.Scheme != "https" || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", "", ErrIssuerNotHTTPS
	}

	return issuer, issuer + MCPPath, nil
}

// New builds the authorization server for the app: keys and the CSRF secret are
// derived from APP_KEY, so a restart or a second instance signs and verifies
// the same way, and rotating APP_KEY ends every access token (they live 15
// minutes) while refresh tokens, which are random and stored hashed, go on.
func New(app *vukem.App, ws *workspaces.Service, cfg Config) (*Service, error) {
	issuer, resource, err := Canonical(app.Config.URL)
	if err != nil {
		return nil, err
	}

	seed, err := app.DeriveKey("mcp-oauth-ed25519", ed25519.SeedSize)
	if err != nil {
		return nil, err
	}

	csrf, err := app.DeriveKey("mcp-oauth-csrf", 32)
	if err != nil {
		return nil, err
	}

	now := cfg.Now
	if now == nil {
		now = app.Now
	}

	store := NewGrantStore(ws.DB, now)
	store.Role = ws.MemberRole

	clients := cfg.Clients
	if clients == nil {
		static := make(oauth.StaticClients, 0, len(cfg.Registered))
		for _, c := range cfg.Registered {
			static = append(static, oauth.Client{ID: c.ID, Name: c.Name, RedirectURIs: slices.Clone(c.RedirectURIs)})
		}

		clients = oauth.Resolvers{static, &oauth.MetadataDocuments{}}
	}

	store.Clients = clients

	svc := &Service{Store: store, WS: ws, Issuer: issuer, Resource: resource, Log: app.Log}

	server, err := oauth.New(oauth.Config{
		Issuer:          issuer,
		Keys:            []jwt.Key{jwt.EdDSA("mcp-1", ed25519.NewKeyFromSeed(seed))},
		Store:           store,
		Clients:         clients,
		Consent:         &Consent{WS: ws, Scopes: cfg.Scopes},
		Resources:       []string{resource},
		ScopesSupported: slices.Clone(cfg.Scopes),
		CSRFKey:         csrf,
		AccessTokenTTL:  AccessTokenTTL,
		Allow:           cfg.Allow,
		Logger:          app.Log,
		Now:             now,
	})
	if err != nil {
		return nil, fmt.Errorf("oauth: %w", err)
	}

	svc.Server = server

	return svc, nil
}
