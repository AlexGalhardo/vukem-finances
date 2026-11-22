package cli_test

import (
	"slices"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/cli"
)

func TestCommandTree(t *testing.T) {
	t.Parallel()

	reg := cli.Describe()

	tree, err := cli.DefaultTree()
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range reg.Operations() {
		c, ok := tree.Find(m.Name)
		if !m.Offered("cli") {
			if ok {
				t.Errorf("%s opts out of the CLI but has a command", m.Name)
			}

			continue
		}

		if !ok {
			t.Errorf("%s has no command", m.Name)
			continue
		}

		if c.Method == "" || c.Path == "" {
			t.Errorf("%s has no REST route, so remote mode cannot reach it", m.Name)
		}

		for i, p := range c.Params {
			var found bool

			for _, f := range c.Fields {
				if f.Name == p {
					found = true

					if f.Positional != i {
						t.Errorf("%s: %s is argument %d, want %d", m.Name, p, f.Positional, i)
					}
				}
			}

			if !found {
				t.Errorf("%s: path parameter %s is not an input field", m.Name, p)
			}
		}
	}

	if c, ok := tree.Find("transactions.create"); !ok || c.Name() != "transactions create" {
		t.Errorf("transactions.create is %q", c.Name())
	}

	c, _ := tree.Find("goals.contributions.add")
	if c.Resource != "goals" || c.Verb != "contributions-add" {
		t.Errorf("goals.contributions.add is %q", c.Name())
	}

	c, _ = tree.Find("transactions.create")

	var flags []string

	for _, f := range c.Fields {
		flags = append(flags, f.Flag)
	}

	for _, want := range []string{"account-uid", "amount-cents", "date", "tags"} {
		if !slices.Contains(flags, want) {
			t.Errorf("transactions create lacks --%s (has %v)", want, flags)
		}
	}
}
