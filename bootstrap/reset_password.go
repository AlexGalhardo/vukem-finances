package bootstrap

import (
	"context"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"

	vukem "github.com/AlexGalhardo/vukem"

	"github.com/AlexGalhardo/vukem-finances/app/accounts"
)

// resetPasswordCommand is the recovery path of a locked-out owner: sign-up is
// closed in the personal edition and mail may not be configured, so the
// operator sets a new password from the server console, for example
//
//	docker compose exec app /app/app user:reset-password -email you@example.com
func resetPasswordCommand() vukem.Command {
	var email *string

	return vukem.CommandFunc{
		CommandName: "user:reset-password",
		Desc:        "Set a new random password for a user and end all of their sessions",
		DefineFlags: func(fs *flag.FlagSet) { email = fs.String("email", "", "the email of the user") },
		Fn: func(ctx context.Context, app *vukem.App, args []string) error {
			if len(args) > 0 || *email == "" {
				return errors.New("user:reset-password needs -email <address>")
			}

			acc, err := accounts.New(app)
			if err != nil {
				return err
			}

			// 26 base32 characters, well over the 12 the forms require.
			password := rand.Text()
			if err := acc.SetPassword(ctx, *email, password); err != nil {
				return fmt.Errorf("user:reset-password: %w", err)
			}

			fmt.Fprintf(app.Stdout(), "user:reset-password: new password for %s: %s\n", *email, password)
			fmt.Fprintln(app.Stdout(), "Sign in with it and change it in the account settings; it is shown only once.")

			return nil
		},
	}
}
