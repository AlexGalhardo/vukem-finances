package tests

import (
	"bytes"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

const inventoryFile = "../resources/operations.json"

// The inventory is committed and checked: a new, changed or removed operation
// fails here until `go run ./cmd/app operations:inventory` writes the file.
func TestOperationInventoryIsFresh(t *testing.T) {
	t.Parallel()

	body, err := bootstrap.InventoryJSON()
	if err != nil {
		t.Fatal(err)
	}

	again, err := bootstrap.InventoryJSON()
	if err != nil || !bytes.Equal(body, again) {
		t.Fatalf("the inventory is not deterministic (err %v)", err)
	}

	matchesFile(t, inventoryFile, body, "go run ./cmd/app operations:inventory")
}
