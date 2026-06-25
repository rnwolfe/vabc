// Package harvest converts Virginia ABC's quarterly XLSX price list into catalog
// products. It is internal so importers of the public vabc/catalog packages never
// inherit the XLSX (excelize) dependency — `go get github.com/rnwolfe/vabc` stays
// HTTP+JSON only. Both cmd/vabc-catalog-gen and the CLI's `catalog refresh` use it.
package harvest

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/rnwolfe/vabc"
)

// ErrNoCodeColumn is returned when the sheet has no recognizable product-code column.
var ErrNoCodeColumn = errors.New("harvest: no product-code column found in the price list")

// FromXLSX parses an ABC quarterly price-list .xlsx file into catalog products.
func FromXLSX(path string) ([]vabc.Product, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("harvest: open %s: %w", path, err)
	}
	defer f.Close()
	return parse(f)
}

// FromBytes parses ABC price-list .xlsx bytes (used by the auto-pull refresh path).
func FromBytes(b []byte) ([]vabc.Product, error) {
	f, err := excelize.OpenReader(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("harvest: open workbook: %w", err)
	}
	defer f.Close()
	return parse(f)
}

// parse walks the first sheet. The real layout has a title banner, then per-section
// category banner rows ("ARMAGNAC/BRANDY/COGNAC"), a header row ("Code | Item Name |
// Size | Proof | … | Price | …"), then product rows. The header may repeat per
// section. Header-mapping (not fixed columns) tolerates quarter-to-quarter changes.
func parse(f *excelize.File) ([]vabc.Product, error) {
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("harvest: workbook has no sheets")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("harvest: read rows: %w", err)
	}

	headerIdx, cols := findHeader(rows)
	if headerIdx < 0 || cols["code"] < 0 {
		return nil, ErrNoCodeColumn
	}
	codeCol := cols["code"]

	var products []vabc.Product
	bannerCat := "" // running category from section-banner rows (no category column)
	for _, row := range rows {
		if nonEmptyCount(row) == 0 {
			continue
		}
		if isHeaderRow(row, codeCol) {
			continue
		}
		code := cell(row, codeCol)
		if isProductCode(code) {
			// Prefer an explicit category column; fall back to the banner section.
			cat := cell(row, cols["category"])
			if cat == "" {
				cat = bannerCat
			}
			p := vabc.Product{
				ProductCode:   pad6(code),
				Name:          cell(row, cols["name"]),
				Category:      cat,
				Size:          cell(row, cols["size"]),
				Proof:         optFloat(cell(row, cols["proof"])),
				RetailPrice:   optFloat(cell(row, cols["price"])),
				DiscountPrice: optFloat(cell(row, cols["discount"])),
				Allocated:     truthy(cell(row, cols["allocated"])),
			}
			if upc := cell(row, cols["upc"]); upc != "" {
				p.UPC = []string{upc}
			}
			products = append(products, p)
			continue
		}
		// A lone-cell row that isn't a product or header is a category banner.
		if t := bannerText(row); t != "" {
			bannerCat = t
		}
	}
	return products, nil
}

// fieldMatchers maps a logical field to header substrings (checked in order).
// More specific fields are resolved before generic ones (e.g. discount before price).
var fieldMatchers = []struct {
	field    string
	contains []string
}{
	{"code", []string{"code", "nc code", "sku"}},
	{"discount", []string{"discount", "sale price"}},
	{"price", []string{"retail", "price", "bottle"}},
	{"proof", []string{"proof"}},
	{"size", []string{"size"}},
	{"upc", []string{"upc"}},
	{"category", []string{"category", "class"}},
	{"type", []string{"type"}},
	{"name", []string{"item name", "brand", "product name", "description", "name", "product"}},
	{"allocated", []string{"allocated", "limited"}},
}

// findHeader returns the index of the header row and a field→column map, scanning
// the first rows for the one that yields a code column.
func findHeader(rows [][]string) (int, map[string]int) {
	best := -1
	var bestCols map[string]int
	limit := len(rows)
	if limit > 30 {
		limit = 30
	}
	for i := 0; i < limit; i++ {
		cols := mapColumns(rows[i])
		if cols["code"] >= 0 {
			return i, cols
		}
		if best < 0 {
			best, bestCols = i, cols
		}
	}
	return best, bestCols
}

func mapColumns(header []string) map[string]int {
	cols := map[string]int{}
	for _, fm := range fieldMatchers {
		cols[fm.field] = -1
	}
	used := map[int]bool{}
	for _, fm := range fieldMatchers {
		for i, h := range header {
			if used[i] {
				continue
			}
			hn := strings.ToLower(strings.TrimSpace(h))
			if hn == "" {
				continue
			}
			for _, want := range fm.contains {
				if strings.Contains(hn, want) {
					cols[fm.field] = i
					used[i] = true
					break
				}
			}
			if cols[fm.field] >= 0 {
				break
			}
		}
	}
	return cols
}

func nonEmptyCount(row []string) int {
	n := 0
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			n++
		}
	}
	return n
}

// isHeaderRow reports whether the row is a (possibly repeated) column header.
func isHeaderRow(row []string, codeCol int) bool {
	return strings.EqualFold(cell(row, codeCol), "code")
}

// isProductCode reports whether s looks like a product code (all digits, len>=3).
func isProductCode(s string) bool {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i]
	}
	if len(s) < 3 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// bannerText returns a category banner's text if the row has exactly one non-empty,
// non-numeric cell; otherwise "".
func bannerText(row []string) string {
	text := ""
	count := 0
	for _, c := range row {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		count++
		text = c
	}
	if count == 1 && !isProductCode(text) {
		return text
	}
	return ""
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func pad6(code string) string {
	code = strings.TrimSpace(code)
	if i := strings.IndexByte(code, '.'); i >= 0 {
		code = code[:i]
	}
	if len(code) < 6 {
		return strings.Repeat("0", 6-len(code)) + code
	}
	return code
}

func optFloat(s string) *float64 {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "$"))
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil {
		return nil
	}
	return &f
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes", "true", "1", "x", "allocated", "limited":
		return true
	}
	return false
}
