package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/AlexGalhardo/vukem/errs"

	"github.com/AlexGalhardo/vukem-finances/app/operations"
)

// The exit codes of the CLI.
const (
	ExitOK       = 0
	ExitFailure  = 1
	ExitUsage    = 2
	ExitAuth     = 3
	ExitNotFound = 4
	ExitConflict = 5
	ExitLimit    = 6
)

// Failure is an error that knows its exit code. Its text is safe to print: it
// never holds a token.
type Failure struct {
	Code int
	Msg  string
}

// Error implements error.
func (f *Failure) Error() string { return f.Msg }

func usagef(format string, a ...any) *Failure {
	return &Failure{Code: ExitUsage, Msg: fmt.Sprintf(format, a...)}
}

// ExitOf is the exit code of an error: 0 for nil.
func ExitOf(err error) int {
	if err == nil {
		return ExitOK
	}

	if f, ok := errors.AsType[*Failure](err); ok {
		return f.Code
	}

	return ExitFailure
}

// ExitForCode maps a stable error code of an operation to an exit code.
func ExitForCode(code string) int {
	switch code {
	case operations.CodeForbidden, operations.CodeMissingScope:
		return ExitAuth
	case operations.CodeNotFound:
		return ExitNotFound
	case operations.CodeConflict, operations.CodeVersionConflict:
		return ExitConflict
	case operations.CodeRateLimited, operations.CodePlanLimit:
		return ExitLimit
	case operations.CodeValidation:
		return ExitUsage
	}

	return ExitFailure
}

// failureOf turns the error of an invocation into a Failure.
func failureOf(err error) *Failure {
	if f, ok := errors.AsType[*Failure](err); ok {
		return f
	}

	oe, ok := errors.AsType[*operations.Error](err)
	if !ok {
		return &Failure{Code: ExitFailure, Msg: "Something went wrong."}
	}

	msg := oe.Message

	if ve, ok := errors.AsType[*errs.ValidationError](oe.Err); ok {
		msg += fieldProblems(ve.Fields)
	}

	return &Failure{Code: ExitForCode(oe.Code), Msg: oe.Code + ": " + msg}
}

func fieldProblems(fields map[string][]string) string {
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}

	sort.Strings(names)

	var parts []string
	for _, n := range names {
		parts = append(parts, n+": "+strings.Join(fields[n], ", "))
	}

	if len(parts) == 0 {
		return ""
	}

	return " (" + strings.Join(parts, "; ") + ")"
}
