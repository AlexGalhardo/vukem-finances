package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/ratelimit"
	"github.com/AlexGalhardo/vukem/router"
	"github.com/AlexGalhardo/vukem/web"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	syncproto "github.com/AlexGalhardo/vukem-finances/app/sync"
)

// Headers of the sync protocol.
const (
	HeaderSyncProtocol = "X-Sync-Protocol"
	HeaderDeviceID     = "X-Device-Id"
)

// SyncDeviceIn is the body of sync.devices.create.
type SyncDeviceIn struct {
	Name     string `json:"name"     validate:"required,max=80" doc:"What the person calls the device" example:"Ana's phone"`
	Platform string `json:"platform" validate:"max=20"          doc:"ios, android, web or other" example:"ios"`
}

// SyncPairIn is the body of sync.pair.
type SyncPairIn struct {
	Code       string `json:"code"        validate:"required,max=40" doc:"The pairing code the Devices page shows" example:"7K2M-9QXA"`
	DeviceName string `json:"device_name" validate:"required,max=80" doc:"What the person calls the device" example:"Ana's phone"`
	Platform   string `json:"platform"    validate:"max=20"          doc:"ios, android, web or other" example:"android"`
}

// SyncDeviceOut is a device that syncs the workspace.
type SyncDeviceOut struct {
	DeviceID    string  `json:"device_id"     example:"01JD5Z8K3M4N5P6Q7R8S9T0V1W"`
	Name        string  `json:"name"          example:"Ana's phone"`
	Platform    string  `json:"platform"      example:"ios"`
	CreatedAt   string  `json:"created_at"    example:"2026-11-21T12:00:00Z"`
	LastSeenAt  *string `json:"last_seen_at"`
	LastPullSeq int64   `json:"last_pull_seq" example:"1042"`
	RevokedAt   *string `json:"revoked_at"`
}

// SyncInfoIn is the empty input of sync.info.
type SyncInfoIn struct{}

// SyncDevicesIn is the empty input of sync.devices.list.
type SyncDevicesIn struct{}

// SyncPullIn is the query of sync.pull.
type SyncPullIn struct {
	Cursor int64 `json:"cursor" doc:"The next_cursor of the previous page; 0 for the first pull" example:"0"`
	Limit  int   `json:"limit"  doc:"Changes per page: 500 by default, at most 1000" example:"500"`
}

// SyncSnapshotIn is the query of sync.snapshot.
type SyncSnapshotIn struct {
	Entity     string `json:"entity"      validate:"required,max=40" doc:"The entity to download, one of GET /sync/info" example:"account"`
	After      string `json:"after"       validate:"max=26"          doc:"The last uid of the previous page" example:""`
	Limit      int    `json:"limit"       doc:"Rows per page: 500 by default, at most 1000" example:"500"`
	SnapshotID string `json:"snapshot_id" validate:"max=60"          doc:"The snapshot_id of the first page, sent back on every later one" example:""`
}

// SyncPushIn is the body of sync.push.
type SyncPushIn struct {
	DeviceID  string                 `json:"device_id"  validate:"max=26" doc:"The device; X-Device-Id is what counts and this must match when set"`
	Mutations []syncproto.MutationIn `json:"mutations"  doc:"At most 200 mutations"`
}

// SyncPushOut is one result per mutation, in request order.
type SyncPushOut struct {
	Results []syncproto.Result `json:"results"`
}

func deviceOut(d models.SyncDevice) SyncDeviceOut {
	return SyncDeviceOut{
		DeviceID: d.DeviceID, Name: d.Name, Platform: d.Platform, CreatedAt: stamp(d.CreatedAt),
		LastSeenAt: stampNull(d.LastSeenAt), LastPullSeq: d.LastPullSeq, RevokedAt: stampNull(d.RevokedAt),
	}
}

