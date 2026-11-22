package sync

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/AlexGalhardo/vukem/auth"
	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Error codes of the protocol that are not mutation errors.
const (
	CodeDeviceRevoked   = "device_revoked"
	CodeResyncRequired  = "resync_required"
	CodeUnsupported     = "protocol_unsupported"
	CodeBadRequest      = "bad_request"
	CodeBatchTooLarge   = "batch_too_large"
	CodeRateLimited     = "rate_limited"
	CodeInvalidSnapshot = "invalid_snapshot"
)

// Limits of the protocol, published by /info.
const (
	ProtocolVersion    = 1
	MaxMutations       = 200
	MaxBodyBytes       = 1 << 20
	DefaultPullLimit   = 500
	MaxPullLimit       = 1000
	DefaultSnapshotRow = 500
	MaxSnapshotRows    = 1000
	PullsPerMinute     = 60
	PushesPerMinute    = 30
	MutationsPerHour   = 5000
	OpRetention        = 30 * 24 * time.Hour
)

// Service is the sync protocol of one app.
type Service struct {
	DB  *database.DB
	WS  *workspaces.Service
	Fin *finance.Service
	Log *slog.Logger
	Now func() time.Time
	// Tokens carries the prefix, the clock and the logger of the app's tokens; Pair issues one.
	Tokens *auth.Tokens
}

// Actor is who calls: the person behind a token, in the workspace the token is
// bound to, with the role they hold there now.
type Actor struct {
	WorkspaceID int64
	UserID      int64
	TokenID     string
	Role        string
}

func fail(status int, code, message string) error {
	return errs.New(status, message).WithExtension("code", code)
}

var (
	errDeviceRevoked = fail(http.StatusForbidden, CodeDeviceRevoked, "This device is unknown or was revoked: stop syncing it.")
	errDeviceName    = fail(http.StatusUnprocessableEntity, "validation_failed", "Name the device with 1 to 80 characters.")
	errPlatform      = fail(http.StatusUnprocessableEntity, "validation_failed", "The platform must be ios, android, web or other.")
)

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}

	return time.Now().UTC()
}

func (s *Service) scoped(ctx context.Context, ex database.Executor) orm.Query[models.SyncDevice] {
	return workspaces.Scoped[models.SyncDevice](ctx, ex)
}

// RegisterDevice makes a device for the calling token.
func (s *Service) RegisterDevice(ctx context.Context, a Actor, name, platform string) (models.SyncDevice, error) {
	name = strings.TrimSpace(name)
	if n := utf8.RuneCountInString(name); n == 0 || n > 80 || !utf8.ValidString(name) {
		return models.SyncDevice{}, errDeviceName
	}

	if platform == "" {
		platform = models.PlatformOther
	}

	if !contains(models.DevicePlatforms(), platform) {
		return models.SyncDevice{}, errPlatform
	}

	d := models.SyncDevice{
		WorkspaceID: a.WorkspaceID, UserID: a.UserID, TokenID: a.TokenID,
		DeviceID: models.NewUID(), Name: name, Platform: platform,
	}

	if err := orm.Create(workspaces.WithTenant(ctx, a.WorkspaceID), s.DB, &d); err != nil {
		return models.SyncDevice{}, err
	}

	return d, nil
}

// Devices lists the devices of the caller in the workspace, newest first.
func (s *Service) Devices(ctx context.Context, a Actor) ([]models.SyncDevice, error) {
	ctx = workspaces.WithTenant(ctx, a.WorkspaceID)

	return s.scoped(ctx, s.DB).Where("user_id", a.UserID).OrderBy("id", orm.Desc).Get(ctx)
}

// Device finds the device an X-Device-Id names. An unknown id, one of another
// token, another workspace or a revoked device is the same answer: 403
// device_revoked. It also stamps last_seen_at.
func (s *Service) Device(ctx context.Context, a Actor, deviceID string) (models.SyncDevice, error) {
	ctx = workspaces.WithTenant(ctx, a.WorkspaceID)

	d, err := s.scoped(ctx, s.DB).Where("device_id", deviceID).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return models.SyncDevice{}, errDeviceRevoked
	}

	if err != nil {
		return models.SyncDevice{}, err
	}

	if d.RevokedAt.Valid || d.TokenID != a.TokenID || d.UserID != a.UserID {
		return models.SyncDevice{}, errDeviceRevoked
	}

	// last_seen_at is a hint for the devices page: write it at most once a
	// minute, and never wait for a busy writer to do it.
	now := s.now()
	if !d.LastSeenAt.Valid || now.Sub(d.LastSeenAt.V) > time.Minute {
		touch, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()

		if _, err := s.scoped(touch, s.DB).Where("id", d.ID).Update(touch, orm.Set{"last_seen_at": now}); err != nil && s.Log != nil {
			s.Log.WarnContext(ctx, "sync: could not record the last sight of a device", "device", d.DeviceID, "error", err)
		}
	}

	return d, nil
}

// RevokeDevice stops one device of a person; the device keeps its local data.
func (s *Service) RevokeDevice(ctx context.Context, workspaceID, userID int64, deviceID string) error {
	ctx = workspaces.WithTenant(ctx, workspaceID)

	n, err := s.scoped(ctx, s.DB).Where("device_id", deviceID).Where("user_id", userID).WhereNull("revoked_at").
		Update(ctx, orm.Set{"revoked_at": s.now()})
	if err == nil && n == 0 {
		return errs.ErrNotFound
	}

	return err
}

// RevokeTokenDevices revokes every device a token registered; the token page
// calls it when the person revokes the token.
func RevokeTokenDevices(ctx context.Context, ex database.Executor, tokenID string, at time.Time) error {
	_, err := ex.ExecContext(ctx,
		"UPDATE sync_devices SET revoked_at = ? WHERE token_id = ? AND revoked_at IS NULL", at, tokenID)

	return err
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}

	return false
}
