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
	date := rt.Catalog.SnapshotDate()
	stale := false
	if t, err := time.Parse("2006-01-02", date); err == nil {
		stale = time.Since(t) > staleAfterDays*24*time.Hour
	}
	return rt.Out.Emit(map[string]any{
		"schemaVersion": vabc.SchemaVersion,
		"snapshotDate":  date,
		"productCount":  rt.Catalog.Count(),
		"source":        rt.Catalog.Source(),
		"stale":         stale,
	})
}

type CatalogRefreshCmd struct {
	FromXLSX string `name:"from-xlsx" help:"Path to a downloaded ABC quarterly price-list .xlsx." required:""`
	Out      string `help:"Override the snapshot output path (default: XDG cache)."`
}

func (c *CatalogRefreshCmd) Run(rt *Runtime) error {
	products, err := harvest.FromXLSX(c.FromXLSX)
	if err != nil {
		return errs.New(errs.ExitRetry, "NOT_IMPLEMENTED", err.Error(),
			"catalog generation is wired in the cli-implement stage")
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
		"source":       "cache:" + out,
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
