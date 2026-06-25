package cli

import (
	"strings"
)

// LotteryCmd groups limited-availability ("lottery" / allocated) reads.
type LotteryCmd struct {
	Check LotteryCheckCmd `cmd:"" help:"Check active limited-availability events for a product."`
}

type LotteryCheckCmd struct {
	Code string `arg:"" help:"6-digit product code."`
}

func (c *LotteryCheckCmd) Run(rt *Runtime) error {
	res, err := rt.Client.LimitedAvailability(rt.Ctx, c.Code)
	if err != nil {
		return liveErr(err)
	}
	// eventLinks are CMS-authored free text — fence titles as untrusted (contract §8)
	// so a downstream agent does not execute embedded instructions.
	if rt.Cfg.WrapUntrusted {
		for i := range res.EventLinks {
			if res.EventLinks[i].Title != "" {
				res.EventLinks[i].Title = fenceUntrusted(res.EventLinks[i].Title)
			}
		}
	}
	// The allocated flag comes from the product's web-catalog record (the live hook
	// only carries active event links). Best-effort; ignore lookup failures.
	code := pad6(c.Code)
	if products, perr := rt.Client.SearchProducts(rt.Ctx, code, 10); perr == nil {
		for _, p := range products {
			if p.ProductCode == code {
				res.Allocated = p.Allocated
				break
			}
		}
	}
	rt.Out.Info("scope: live limited-availability hook for product %s", c.Code)
	return rt.Out.Emit(res)
}

// fenceUntrusted wraps target-originated free text so an agent treats it as data.
func fenceUntrusted(s string) string {
	return "⟦UNTRUSTED⟧ " + s + " ⟦/UNTRUSTED⟧"
}

// pad6 normalizes a product code to 6-digit zero-padded form (matches the API/catalog).
func pad6(code string) string {
	if len(code) < 6 {
		return strings.Repeat("0", 6-len(code)) + code
	}
	return code
}
