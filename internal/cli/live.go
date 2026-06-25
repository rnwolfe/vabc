package cli

import (
	"errors"
	"strconv"
	"strings"

	"github.com/rnwolfe/vabc"
	"github.com/rnwolfe/vabc/internal/errs"
)

// liveErr maps an upstream client error to a structured CLIError + exit code.
// cli-implement extends this as it wires real HTTP status handling (400 invalid
// store → NOT_FOUND, WAF/429 → RATE_LIMITED, 5xx/transient CF → RETRYABLE).
func liveErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, vabc.ErrNotImplemented):
		return errs.New(errs.ExitRetry, "NOT_IMPLEMENTED",
			"the live Virginia ABC API is not wired yet (scaffold placeholder)",
			"this command is implemented in the cli-implement stage")
	default:
		return errs.New(errs.ExitRetry, "UPSTREAM_ERROR", err.Error(),
			"retry; if it persists, run `vabc doctor`")
	}
}

// parseLatLng parses a "lat,lng" string. A bare ZIP is not yet resolvable
// (cli-implement adds geocoding); callers surface a helpful usage error.
func parseLatLng(s string) (lat, lng float64, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(s), ",", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return a, b, true
}
