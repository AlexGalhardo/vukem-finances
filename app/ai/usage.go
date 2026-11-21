package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Usage statuses. A reservation counts against the monthly limit while the
// call runs; a failed call is given back.
const (
	UsageReserved = "reserved"
	UsageOK       = "ok"
	UsageFailed   = "failed"
)

// CallsPerMinute is the per-workspace rate limit.
const CallsPerMinute = 20

// Refusals of Client.Call, before any request leaves the server.
var (
	ErrDisabled      = errors.New("AI is turned off for this workspace")
	ErrFeatureOff    = errors.New("this AI feature is turned off")
	ErrNoCredential  = errors.New("no AI key is saved for this workspace")
	ErrTooManyCalls  = errors.New("too many AI calls this minute: try again shortly")
	ErrMonthlyLimit  = errors.New("the monthly AI call limit is reached")
	errUsageRowLost  = errors.New("ai: usage reservation was not recorded")
	errNoProviderFor = errors.New("ai: no provider factory configured")
)

// ProviderConfig is what a provider needs to talk to its service. The key
// lives in it only for the length of one call.
type ProviderConfig struct {
	Kind    string
	Key     Key
	BaseURL string
	Model   string
}

// Factory builds the provider of a credential. NewProvider is the production one.
type Factory func(ProviderConfig) (Provider, error)

// Client is the one door every AI feature goes through.
type Client struct {
	DB       *database.DB
	Vault    *Vault
	Settings *SettingsStore
	Factory  Factory
	Log      *slog.Logger

	now  func() time.Time
	mu   sync.Mutex
	hits map[int64][]time.Time
}

// NewClient builds the client. now defaults to the wall clock and log to slog.Default.
func NewClient(db *database.DB, vault *Vault, settings *SettingsStore, factory Factory, log *slog.Logger, now func() time.Time) *Client {
	if now == nil {
		now = time.Now
	}

	if log == nil {
		log = slog.Default()
	}

	return &Client{DB: db, Vault: vault, Settings: settings, Factory: factory, Log: log, now: now, hits: map[int64][]time.Time{}}
}

// Call runs req for workspace after checking, in order: AI is on, the feature
// is on, a key is saved, the per-minute rate, the monthly limit. Only then does
// it call the provider, and it records the usage either way. A refusal makes no
// outbound request.
func (c *Client) Call(ctx context.Context, workspace int64, req Request) (Response, error) {
	ctx = workspaces.WithTenant(ctx, workspace)

	settings, err := c.Settings.Get(ctx)
	if err != nil {
		return Response{}, err
	}

	if !settings.Enabled {
		return Response{}, ErrDisabled
	}

	if !settings.FeatureOn(req.Feature) {
		return Response{}, ErrFeatureOff
	}

	cred, ok, err := c.Vault.Active(ctx)
	if err != nil {
		return Response{}, err
	}

	if !ok {
		return Response{}, ErrNoCredential
	}

	if !c.allow(workspace) {
		return Response{}, ErrTooManyCalls
	}

	month, err := c.month(ctx)
	if err != nil {
		return Response{}, err
	}

	id, err := c.reserve(ctx, workspace, month, req.Feature, cred, settings.MonthlyCallLimit)
	if err != nil {
		return Response{}, err
	}

	resp, err := c.run(ctx, cred, req)
	if err != nil {
		c.finish(ctx, id, UsageFailed, Response{})
		c.logFailure(ctx, cred.Provider, req.Feature, err)

		return Response{}, publicError(err)
	}

	c.finish(ctx, id, UsageOK, resp)

	if err := c.Vault.Touch(ctx, cred.ID); err != nil {
		c.Log.WarnContext(ctx, "ai: could not record the key use", "workspace", workspace)
	}

	return resp, nil
}

// Check calls the cheapest endpoint of the workspace's saved key and counts
// nothing against the monthly limit. The owner uses it from the settings page.
func (c *Client) Check(ctx context.Context, workspace int64) error {
	ctx = workspaces.WithTenant(ctx, workspace)

	cred, ok, err := c.Vault.Active(ctx)
	if err != nil {
		return err
	}

	if !ok {
		return ErrNoCredential
	}

	if !c.allow(workspace) {
		return ErrTooManyCalls
	}

	key, err := c.Vault.Open(cred)
	if err != nil {
		return err
	}

	p, err := c.provider(cred, key)
	if err != nil {
		return err
	}

	if err := p.Check(ctx); err != nil {
		c.logFailure(ctx, cred.Provider, "check", err)
		return publicError(err)
	}

	return nil
}

func (c *Client) provider(cred models.AICredential, key Key) (Provider, error) {
	if c.Factory == nil {
		return nil, errNoProviderFor
	}

	return c.Factory(ProviderConfig{Kind: cred.Provider, Key: key, BaseURL: cred.BaseURL, Model: cred.DefaultModel})
}

// run opens the key, builds the provider and makes the call. The key is a
// local of this function and goes out of scope with it.
func (c *Client) run(ctx context.Context, cred models.AICredential, req Request) (Response, error) {
	key, err := c.Vault.Open(cred)
	if err != nil {
		return Response{}, err
	}

	p, err := c.provider(cred, key)
	if err != nil {
		return Response{}, err
	}

	return p.Complete(ctx, req)
}

// allow is a sliding window of CallsPerMinute per workspace, in memory: the
// app runs as one replica.
func (c *Client) allow(workspace int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	kept := c.hits[workspace][:0]

	for _, at := range c.hits[workspace] {
		if now.Sub(at) < time.Minute {
			kept = append(kept, at)
		}
	}

	if len(kept) >= CallsPerMinute {
		c.hits[workspace] = kept
		return false
	}

	c.hits[workspace] = append(kept, now)

	return true
}

