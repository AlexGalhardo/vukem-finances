package main

import (
	"fmt"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	docsURL = "https://github.com/AlexGalhardo/vukem-finances/blob/main/docs/desktop.md"
	license = "MIT"
)

// menuActions are what the menu does that needs the running app.
type menuActions struct {
	exportBackup   func()
	openDataFolder func()
	version        string
}

// buildMenu is the native menu. There is no developer-tools item and the app
// registers no OS-level global shortcuts; the keyboard shortcuts of the web
// shell stay in the page.
func buildMenu(app *application.App, a menuActions) *application.Menu {
	menu := app.Menu.New()

	if runtime.GOOS == "darwin" {
		menu.AddRole(application.AppMenu)
	}

	file := menu.AddSubmenu("File")
	file.Add("Export backup...").OnClick(func(*application.Context) { a.exportBackup() })
	file.Add("Open data folder").OnClick(func(*application.Context) { a.openDataFolder() })
	file.AddSeparator()
	file.AddRole(application.Quit)

	edit := menu.AddSubmenu("Edit")
	edit.AddRole(application.Cut)
	edit.AddRole(application.Copy)
	edit.AddRole(application.Paste)
	edit.AddRole(application.SelectAll)

	view := menu.AddSubmenu("View")
	view.AddRole(application.Reload)
	view.AddSeparator()
	view.AddRole(application.ZoomIn)
	view.AddRole(application.ZoomOut)
	view.AddRole(application.ResetZoom)

	help := menu.AddSubmenu("Help")
	help.Add("Documentation").OnClick(func(*application.Context) { _ = app.Browser.OpenURL(docsURL) })
	help.Add("About vukem-finances").OnClick(func(*application.Context) {
		app.Dialog.Info().
			SetTitle("About vukem-finances").
			SetMessage(fmt.Sprintf("vukem-finances %s\nLicense: %s\n\nYour data stays on this computer.", a.version, license)).
			Show()
	})

	return menu
}
