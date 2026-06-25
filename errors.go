package vabc

import (
	"fmt"
	"time"
)

// ErrKind classifies an API failure so the CLI can map it to a stable exit code
// without parsing messages. Importers can switch on it too.
type ErrKind string

const (
	// KindNotFound: the target reports the resource does not exist (e.g. an invalid
	// store number returns HTTP 400 "No Store exists ...").
	KindNotFound ErrKind = "not_found"
	// KindRateLimited: throttled or blocked (HTTP 429, a WAF challenge, or the local
	// circuit-breaker). RetryAfter, when set, says how long to wait.
	KindRateLimited ErrKind = "rate_limited"
	// KindRetryable: a transient upstream/network error (5xx, timeout) worth retrying.
	KindRetryable ErrKind = "retryable"
	// KindSchemaDrift: the response did not match the expected shape — an upstream
	// change, surfaced explicitly instead of as a decode panic.
	KindSchemaDrift ErrKind = "schema_drift"
)

// APIError is a typed, classified error from the Virginia ABC client.
type APIError struct {
	Kind       ErrKind
	Status     int           // HTTP status, when applicable (0 otherwise)
	Msg        string        // human-readable summary
	RetryAfter time.Duration // for KindRateLimited: suggested wait before retrying
	Err        error         // wrapped cause, if any
}

func (e *APIError) Error() string {
	if e.Status != 0 {
		return fmt.Sprintf("%s (status %d): %s", e.Kind, e.Status, e.Msg)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Msg)
}

func (e *APIError) Unwrap() error { return e.Err }

// RetryAfterSeconds returns the suggested retry delay in whole seconds (0 if none),
// for surfacing to an agent that wants to schedule a retry.
func (e *APIError) RetryAfterSeconds() int {
	if e.RetryAfter <= 0 {
		return 0
	}
	return int((e.RetryAfter + time.Second - 1) / time.Second)
}

func notFound(status int, msg string) *APIError {
	return &APIError{Kind: KindNotFound, Status: status, Msg: msg}
}

func rateLimited(status int, msg string, retryAfter time.Duration) *APIError {
	return &APIError{Kind: KindRateLimited, Status: status, Msg: msg, RetryAfter: retryAfter}
}

func retryable(status int, msg string, err error) *APIError {
	return &APIError{Kind: KindRetryable, Status: status, Msg: msg, Err: err}
}

func schemaDrift(msg string, err error) *APIError {
	return &APIError{Kind: KindSchemaDrift, Msg: msg, Err: err}
}
