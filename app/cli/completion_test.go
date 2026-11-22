package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
)

func TestCompletion(t *testing.T) {
	t.Parallel()

	c, _ := newCLI(t, nil, nil)

	for _, shell := range cli.Shells {
		t.Run(shell, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder

			cc := &cli.CLI{Tree: c.Tree, Env: cli.Env{Stdout: &out}}
			if err := cc.Run(t.Context(), "completion", []string{shell}); err != nil {
				t.Fatal(err)
			}

			script := out.String()
			if !strings.Contains(script, cli.DefaultProgram) {
				t.Fatalf("no program name in the %s script", shell)
			}

			for _, cmd := range c.Tree.Commands {
				for _, want := range []string{cmd.Resource, cmd.Verb} {
					if !strings.Contains(script, want) {
						t.Fatalf("%s script has no %q (%s)", shell, want, cmd.Name())
					}
				}

				for _, f := range cmd.Fields {
					if f.Positional < 0 && !strings.Contains(script, f.Flag) {
						t.Fatalf("%s script has no flag %q (%s)", shell, f.Flag, cmd.Name())
					}
				}
			}

			for _, want := range []string{"login", "logout", "status", "table", "json", "csv", "yes"} {
				if !strings.Contains(script, want) {
					t.Fatalf("%s script has no %q", shell, want)
				}
			}
		})
	}

	t.Run("a program name is used", func(t *testing.T) {
		t.Parallel()

		c, tm := newCLI(t, nil, nil)

		if code := run(t, c, tm, "completion", "bash", "vf"); code != 0 || !strings.Contains(tm.out.String(), "complete -F _vf_complete vf") {
			t.Fatalf("exit %d: %s", code, tm.out.String())
		}
	})

	t.Run("bad input is a usage error", func(t *testing.T) {
		t.Parallel()

		c, tm := newCLI(t, nil, nil)

		for _, args := range [][]string{{"completion"}, {"completion", "tcsh"}, {"completion", "bash", "x; rm -rf ~"}, {"completion", "bash", "a", "b"}} {
			if code := run(t, c, tm, args...); code != cli.ExitUsage {
				t.Errorf("%v: exit %d, want %d", args, code, cli.ExitUsage)
			}
		}
	})

	t.Run("the bash script is valid bash", func(t *testing.T) {
		t.Parallel()

		bash, err := exec.LookPath("bash")
		if err != nil {
			t.Skip("bash is not installed")
		}

		script, err := c.Completion("bash", cli.DefaultProgram)
		if err != nil {
			t.Fatal(err)
		}

		file := filepath.Join(t.TempDir(), "c.bash")
		if err := os.WriteFile(file, []byte(script), 0o600); err != nil {
			t.Fatal(err)
		}

		if out, err := exec.Command(bash, "-n", file).CombinedOutput(); err != nil {
			t.Fatalf("bash -n: %v: %s", err, out)
		}
	})
}
