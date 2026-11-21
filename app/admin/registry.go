package admin

import (
	vadmin "github.com/AlexGalhardo/vukem/admin"
	"github.com/AlexGalhardo/vukem/authz"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

// Resources lists the panel's resources; vukem make:admin-resource appends
// here.
func Resources() []vadmin.AnyResource {
	return []vadmin.AnyResource{
		User(),
		Workspace(),
		WorkspaceSubscription(),
	}
}

// Policies lists what admins may do per model; vukem make:admin-resource
// appends here. A resource without a policy denies every action.
func Policies(admins Admins) []authz.Policy {
	return []authz.Policy{
		authz.For[*models.User](UserPolicy{Admins: admins}),
		authz.For[*models.Workspace](WorkspacePolicy{Admins: admins}),
		authz.For[*models.WorkspaceSubscription](WorkspaceSubscriptionPolicy{Admins: admins}),
	}
}
