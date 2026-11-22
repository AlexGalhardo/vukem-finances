// Package operations is the registry of everything a person or an agent can
// do with their finances (epic 30). An operation is declared once, with its
// name, scope, kind and typed input and output; the REST API, and later the
// CLI and MCP, are adapters that call Invoke. Invoke is the only way in: it
// checks the edition, the scope and the role, validates the input, applies the
// idempotency key of a write, runs the operation and writes the audit entry.
//
// An operation stays thin: the rules live in app/finance, app/reports and the
// other services. Run never takes a workspace id from its input: the
// workspace is the one of the Actor.
package operations
