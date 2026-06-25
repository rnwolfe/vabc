package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/rnwolfe/vabc"
	"github.com/rnwolfe/vabc/internal/errs"
	"github.com/rnwolfe/vabc/internal/geocode"
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

// resolveLocation turns a "lat,lng", a 5-digit ZIP, or a street address into
// coordinates plus a human label, so distances are measured from the user's actual
// location. ZIPs use an embedded centroid table (offline); addresses use the free
// US Census geocoder.
func resolveLocation(ctx context.Context, loc string) (lat, lng float64, label string, err error) {
	p, label, gerr := geocode.Resolve(ctx, loc)
	if gerr != nil {
		return 0, 0, "", errs.New(errs.ExitNotFound, "GEOCODE_FAILED", gerr.Error(),
			"pass a 5-digit ZIP, a full street address, or \"lat,lng\"")
	}
	return p.Lat, p.Lng, label, nil
}
