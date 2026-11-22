package alerts

import (
	"log/slog"
	"time"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/database"

	"github.com/AlexGalhardo/vukem-finances/app/finance"
	"github.com/AlexGalhardo/vukem-finances/app/workspaces"
)

// Service evaluates and delivers the alerts of a workspace. Every method that
// reads or writes takes a context carrying the workspace (workspaces.WithTenant).
type Service struct {
	DB  *database.DB
	Fin *finance.Service
	WS  *workspaces.Service
	Log *slog.Logger
	URL string

	// InApp and Email are the ways a notification reaches a person (epic 17
	// adds Telegram next to them).
	InApp Channel
	Email Channel

	now func() time.Time
}

// New builds the service from the app and the services it reads through.
func New(app *vukem.App, ws *workspaces.Service, fin *finance.Service) *Service {
	return &Service{
		DB: ws.DB, Fin: fin, WS: ws, Log: app.Log, URL: ws.URL,
		InApp: InApp{}, Email: &Email{WS: ws, URL: ws.URL}, now: app.Now,
	}
}