// syncActor checks the protocol version and returns who calls.
func (a *API) syncActor(c *web.Ctx) (syncproto.Actor, error) {
	raw := strings.TrimSpace(c.Header(HeaderSyncProtocol))

	if raw == "" {
		return syncproto.Actor{}, errs.New(http.StatusBadRequest, "Send the header "+HeaderSyncProtocol+": 1.").
			WithExtension("code", CodeBadRequest)
	}

	version, err := strconv.Atoi(raw)
	if err != nil || version < syncproto.ProtocolVersion || version > syncproto.ProtocolVersion {
		return syncproto.Actor{}, errs.New(http.StatusUpgradeRequired, "This server speaks sync protocol 1.").
			WithExtension("code", syncproto.CodeUnsupported).
			WithExtension("min", syncproto.ProtocolVersion).WithExtension("max", syncproto.ProtocolVersion)
	}

	p, ok := PrincipalFrom(c.Context())
	if !ok {
		return syncproto.Actor{}, errs.New(http.StatusUnauthorized, "A valid token is required.").WithExtension("code", CodeUnauthorized)
	}

	return syncproto.Actor{WorkspaceID: p.WorkspaceID, UserID: p.User.ID, TokenID: p.Token.ID, Role: p.Role}, nil
}

// syncDevice is syncActor plus the device of X-Device-Id, held to limiter.
func (a *API) syncDevice(c *web.Ctx, limiter ratelimit.Limiter) (syncproto.Actor, models.SyncDevice, error) {
	actor, err := a.syncActor(c)
	if err != nil {
		return actor, models.SyncDevice{}, err
	}

	dev, err := a.Sync.Device(c.Context(), actor, strings.TrimSpace(c.Header(HeaderDeviceID)))
	if err != nil {
		return actor, dev, err
	}

	if limiter != nil {
		res, err := limiter.Allow(c.Context(), "sync:"+dev.DeviceID)
		if err != nil {
			return actor, dev, errs.New(http.StatusServiceUnavailable, "Try again in a moment.").WithExtension("code", CodeUnavailable)
		}

		if !res.Allowed {
			c.Writer().Header().Set("Retry-After", strconv.Itoa(int(res.RetryAfter.Seconds())+1))

			return actor, dev, errs.New(http.StatusTooManyRequests, "Too many requests.").WithExtension("code", CodeRateLimited)
		}
	}

	return actor, dev, nil
}

func (a *API) syncLog(c *web.Ctx, what string, dev models.SyncDevice, started time.Time, attrs ...any) {
	if a.Log == nil {
		return
	}

	attrs = append([]any{"workspace", dev.WorkspaceID, "device", dev.DeviceID, "ms", time.Since(started).Milliseconds()}, attrs...)
	a.Log.InfoContext(c.Context(), "sync "+what, attrs...)
}

func (a *API) syncInfo(c *web.Ctx, _ SyncInfoIn) (syncproto.Info, error) {
	actor, err := a.syncActor(c)
	if err != nil {
		return syncproto.Info{}, err
	}

	return a.Sync.Info(c.Context(), actor)
}

func (a *API) syncDevicesCreate(c *web.Ctx, in SyncDeviceIn) (SyncDeviceOut, error) {
	actor, err := a.syncActor(c)
	if err != nil {
		return SyncDeviceOut{}, err
	}

	d, err := a.Sync.RegisterDevice(c.Context(), actor, in.Name, in.Platform)
	if err != nil {
		return SyncDeviceOut{}, err
	}

	return deviceOut(d), nil
}

func (a *API) syncDevicesList(c *web.Ctx, _ SyncDevicesIn) (List[SyncDeviceOut], error) {
	actor, err := a.syncActor(c)
	if err != nil {
		return List[SyncDeviceOut]{}, err
	}

	devices, err := a.Sync.Devices(c.Context(), actor)
	if err != nil {
		return List[SyncDeviceOut]{}, err
	}

	out := make([]SyncDeviceOut, 0, len(devices))
	for _, d := range devices {
		out = append(out, deviceOut(d))
	}

	return List[SyncDeviceOut]{Data: out}, nil
}

func (a *API) syncPull(c *web.Ctx, in SyncPullIn) (syncproto.PullResult, error) {
	started := time.Now()

	actor, dev, err := a.syncDevice(c, a.PullLimiter)
	if err != nil {
		return syncproto.PullResult{}, err
	}

	res, err := a.Sync.Pull(c.Context(), actor, dev, in.Cursor, in.Limit)
	if err == nil {
		a.syncLog(c, "pull", dev, started, "changes", len(res.Changes), "cursor", in.Cursor, "next", res.NextCursor)
	}

	return res, err
}

func (a *API) syncSnapshot(c *web.Ctx, in SyncSnapshotIn) (syncproto.SnapshotPage, error) {
	started := time.Now()

	actor, dev, err := a.syncDevice(c, a.PullLimiter)
	if err != nil {
		return syncproto.SnapshotPage{}, err
	}

	page, err := a.Sync.Snapshot(c.Context(), actor, in.Entity, in.After, in.SnapshotID, in.Limit)
	if err == nil {
		a.syncLog(c, "snapshot", dev, started, "entity", in.Entity, "rows", len(page.Rows))
	}

	return page, err
}

