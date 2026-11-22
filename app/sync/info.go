package sync

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Range is the protocol versions a server speaks.
type Range struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// WorkspaceInfo names the workspace a token syncs.
type WorkspaceInfo struct {
	UID      string `json:"uid"`
	Currency string `json:"currency"`
	Timezone string `json:"timezone"`
}

// RefInfo is a reference field and the entity it points at.
type RefInfo struct {
	Field  string `json:"field"`
	Entity string `json:"entity"`
}

// EntityInfo is one synced entity as the server knows it.
type EntityInfo struct {
	Name   string      `json:"name"`
	Fields []FieldSpec `json:"fields"`
	Refs   []RefInfo   `json:"refs"`
}

// LimitsInfo is what the server enforces.
type LimitsInfo struct {
	MaxMutations       int   `json:"max_mutations"`
	MaxBodyBytes       int   `json:"max_body_bytes"`
	PullDefaultLimit   int   `json:"pull_default_limit"`
	PullMaxLimit       int   `json:"pull_max_limit"`
	SnapshotMaxRows    int   `json:"snapshot_max_rows"`
	PullsPerMinute     int   `json:"pulls_per_minute"`
	PushesPerMinute    int   `json:"pushes_per_minute"`
	MutationsPerHour   int   `json:"mutations_per_hour"`
	OpRetentionSeconds int64 `json:"op_retention_seconds"`
}

// Info is the answer of GET /info.
type Info struct {
	Protocol   Range         `json:"protocol"`
	ServerTime time.Time     `json:"server_time"`
	Workspace  WorkspaceInfo `json:"workspace"`
	Entities   []EntityInfo  `json:"entities"`
	Limits     LimitsInfo    `json:"limits"`
}

// Info describes the protocol, the workspace and every entity of the registry
// in dependency order. It fails when a registered field has no type: a client
// must never be told about a field the server cannot validate.
func (s *Service) Info(ctx context.Context, a Actor) (Info, error) {
	ctx = workspaces.WithTenant(ctx, a.WorkspaceID)

	ws, err := s.WS.Current(ctx)
	if err != nil {
		return Info{}, err
	}

	info := Info{
		Protocol:   Range{Min: ProtocolVersion, Max: ProtocolVersion},
		ServerTime: s.now(),
		Workspace:  WorkspaceInfo{UID: strconv.FormatInt(ws.ID, 10), Currency: ws.Currency, Timezone: ws.Timezone},
		Limits: LimitsInfo{
			MaxMutations: MaxMutations, MaxBodyBytes: MaxBodyBytes, PullDefaultLimit: DefaultPullLimit,
			PullMaxLimit: MaxPullLimit, SnapshotMaxRows: MaxSnapshotRows, PullsPerMinute: PullsPerMinute,
			PushesPerMinute: PushesPerMinute, MutationsPerHour: MutationsPerHour,
			OpRetentionSeconds: int64(OpRetention / time.Second),
		},
	}

	for _, e := range Order() {
		fields, missing := Fields(e)
		if len(missing) > 0 {
			return Info{}, fmt.Errorf("sync: entity %s has fields without a type: %v", e.Name, missing)
		}

		ei := EntityInfo{Name: e.Name, Fields: fields, Refs: []RefInfo{}}

		for _, r := range e.Refs {
			ei.Refs = append(ei.Refs, RefInfo{Field: wireName(r.Column, true), Entity: r.Entity})
		}

		info.Entities = append(info.Entities, ei)
	}

	return info, nil
}
