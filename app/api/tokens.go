package api

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/config"
)

var (
	errTokenName   = errs.New(http.StatusUnprocessableEntity, "Name the token with 1 to 80 characters.")
	errTokenScopes = errs.New(http.StatusUnprocessableEntity, "Choose at least one scope from the list.")
	errTokenExpiry = errs.New(http.StatusUnprocessableEntity, "The expiry must be 30, 90 or 365 days.")
	errTokenLimit  = errs.New(http.StatusUnprocessableEntity,
		"You hold the most live tokens this workspace allows: revoke one first.")
)

// TokenInput is a new token as the form sends it.
type TokenInput struct {
	Name   string
	Scopes []string
	Days   int
}

// TokenService issues, lists and revokes the API tokens of a person in one
// workspace. The secret is returned once by Create and never stored.
type TokenService struct {
	DB *database.DB
	// Prototype carries the prefix, the clock and the logger of the app's tokens.
	Prototype *auth.Tokens
}

func (s *TokenService) tokens(ex database.Executor) *auth.Tokens {
	t := auth.NewTokens(ex)
	t.Prefix, t.Now, t.Log = s.Prototype.Prefix, s.Prototype.Now, s.Prototype.Log

	return t
}

func (s *TokenService) now() time.Time {
	if s.Prototype.Now != nil {
		return s.Prototype.Now().UTC()
	}

	return time.Now().UTC()
}

// Live reports whether a token still works at now.
func Live(t auth.Token, now time.Time) bool {
	return t.RevokedAt == nil && (t.ExpiresAt == nil || t.ExpiresAt.After(now))
}

func userKey(u models.User) string { return strconv.FormatInt(u.ID, 10) }

func workspaceKey(id int64) string { return strconv.FormatInt(id, 10) }

// Create issues a token for u in workspaceID. It returns the plain token,
// which is shown once.
func (s *TokenService) Create(ctx context.Context, u models.User, workspaceID int64, in TokenInput) (string, auth.Token, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return "", auth.Token{}, errTokenName
	}

	if !slices.Contains(config.TokenExpiryDays, in.Days) {
		return "", auth.Token{}, errTokenExpiry
	}

	chosen := make([]string, 0, len(in.Scopes))

	for _, sc := range Scopes() {
		if slices.Contains(in.Scopes, sc.Name) {
			chosen = append(chosen, sc.Name)
		}
	}

	for _, requested := range in.Scopes {
		if !ValidScope(requested) {
			return "", auth.Token{}, errTokenScopes
		}
	}

	if len(chosen) == 0 {
		return "", auth.Token{}, errTokenScopes
	}

	var (
		plain string
		tok   auth.Token
	)

	// One transaction, so two requests cannot both pass the count (SQLite
	// serializes writers; the other databases take the row lock of the user).
	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		live, err := s.liveCount(ctx, tx, u, workspaceID)
		if err != nil {
			return err
		}

		if live >= config.TokenMaxLive {
			return errTokenLimit
		}

		plain, tok, err = s.tokens(tx).CreateWith(ctx, userKey(u), name, chosen,
			time.Duration(in.Days)*24*time.Hour, auth.CreateOptions{Resource: workspaceKey(workspaceID)})

		return err
	})
	if err != nil {
		return "", auth.Token{}, err
	}

	return plain, tok, nil
}

func (s *TokenService) liveCount(ctx context.Context, ex database.Executor, u models.User, workspaceID int64) (int, error) {
	all, err := s.tokens(ex).List(ctx, userKey(u))
	if err != nil {
		return 0, err
	}

	now, n := s.now(), 0

	for _, t := range all {
		if t.Resource == workspaceKey(workspaceID) && Live(t, now) {
			n++
		}
	}

	return n, nil
}

// List returns the tokens u made for workspaceID, newest first, revoked and
// expired ones included. It never carries a secret.
func (s *TokenService) List(ctx context.Context, u models.User, workspaceID int64) ([]auth.Token, error) {
	all, err := s.tokens(s.DB).List(ctx, userKey(u))
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(all, func(t auth.Token) bool { return t.Resource != workspaceKey(workspaceID) }), nil
}

// Revoke revokes a live token of u in workspaceID: the answer for the token of
// another person, another workspace, an unknown id or one already revoked is
// the same 404.
func (s *TokenService) Revoke(ctx context.Context, u models.User, workspaceID int64, id string) error {
	mine, err := s.List(ctx, u, workspaceID)
	if err != nil {
		return err
	}

	if !slices.ContainsFunc(mine, func(t auth.Token) bool { return t.ID == id && t.RevokedAt == nil }) {
		return errs.ErrNotFound
	}

	ok, err := s.tokens(s.DB).RevokeFor(ctx, userKey(u), id)
	if err != nil {
		return err
	}

	if !ok {
		return errs.ErrNotFound
	}

	return nil
}
