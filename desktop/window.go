package main

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	appconfig "github.com/AlexGalhardo/vukem-finances/config"
)

// appID is the application id of the installers and of the single-instance lock.
const appID = "dev.vukem.finances"

// runWindowed is the normal run: the same server as the headless one, shown in a
// native window. Closing the window quits the app, and the server stops with it.
func runWindowed(ctx context.Context, f flags) error {
	paths, err := resolvePaths(f)
	if err != nil {
		return err
	}

	var window *application.WebviewWindow

	app := application.New(application.Options{
		Name:        "vukem-finances",
		Description: "Personal finances, on your computer.",
		// A second launch exits here and focuses the first window.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: appID,
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if window != nil {
					window.Restore()
					window.Focus()
				}
			},
		},
		Windows: application.WindowsOptions{
			// The default is next to the executable, which an installer may
			// have put in a folder the user cannot write.
			WebviewUserDataPath: paths.Data + string(os.PathSeparator) + "webview",
		},
		Mac: application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})

	// The OS file lock covers what the Wails instance check does not: any other
	// process (or the headless mode) writing the same data directory.
	lock, err := AcquireLock(paths.Lock)
	if err != nil {
		return err
	}

	defer func() { _ = lock.Release() }()

	native := &native{app: app}
	sh := &shell{tmpDir: paths.Data, openURL: app.Browser.OpenURL, chooseSavePath: native.chooseSavePath}

	srv, err := StartServer(ctx, paths, startOptions{shell: sh})
	if err != nil {
		return err
	}

	var stopped atomic.Bool

	stop := func() {
		if !stopped.CompareAndSwap(false, true) {
			return
		}

		stopCtx, cancel := context.WithTimeout(context.Background(), stopTimeout)
		defer cancel()

		_ = srv.Stop(stopCtx)
	}

	app.OnShutdown(stop)

	app.Menu.Set(buildMenu(app, menuActions{
		version:        appconfig.Version,
		openDataFolder: func() { _ = app.Browser.OpenFile(paths.Data) },
		exportBackup:   func() { native.exportBackup(ctx, srv, paths) },
	}))

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:               "main",
		Title:              "vukem-finances",
		URL:                srv.EntryURL(),
		Width:              1280,
		Height:             820,
		MinWidth:           720,
		MinHeight:          520,
		UseApplicationMenu: true,
	})

	// The server ending by itself (not because of Stop) is a failure the person
	// must see: there is nothing left behind the window.
	go func() {
		err := <-srv.Done()
		srv.done <- err

		if err != nil && !stopped.Load() {
			app.Dialog.Error().SetTitle("vukem-finances").SetMessage("The app stopped: " + err.Error()).Show()
			app.Quit()
		}
	}()

	return app.Run()
}

// native is the Wails side of the native dialogs.
type native struct{ app *application.App }

func (n *native) chooseSavePath(name string) (string, error) {
	return n.app.Dialog.SaveFile().CanCreateDirectories(true).SetFilename(name).PromptForSingleSelection()
}

func (n *native) exportBackup(ctx context.Context, srv *Server, paths Paths) {
	fail := func(err error) {
		n.app.Dialog.Error().SetTitle("Export backup").SetMessage("The backup could not be saved: " + err.Error()).Show()
	}

	name := "vukem-finances-backup-" + time.Now().Format("2006-01-02") + ".zip"

	dest, err := n.app.Dialog.SaveFile().
		CanCreateDirectories(true).
		SetFilename(name).
		AddFilter("Zip archive", "*.zip").
		PromptForSingleSelection()
	if err != nil {
		fail(err)
		return
	}

	if dest == "" {
		return
	}

	db, err := srv.app.DB()
	if err != nil {
		fail(err)
		return
	}

	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		fail(err)
		return
	}

	err = WriteBackupZip(ctx, db.SQL(), paths, out)
	if cerr := out.Close(); err == nil {
		err = cerr
	}

	if err != nil {
		_ = os.Remove(dest)
		fail(err)

		return
	}

	n.app.Dialog.Info().SetTitle("Export backup").
		SetMessage(fmt.Sprintf("Backup saved to %s.\nIt holds your data and your key: keep it private.", dest)).Show()
}
