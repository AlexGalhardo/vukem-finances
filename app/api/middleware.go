package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Stable problem codes (the `code` member of an error, docs/api.md).
const (
	CodeBadRequest      = "bad_request"
	CodeUnauthorized    = "unauthorized"
	CodePlanLimit       = "plan_limit"
	CodeMissingScope    = "missing_scope"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeVersionConflict = "version_conflict"
	CodeValidation      = "validation_failed"
	CodeRateLimited     = "rate_limited"
	CodeUnavailable     = "unavailable"
)

// CodeOf is the default code of a status.
func CodeOf(status int) string {
	switch status {
	case http.StatusBadRequest:
		return CodeBadRequest
	case http.StatusUnauthorized:
		return CodeUnauthorized
	case http.StatusPaymentRequired:
		return CodePlanLimit
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusUnprocessableEntity:
		return CodeValidation
	case http.StatusTooManyRequests:
		return CodeRateLimited
	case http.StatusServiceUnavailable:
		return CodeUnavailable
	default:
		return ""
	}
}

// Principal is who a request acts as: the token, the person who made it, the
// workspace it is bound to and the role the person holds there now.
type Principal struct {
	Token       auth.Token
	User        models.User
	WorkspaceID int64
	Role        string
}

type principalKey struct{}

// PrincipalFrom returns the principal the Authenticate middleware set.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

// API holds what the middleware and the handlers share.
type API struct {
	Users  *auth.ORMProvider[models.User]
	Tokens *auth.Tokens
	WS     *workspaces.Service
	Sync   *syncproto.Service
	// Ops is the registry every operation of the API goes through.
	Ops *operations.Catalog
	// Exports holds the downloads of a workspace to the limit they share with
	// the exports of the web app.
	Exports func(http.Handler) http.Handler
	// PullLimiter and PushLimiter hold a device to the sync rate limits.
	PullLimiter, PushLimiter ratelimit.Limiter
	Log                      *slog.Logger
}

func fail(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	web.Fail(w, r, errs.New(status, message).WithExtension("code", code))
}

// Authenticate is the first half of the chain: the token, its workspace, the
// role of its person. Failed lookups count against the client address
// (failed), and an address over the limit is refused before any lookup.
//
// The order never leaks: 401 for a missing, unknown, expired or revoked token;
// 404 when the person is no longer in the workspace the token is bound to.
func (a *API) Authenticate(failed ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ip := ratelimit.ByIP(r)

			if peeker, ok := failed.(ratelimit.Peeker); ok {
				res, err := peeker.Peek(ctx, ip)
				if err != nil {
					fail(w, r, http.StatusServiceUnavailable, CodeUnavailable, "Try again in a moment.")
					return
				}

				if !res.Allowed {
					web.SecurityEvent(r, "api.token.lookup_limited")
					tooMany(w, r, res)

					return
				}
			}

			p, status, err := a.principal(r)
			switch {
			case err != nil:
				fail(w, r, http.StatusServiceUnavailable, CodeUnavailable, "Try again in a moment.")
				return
			case status == http.StatusUnauthorized:
				_, _ = failed.Allow(ctx, ip)

				w.Header().Set("WWW-Authenticate", "Bearer")
				fail(w, r, status, CodeUnauthorized, "A valid token is required.")

				return
			case status != 0:
				fail(w, r, status, CodeNotFound, "Not found.")
				return
			}

			ctx = workspaces.WithTenant(ctx, p.WorkspaceID)
			ctx = workspaces.WithRole(ctx, p.Role)
			ctx = context.WithValue(ctx, principalKey{}, p)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// principal resolves the bearer token of r. A non-zero status is the answer;
// err is a failing store, not a bad credential.
func (a *API) principal(r *http.Request) (Principal, int, error) {
	ctx := r.Context()

	plain, ok := auth.BearerToken(r)
	if !ok {
		return Principal{}, http.StatusUnauthorized, nil
	}

	tok, err := a.Tokens.Find(ctx, plain)
	if errors.Is(err, auth.ErrInvalidToken) {
		web.SecurityEvent(r, "api.token.rejected")
		return Principal{}, http.StatusUnauthorized, nil
	}

	if err != nil {
		return Principal{}, 0, err
	}

	ws, werr := strconv.ParseInt(tok.Resource, 10, 64)
	// A token with no workspace is not one this app issues: refuse it as unknown.
	if werr != nil || ws <= 0 || slices.Contains(tok.Abilities, "*") {
		return Principal{}, http.StatusUnauthorized, nil
	}

	user, err := a.Users.ByID(ctx, tok.UserID)
	if err != nil {
		if isNotFound(err) {
			return Principal{}, http.StatusUnauthorized, nil
		}

		return Principal{}, 0, err
	}

	role, member, err := a.WS.MemberRole(ctx, user.ID, ws)
	if err != nil {
		return Principal{}, 0, err
	}

	if !member {
		return Principal{}, http.StatusNotFound, nil
	}

	return Principal{Token: tok, User: user, WorkspaceID: ws, Role: role}, 0, nil
}

func tooMany(w http.ResponseWriter, r *http.Request, res ratelimit.Result) {
	retry := int(res.RetryAfter.Seconds()) + 1
	w.Header().Set("Retry-After", strconv.Itoa(retry))
	web.Fail(w, r, errs.New(http.StatusTooManyRequests, "Too many requests.").WithExtension("code", CodeRateLimited))
}

// TokenKey keys a rate limit by the token of the request, after Authenticate.
func TokenKey(r *http.Request) string {
	if p, ok := PrincipalFrom(r.Context()); ok {
		return "api-token:" + p.Token.ID
	}

	return ratelimit.ByIP(r)
}

// Require is the second half of the chain, per operation: the token must hold
// the scope of op, and the role of its person must hold the ability the scope
// needs. A scope is a ceiling, never a grant. An operation of the registry is
// held to the same gate Invoke applies (Registry.Authorize), here early, so a
// refused call is refused before its body is read.
func (a *API) Require(op Operation) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if op.Scope == "" {
				next.ServeHTTP(w, r)
				return
			}

			p, ok := PrincipalFrom(r.Context())
			if !ok {
				fail(w, r, http.StatusUnauthorized, CodeUnauthorized, "A valid token is required.")
				return
			}

			if meta, registered := a.Ops.Named(op.Name); registered {
				if err := a.Ops.Authorize(meta, actorOf(p)); err != nil {
					web.Fail(w, r, restError(err))
					return
				}

				next.ServeHTTP(w, r)

				return
			}

			if !slices.Contains(p.Token.Abilities, op.Scope) {
				fail(w, r, http.StatusForbidden, CodeMissingScope, "The token lacks the scope "+op.Scope+".")
				return
			}

			if !workspaces.RoleCan(p.Role, AbilityOf(op.Scope)) {
				fail(w, r, http.StatusForbidden, CodeForbidden, "Your role in this workspace cannot do that.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isNotFound(err error) bool {
	return errors.Is(err, errs.ErrNotFound) || errors.Is(err, auth.ErrUserNotFound)
}
