package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

const plansFile = "../site/generated/plans.json"

// The site reads the plan limits from a committed file: a changed limit fails
// here until `go run ./cmd/app plans:export` writes it again.
func TestPlansExportIsFresh(t *testing.T) {
	t.Parallel()

	body, err := bootstrap.PlansJSON()
	if err != nil {
		t.Fatal(err)
	}

	matchesFile(t, plansFile, body, "go run ./cmd/app plans:export")
}
