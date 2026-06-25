package cli

import (
	"context"

	"github.com/rnwolfe/vabc/internal/errs"
)

// InventoryCmd groups live inventory reads against Virginia ABC.
type InventoryCmd struct {
	Check     InventoryCheckCmd     `cmd:"" help:"Live per-store availability of a product, with nearby stores."`
	Warehouse InventoryWarehouseCmd `cmd:"" help:"Statewide central-warehouse stock of a product."`
}

type InventoryCheckCmd struct {
	Code  string `arg:"" help:"6-digit product code (e.g. 010807)."`
	Store int    `help:"Anchor store number (e.g. 219)."`
	Near  string `help:"Resolve the nearest store from \"lat,lng\" instead of --store."`
}

func (c *InventoryCheckCmd) Run(rt *Runtime) error {
	ctx := context.Background()
	store := c.Store

	if c.Near != "" {
		lat, lng, ok := parseLatLng(c.Near)
		if !ok {
			return errs.New(errs.ExitUsage, "USAGE",
				"--near expects \"lat,lng\" (ZIP resolution is added in cli-implement)",
				"use --near 38.91,-77.23 or pass --store <number>")
		}
		stores, err := rt.Client.StoreNear(ctx, lat, lng, 1)
		if err != nil {
			return liveErr(err)
		}
		if len(stores) == 0 {
			return errs.New(errs.ExitEmpty, "NO_NEARBY_STORE", "no store found near that location",
				"widen the search or pass --store <number>")
		}
		store = stores[0].StoreNumber
	}

	if store == 0 {
		return errs.New(errs.ExitUsage, "USAGE", "an anchor store is required",
			"pass --store <number> or --near \"lat,lng\"")
	}

	res, err := rt.Client.StoreNearby(ctx, store, c.Code)
	if err != nil {
		return liveErr(err)
	}
	rt.Out.Info("scope: live inventory for product %s anchored at store %d", c.Code, store)
	return rt.Out.Emit(res)
}

type InventoryWarehouseCmd struct {
	Code string `arg:"" help:"6-digit product code."`
}

func (c *InventoryWarehouseCmd) Run(rt *Runtime) error {
	res, err := rt.Client.Warehouse(context.Background(), c.Code)
	if err != nil {
		return liveErr(err)
	}
	rt.Out.Info("scope: live statewide warehouse stock for product %s", c.Code)
	return rt.Out.Emit(res)
}
