package tests

import (
	"os"
	"strings"
	"testing"
)

// TestDataLifecycle_BackupsAreDocumentedAsRetainedUntilRotated: deletion cannot
// reach a backup file, so the operator's guide must say that a snapshot keeps
// what was deleted until the operator rotates it, and the threat model lists it.
func TestDataLifecycle_BackupsAreDocumentedAsRetainedUntilRotated(t *testing.T) {
	t.Parallel()

	guide, err := os.ReadFile("../deploy/README.md")
	if err != nil {
		t.Fatal(err)
	}

	threats, err := os.ReadFile("../docs/security.md")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(strings.ToLower(string(threats)), "backups are retained until rotated") {
		t.Error("docs/security.md does not say that backups are retained until rotated")
	}

	text := strings.ToLower(string(guide))
	if !strings.Contains(text, "retained until") || !strings.Contains(text, "deleted") {
		t.Error("deploy/README.md does not say that a backup keeps deleted data until it is rotated")
	}
}
