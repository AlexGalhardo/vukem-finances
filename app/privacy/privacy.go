package privacy

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// GracePeriod is how long a deletion waits before the purge removes the rows.
const GracePeriod = 30 * 24 * time.Hour

// Audit events. The record holds the event, the time and a hashed subject.
const (
	EventWorkspaceRequested = "workspace.deletion_requested"
	EventWorkspaceCancelled = "workspace.deletion_cancelled"
	EventWorkspacePurged    = "workspace.purged"
	EventAccountRequested   = "account.deletion_requested"
	EventAccountCancelled   = "account.deletion_cancelled"
	EventAccountPurged      = "account.purged"
	EventLocalErased        = "local.erased"
)

// SubscriptionCanceller cancels what a workspace pays for. It must be
// idempotent: a second call for a canceled subscription is a no-op.
type SubscriptionCanceller interface {
	CancelWorkspaceSubscriptions(ctx context.Context, workspaceID int64) error
}

// AccessRevoker ends every credential of a person at once: sessions, remember
// tokens, API tokens and OAuth grants.
type AccessRevoker interface {
	RevokeAllAccess(ctx context.Context, userID int64) error
}

// Service holds the deletion, export and purge rules.
type Service struct {
	DB  *database.DB
	Log *slog.Logger
	WS  *workspaces.Service
	// Fin builds the workspace part of the export; the purge does not need it.
	Fin *finance.Service
	// Billing is nil in the personal edition, which has no billing.
	Billing SubscriptionCanceller
	// Access is the revocation of a person's credentials.
	Access AccessRevoker
	// RetentionYears is PAYMENT_RECORD_RETENTION_YEARS.
	RetentionYears int

	key []byte
	now func() time.Time
}

// New builds the service from the app. The caller sets Billing (saas) and
// Access, which live in packages this one does not import.
func New(app *vukem.App, ws *workspaces.Service) (*Service, error) {
	years, err := config.PaymentRecordRetentionYears(app.Source)
	if err != nil {
		return nil, err
	}

	key, err := app.DeriveKey("privacy.subject", 32)
	if err != nil {
		return nil, err
	}

	return &Service{DB: ws.DB, Log: app.Log, WS: ws, RetentionYears: years, key: key, now: app.Now}, nil
}

// Ref is the non-reversible reference of a subject ("workspace", "user"): an
// HMAC under a key derived from APP_KEY, so a reference cannot be turned back
// into an id by trying ids, and cannot be matched without the key.
func (s *Service) Ref(kind string, id int64) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(kind + ":" + strconv.FormatInt(id, 10)))

	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Service) record(ctx context.Context, ex database.Executor, event, ref string) error {
	return orm.Create(ctx, ex, &models.PrivacyAudit{Event: event, SubjectRef: ref})
}

// Audit lists the audit events of a subject reference, oldest first.
func (s *Service) Audit(ctx context.Context, ref string) ([]models.PrivacyAudit, error) {
	return orm.QueryOf[models.PrivacyAudit](s.DB).Where("subject_ref", ref).OrderBy("id", orm.Asc).Get(ctx)
}
