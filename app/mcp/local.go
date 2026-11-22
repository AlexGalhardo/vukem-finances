package mcp

import (
	"context"
	"slices"
	"time"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

func timeNow() time.Time { return time.Now().UTC() }

// Scopes are the scopes the local owner acts with: those of every tool this
// configuration offers and of every read operation (the resources read them).
// A write scope is never held unless a toolset offers a write tool, so the
// default server cannot write even if a tool were registered by mistake.
func Scopes(cat *operations.Catalog, cfg config.MCP) []string {
	var out []string

	edition := cat.Config().Edition

	for _, m := range cat.Operations() {
		if m.Kind == operations.KindRead && m.Offered(operations.SurfaceMCP) || Offered(cfg, edition, m) {
			if !slices.Contains(out, m.Scope) {
				out = append(out, m.Scope)
			}
		}
	}

	slices.Sort(out)

	return out
}

// LocalActor is the actor of the person at the console of a personal
// installation. It resolves the owner on every call, so nothing is cached.
func LocalActor(ws *workspaces.Service, cat *operations.Catalog, cfg config.MCP) ActorFunc {
	scopes := Scopes(cat, cfg)

	return func(ctx context.Context) (operations.Actor, error) {
		return operations.LocalOwner(ctx, ws, operations.SurfaceMCP, scopes)
	}
}
