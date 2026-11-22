package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AlexGalhardo/vukem-finances/app/pages"
)

// The app has no landing page in either edition: "/" only sends the visitor on, always to
// a path of this same server (the personal edition runs offline).
func TestRootRedirects(t *testing.T) {
	t.Parallel()

	saas := newSaaSApp(t)
	saas.Client(t).Get("/").AssertRedirect("/login")
	saas.Client(t).Inertia("").Get("/").AssertRedirect("/login")
	signUp(t, saas, "Ana", "ana@example.com").Client.Get("/").AssertRedirect("/dashboard")

	personal := newPersonalApp(t)
	personal.Client(t).Get("/").AssertRedirect("/setup")

	ana := signUp(t, personal, "Ana", "ana@example.com")
	personal.Client(t).Get("/").AssertRedirect("/login")
	ana.Client.Get("/").AssertRedirect("/dashboard")

	app := newSaaSApp(t)
	app.Client(t).Get("/up").AssertOK()
}

func TestNoHomePage(t *testing.T) {
	t.Parallel()

	for _, p := range pages.All() {
		if p.Component == "Home" {
			t.Fatal("the app has a Home page: the landing page lives in site/")
		}
	}

	if _, err := os.Stat(filepath.Join("..", "resources", "js", "pages", "Home.tsx")); !os.IsNotExist(err) {
		t.Fatalf("resources/js/pages/Home.tsx must not exist (err %v)", err)
	}
}

func TestErrorPage(t *testing.T) {
	t.Parallel()

	app := newSaaSApp(t)

	// Errors on Inertia visits render resources/js/pages/Error; full page
	// loads keep the HTML status page.
	app.Client(t).Inertia("").Get("/no-such-page").AssertStatus(404).
		AssertInertia("Error").
		PropEquals("status", 404).
		PropEquals("message", "Not Found")
	app.Client(t).Get("/no-such-page").AssertStatus(404).AssertHeader("X-Inertia", "")
}
