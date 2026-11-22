package mcp

import (
	"errors"
	"fmt"

	vukem "github.com/AlexGalhardo/vukem"
	vmcp "github.com/AlexGalhardo/vukem/mcp"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
	"github.com/AlexGalhardo/vukem-finances/app/reports"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
	"github.com/AlexGalhardo/vukem-finances/config"
)

// ErrNotPersonal is returned when the stdio server is asked for outside the
// personal edition: it acts as the local owner, which only that edition has.
var ErrNotPersonal = errors.New("mcp:serve --stdio runs only with APP_EDITION=personal: it acts as the local owner of one installation")

// CatalogOf builds the operation catalog of a booted app.
func CatalogOf(app *vukem.App) (*operations.Catalog, *workspaces.Service, error) {
	ws, err := workspaces.New(app)
	if err != nil {
		return nil, nil, fmt.Errorf("opening the workspaces: %w", err)
	}

	fin, err := finance.New(app, ws)
	if err != nil {
		return nil, nil, fmt.Errorf("opening the finances: %w", err)
	}

	store, err := app.Cache()
	if err != nil {
		return nil, nil, fmt.Errorf("opening the cache: %w", err)
	}

	cat := operations.NewCatalog(
		operations.Config{Edition: ws.Edition, Store: store.Store(), Log: app.Log, Now: app.Now},
		operations.Services{WS: ws, Fin: fin, Rep: reports.New(fin)},
	)

	return cat, ws, nil
}

// NewStdio builds the server of the personal edition for a client on the
// standard streams. It refuses any other edition before anything is served.
func NewStdio(app *vukem.App) (*vmcp.Server, error) {
	edition, err := config.LoadEdition(app.Source)
	if err != nil {
		return nil, err
	}

	if !edition.Personal() {
		return nil, ErrNotPersonal
	}

	cfg, err := config.LoadMCP(app.Source)
	if err != nil {
		return nil, err
	}

	cat, ws, err := CatalogOf(app)
	if err != nil {
		return nil, err
	}

	return New(Options{
		Catalog: cat, Actor: LocalActor(ws, cat, cfg), Config: cfg, Version: config.Version, Logger: app.Log,
	})
}
