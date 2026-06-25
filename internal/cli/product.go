package cli

import (
	"strings"

	"github.com/rnwolfe/vabc"
	"github.com/rnwolfe/vabc/internal/errs"
)

// ProductCmd groups product reads. Search/get query the live web catalog (the
// site's Coveo index) — full coverage and current, including new/online-only SKUs.
type ProductCmd struct {
	Search ProductSearchCmd `cmd:"" help:"Search products by keyword (live web catalog)."`
	Get    ProductGetCmd    `cmd:"" help:"Get one product by 6-digit product code."`
}

type ProductSearchCmd struct {
	Query     string `arg:"" optional:"" help:"Keyword matched against product name (omit to browse)."`
	Type      string `help:"Filter by product category/type, e.g. bourbon, gin."`
	Allocated bool   `help:"Only allocated / limited-availability products."`
}

func (c *ProductSearchCmd) Run(rt *Runtime) error {
	products, err := rt.Client.SearchProducts(rt.Ctx, c.Query, rt.Cfg.Limit)
	if err != nil {
		return liveErr(err)
	}
	return rt.Out.Emit(filterProducts(products, c.Type, c.Allocated))
}

type ProductGetCmd struct {
	Code string `arg:"" help:"6-digit product code (e.g. 010807)."`
}

func (c *ProductGetCmd) Run(rt *Runtime) error {
	code := pad6(c.Code)
	products, err := rt.Client.SearchProducts(rt.Ctx, code, 10)
	if err != nil {
		return liveErr(err)
	}
	for _, p := range products {
		if p.ProductCode == code {
			return rt.Out.Emit(p)
		}
	}
	return errs.NotFound("product", code)
}

// filterProducts applies the --type and --allocated filters to search results.
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
