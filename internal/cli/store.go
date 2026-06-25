package cli

import (
	"context"
	"strconv"

	"github.com/rnwolfe/vabc/internal/errs"
)

// StoreCmd groups Virginia ABC store-locator reads (ArcGIS-backed).
type StoreCmd struct {
	List StoreListCmd `cmd:"" help:"List all Virginia ABC stores."`
	Get  StoreGetCmd  `cmd:"" help:"Get one store by store number."`
	Near StoreNearCmd `cmd:"" help:"Find stores nearest a location."`
}

type StoreListCmd struct{}

func (c *StoreListCmd) Run(rt *Runtime) error {
	stores, err := rt.Client.Stores(context.Background())
	if err != nil {
		return liveErr(err)
	}
	return rt.Out.Emit(stores)
}

type StoreGetCmd struct {
	Number int `arg:"" help:"ABC store number (e.g. 219)."`
}

func (c *StoreGetCmd) Run(rt *Runtime) error {
	stores, err := rt.Client.Stores(context.Background())
	if err != nil {
		return liveErr(err)
	}
	for _, s := range stores {
		if s.StoreNumber == c.Number {
			return rt.Out.Emit(s)
		}
	}
	return errs.NotFound("store", strconv.Itoa(c.Number))
}

type StoreNearCmd struct {
	Location string `arg:"" help:"A 5-digit ZIP or \"lat,lng\" to search near."`
}

func (c *StoreNearCmd) Run(rt *Runtime) error {
	ctx := context.Background()
	lat, lng, err := resolveLocation(ctx, rt, c.Location)
	if err != nil {
		return err
	}
	stores, err := rt.Client.StoreNear(ctx, lat, lng, rt.Cfg.Limit)
	if err != nil {
		return liveErr(err)
	}
	return rt.Out.Emit(stores)
}