func (a *API) syncPush(c *web.Ctx, in SyncPushIn) (SyncPushOut, error) {
	started := time.Now()

	actor, dev, err := a.syncDevice(c, a.PushLimiter)
	if err != nil {
		return SyncPushOut{}, err
	}

	if in.DeviceID != "" && in.DeviceID != dev.DeviceID {
		return SyncPushOut{}, errs.New(http.StatusBadRequest, "device_id does not match "+HeaderDeviceID+".").WithExtension("code", CodeBadRequest)
	}

	results, err := a.Sync.Push(c.Context(), actor, dev, in.Mutations)
	if err != nil {
		return SyncPushOut{}, err
	}

	rejected := 0

	for _, r := range results {
		if r.Status == syncproto.StatusRejected {
			rejected++
		}
	}

	a.syncLog(c, "push", dev, started, "mutations", len(in.Mutations), "rejected", rejected)

	return SyncPushOut{Results: results}, nil
}

// syncPair is the one operation that takes no token: the code is the credential.
// Every attempt of an address counts, and so does every failed attempt of the
// whole server, so a code cannot be guessed. The code and the token are never logged.
func (a *API) syncPair(c *web.Ctx, in SyncPairIn) (syncproto.PairResult, error) {
	ctx := c.Context()

	if a.PairLimiter != nil {
		res, err := a.PairLimiter.Allow(ctx, "pair:"+ratelimit.ByIP(c.Request()))
		if err != nil {
			return syncproto.PairResult{}, errs.New(http.StatusServiceUnavailable, "Try again in a moment.").WithExtension("code", CodeUnavailable)
		}

		if !res.Allowed {
			c.Writer().Header().Set("Retry-After", strconv.Itoa(int(res.RetryAfter.Seconds())+1))

			return syncproto.PairResult{}, errs.New(http.StatusTooManyRequests, "Too many requests.").WithExtension("code", CodeRateLimited)
		}
	}

	if err := pairFailureGate(ctx, a.PairFailLimiter, c.Writer().Header()); err != nil {
		return syncproto.PairResult{}, err
	}

	out, err := a.Sync.Pair(ctx, in.Code, in.DeviceName, in.Platform)
	if err != nil {
		if a.PairFailLimiter != nil && syncproto.IsInvalidPairing(err) {
			_, _ = a.PairFailLimiter.Allow(ctx, "pair-failed")
		}

		return syncproto.PairResult{}, err
	}

	c.Writer().Header().Set("Cache-Control", "no-store")

	return out, nil
}

// pairFailureGate holds pairing to the server-wide cap on failed attempts. The cap needs a limiter that
// can look without spending (a Peeker). One that cannot would leave the cap off without a word, so the
// gate fails closed with 503 instead. A nil limiter means the cap is not configured at all.
func pairFailureGate(ctx context.Context, l ratelimit.Limiter, header http.Header) error {
	if l == nil {
		return nil
	}

	unavailable := errs.New(http.StatusServiceUnavailable, "Try again in a moment.").WithExtension("code", CodeUnavailable)

	peeker, ok := l.(ratelimit.Peeker)
	if !ok {
		return unavailable
	}

	res, err := peeker.Peek(ctx, "pair-failed")
	if err != nil {
		return unavailable
	}

	if !res.Allowed {
		header.Set("Retry-After", strconv.Itoa(int(res.RetryAfter.Seconds())+1))

		return errs.New(http.StatusTooManyRequests, "Too many requests.").WithExtension("code", CodeRateLimited)
	}

	return nil
}

// PairRoutes registers sync.pair outside the authenticated group.
func (a *API) PairRoutes(g *router.Group) {
	register(a, g, nil, "sync.pair", a.syncPair)
}

func (a *API) syncRoutes(g *router.Group, idem router.Middleware) {
	register(a, g, idem, "sync.info", a.syncInfo)
	register(a, g, idem, "sync.devices.create", a.syncDevicesCreate)
	register(a, g, idem, "sync.devices.list", a.syncDevicesList)
	register(a, g, idem, "sync.pull", a.syncPull)
	register(a, g, idem, "sync.snapshot", a.syncSnapshot)
	register(a, g, idem, "sync.push", a.syncPush)
}
