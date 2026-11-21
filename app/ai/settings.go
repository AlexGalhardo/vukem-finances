package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/AlexGalhardo/vukem/database"
	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"

	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Features an owner can switch on.
const (
	FeatureCategorize = "categorize"
	FeatureAssistant  = "assistant"
)

// Default and bounds of the monthly call limit.
const (
	DefaultMonthlyLimit = 2000
	maxMonthlyLimit     = 1_000_000
)

var (
	errConsent = errs.New(http.StatusUnprocessableEntity,
		"Confirm what is sent to the provider before turning AI on.")
	errLimit = errs.New(http.StatusUnprocessableEntity, "The monthly limit must be between 1 and 1,000,000 calls.")
)

// Settings is whether AI is on for a workspace. The zero value is the default:
// off, every feature off.
type Settings struct {
	Enabled          bool            `json:"enabled"`
	Features         map[string]bool `json:"features"`
	MonthlyCallLimit int             `json:"monthlyCallLimit"`
	EnabledAt        string          `json:"enabledAt"`
}

// FeatureOn reports whether AI and the feature are both on.
func (s Settings) FeatureOn(feature string) bool { return s.Enabled && s.Features[feature] }

// SettingsStore reads and writes workspace_ai_settings.
type SettingsStore struct {
	DB  *database.DB
	now func() time.Time
}

// NewSettingsStore builds the store; now defaults to the wall clock.
func NewSettingsStore(db *database.DB, now func() time.Time) *SettingsStore {
	if now == nil {
		now = time.Now
	}

	return &SettingsStore{DB: db, now: now}
}

func view(row models.WorkspaceAISettings) Settings {
	features := map[string]bool{FeatureCategorize: false, FeatureAssistant: false}

	var stored map[string]bool
	if json.Unmarshal([]byte(row.Features), &stored) == nil {
		for name := range features {
			features[name] = stored[name]
		}
	}

	out := Settings{Enabled: row.Enabled, Features: features, MonthlyCallLimit: row.MonthlyCallLimit}
	if row.EnabledAt.Valid {
		out.EnabledAt = row.EnabledAt.V.UTC().Format("2006-01-02 15:04")
	}

	return out
}

func (s *SettingsStore) row(ctx context.Context, ex database.Executor) (models.WorkspaceAISettings, bool, error) {
	row, err := workspaces.Scoped[models.WorkspaceAISettings](ctx, ex).First(ctx)
	if errors.Is(err, orm.ErrNotFound) {
		return models.WorkspaceAISettings{Features: "{}", MonthlyCallLimit: DefaultMonthlyLimit}, false, nil
	}

	return row, err == nil, err
}

// Get returns the settings of the workspace of ctx.
func (s *SettingsStore) Get(ctx context.Context) (Settings, error) {
	row, _, err := s.row(ctx, s.DB)

	return view(row), err
}

// UpdateInput is what the owner submits. Consent must be true to turn AI on.
type UpdateInput struct {
	Enabled          bool
	Features         map[string]bool
	MonthlyCallLimit int
	Consent          bool
}

// Update saves the settings of the workspace of ctx and writes the audit
// event. Turning AI on needs the consent flag; turning it off needs nothing
// and takes effect for the next call.
func (s *SettingsStore) Update(ctx context.Context, actor int64, in UpdateInput) (Settings, error) {
	if in.MonthlyCallLimit < 1 || in.MonthlyCallLimit > maxMonthlyLimit {
		return Settings{}, errLimit
	}

	var out Settings

	err := s.DB.Tx(ctx, func(tx *database.Tx) error {
		row, exists, err := s.row(ctx, tx)
		if err != nil {
			return err
		}

		turningOn := in.Enabled && !row.Enabled
		if turningOn && !in.Consent {
			return errConsent
		}

		features := map[string]bool{
			FeatureCategorize: in.Features[FeatureCategorize], FeatureAssistant: in.Features[FeatureAssistant],
		}

		encoded, err := json.Marshal(features)
		if err != nil {
			return err
		}

		sameState := row.Enabled == in.Enabled

		row.Enabled, row.Features, row.MonthlyCallLimit = in.Enabled, string(encoded), in.MonthlyCallLimit
		if turningOn {
			row.EnabledBy = actor
			row.EnabledAt = orm.NullTime{}
			row.EnabledAt.V, row.EnabledAt.Valid = s.now().UTC(), true
		}

		if exists {
			err = orm.Update(ctx, tx, &row)
		} else {
			err = orm.Create(ctx, tx, &row)
		}

		if err != nil {
			return err
		}

		action := AuditSettingsChange

		switch {
		case !sameState && in.Enabled:
			action = AuditEnabled
		case !sameState:
			action = AuditDisabled
		}

		out = view(row)

		return orm.Create(ctx, tx, &models.AuditEvent{ActorUserID: actor, Action: action, Subject: "ai"})
	})

	return out, err
}
