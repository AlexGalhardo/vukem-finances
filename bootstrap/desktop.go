package bootstrap

import (
	"context"
	"net"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/router"

	"github.com/AlexGalhardo/vukem-finances/app/desktopgate"
)

// DesktopOptions mounts the desktop gate: the app's own HTTP runner is turned
// off and a runner that serves the same handler behind the gate on ln takes its
// place. The gate wraps the whole handler, so even /up and the static files
// answer 403 without the window's cookie. The caller owns ln (bind it to
// 127.0.0.1 first, so there is no port race) and passes the result to New.
func DesktopOptions(gate *desktopgate.Gate, ln net.Listener) []vukem.Option {
	return []vukem.Option{
		vukem.WithoutHTTP(),
		vukem.WithRoutesAfterBoot(func(_ *router.Router, app *vukem.App) error {
			app.AddRunner("http", vukem.RunnerFunc(func(ctx context.Context) error {
				h, err := app.Handler()
				if err != nil {
					return err
				}

				srv := vukem.NewHTTPServer(ln.Addr().String(), gate.Wrap(h))
				srv.ShutdownTimeout = app.Config.ShutdownTimeout
				srv.Server.MaxHeaderBytes = app.Config.MaxHeaderBytes
				srv.Listener = ln

				return srv.Run(ctx)
			}))

			return nil
		}),
	}
}
