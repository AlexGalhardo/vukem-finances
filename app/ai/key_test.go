package ai_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/crypt"
	"github.com/AlexGalhardo/vukem/database"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

func newVault(t *testing.T, s site) *ai.Vault {
	t.Helper()

	v, err := ai.NewVault(s.app.App, false)
	if err != nil {
		t.Fatal(err)
	}

	return v
}

func saveTestKey(t *testing.T, v *ai.Vault, s site) models.AICredential {
	t.Helper()

	c, err := v.Save(s.ctx, s.user.ID, ai.SaveInput{Provider: ai.KindAnthropic, Key: ai.NewKey(testKey)})
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestKeySealing(t *testing.T) {
	t.Parallel()

	s := newSite(t, newApp(t, nil), "ana@example.com")
	v := newVault(t, s)
	c := saveTestKey(t, v, s)

	if strings.Contains(c.KeyCiphertext, testKey) || c.KeyCiphertext == "" {
		t.Fatalf("the ciphertext %q is empty or holds the key", c.KeyCiphertext)
	}

	if c.KeyHint != "WXYZ" {
		t.Errorf("hint = %q, want WXYZ", c.KeyHint)
	}

	// The row in the database holds no clear key either.
	var raw string
	if err := mustDB(t, s).QueryRowContext(t.Context(), "SELECT key_ciphertext FROM ai_credentials WHERE id = ?", c.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(raw, "0123456789") {
		t.Fatal("the key is in the database in clear")
	}

	active, ok, err := v.Active(s.ctx)
	if err != nil || !ok {
		t.Fatalf("Active = %v, %v", ok, err)
	}

	got, err := v.Open(active)
	if err != nil || got.Reveal() != testKey {
		t.Fatalf("Open = %v, %v", got.IsZero(), err)
	}

	// A new key revokes the previous one and erases its ciphertext.
	next, err := v.Save(s.ctx, s.user.ID, ai.SaveInput{Provider: ai.KindOpenAI, Key: ai.NewKey("sk-placeholder-other-0123456789-ABCD")})
	if err != nil {
		t.Fatal(err)
	}

	old, err := workspaces.Scoped[models.AICredential](s.ctx, mustDB(t, s)).Where("id", c.ID).First(t.Context())
	if err != nil || !old.RevokedAt.Valid || old.KeyCiphertext != "" {
		t.Fatalf("previous credential = %+v, %v; want revoked with no ciphertext", old, err)
	}

	if cur, _, _ := v.Active(s.ctx); cur.ID != next.ID {
		t.Errorf("active = %d, want %d", cur.ID, next.ID)
	}
}

func mustDB(t *testing.T, s site) *database.DB {
	t.Helper()

	db, err := s.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestKeyBoundToWorkspace(t *testing.T) {
	t.Parallel()

	app := newApp(t, nil)
	a := newSite(t, app, "ana@example.com")
	b := newSite(t, app, "bob@example.com")
	v := newVault(t, a)
	ca := saveTestKey(t, v, a)
	cb := saveTestKey(t, v, b)

	// The ciphertext of A's row copied onto B's row (and the reverse) must not open.
	moved := cb
	moved.KeyCiphertext = ca.KeyCiphertext

	if _, err := v.Open(moved); err == nil {
		t.Fatal("a ciphertext moved to another workspace's row opened")
	}

	// Moved to another row of the same workspace is refused too.
	sameWorkspace := ca
	sameWorkspace.ID = ca.ID + 100

	if _, err := v.Open(sameWorkspace); err == nil {
		t.Fatal("a ciphertext moved to another row id opened")
	}

	if _, err := v.Open(ca); err != nil {
		t.Fatalf("the original no longer opens: %v", err)
	}
}

func TestKeyRedacted(t *testing.T) {
	t.Parallel()

	k := ai.NewKey(testKey)

	for _, got := range []string{
		fmt.Sprintf("%v %+v %#v %s %q %x", k, k, k, k, k, k),
		fmt.Sprint(struct{ K ai.Key }{k}),
		fmt.Sprintf("%+v", struct{ K ai.Key }{k}),
		fmt.Sprintf("%+v", &struct{ K ai.Key }{k}),
	} {
		if strings.Contains(got, "0123456789") || strings.Contains(got, "WXYZ") {
			t.Errorf("formatted key leaks: %s", got)
		}
	}

	js, err := json.Marshal(map[string]any{"key": k, "nested": struct{ K ai.Key }{k}})
	if err != nil || strings.Contains(string(js), "0123456789") {
		t.Errorf("json = %s, %v", js, err)
	}

	var buf bytes.Buffer

	slog.New(slog.NewJSONHandler(&buf, nil)).Info("saved", "key", k)
	slog.New(slog.NewTextHandler(&buf, nil)).Info("saved", slog.Any("key", k))

	if strings.Contains(buf.String(), "0123456789") {
		t.Errorf("log leaks the key: %s", buf.String())
	}

	if k.Reveal() != testKey {
		t.Error("Reveal must still return the key")
	}

	if h := ai.NewKey("short").Hint(); h != "" {
		t.Errorf("a short key has the hint %q", h)
	}
}

func TestResealAfterRotation(t *testing.T) {
	t.Parallel()

	s := newSite(t, newApp(t, nil), "ana@example.com")
	oldKey := bytes.Repeat([]byte{1}, crypt.KeySize)
	newKey := bytes.Repeat([]byte{2}, crypt.KeySize)

	before, err := crypt.New(oldKey)
	if err != nil {
		t.Fatal(err)
	}

	v := newVault(t, s)
	v.Crypt = before
	c := saveTestKey(t, v, s)

	// After the rotation the new key seals and the old one only opens.
	after, err := crypt.New(newKey, oldKey)
	if err != nil {
		t.Fatal(err)
	}

	v.Crypt = after

	if !after.NeedsRotation(c.KeyCiphertext) {
		t.Fatal("the ciphertext should need rotation")
	}

	n, err := v.Reseal(t.Context())
	if err != nil || n != 1 {
		t.Fatalf("Reseal = %d, %v", n, err)
	}

	cur, _, _ := v.Active(s.ctx)
	if after.NeedsRotation(cur.KeyCiphertext) {
		t.Error("the ciphertext still needs rotation")
	}

	// Only the new key opens it now.
	if k, err := v.Open(cur); err != nil || k.Reveal() != testKey {
		t.Fatalf("Open after reseal: %v", err)
	}

	only, _ := crypt.New(newKey)
	v.Crypt = only

	if _, err := v.Open(cur); err != nil {
		t.Fatalf("the new key alone must open the resealed credential: %v", err)
	}

	if n, err := v.Reseal(t.Context()); err != nil || n != 0 {
		t.Errorf("a second Reseal changed %d, %v", n, err)
	}

}
