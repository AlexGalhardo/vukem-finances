package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/crypt"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Actions written to audit_events. None of them carries the key.
const (
	AuditKeySaved       = "ai.key_saved"
	AuditKeyDeleted     = "ai.key_deleted"
	AuditEnabled        = "ai.enabled"
	AuditDisabled       = "ai.disabled"
	AuditSettingsChange = "ai.settings_changed"
)

const maxKeyLength = 512

var (
	errNoKey      = errs.New(http.StatusUnprocessableEntity, "Enter the API key.")
	errKeyTooLong = errs.New(http.StatusUnprocessableEntity, "That does not look like an API key.")
	errProvider   = errs.New(http.StatusUnprocessableEntity, "Choose a supported provider.")
	errNoCrypt    = errs.New(http.StatusServiceUnavailable, "Encryption is not configured on this server (APP_KEY).")
	errNoRow      = errs.New(http.StatusNotFound, "Not found.")
)

// Vault stores the keys of the workspaces encrypted.
type Vault struct {
	DB    *database.DB
	Crypt *crypt.Encrypter
	// AllowPrivate is AI_ALLOW_PRIVATE_HOSTS in the personal edition.
	AllowPrivate bool

	now func() time.Time
}

// NewVault builds the vault from the app.
func NewVault(app *vukem.App, allowPrivate bool) (*Vault, error) {
	db, err := app.DB()
	if err != nil {
		return nil, err
	}

	return &Vault{DB: db, Crypt: app.Crypt, AllowPrivate: allowPrivate, now: app.Now}, nil
}

// additionalData binds a ciphertext to its workspace and row: moved to
// another row, it no longer opens.
func additionalData(workspaceID, credentialID int64) []byte {
	return []byte("ai-key:" + strconv.FormatInt(workspaceID, 10) + ":" + strconv.FormatInt(credentialID, 10))
}

// SaveInput is a key the owner submits.
type SaveInput struct {
	Provider string
	Label    string
	Key      Key
	BaseURL  string
	Model    string
}

// Save seals the key and makes it the active credential of the workspace of
// ctx; the previous one is revoked (its ciphertext is erased). The change and
// its audit event stand or fall together.
func (v *Vault) Save(ctx context.Context, actor int64, in SaveInput) (models.AICredential, error) {
	var cred models.AICredential

	workspace, ok := workspaces.TenantID(ctx)
	if !ok {
		return cred, errNoRow
	}

	if v.Crypt == nil {
		return cred, errNoCrypt
	}

	in.Provider = strings.TrimSpace(in.Provider)
	in.BaseURL = strings.TrimSpace(in.BaseURL)
	in.Key = NewKey(strings.TrimSpace(in.Key.Reveal()))

	switch {
	case !ValidKind(in.Provider):
		return cred, errProvider
	case in.Key.IsZero():
		return cred, errNoKey
	case len(in.Key.Reveal()) > maxKeyLength:
		return cred, errKeyTooLong
	}

	if err := CheckBaseURL(in.Provider, in.BaseURL, v.AllowPrivate); err != nil {
		return cred, err
	}

	err := v.DB.Tx(ctx, func(tx *database.Tx) error {
		_, err := workspaces.Scoped[models.AICredential](ctx, tx).WhereNull("revoked_at").
			Update(ctx, orm.Set{"revoked_at": v.clock(), "key_ciphertext": ""})
		if err != nil {
			return err
		}

		cred = models.AICredential{
			Provider: in.Provider, Label: truncate(strings.TrimSpace(in.Label), 80), KeyHint: in.Key.Hint(),
			BaseURL: in.BaseURL, DefaultModel: truncate(strings.TrimSpace(in.Model), 100), CreatedBy: actor,
		}
		if err := orm.Create(ctx, tx, &cred); err != nil {
			return err
		}

		cred.KeyCiphertext = v.Crypt.Seal([]byte(in.Key.Reveal()), additionalData(workspace, cred.ID))
		if err := orm.UpdateOnly(ctx, tx, &cred, "key_ciphertext"); err != nil {
			return err
		}

		return orm.Create(ctx, tx, &models.AuditEvent{ActorUserID: actor, Action: AuditKeySaved, Subject: in.Provider})
	})

	return cred, err
}

