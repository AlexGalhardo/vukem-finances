package bootstrap

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"runtime"
	"strconv"
	"text/tabwriter"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/config"
)

// aboutCommand replaces the framework about command to add the version and
// the edition. vukem.WithVersion would also print the version in GET /up,
// which this app keeps private (see config.Version).
func aboutCommand() vukem.Command {
	var asJSON *bool

	return vukem.CommandFunc{
		CommandName: "about",
		Desc:        "Show application, version, edition and environment",
		DefineFlags: func(fs *flag.FlagSet) { asJSON = fs.Bool("json", false, "print JSON") },
		Fn: func(_ context.Context, app *vukem.App, _ []string) error {
			edition, err := config.LoadEdition(app.Source)
			if err != nil {
				return err
			}

			info := [][2]string{
				{"Name", app.Config.Name},
				{"Version", config.Version},
				{"Edition", string(edition)},
				{"Environment", string(app.Config.Env)},
				{"Debug", strconv.FormatBool(app.Config.Debug)},
				{"URL", app.Config.URL},
				{"Address", app.Addr()},
				{"Go", runtime.Version()},
				{"APP_KEY", app.Config.Key.String()},
				{"Routes", strconv.Itoa(len(app.Router.Routes()))},
			}

			if *asJSON {
				m := map[string]string{}
				for _, kv := range info {
					m[kv[0]] = kv[1]
				}

				return json.NewEncoder(app.Stdout()).Encode(m)
			}

			tw := tabwriter.NewWriter(app.Stdout(), 0, 4, 2, ' ', 0)
			for _, kv := range info {
				fmt.Fprintf(tw, "%s\t%s\n", kv[0], kv[1])
			}

			return tw.Flush()
		},
	}
}
