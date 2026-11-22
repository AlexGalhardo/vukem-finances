package operations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/config"
)

func TestInventoryIsSortedDeterministicAndHonestAboutSurfaces(t *testing.T) {
	t.Parallel()

	run := func(context.Context, operations.Actor, echoIn) (echoOut, error) { return echoOut{}, nil }
	reg := operations.NewRegistry(operations.Config{Edition: config.EditionSaaS})

	// Declared out of order on purpose.
	operations.Define(reg, operations.Operation[echoIn, echoOut]{
		Name: "zeta.show", Summary: "Z", Scope: operations.ScopeTagsRead, Kind: operations.KindRead, Run: run,
		OptOut: map[operations.Surface]string{operations.SurfaceMCP: "a human flow"},
	})
	operations.Define(reg, operations.Operation[echoIn, echoOut]{
		Name: "alpha.create", Summary: "A", Scope: operations.ScopeTagsWrite, Kind: operations.KindWrite, Idempotent: true, Run: run,
		Editions: []config.Edition{config.EditionSaaS},
	})

	rest := func(name string) (operations.Route, bool) {
		return operations.Route{Method: "GET", Path: "/x/" + name}, true
	}

	first, err := operations.BuildInventory(reg, rest).JSON()
	if err != nil {
		t.Fatal(err)
	}

	second, _ := operations.BuildInventory(reg, rest).JSON()
	if !bytes.Equal(first, second) || first[len(first)-1] != '\n' {
		t.Fatal("the inventory is not stable")
	}

	var inv operations.Inventory
	if err := json.Unmarshal(first, &inv); err != nil {
		t.Fatal(err)
	}

	if len(inv.Operations) != 2 || inv.Operations[0].Name != "alpha.create" || inv.Operations[1].Name != "zeta.show" {
		t.Fatalf("operations = %+v, want alpha.create then zeta.show", inv.Operations)
	}

	if got := inv.Operations[0].Editions; len(got) != 1 || got[0] != "saas" {
		t.Errorf("editions = %v, want [saas]", got)
	}

	zeta := inv.Operations[1]
	if s := zeta.Surfaces[2]; s.Surface != "mcp" || s.Offered || s.Reason != "a human flow" {
		t.Errorf("mcp surface = %+v, want an opt-out with its reason", s)
	}

	if s := zeta.Surfaces[0]; s.Route != "GET /x/zeta.show" || !s.Offered {
		t.Errorf("rest surface = %+v", s)
	}

	if len(zeta.Input) != 1 || zeta.Input[0].Name != "name" || !zeta.Input[0].Required {
		t.Errorf("input = %+v, want the required name", zeta.Input)
	}
}
