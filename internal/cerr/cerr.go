// Package cerr defines the CLI's stable error codes and their exit codes
// (spec §4.3). Every error the user or an agent sees carries a code, a
// message that says what happened and what to do, and optional extra fields
// (action_url, status, retry_after…) that go out as-is in --json.
package cerr

import (
	"errors"
	"sort"
)

// Error is a CLI error with a stable code.
type Error struct {
	Code    string
	Message string
	Extra   map[string]any
}

func (e *Error) Error() string { return e.Message }

// New returns an error with a code and a message.
func New(code, msg string) *Error { return &Error{Code: code, Message: msg} }

// With adds an extra field (it goes into the JSON error object).
func (e *Error) With(key string, v any) *Error {
	if e.Extra == nil {
		e.Extra = map[string]any{}
	}
	e.Extra[key] = v
	return e
}

// Str returns an extra field as a string ("" if missing).
func (e *Error) Str(key string) string {
	s, _ := e.Extra[key].(string)
	return s
}

// Codes and their exit codes (spec §4.3).
const (
	ServerError       = "server_error"
	Network           = "network"
	Usage             = "usage"
	MissingFlag       = "missing_flag"
	NotLinked         = "not_linked"
	ProjectNotFound   = "project_not_found"
	NeedsLogin        = "needs_login"
	SessionExpired    = "session_expired"
	Forbidden         = "forbidden"
	PlanLimit         = "plan_limit"
	ProjectNotRunning = "project_not_running"
	RateLimited       = "rate_limited"
	UpgradeRequired   = "upgrade_required"
	LoginPending      = "login_pending"
	Provisioning      = "provisioning"
	// Domain errors that are not in the §4.3 table keep exit 1 (deploy
	// blocked by secrets, SQL errors, failed builds…).
	Blocked  = "blocked"
	SQLError = "sql_error"
	Failed   = "failed"
)

var exitCodes = map[string]int{
	ServerError: 1, Network: 1,
	Usage: 2, MissingFlag: 2, NotLinked: 2, ProjectNotFound: 2,
	NeedsLogin: 3, SessionExpired: 3,
	Forbidden:         4,
	PlanLimit:         5,
	ProjectNotRunning: 6,
	RateLimited:       7,
	UpgradeRequired:   8,
	LoginPending:      10, Provisioning: 10,
}

// ExitCode maps an error code to the process exit code (1 if unknown).
func ExitCode(code string) int {
	if n, ok := exitCodes[code]; ok {
		return n
	}
	return 1
}

// As returns err as *Error, wrapping unknown errors as server_error.
func As(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{Code: ServerError, Message: err.Error()}
}

// Keys returns the extra keys sorted (stable human output).
func (e *Error) Keys() []string {
	keys := make([]string, 0, len(e.Extra))
	for k := range e.Extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
