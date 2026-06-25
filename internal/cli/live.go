package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/rnwolfe/vabc"
	"github.com/rnwolfe/vabc/internal/errs"
)

// liveErr maps an upstream client error to a structured CLIError + exit code.
func liveErr(err error) error {
	if err == nil {
		return nil
	}
	var ae *vabc.APIError
	if errors.As(err, &ae) {
		switch ae.Kind {
		case vabc.KindNotFound:
			return errs.New(errs.ExitNotFound, "NOT_FOUND", ae.Msg,
				"check the store number / product code")
		case vabc.KindRateLimited:
			rem := "the upstream rate-limited or blocked the request; wait and retry"
			if s := ae.RetryAfterSeconds(); s > 0 {
				rem = fmt.Sprintf("blocked; retry after ~%ds (or pass --wait to wait it out)", s)
			}
			return errs.New(errs.ExitRate, "RATE_LIMITED", ae.Msg, rem)
		case vabc.KindSchemaDrift:
			return errs.New(errs.ExitGeneric, "SCHEMA_DRIFT", ae.Msg,
				"the Virginia ABC response shape changed; please file an issue")
		default: // KindRetryable
			return errs.New(errs.ExitRetry, "UPSTREAM_ERROR", ae.Msg,
				"transient upstream error; retry shortly")
		}
	}
	return errs.New(errs.ExitRetry, "UPSTREAM_ERROR", err.Error(),
		"retry; if it persists, run `vabc doctor`")
}

// parseLatLng parses a "lat,lng" string.
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

func isZip(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 5 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// resolveLocation turns a "lat,lng" or 5-digit ZIP into coordinates. A ZIP is
// resolved to a store in that ZIP (no external geocoder dependency).
func resolveLocation(ctx context.Context, rt *Runtime, loc string) (lat, lng float64, err error) {
	if lat, lng, ok := parseLatLng(loc); ok {
		return lat, lng, nil
	}
	if isZip(loc) {
		stores, e := rt.Client.Stores(ctx)
		if e != nil {
			return 0, 0, liveErr(e)
		}
		for _, s := range stores {
			if s.Zip == loc || strings.HasPrefix(s.Zip, loc) {
				return s.Lat, s.Lng, nil
			}
		}
		return 0, 0, errs.New(errs.ExitNotFound, "NO_STORE_IN_ZIP",
			"no Virginia ABC store found in ZIP "+loc,
			"pass coordinates instead, e.g. --near 38.91,-77.23")
	}
	return 0, 0, errs.New(errs.ExitUsage, "USAGE",
		"location must be a 5-digit ZIP or \"lat,lng\"",
		"e.g. 22182 or 38.91,-77.23")
}
