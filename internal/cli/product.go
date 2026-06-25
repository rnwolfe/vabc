package cli

import (
	"strings"

	"github.com/rnwolfe/vabc"
	"github.com/rnwolfe/vabc/catalog"
	"github.com/rnwolfe/vabc/internal/errs"
)

// ProductCmd groups product reads. Search/get use the live Coveo web catalog by
// default (full coverage, current) and fall back to the embedded snapshot offline.
type ProductCmd struct {
	Search ProductSearchCmd `cmd:"" help:"Search products by keyword (live web catalog)."`
	Get    ProductGetCmd    `cmd:"" help:"Get one product by 6-digit product code."`
}

type ProductSearchCmd struct {
	Query     string `arg:"" optional:"" help:"Keyword matched against product name (omit to browse)."`
	Type      string `help:"Filter by product category/type, e.g. bourbon, gin."`
	Allocated bool   `help:"Only allocated / limited-availability products."`
	Offline   bool   `help:"Search the embedded snapshot only (no network)."`
}

func (c *ProductSearchCmd) Run(rt *Runtime) error {
	// Live path: the Coveo web catalog covers more than the downloadable price list.
	if !c.Offline {
		products, err := rt.Client.SearchProducts(rt.Ctx, c.Query, rt.Cfg.Limit)
		if err == nil {
			return rt.Out.Emit(filterProducts(products, c.Type, c.Allocated))
		}
		// Fall back to the offline snapshot when the live catalog is unreachable.
		rt.Out.Info("note: live catalog unavailable (%s); searching the offline snapshot", liveReason(err))
	}
	return c.searchSnapshot(rt)
}

func (c *ProductSearchCmd) searchSnapshot(rt *Runtime) error {
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
			"run: vabc catalog refresh")
	}
	rt.warnIfStale()
	return rt.Out.Emit(products)
}

type ProductGetCmd struct {
	Code    string `arg:"" help:"6-digit product code (e.g. 010807)."`
	Offline bool   `help:"Look up in the embedded snapshot only (no network)."`
}

func (c *ProductGetCmd) Run(rt *Runtime) error {
	code := pad6(c.Code)

	// Snapshot first: fast, offline, covers the common ~4,900 products.
	if rt.Catalog != nil {
		if p, ok, _ := rt.Catalog.Get(code); ok {
			return rt.Out.Emit(p)
		}
	}
	// Long tail (e.g. new / online-only SKUs): resolve live via Coveo.
	if !c.Offline {
		products, err := rt.Client.SearchProducts(rt.Ctx, code, 10)
		if err == nil {
			for _, p := range products {
				if p.ProductCode == code {
					return rt.Out.Emit(p)
				}
			}
		}
	}
	return errs.NotFound("product", code)
}

// filterProducts applies the --type and --allocated filters to live results.
func filterProducts(products []vabc.Product, typ string, allocatedOnly bool) []vabc.Product {
	typ = strings.ToLower(strings.TrimSpace(typ))
	out := make([]vabc.Product, 0, len(products))
	for _, p := range products {
		if allocatedOnly && !p.Allocated {
			continue
		}
		if typ != "" &&
			!strings.Contains(strings.ToLower(p.Category), typ) &&
			!strings.Contains(strings.ToLower(p.Type), typ) &&
			!strings.Contains(strings.ToLower(p.Name), typ) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// liveReason returns a short reason string for the fallback note.
func liveReason(err error) string {
	if ce := liveErr(err); ce != nil {
		return ce.Error()
	}
	return err.Error()
}
