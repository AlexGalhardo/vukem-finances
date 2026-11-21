package ai

import (
	"github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/config"
)

// Service bundles what the routes and the features need.
type Service struct {
	Client   *Client
	Vault    *Vault
	Settings *SettingsStore
	// PrivateHosts is AI_ALLOW_PRIVATE_HOSTS, already limited to the personal edition.
	PrivateHosts bool
}

// New builds the AI layer from the app: the vault over APP_KEY, the settings
// store, and the client with the production providers. The per-minute rate
// limit lives in the Client, so build it once and share it.
func New(app *vukem.App) (*Service, error) {
	edition, err := config.LoadEdition(app.Source)
	if err != nil {
		return nil, err
	}

	private := config.LoadAI(app.Source, edition).AllowPrivateHosts

	vault, err := NewVault(app, private)
	if err != nil {
		return nil, err
	}

	settings := NewSettingsStore(vault.DB, app.Now)

	return &Service{
		Client:   NewClient(vault.DB, vault, settings, NewProvider(private), app.Log, app.Now),
		Vault:    vault,
		Settings: settings, PrivateHosts: private,
	}, nil
}
