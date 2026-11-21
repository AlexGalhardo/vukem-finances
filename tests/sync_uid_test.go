package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem/ulid"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func TestSyncedModelsStampUID(t *testing.T) {
	t.Parallel()

	ctx := workspaces.WithTenant(t.Context(), 7)

	a := models.Account{}
	if err := a.BeforeCreate(ctx); err != nil || !ulid.Valid(a.UID) {
		t.Fatalf("uid = %q, err %v; want a generated ULID", a.UID, err)
	}

	mine := ulid.NewGenerator().New()
	c := models.Category{UID: mine}

	if err := c.BeforeCreate(ctx); err != nil || c.UID != mine {
		t.Fatalf("uid = %q, err %v; want the client's %q kept", c.UID, err, mine)
	}
}
