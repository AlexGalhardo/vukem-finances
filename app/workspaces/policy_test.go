package workspaces

import (
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/models"
)

func TestPolicyTable(t *testing.T) {
	t.Parallel()

	want := map[string][]string{
		AbilityRead:          {models.RoleOwner, models.RoleAdmin, models.RoleMember, models.RoleViewer},
		AbilityRecord:        {models.RoleOwner, models.RoleAdmin, models.RoleMember},
		AbilityManageData:    {models.RoleOwner, models.RoleAdmin},
		AbilityManageMembers: {models.RoleOwner, models.RoleAdmin},
		AbilityManageBilling: {models.RoleOwner},
		AbilityTransfer:      {models.RoleOwner},
		AbilityManageAI:      {models.RoleOwner},
	}

	if len(grants) != len(want) {
		t.Errorf("grants has %d abilities, want %d", len(grants), len(want))
	}

	for ability, roles := range want {
		if !slices.Equal(grants[ability], roles) {
			t.Errorf("%s is granted to %v, want %v", ability, grants[ability], roles)
		}
	}
}
