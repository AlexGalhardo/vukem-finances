package bootstrap

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/ai"
	"github.com/AlexGalhardo/vukem-finances/app/ai/categorize"
	"github.com/AlexGalhardo/vukem-finances/app/ai/fake"
)

// aiEvalCommand measures the categorizer on testdata/categorization-eval.jsonl
// and prints the accuracy and the tokens used. It is manual and never runs in
// CI: with --workspace it spends real calls of that workspace's own key.
func aiEvalCommand() vukem.Command {
	var provider, file *string

	var workspace *int64

	return vukem.CommandFunc{
		CommandName: "ai:eval",
		Desc:        "Measure AI categorization on the evaluation set (--provider=fake needs no key)",
		DefineFlags: func(fs *flag.FlagSet) {
			provider = fs.String("provider", "configured", "fake (deterministic, no key) or configured (the key of --workspace)")
			workspace = fs.Int64("workspace", 0, "workspace whose key and limits are used with --provider=configured")
			file = fs.String("file", "testdata/categorization-eval.jsonl", "evaluation set")
		},
		Fn: func(ctx context.Context, app *vukem.App, _ []string) error {
			f, err := os.Open(*file)
			if err != nil {
				return err
			}
			defer f.Close()

			rows, err := categorize.ReadEval(f)
			if err != nil {
				return err
			}

			var (
				completer categorize.Completer
				id        int64
			)

			switch *provider {
			case "fake":
				completer = fake.DefaultClassifier()
			case "configured":
				if *workspace <= 0 {
					return errors.New("ai:eval: --workspace is required with --provider=configured")
				}

				svc, err := ai.Shared(app)
				if err != nil {
					return err
				}

				completer, id = svc.Client, *workspace
			default:
				return fmt.Errorf("ai:eval: unknown provider %q (fake or configured)", *provider)
			}

			res, err := categorize.Evaluate(ctx, completer, id, rows)
			fmt.Fprintln(app.Stdout(), res.Line())

			return err
		},
	}
}