// month is "2026-11" in the timezone of the workspace.
func (c *Client) month(ctx context.Context) (string, error) {
	id, _ := workspaces.TenantID(ctx)

	ws, err := orm.QueryOf[models.Workspace](c.DB).Find(ctx, id)
	if err != nil {
		return "", fmt.Errorf("ai: load the workspace: %w", err)
	}

	loc, err := time.LoadLocation(ws.Timezone)
	if err != nil {
		loc = time.UTC
	}

	return c.now().In(loc).Format("2006-01"), nil
}

// reserve counts the call against the monthly limit and records it in one
// statement: the insert happens only when fewer than limit calls are counted,
// so concurrent calls at limit minus one let exactly one through.
func (c *Client) reserve(
	ctx context.Context, workspace int64, month, feature string, cred models.AICredential, limit int,
) (int64, error) {
	now := c.now().UTC()

	res, err := c.DB.ExecContext(ctx, `INSERT INTO ai_usage
		(workspace_id, month, feature, provider, model, input_tokens, output_tokens, status, created_at, updated_at)
		SELECT ?, ?, ?, ?, ?, 0, 0, ?, ?, ?
		WHERE (SELECT COUNT(*) FROM ai_usage WHERE workspace_id = ? AND month = ? AND status IN (?, ?)) < ?`,
		workspace, month, feature, cred.Provider, cred.DefaultModel, UsageReserved, now, now,
		workspace, month, UsageReserved, UsageOK, limit)
	if err != nil {
		return 0, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	if n == 0 {
		return 0, ErrMonthlyLimit
	}

	id, err := res.LastInsertId()
	if err != nil || id == 0 {
		return 0, errUsageRowLost
	}

	return id, nil
}

func (c *Client) finish(ctx context.Context, id int64, status string, resp Response) {
	set := orm.Set{"status": status, "input_tokens": resp.InputTokens, "output_tokens": resp.OutputTokens}
	if resp.Model != "" {
		set["model"] = resp.Model
	}

	if _, err := workspaces.Scoped[models.AIUsage](ctx, c.DB).Where("id", id).Update(ctx, set); err != nil {
		c.Log.ErrorContext(ctx, "ai: could not finalize the usage row", "usage", id)
	}
}

// publicError is what a caller may show: one of the generic sentinels. Anything
// else (a transport error, a decryption failure) becomes ErrUnavailable, so no
// text from upstream or from a failed request travels on.
func publicError(err error) error {
	for _, known := range []error{ErrRejected, ErrRateLimited, ErrBadRequest, ErrUnavailable} {
		if errors.Is(err, known) {
			return known
		}
	}

	return ErrUnavailable
}

var errorKind = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,39}$`)

// logFailure writes what is safe to keep: the provider, the feature and the
// error class. Never the text of the error: a provider can echo the key or the
// prompt in it. A diagnostic carried by a ProviderError (status, error type) is
// the exception, since it is checked against a strict shape first.
func (c *Client) logFailure(ctx context.Context, provider, feature string, err error) {
	attrs := []any{"provider", provider, "feature", feature, "class", publicError(err).Error()}

	if pe, ok := errors.AsType[*ProviderError](err); ok {
		attrs = append(attrs, "status", pe.Status)
		if errorKind.MatchString(strings.ToLower(pe.Type)) {
			attrs = append(attrs, "type", strings.ToLower(pe.Type))
		}
	}

	c.Log.WarnContext(ctx, "ai call failed", attrs...)
}

// ProviderError is a failed provider response. Its text is generic; Status and
// Type (the provider's error type, kept only when it has a plain shape) are for logs.
type ProviderError struct {
	Status int
	Type   string
	Class  error
}

// Error implements error with the generic text only.
func (e *ProviderError) Error() string { return e.Class.Error() }

// Unwrap lets errors.Is match the generic sentinel.
func (e *ProviderError) Unwrap() error { return e.Class }

// MonthUsage is a summary of the calls of a month, for the settings page.
type MonthUsage struct {
	Month        string       `json:"month"`
	Calls        int64        `json:"calls"`
	InputTokens  int64        `json:"inputTokens"`
	OutputTokens int64        `json:"outputTokens"`
	ByFeature    []FeatureUse `json:"byFeature"`
}

// FeatureUse is the usage of one feature in a month.
type FeatureUse struct {
	Feature      string `json:"feature"`
	Provider     string `json:"provider"`
	Calls        int64  `json:"calls"`
	InputTokens  int64  `json:"inputTokens"`
	OutputTokens int64  `json:"outputTokens"`
}

// Usage sums the counted calls of this month of the workspace of ctx.
func (c *Client) Usage(ctx context.Context) (MonthUsage, error) {
	month, err := c.month(ctx)
	if err != nil {
		return MonthUsage{}, err
	}

	rows, err := workspaces.Scoped[models.AIUsage](ctx, c.DB).Where("month", month).
		WhereIn("status", UsageReserved, UsageOK).Get(ctx)
	if err != nil {
		return MonthUsage{}, err
	}

	out := MonthUsage{Month: month, ByFeature: []FeatureUse{}}
	index := map[string]int{}

	for _, r := range rows {
		key := r.Feature + "/" + r.Provider

		i, seen := index[key]
		if !seen {
			i = len(out.ByFeature)
			index[key] = i
			out.ByFeature = append(out.ByFeature, FeatureUse{Feature: r.Feature, Provider: r.Provider})
		}

		out.ByFeature[i].Calls++
		out.ByFeature[i].InputTokens += int64(r.InputTokens)
		out.ByFeature[i].OutputTokens += int64(r.OutputTokens)
		out.Calls++
		out.InputTokens += int64(r.InputTokens)
		out.OutputTokens += int64(r.OutputTokens)
	}

	return out, nil
}
