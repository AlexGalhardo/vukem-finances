package tests

import (
	"bytes"
	"strings"
	"testing"

	vukem "github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/bootstrap"
	"github.com/AlexGalhardo/vukem-finances/config"
)

func TestAboutReportsVersionAndEdition(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	app := vukemtest.New(t, bootstrap.New, vukemtest.Options{
		Env:        map[string]string{"APP_EDITION": "personal"},
		AppOptions: []vukem.Option{vukem.WithStdout(&out)},
	})

	if err := app.Call(t.Context(), "about", nil); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{config.Version, "personal"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("about output lacks %q:\n%s", want, out.String())
		}
	}

	up := app.Client(t).Get("/up")
	up.AssertStatus(200)

	if strings.Contains(up.Body(), config.Version) {
		t.Fatalf("/up discloses the version: %s", up.Body())
	}
}
