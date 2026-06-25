// Package errs defines the stable exit-code table and the structured CLI error type.
// Exit codes are a contract: distinct, documented, and append-only. See contract.md §4.
package errs

// Stable exit codes.
const (
	ExitOK              = 0
	ExitGeneric         = 1
	ExitUsage           = 2
	ExitEmpty           = 3
	ExitAuth            = 4
	ExitNotFound        = 5
	ExitPerm            = 6
	ExitRate            = 7
	ExitRetry           = 8
	ExitConfig          = 10
	// ExitCatalogUnavailable: the catalog snapshot is missing or unreadable (vabc-specific).
	ExitCatalogUnavailable = 11
	ExitMutationBlocked    = 12
	ExitInputRequired      = 13
	// ExitCatalogStale: the catalog snapshot is older than the freshness threshold (vabc-specific).
	ExitCatalogStale = 14
	ExitCancelled    = 130
)

// Table returns the exit-code table for the `schema` command.
func Table() map[string]int {
	return map[string]int{
		"ok":               ExitOK,
		"generic_error":    ExitGeneric,
		"usage":            ExitUsage,
		"empty_results":    ExitEmpty,
		"auth_required":    ExitAuth,
		"not_found":        ExitNotFound,
		"permission":       ExitPerm,
		"rate_limited":     ExitRate,
		"retryable":        ExitRetry,
		"config_error":        ExitConfig,
		"catalog_unavailable": ExitCatalogUnavailable,
		"mutation_blocked":    ExitMutationBlocked,
		"input_required":      ExitInputRequired,
		"catalog_stale":       ExitCatalogStale,
		"cancelled":           ExitCancelled,
	}
}

// CLIError is a structured error carrying a machine-readable code, a remediation hint,
// and the process exit code to return.
type CLIError struct {
	Message     string
	Code        string
	Remediation string
	Exit        int
}

func (e *CLIError) Error() string { return e.Message }

// New constructs a CLIError.
func New(exit int, code, msg, remediation string) *CLIError {
	return &CLIError{Message: msg, Code: code, Remediation: remediation, Exit: exit}
}

// MutationBlocked is returned when a mutating op runs without --allow-mutations.
func MutationBlocked(op string) *CLIError {
	return New(ExitMutationBlocked, "MUTATION_BLOCKED",
		op+" is a mutating operation and is blocked by default",
		"re-run with --allow-mutations (add --dry-run to preview)")
}

// NotFound is returned when a resource id does not exist.
func NotFound(kind, id string) *CLIError {
	return New(ExitNotFound, "NOT_FOUND", kind+" "+id+" not found",
		"list available "+kind+"s to find a valid id")
}

// InputRequired is returned when --no-input is set but input is needed.
func InputRequired(what string) *CLIError {
	return New(ExitInputRequired, "INPUT_REQUIRED", what+" is required",
		"pass it as a flag/argument (running with --no-input, so prompts are disabled)")
}

// CatalogUnavailable is returned when no catalog snapshot could be loaded.
func CatalogUnavailable() *CLIError {
	return New(ExitCatalogUnavailable, "CATALOG_UNAVAILABLE",
		"no product catalog snapshot is available",
		"run: vabc catalog refresh --from-xlsx <price-list.xlsx>")
}
