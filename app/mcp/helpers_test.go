package mcp_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	vmcp "github.com/AlexGalhardo/vukem/mcp"
	"github.com/AlexGalhardo/vukem/mcp/mcptest"
	"github.com/AlexGalhardo/vukem/orm"
	"github.com/AlexGalhardo/vukem/vukemtest"

	appmcp "github.com/AlexGalhardo/vukem-finances/app/mcp"
	"github.com/AlexGalhardo/vukem-finances/app/models"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/bootstrap"
	"github.com/AlexGalhardo/vukem-finances/config"
)

type fixture struct {
	app    *vukemtest.App
	cat    *operations.Catalog
	srv    *vmcp.Server
	client *mcptest.Client
	owner  models.User
}

// newFixture signs up the owner of a personal installation and connects an
// in-memory client to a server with cfg. prepare may add operations to the
// catalog before the tools are built.
func newFixture(t *testing.T, cfg config.MCP, prepare func(*operations.Catalog), opts ...mcptest.Option) *fixture {
	t.Helper()

	return newSizedFixture(t, cfg, 500, prepare, opts...)
}

// newSizedFixture is newFixture with the page size of the lists.
func newSizedFixture(t *testing.T, cfg config.MCP, pageSize int, prepare func(*operations.Catalog), opts ...mcptest.Option) *fixture {
	t.Helper()

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{Env: map[string]string{"APP_EDITION": "personal"}})

	db, err := app.DB()
	if err != nil {
		t.Fatal(err)
	}

	u := models.User{Name: "Ana", Email: "ana@example.com", Password: strings.Repeat("!", 8)}
	u.EmailVerifiedAt.V, u.EmailVerifiedAt.Valid = time.Now().UTC(), true

	if err := orm.Create(t.Context(), db, &u); err != nil {
		t.Fatal(err)
	}

	ws, err := workspaces.New(app.App)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ws.Create(t.Context(), u, "Ana household", "USD"); err != nil {
		t.Fatal(err)
	}

	cat, _, err := appmcp.CatalogOf(app.App)
	if err != nil {
		t.Fatal(err)
	}

	if prepare != nil {
		prepare(cat)
	}

	srv, err := appmcp.New(appmcp.Options{
		Catalog: cat, Actor: appmcp.LocalActor(ws, cat, cfg), Config: cfg, Version: "1.2.3", PageSize: pageSize,
	})
	if err != nil {
		t.Fatal(err)
	}

	return &fixture{app: app, cat: cat, srv: srv, client: mcptest.NewClient(t, srv, opts...), owner: u}
}

func readOnly() config.MCP { return config.MCP{Toolsets: []string{"read"}} }

func everything(destructive bool) config.MCP {
	return config.MCP{Toolsets: []string{"all"}, AllowDestructive: destructive}
}

// seed runs an operation as the owner with every scope, outside the server.
func (f *fixture) seed(t *testing.T, op string, input any) json.RawMessage {
	t.Helper()

	var scopes []string
	for _, s := range operations.Scopes() {
		scopes = append(scopes, s.Name)
	}

	ws, err := workspaces.New(f.app.App)
	if err != nil {
		t.Fatal(err)
	}

	actor, err := operations.LocalOwner(t.Context(), ws, operations.SurfaceMCP, scopes)
	if err != nil {
		t.Fatal(err)
	}

	dyn, ok := f.cat.Dynamic(op)
	if !ok {
		t.Fatalf("no operation %s", op)
	}

	body, _ := json.Marshal(input)

	res, err := dyn.CallJSON(t.Context(), actor, body)
	if err != nil {
		t.Fatalf("seed %s: %v", op, err)
	}

	out, _ := json.Marshal(res.Value)

	return out
}

func uidOf(t *testing.T, raw json.RawMessage) string {
	t.Helper()

	var v struct {
		UID string `json:"uid"`
	}

	if err := json.Unmarshal(raw, &v); err != nil || v.UID == "" {
		t.Fatalf("no uid in %s", raw)
	}

	return v.UID
}

// count is the number of rows of a table.
func (f *fixture) count(t *testing.T, table string) int {
	t.Helper()

	db, err := f.app.DB()
	if err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.SQL().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}

	return n
}

// toolNames lists the tools the server offers.
func (f *fixture) toolNames(t *testing.T) []string {
	t.Helper()

	var res struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}

	if err := f.client.Call("tools/list", nil).Decode(&res); err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(res.Tools))
	for _, tl := range res.Tools {
		names = append(names, tl.Name)
	}

	return names
}
