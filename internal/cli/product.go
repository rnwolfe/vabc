package cli

import (
	"github.com/rnwolfe/vabc/catalog"
	"github.com/rnwolfe/vabc/internal/errs"
)

// ProductCmd groups product catalog reads. Backed by the local catalog snapshot
// (Virginia ABC has no live, agent-usable search API), not live calls.
type ProductCmd struct {
	Search ProductSearchCmd `cmd:"" help:"Search the catalog snapshot by keyword."`
	Get    ProductGetCmd    `cmd:"" help:"Get one product by 6-digit product code."`
}

type ProductSearchCmd struct {
	Query     string `arg:"" optional:"" help:"Keyword matched against product name or code (omit to browse)."`
	Type      string `help:"Filter by product type or category, e.g. bourbon, gin."`
	Allocated bool   `help:"Only allocated / limited-availability products."`
}

func (c *ProductSearchCmd) Run(rt *Runtime) error {
	if rt.Catalog == nil {
		return errs.CatalogUnavailable()
	}
	opts := catalog.SearchOpts{Query: c.Query, Type: c.Type}
	if c.Allocated {
		t := true
		opts.Allocated = &t
	}
	products, err := rt.Catalog.Search(opts)
	if err != nil {
		return errs.New(errs.ExitCatalogUnavailable, "CATALOG_ERROR", err.Error(),
			"run: vabc catalog refresh --from-xlsx <price-list.xlsx>")
	}
	rt.Out.Info("scope: catalog snapshot %s (%s); run `vabc catalog refresh` for newer data",
		rt.Catalog.SnapshotDate(), rt.Catalog.Source())
	return rt.Out.Emit(products)
}

type ProductGetCmd struct {
	Code string `arg:"" help:"6-digit product code (e.g. 010807)."`
}

func (c *ProductGetCmd) Run(rt *Runtime) error {
	if rt.Catalog == nil {
		return errs.CatalogUnavailable()
	}
	p, ok, err := rt.Catalog.Get(c.Code)
	if err != nil {
		return errs.New(errs.ExitCatalogUnavailable, "CATALOG_ERROR", err.Error(),
			"run: vabc catalog refresh --from-xlsx <price-list.xlsx>")
	}
	if !ok {
		return errs.NotFound("product", c.Code)
	}
	rt.Out.Info("scope: catalog snapshot %s (%s)", rt.Catalog.SnapshotDate(), rt.Catalog.Source())
	return rt.Out.Emit(p)
}
