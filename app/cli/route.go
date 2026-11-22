package cli

import (
	"github.com/AlexGalhardo/vukem-finances/app/api"
	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// RESTRoute is where the REST API serves an operation, the path relative to
// the API base.
func RESTRoute(op string) (string, string, bool) {
	o, ok := api.OperationNamed(op)

	return o.Method, o.Path, ok
}

// Describe returns the registry that names every operation, with no services.
func Describe() *operations.Registry { return api.Describe().Registry }

// DefaultTree is the tree of the registry of the application.
func DefaultTree() (*Tree, error) { return BuildTree(Describe(), RESTRoute) }
