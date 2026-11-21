package workspaces

import (
	"context"
	"net/http"
	"slices"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/authz"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// Abilities, checked with Service.Can in the controllers.
const (
	// AbilityRead: see everything of the workspace.
	AbilityRead = "workspace.read"
	// AbilityRecord: add, edit and delete transactions.
	AbilityRecord = "transactions.record"
	// AbilityManageData: accounts, categories, budgets and imports.
	AbilityManageData = "data.manage"
	// AbilityManageMembers: invite and remove people, rename the workspace.
	AbilityManageMembers = "members.manage"
	// AbilityManageBilling: choose a plan, open the billing portal.
	AbilityManageBilling = "billing.manage"
)

// grants says which roles hold each ability.
var grants = map[string][]string{
	AbilityRead:          {models.RoleOwner, models.RoleAdmin, models.RoleMember, models.RoleViewer},
	AbilityRecord:        {models.RoleOwner, models.RoleAdmin, models.RoleMember},
	AbilityManageData:    {models.RoleOwner, models.RoleAdmin},
	AbilityManageMembers: {models.RoleOwner, models.RoleAdmin},
	AbilityManageBilling: {models.RoleOwner},
}

func newGate() *authz.Gate[models.User] {
	gate := authz.New(func(ctx context.Context) (models.User, error) {
		return auth.UserFrom[models.User](ctx, "web")
	})

	for ability, roles := range grants {
		gate.Define(ability, func(ctx context.Context, _ models.User) bool {
			return slices.Contains(roles, Role(ctx))
		})
	}

	return gate
}

// Can returns nil when the signed-in user may perform ability in the
// workspace of ctx, else a 403.
func (s *Service) Can(ctx context.Context, ability string) error {
	return s.Gate.Authorize(ctx, ability, nil)
}

// Require is a route middleware that answers 403 unless the role of the
// user holds ability. The tenant middleware must run before it.
func (s *Service) Require(ability string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := s.Can(r.Context(), ability); err != nil {
				web.Fail(w, r, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
