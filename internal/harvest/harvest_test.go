package harvest

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// writeXLSX builds a synthetic price-list workbook so the parser is tested without
// the real (Cloudflare-gated) file. Columns are deliberately reordered vs. a naive
// layout to prove header-mapping (not fixed columns).
func writeXLSX(t *testing.T, rows [][]string) string {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for r, row := range rows {
		for c, val := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			_ = f.SetCellValue(sheet, cell, val)
		}
	}
	path := filepath.Join(t.TempDir(), "pricelist.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFromXLSX(t *testing.T) {
	path := writeXLSX(t, [][]string{
		{"Some Title Row", "", "", ""},
		{"Brand", "Code", "Size", "Retail Price", "Proof", "Allocated", "Category"},
		{"Crown Royal Regal Apple", "10807", "750ml", "$24.99", "70", "N", "Whisky"},
		{"Sample Allocated Bourbon", "12345", "750ml", "89.99", "100", "Y", "Bourbon"},
		{"", "", "", "", "", "", ""}, // blank row skipped
	})

	products, err := FromXLSX(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 2 {
		t.Fatalf("want 2 products, got %d: %+v", len(products), products)
	}

	p := products[0]
	if p.ProductCode != "010807" {
		t.Fatalf("code not zero-padded: %q", p.ProductCode)
	}
	if p.Name != "Crown Royal Regal Apple" || p.Size != "750ml" || p.Category != "Whisky" {
		t.Fatalf("fields mismapped: %+v", p)
	}
	if p.RetailPrice == nil || *p.RetailPrice != 24.99 {
		t.Fatalf("price not parsed (stripped $): %+v", p.RetailPrice)
	}
	if p.Proof == nil || *p.Proof != 70 {
		t.Fatalf("proof not parsed: %+v", p.Proof)
	}
	if p.Allocated {
		t.Fatalf("row 1 should not be allocated")
	}
	if !products[1].Allocated {
		t.Fatalf("row 2 should be allocated")
	}
}

func TestFromXLSXNoCodeColumn(t *testing.T) {
	path := writeXLSX(t, [][]string{
		{"Brand", "Size", "Price"},
		{"Something", "750ml", "9.99"},
	})
	if _, err := FromXLSX(path); err != ErrNoCodeColumn {
		t.Fatalf("want ErrNoCodeColumn, got %v", err)
	}
}
