package mcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	vmcp "github.com/AlexGalhardo/vukem/mcp"
	"github.com/AlexGalhardo/vukem/oauth"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/ratelimit"

	appoauth "github.com/AlexGalhardo/vukem-finances/app/oauth"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// RemoteOptions are what the remote endpoint needs.
type RemoteOptions struct {
	Catalog *operations.Catalog
	WS      *workspaces.Service
	OAuth   *appoauth.Service
	Config  config.MCP
	Version string
	Log     *slog.Logger
	// Reads and Writes limit the calls of one grant; nil does not limit.
	Reads, Writes ratelimit.Limiter
}

// RemoteScopes are the scopes a client may ask the remote endpoint for: those
// of the operations the server offers as tools or reads as resources. The sync
// protocol and the downloads are not among them.
func RemoteScopes(cat *operations.Catalog, cfg config.MCP) []string {
	return Scopes(cat, cfg)
}

// NewRemote builds the Streamable HTTP endpoint of the server. Every request
// carries an access token for this resource, from this issuer, for a person who
// is a member of the workspace of the grant now: the verifier and the actor
// read the membership again on each call, so removing a member counts at once.
func NewRemote(o RemoteOptions) (*vmcp.HTTPHandler, error) {
	r := &remote{ws: o.WS, as: o.OAuth.Server, log: o.Log, reads: o.Reads, writes: o.Writes}

	srv, err := New(Options{
		Catalog: o.Catalog, Actor: r.actor, Config: o.Config, Version: o.Version, Logger: o.Log,
		Before: r.before, Done: r.done,
	})
	if err != nil {
		return nil, err
	}

	return srv.HTTPHandler(vmcp.HTTPConfig{
		AllowedOrigins:       []string{o.OAuth.Issuer},
		Verifier:             r,
		Resource:             o.OAuth.Resource,
		AuthorizationServers: []string{o.OAuth.Issuer},
		ScopesSupported:      RemoteScopes(o.Catalog, o.Config),
		ResourceName:         "vukem-finances",
	})
}

type remote struct {
	ws            *workspaces.Service
	as            *oauth.Server
	log           *slog.Logger
	reads, writes ratelimit.Limiter
}

// VerifyAccessToken implements vmcp.TokenVerifier: the framework checks the
// signature, the issuer, the expiry, the audience and that the grant is not
// revoked; the membership is checked here, so a removed member is refused with
// a 401 on the next request.
func (r *remote) VerifyAccessToken(ctx context.Context, token, resource string) (*vmcp.TokenInfo, error) {
	at, err := r.as.VerifyAccessToken(ctx, token, resource)
	if errors.Is(err, oauth.ErrInvalidToken) {
		return nil, vmcp.ErrInvalidToken
	}

	if err != nil {
		return nil, err
	}

	sub, ok := appoauth.ParseSubject(at.Subject)
	if !ok {
		return nil, vmcp.ErrInvalidToken
	}

	_, member, err := r.ws.MemberRole(ctx, sub.UserID, sub.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if !member {
		return nil, vmcp.ErrInvalidToken
	}

	return &vmcp.TokenInfo{
		Subject: at.Subject, ClientID: at.ClientID, Scopes: at.Scopes, Resource: at.Resource, ExpiresAt: at.ExpiresAt,
	}, nil
}

// actor is the actor of a call: the person and workspace of the grant, the role
// they hold now, and the scopes the token carries. The scopes are a ceiling,
// intersected with the role by Invoke.
func (r *remote) actor(ctx context.Context) (operations.Actor, error) {
	info, ok := vmcp.TokenInfoFrom(ctx)
	if !ok {
		return operations.Actor{}, forbidden("no access token")
	}

	sub, ok := appoauth.ParseSubject(info.Subject)
	if !ok {
		return operations.Actor{}, forbidden("the grant names no workspace")
	}

	role, member, err := r.ws.MemberRole(ctx, sub.UserID, sub.WorkspaceID)
	if err != nil {
		return operations.Actor{}, err
	}

	if !member {
		return operations.Actor{}, forbidden("you are no longer a member of this workspace")
	}

	user, err := orm.QueryOf[models.User](r.ws.DB).Find(ctx, sub.UserID)
	if err != nil {
		return operations.Actor{}, err
	}

	if user.PendingDeletion() {
		return operations.Actor{}, forbidden("this account is scheduled for deletion")
	}

	return operations.NewActor(operations.SurfaceMCP, user, sub.WorkspaceID, role, info.Scopes,
		"oauth:"+info.ClientID+":"+info.Subject), nil
}

func forbidden(msg string) error {
	return &operations.Error{Code: operations.CodeForbidden, Message: msg}
}

// before limits the grant and keeps destructive tools for owners and admins: a
// member may hold a write scope, but a tool that removes data needs more than
// the scope.
func (r *remote) before(ctx context.Context, m operations.Meta, actor operations.Actor) error {
	if m.Kind == operations.KindDestructive && actor.Role() != models.RoleOwner && actor.Role() != models.RoleAdmin {
		return forbidden("only an owner or an admin can use a tool that removes data")
	}

	limiter := r.reads
	if m.Kind != operations.KindRead {
		limiter = r.writes
	}

	if limiter == nil {
		return nil
	}

	info, _ := vmcp.TokenInfoFrom(ctx)
	key := "mcp:" + info.Subject + ":" + info.ClientID + ":" + strconv.FormatBool(m.Kind != operations.KindRead)

	res, err := limiter.Allow(ctx, key)
	if err != nil {
		return fmt.Errorf("mcp rate limit: %w", err)
	}

	if !res.Allowed {
		return &operations.Error{Code: operations.CodeRateLimited, Message: "Too many calls: try again in " +
			strconv.Itoa(int(res.RetryAfter.Seconds())+1) + " seconds."}
	}

	return nil
}

// done writes the audit entry of every call: the client, the tool and how it
// ended. It never holds an input, an output, an amount or a description.
func (r *remote) done(ctx context.Context, m operations.Meta, err error) {
	if r.log == nil {
		return
	}

	info, _ := vmcp.TokenInfoFrom(ctx)

	attrs := []any{"client", info.ClientID, "tool", ToolName(m.Name), "kind", string(m.Kind)}

	if sub, ok := appoauth.ParseSubject(info.Subject); ok {
		attrs = append(attrs, "workspace", sub.WorkspaceID, "user", sub.UserID)
	}

	if err != nil {
		code := operations.CodeInternal
		if e, ok := errors.AsType[*operations.Error](err); ok {
			code = e.Code
		}

		r.log.WarnContext(ctx, "mcp call", append(attrs, "outcome", "refused", "code", code)...)

		return
	}

	r.log.InfoContext(ctx, "mcp call", append(attrs, "outcome", "ok")...)
}

// Allow is the per-address limit of the endpoint, as a middleware.
func Allow(l ratelimit.Limiter) func(http.Handler) http.Handler {
	return ratelimit.Middleware(l, ratelimit.ByIP)
}