// Active returns the credential in use in the workspace of ctx.
func (v *Vault) Active(ctx context.Context) (models.AICredential, bool, error) {
	cred, err := workspaces.Scoped[models.AICredential](ctx, v.DB).WhereNull("revoked_at").
		OrderBy("id", orm.Desc).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return cred, false, nil
	}

	return cred, err == nil, err
}

// Open decrypts the key of cred. The key is meant to live only as long as the
// request it signs.
func (v *Vault) Open(cred models.AICredential) (Key, error) {
	if v.Crypt == nil {
		return Key{}, errNoCrypt
	}

	plain, err := v.Crypt.Open(cred.KeyCiphertext, additionalData(cred.WorkspaceID, cred.ID))
	if err != nil {
		return Key{}, fmt.Errorf("ai: open the key of credential %d: %w", cred.ID, err)
	}

	return NewKey(string(plain)), nil
}

// Revoke deletes the key of credential id in the workspace of ctx (404 for
// any other workspace's id): the row stays for the history, the ciphertext goes.
func (v *Vault) Revoke(ctx context.Context, actor, id int64) error {
	return v.DB.Tx(ctx, func(tx *database.Tx) error {
		cred, err := workspaces.Scoped[models.AICredential](ctx, tx).Where("id", id).WhereNull("revoked_at").First(ctx)
		if errors.Is(err, orm.ErrNotFound) {
			return errNoRow
		}

		if err != nil {
			return err
		}

		if _, err := workspaces.Scoped[models.AICredential](ctx, tx).Where("id", id).
			Update(ctx, orm.Set{"revoked_at": v.clock(), "key_ciphertext": ""}); err != nil {
			return err
		}

		return orm.Create(ctx, tx, &models.AuditEvent{ActorUserID: actor, Action: AuditKeyDeleted, Subject: cred.Provider})
	})
}

// Touch records that the credential was used.
func (v *Vault) Touch(ctx context.Context, id int64) error {
	_, err := workspaces.Scoped[models.AICredential](ctx, v.DB).Where("id", id).
		Update(ctx, orm.Set{"last_used_at": v.clock()})

	return err
}

// Reseal re-encrypts, with the current APP_KEY, every active credential of
// every workspace that was sealed with an older one. It returns how many it
// changed. A credential that does not open is left alone and counted in the error.
func (v *Vault) Reseal(ctx context.Context) (int, error) {
	if v.Crypt == nil {
		return 0, nil
	}

	creds, err := orm.QueryOf[models.AICredential](v.DB).WhereNull("revoked_at").Get(ctx)
	if err != nil {
		return 0, err
	}

	done, failed := 0, 0

	for _, c := range creds {
		if !v.Crypt.NeedsRotation(c.KeyCiphertext) {
			continue
		}

		sealed, err := v.Crypt.Rotate(c.KeyCiphertext, additionalData(c.WorkspaceID, c.ID))
		if err != nil {
			failed++
			continue
		}

		if _, err := orm.QueryOf[models.AICredential](v.DB).Where("id", c.ID).
			Update(ctx, orm.Set{"key_ciphertext": sealed}); err != nil {
			return done, err
		}

		done++
	}

	if failed > 0 {
		return done, fmt.Errorf("ai: %d credentials could not be re-sealed (is the old APP_KEY still in APP_PREVIOUS_KEYS?)", failed)
	}

	return done, nil
}

func (v *Vault) clock() time.Time {
	if v.now != nil {
		return v.now().UTC()
	}

	return time.Now().UTC()
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}

	return s
}
