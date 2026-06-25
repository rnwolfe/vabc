package vabc

import (
	"context"
	"fmt"
	"strconv"
)

// rawStoreInfo is the per-store block returned by the inventory endpoints. Field
// names follow the upstream JSON (case-insensitively matched).
type rawStoreInfo struct {
	StoreID        int     `json:"storeId"`
	Quantity       int     `json:"quantity"`
	Distance       float64 `json:"distance"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	Address        string  `json:"address"`
	Address1       string  `json:"address1"`
	Address2       *string `json:"address2"`
	City           string  `json:"city"`
	State          string  `json:"state"`
	Zip            string  `json:"zip"`
	URL            string  `json:"url"`
	Hours          string  `json:"hours"`
	ShoppingCenter string  `json:"shoppingCenter"`
	PhoneNumber    struct {
		FormattedPhoneNumber string `json:"FormattedPhoneNumber"`
	} `json:"PhoneNumber"`
}

func (r rawStoreInfo) toStoreStock() StoreStock {
	addr2 := ""
	if r.Address2 != nil {
		addr2 = *r.Address2
	}
	dist := r.Distance
	return StoreStock{
		Store: Store{
			StoreNumber:    r.StoreID,
			Address:        r.Address,
			Address1:       r.Address1,
			Address2:       addr2,
			City:           r.City,
			State:          r.State,
			Zip:            r.Zip,
			Phone:          r.PhoneNumber.FormattedPhoneNumber,
			Lat:            r.Latitude,
			Lng:            r.Longitude,
			Hours:          r.Hours,
			ShoppingCenter: r.ShoppingCenter,
			URL:            r.URL,
			Distance:       &dist,
		},
		Quantity: r.Quantity,
	}
}

type rawInventory struct {
	Products []struct {
		ProductID    string         `json:"productId"`
		StoreInfo    rawStoreInfo   `json:"storeInfo"`
		NearbyStores []rawStoreInfo `json:"nearbyStores"`
	} `json:"products"`
}

// StoreNearby returns the anchor store's stock plus nearby stores stocking it.
func (c *httpClient) StoreNearby(ctx context.Context, storeNumber int, productCode string) (InventoryResult, error) {
	code := pad6(productCode)
	url := fmt.Sprintf("%s/webapi/inventory/storeNearby?storeNumber=%d&productCode=%s",
		c.baseURL, storeNumber, code)
	var raw rawInventory
	if err := c.getJSON(ctx, url, &raw); err != nil {
		return InventoryResult{}, err
	}
	if len(raw.Products) == 0 {
		// Valid store + unknown product still returns a product row; an empty list
		// is unexpected. Surface a not-found rather than an empty success.
		return InventoryResult{}, notFound(0, "no inventory record for product "+code+" at store "+strconv.Itoa(storeNumber))
	}
	p := raw.Products[0]
	res := InventoryResult{
		ProductCode:  code,
		Store:        p.StoreInfo.toStoreStock(),
		NearbyStores: make([]StoreStock, 0, len(p.NearbyStores)),
	}
	for _, ns := range p.NearbyStores {
		res.NearbyStores = append(res.NearbyStores, ns.toStoreStock())
	}
	return res, nil
}

// MyStore returns one store's stock of a product (leaner endpoint).
func (c *httpClient) MyStore(ctx context.Context, storeNumber int, productCode string) (StoreStock, error) {
	code := pad6(productCode)
	// Params are plural-named but singular server-side; never pass comma lists.
	url := fmt.Sprintf("%s/webapi/inventory/mystore?storeNumbers=%d&productCodes=%s",
		c.baseURL, storeNumber, code)
	var raw rawInventory
	if err := c.getJSON(ctx, url, &raw); err != nil {
		return StoreStock{}, err
	}
	if len(raw.Products) == 0 {
		return StoreStock{}, notFound(0, "no inventory record for product "+code+" at store "+strconv.Itoa(storeNumber))
	}
	return raw.Products[0].StoreInfo.toStoreStock(), nil
}

type rawWarehouse struct {
	// The upstream returns the count as a string, e.g. {"warehouseInventory":"100"}.
	WarehouseInventory string `json:"warehouseInventory"`
}

// Warehouse returns statewide central-warehouse stock for a product.
func (c *httpClient) Warehouse(ctx context.Context, productCode string) (WarehouseResult, error) {
	code := pad6(productCode)
	url := fmt.Sprintf("%s/webapi/inventory/store?productId=%s", c.baseURL, code)
	var raw rawWarehouse
	if err := c.getJSON(ctx, url, &raw); err != nil {
		return WarehouseResult{}, err
	}
	return WarehouseResult{ProductCode: code, WarehouseInventory: atoiSafe(raw.WarehouseInventory)}, nil
}
