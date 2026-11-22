package privacy_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestExport(t *testing.T) {
	t.Parallel()

	app := newApp(t)
	w := newWorld(t, app, "ana@example.com")

	// A workspace the person is only a member of must not leave with the export.
	other := newWorld(t, app, "bob@example.com")
	other.joinAs(t, w.owner.ID, "member")

	var buf bytes.Buffer
	if err := w.svc.Export(t.Context(), &buf, w.owner); err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Format      string                 `json:"format"`
		Profile     struct{ Email string } `json:"profile"`
		Memberships []struct {
			Role string `json:"role"`
		} `json:"memberships"`
		Workspaces []struct {
			Transactions []struct{ Description string } `json:"transactions"`
		} `json:"workspaces"`
	}

	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("the export is not JSON: %v\n%s", err, buf.String())
	}

	if doc.Format != "vukem-finances-personal-export" || doc.Profile.Email != "ana@example.com" {
		t.Errorf("unexpected header: %+v", doc)
	}

	if len(doc.Memberships) != 2 {
		t.Errorf("memberships = %d, want 2", len(doc.Memberships))
	}

	if len(doc.Workspaces) != 1 || len(doc.Workspaces[0].Transactions) != 1 || doc.Workspaces[0].Transactions[0].Description != "Coffee" {
		t.Errorf("only the owned workspace with its data must be exported: %+v", doc.Workspaces)
	}

	body := buf.String()

	for _, secret := range []string{secretHash, secretTwoFA, secretRecovery, secretAIKey, w.token, w.token[strings.Index(w.token, ".")+1:]} {
		if strings.Contains(body, secret) {
			t.Errorf("the export holds a secret value: %q", secret)
		}
	}

	for _, key := range []string{"password", "two_factor_secret", "recovery", "hash", "key_ciphertext"} {
		if strings.Contains(strings.ToLower(body), `"`+key) {
			t.Errorf("the export names a secret field: %s", key)
		}
	}
}
