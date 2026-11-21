package tests

import (
	"maps"
	"testing"

	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/bootstrap"
)

// newEditionApp builds the app in edition with extra environment values.
func newEditionApp(t *testing.T, edition string, env ...map[string]string) *vukemtest.App {
	t.Helper()

	opts := vukemtest.Options{Env: map[string]string{"APP_EDITION": edition}}
	for _, e := range env {
		maps.Copy(opts.Env, e)
	}

	return vukemtest.New(t, bootstrap.New, opts)
}

// newSaaSApp builds the multi-tenant edition: the one the finance suite covers.
func newSaaSApp(t *testing.T, env ...map[string]string) *vukemtest.App {
	t.Helper()

	return newEditionApp(t, "saas", env...)
}

// newPersonalApp builds the self-hosted edition.
func newPersonalApp(t *testing.T, env ...map[string]string) *vukemtest.App {
	t.Helper()

	return newEditionApp(t, "personal", env...)
}
