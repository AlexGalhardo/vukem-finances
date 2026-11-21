package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AlexGalhardo/vukem/openapi"
	"github.com/AlexGalhardo/vukem/vukemtest"

	"github.com/AlexGalhardo/vukem-finances/app/api"
)

const (
	openAPIFile = "../docs/openapi.json"
	clientFile  = "../clients/typescript/api.ts"
)

func apiDocument(t *testing.T, app *vukemtest.App) *openapi.Document {
	t.Helper()

	doc, err := api.BuildDocument(app.Router.Routes())
	if err != nil {
		t.Fatal(err)
	}

	return doc
}

// matchesFile fails when got differs from the file; UPDATE_OPENAPI=1 rewrites it.
func matchesFile(t *testing.T, path string, got []byte, how string) {
	t.Helper()

	if os.Getenv("UPDATE_OPENAPI") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v (run %s)", path, err, how)
	}

	// Git on Windows may check the file out with CRLF.
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))

	if !bytes.Equal(want, got) {
		t.Errorf("%s is stale: run %s", path, how)
	}
}

const updateHow = "UPDATE_OPENAPI=1 go test ./tests/... -run 'TestOpenAPISnapshot|TestTypeScriptClientIsFresh'"

func TestOpenAPISnapshot(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	body, err := api.RenderDocument(apiDocument(t, app))
	if err != nil {
		t.Fatal(err)
	}

	matchesFile(t, openAPIFile, body, updateHow)
}

func TestTypeScriptClientIsFresh(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	ts, err := openapi.TypeScriptClientWith(apiDocument(t, app), openapi.ClientOptions{Args: "(see docs/api.md)"})
	if err != nil {
		t.Fatal(err)
	}

	matchesFile(t, clientFile, []byte(ts), updateHow)
}

func TestOpenAPIRoute(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	want, err := api.RenderDocument(apiDocument(t, app))
	if err != nil {
		t.Fatal(err)
	}

	// Public: no token. It describes the operations and carries their scopes.
	res := app.Client(t).Get("/api/v1/openapi.json").AssertOK().AssertHeaderContains("Content-Type", "application/json")
	if res.Body() != string(want) {
		t.Error("the served document differs from the generated one")
	}

	var doc struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]map[string]struct {
			ID    string `json:"operationId"`
			Scope string `json:"x-required-scope"`
		} `json:"paths"`
		Components struct {
			SecuritySchemes map[string]any `json:"securitySchemes"`
		} `json:"components"`
	}

	if err := json.Unmarshal([]byte(res.Body()), &doc); err != nil {
		t.Fatal(err)
	}

	if _, ok := doc.Components.SecuritySchemes["bearer"]; !ok || !strings.HasPrefix(doc.OpenAPI, "3.") {
		t.Errorf("openapi = %q, security schemes = %v", doc.OpenAPI, doc.Components.SecuritySchemes)
	}

	seen := map[string]bool{}

	for path, methods := range doc.Paths {
		for method, op := range methods {
			o, ok := api.OperationNamed(op.ID)
			if !ok || o.Scope != op.Scope || !strings.EqualFold(o.Method, method) || o.Path != path {
				t.Errorf("%s %s (%s) does not match the operations table", method, path, op.ID)
			}

			seen[op.ID] = true
		}
	}

	for _, o := range api.Operations() {
		if !seen[o.Name] {
			t.Errorf("operation %s is in the table and not in the document", o.Name)
		}
	}
}

func TestNoAPIDocsPage(t *testing.T) {
	t.Parallel()

	app := newApp(t)

	app.Client(t).Get("/api/docs").AssertNotFound()
	app.Client(t).Get("/api/v1/docs").AssertNotFound()

	for _, r := range app.Router.Routes() {
		if strings.HasPrefix(r.Path, "/api/docs") || strings.Contains(r.Path, "scalar") {
			t.Errorf("route %s %s serves an API reference page", r.Method, r.Path)
		}
	}

	// The framework's apidocs package renders with Scalar (D-021): nothing imports it.
	for _, dir := range []string{"app", "routes", "bootstrap", "cmd", "config"} {
		err := filepath.WalkDir(filepath.Join("..", dir), func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}

			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			if bytes.Contains(src, []byte("vukem/apidocs")) {
				t.Errorf("%s imports the apidocs package", path)
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
