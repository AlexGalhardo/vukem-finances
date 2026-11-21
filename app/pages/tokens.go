package pages

import "github.com/AlexGalhardo/vukem/inertia"

// Tokens lists the API tokens the person made for the current workspace.
var Tokens = inertia.Define[TokensProps]("Tokens")

// TokenView is a token as the page may show it: never the secret.
type TokenView struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Scopes     []string `json:"scopes"`
	LastUsedAt string   `json:"lastUsedAt"`
	ExpiresAt  string   `json:"expiresAt"`
	RevokedAt  string   `json:"revokedAt"`
	// Status is live, expired or revoked.
	Status string `json:"status"`
}

// CreatedToken is the one time answer of a creation: the plain token.
type CreatedToken struct {
	Name   string `json:"name"`
	Secret string `json:"secret"`
}

// TokensProps are the props of Tokens. Created is set only on the response
// that makes a token, and never again.
type TokensProps struct {
	Tokens     []TokenView   `json:"tokens"`
	Scopes     []string      `json:"scopes"`
	ExpiryDays []int         `json:"expiryDays"`
	Max        int           `json:"max"`
	Created    *CreatedToken `json:"created"`
}
