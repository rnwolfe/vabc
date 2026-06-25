// Command vabc-catalog-gen regenerates the embedded catalog snapshot
// (catalog/data/catalog.json) from a Virginia ABC quarterly XLSX price list.
//
// Usage:
//
//	vabc-catalog-gen --from-xlsx ./q3-2026-price-list.xlsx --out catalog/data/catalog.json
//
// Run by the maintainer (or a scheduled CI job) once per quarter; the regenerated
// snapshot is committed and ships embedded in the next release.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rnwolfe/vabc"
	"github.com/rnwolfe/vabc/internal/harvest"
)

func main() {
	in := flag.String("from-xlsx", "", "path to the ABC quarterly price-list .xlsx (required)")
	out := flag.String("out", "catalog/data/catalog.json", "output snapshot path")
	date := flag.String("date", "", "snapshot date YYYY-MM-DD (default: file's quarter / today)")
	flag.Parse()

	if *in == "" {
		fmt.Fprintln(os.Stderr, "error: --from-xlsx is required")
		os.Exit(2)
	}

	products, err := harvest.FromXLSX(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}

	snap := struct {
		SchemaVersion int            `json:"schemaVersion"`
		SnapshotDate  string         `json:"snapshotDate"`
		Products      []vabc.Product `json:"products"`
	}{
		SchemaVersion: vabc.SchemaVersion,
		SnapshotDate:  snapshotDate(*date),
		Products:      products,
	}

	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %d products to %s\n", len(products), *out)
}

func snapshotDate(override string) string {
	if override != "" {
		return override
	}
	return time.Now().UTC().Format("2006-01-02")
}
