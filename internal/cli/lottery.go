package cli

import "context"

// LotteryCmd groups limited-availability ("lottery" / allocated) reads.
type LotteryCmd struct {
	Check LotteryCheckCmd `cmd:"" help:"Check active limited-availability events for a product."`
}

type LotteryCheckCmd struct {
	Code string `arg:"" help:"6-digit product code."`
}

func (c *LotteryCheckCmd) Run(rt *Runtime) error {
	res, err := rt.Client.LimitedAvailability(context.Background(), c.Code)
	if err != nil {
		return liveErr(err)
	}
	// NOTE(cli-implement): eventLinks are CMS-authored free text — fence them as
	// untrusted by default in agent mode (contract §8) before emitting.
	rt.Out.Info("scope: live limited-availability hook for product %s", c.Code)
	return rt.Out.Emit(res)
}
