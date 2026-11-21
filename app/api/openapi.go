package api

import (
	"encoding/json"
	"fmt"

	"github.com/AlexGalhardo/vukem/openapi"
	"github.com/AlexGalhardo/vukem/router"
)

// Version is the version of the API document (the major is in the path).
const Version = "1.0.0"

const description = "The public API of vukem-finances: read and write the data of one workspace with a bearer token " +
	"that holds only the scopes it needs. Every operation names its scope in `x-required-scope`; a scope is a ceiling, " +
	"never a grant: the role of the person who made the token limits it at every request. Money is an integer `*_cents` " +
	"in the workspace currency. See docs/api.md for conventions, errors, idempotency, rate limits and versioning."

// BuildDocument builds the OpenAPI document of the API from the routes of the
// app (router.Routes()). Paths are relative to BasePath, which is the server.
func BuildDocument(routes []router.RouteInfo) (*openapi.Document, error) {
	return openapi.Build(routes, openapi.Info{
		Title:           "vukem-finances API",
		Version:         Version,
		Description:     description,
		Servers:         []string{BasePath},
		StripPrefix:     BasePath,
		RequiredOutputs: true,
		TypedOnly:       true,
		License:         &openapi.License{Name: "MIT", Identifier: "MIT"},
	})
}

// RenderDocument writes doc as indented JSON, with the scope of every
// operation as x-required-scope (an operation of any valid token has none).
// The output is stable, so a snapshot test can compare it byte for byte.
func RenderDocument(doc *openapi.Document) ([]byte, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}

	var tree map[string]any
	if err := json.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}

	paths, _ := tree["paths"].(map[string]any)

	for path, methods := range paths {
		ops, _ := methods.(map[string]any)

		for method, v := range ops {
			op, _ := v.(map[string]any)
			id, _ := op["operationId"].(string)

			o, ok := OperationNamed(id)
			if !ok {
				return nil, fmt.Errorf("api: %s %s (%s) is not in the operations table", method, path, id)
			}

			if o.Scope != "" {
				op["x-required-scope"] = o.Scope
			}
		}
	}

	out, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(out, '\n'), nil
}
