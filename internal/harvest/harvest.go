// Package harvest converts Virginia ABC's quarterly XLSX price list into catalog
// products. It is internal so importers of the public vabc/catalog packages never
// inherit the XLSX (excelize) dependency — `go get github.com/rnwolfe/vabc` stays
// HTTP+JSON only. Both cmd/vabc-catalog-gen and the CLI's `catalog refresh` use it.
package harvest

import (
	"errors"

	"github.com/rnwolfe/vabc"
)

// ErrNotImplemented is returned until cli-implement adds the excelize-based parser.
var ErrNotImplemented = errors.New("harvest: XLSX parsing not implemented yet (wired by cli-implement)")

// FromXLSX parses an ABC quarterly price-list .xlsx into catalog products.
//
// TODO(cli-implement): add github.com/xuri/excelize/v2, open the workbook, map the
// price-list columns (product code, name, category/type, size, proof, retail/discount
// price, allocated flag, UPCs) onto vabc.Product, and return the rows. Keep excelize
// confined to this package.
func FromXLSX(path string) ([]vabc.Product, error) {
	return nil, ErrNotImplemented
}
