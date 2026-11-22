package tests

import (
	"testing"

	"github.com/AlexGalhardo/vukem-finances/config"
)

func TestReleaseVersionOnTheScreens(t *testing.T) {
	t.Parallel()

	t.Run("personal shows it on a public screen", func(t *testing.T) {
		t.Parallel()

		newPersonalApp(t).Client(t).Inertia("").Get("/setup").AssertOK().AssertInertia("Setup").
			PropEquals("release", config.DisplayVersion())
	})

	t.Run("saas hides it from a visitor and shows it after sign-in", func(t *testing.T) {
		t.Parallel()

		app := newSaaSApp(t)

		login := app.Client(t).Inertia("").Get("/login").AssertOK()
		login.AssertDontSee(`"release":"` + config.DisplayVersion())

		ana := signUp(t, app, "Ana", "ana@example.com")
		ana.Client.Inertia("").Get("/dashboard").AssertOK().AssertInertia("Dashboard").
			PropEquals("release", config.DisplayVersion())
	})
}
