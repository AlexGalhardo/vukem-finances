package migrations

import (
	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/schema"
)

// m20261122120000AddTokenResource adds access_tokens.resource: the id of the
// workspace an API token works in (the framework stores it and the API
// middleware compares it).
func m20261122120000AddTokenResource() schema.Migration {
	return auth.TokensResourceMigration("20261122120000_add_token_resource")
}
