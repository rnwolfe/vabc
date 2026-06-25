package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rnwolfe/vabc"
	"github.com/rnwolfe/vabc/internal/errs"
	"github.com/rnwolfe/vabc/internal/harvest"
)

// staleAfterDays is the freshness threshold; the ABC price list refreshes quarterly.
const staleAfterDays = 120

// catalogStale reports whether the loaded snapshot is older than the threshold.
func (rt *Runtime) catalogStale() bool {
	if rt.Catalog == nil {
		return false
	}
	t, err := time.Parse("2006-01-02", rt.Catalog.SnapshotDate())
	if err != nil {
		return false
	}
	return time.Since(t) > staleAfterDays*24*time.Hour
}

// warnIfStale emits a one-line refresh hint to stderr only when the catalog is
// stale. A current snapshot prints nothing (no per-call scope noise).
func (rt *Runtime) warnIfStale() {
	if rt.catalogStale() {
		rt.Out.Info("note: catalog snapshot is from %s; run `vabc catalog refresh` to update",
			rt.Catalog.SnapshotDate())
	}
}

// CatalogCmd manages the local product catalog snapshot.
type CatalogCmd struct {
	Status  CatalogStatusCmd  `cmd:"" help:"Show catalog snapshot freshness and source."`
	Refresh CatalogRefreshCmd `cmd:"" help:"Rebuild the local snapshot from a downloaded ABC price list."`
}

type CatalogStatusCmd struct{}

func (c *CatalogStatusCmd) Run(rt *Runtime) error {
	if rt.Catalog == nil {
		return errs.CatalogUnavailable()
	}
	return rt.Out.Emit(map[string]any{
		"schemaVersion": vabc.SchemaVersion,
		"snapshotDate":  rt.Catalog.SnapshotDate(),
		"productCount":  rt.Catalog.Count(),
		"source":        rt.Catalog.Source(),
		"stale":         rt.catalogStale(),
	})
}

type CatalogRefreshCmd struct {
	FromXLSX string `name:"from-xlsx" help:"Use a local ABC price-list .xlsx instead of auto-downloading the latest."`
	Out      string `help:"Override the snapshot output path (default: XDG cache)."`
}

func (c *CatalogRefreshCmd) Run(rt *Runtime) error {
	var (
		products []vabc.Product
		source   string
		err      error
	)
	if c.FromXLSX != "" {
		products, err = harvest.FromXLSX(c.FromXLSX)
		source = "file:" + c.FromXLSX
		if err != nil {
			return errs.New(errs.ExitConfig, "PARSE_ERROR", err.Error(),
				"ensure the file is an ABC quarterly price-list .xlsx")
		}
	} else {
		rt.Out.Info("downloading the latest Virginia ABC price list…")
		var data []byte
		data, source, err = vabc.FetchLatestPriceList(rt.Ctx, clientOptions(rt.Cfg)...)
		if err != nil {
			return liveErr(err)
		}
		products, err = harvest.FromBytes(data)
		if err != nil {
			return errs.New(errs.ExitConfig, "PARSE_ERROR", err.Error(),
				"the downloaded price list could not be parsed; try --from-xlsx with a manual download")
		}
	}
	if len(products) == 0 {
		return errs.New(errs.ExitConfig, "EMPTY_CATALOG", "no products parsed from the price list",
			"the price-list format may have changed; please file an issue")
	}

	out := c.Out
	if out == "" {
		out = defaultCachePath()
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
		return errs.New(errs.ExitConfig, "WRITE_ERROR", err.Error(), "check the output directory is writable")
	}
	snap := map[string]any{
		"schemaVersion": vabc.SchemaVersion,
		"snapshotDate":  time.Now().UTC().Format("2006-01-02"),
		"source":        source,
		"products":      products,
	}
	b, _ := json.MarshalIndent(snap, "", "  ")
	if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
		return errs.New(errs.ExitConfig, "WRITE_ERROR", err.Error(), "check the output path is writable")
	}
	return rt.Out.Emit(map[string]any{
		"ok":           true,
		"snapshotDate": snap["snapshotDate"],
		"productCount": len(products),
		"source":       source,
		"written":      out,
	})
}

// defaultCachePath is where `catalog refresh` writes by default.
func defaultCachePath() string {
	if p := os.Getenv("VABC_CATALOG"); p != "" {
		return p
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "vabc-catalog.json"
	}
	return filepath.Join(dir, "vabc", "catalog.json")
}
