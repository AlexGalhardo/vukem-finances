package operations

import (
	"errors"
	"net/http"
	"slices"

	"github.com/AlexGalhardo/vukem/errs"
	"github.com/AlexGalhardo/vukem/orm"
)

// The stable error codes of an operation. An adapter maps them to its own
// vocabulary (a status, an exit code, a JSON-RPC error); nothing else about an
// error is a contract. missing_scope and version_conflict refine forbidden and
// conflict: the REST API has always named them.
const (
	CodeNotFound        = "not_found"
	CodeForbidden       = "forbidden"
	CodeValidation      = "validation_failed"
	CodeConflict        = "conflict"
	CodeRateLimited     = "rate_limited"
	CodePlanLimit       = "plan_limit"
	CodeInternal        = "internal"
	CodeMissingScope    = "missing_scope"
	CodeVersionConflict = "version_conflict"
)

// Codes lists every stable error code.
func Codes() []string {
	return []string{
		CodeNotFound, CodeForbidden, CodeValidation, CodeConflict, CodeRateLimited, CodePlanLimit, CodeInternal,
		CodeMissingScope, CodeVersionConflict,
	}
}

// Error is what Invoke returns for every failure. Code and Message are safe
// to show to the caller; Err is the cause and never leaves the server (the
// REST adapter hands it to the framework, which hides what must stay hidden).
type Error struct {
	Code    string
	Message string
	Err     error
}

// Error implements error.
func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Unwrap returns the cause.
func (e *Error) Unwrap() error { return e.Err }

// fail makes an Error whose cause is the HTTP error a REST client receives.
func fail(status int, code, message string) *Error {
	return &Error{Code: code, Message: message, Err: errs.New(status, message).WithExtension("code", code)}
}

// asError turns any error of a Run into an *Error with a stable code. An
// error that is not a known failure is internal, with a generic message: the
// text of the cause is for the server log.
func asError(err error) *Error {
	if err == nil {
		return nil
	}

	if e, ok := errors.AsType[*Error](err); ok {
		return e
	}

	if errors.Is(err, orm.ErrNotFound) {
		return &Error{Code: CodeNotFound, Message: "Not found.", Err: err}
	}

	status := errs.Status(err)
	code := codeOf(status)

	if he, ok := errors.AsType[*errs.HTTPError](err); ok {
		if named, _ := he.Extensions["code"].(string); slices.Contains(Codes(), named) {
			code = named
		}
	}

	message := "Something went wrong on our side."

	if code != CodeInternal {
		message = clientText(err)
	}

	return &Error{Code: code, Message: message, Err: err}
}

func clientText(err error) string {
	if he, ok := errors.AsType[*errs.HTTPError](err); ok {
		return he.Text()
	}

	return err.Error()
}

// codeOf is the code of a status.
func codeOf(status int) string {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return CodeForbidden
	case http.StatusPaymentRequired:
		return CodePlanLimit
	case http.StatusNotFound, http.StatusMethodNotAllowed:
		return CodeNotFound
	case http.StatusConflict:
		return CodeConflict
	case http.StatusTooManyRequests:
		return CodeRateLimited
	}

	if status >= http.StatusInternalServerError {
		return CodeInternal
	}

	return CodeValidation
}

// IsCode reports whether err is an operation error with code.
func IsCode(err error, code string) bool {
	e, ok := errors.AsType[*Error](err)

	return ok && e.Code == code
}
