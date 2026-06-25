package vabc

// Product is a catalog record, keyed by ProductCode (6-digit, zero-padded).
// Populated from the catalog snapshot, not from a live API. Optional numeric
// fields are pointers so "unknown" is distinct from zero.
type Product struct {
	ProductCode     string   `json:"productCode"`
	Name            string   `json:"name"`
	Category        string   `json:"category,omitempty"`
	Type            string   `json:"type,omitempty"`
	Proof           *float64 `json:"proof,omitempty"`
	Size            string   `json:"size,omitempty"`
	RetailPrice     *float64 `json:"retailPrice,omitempty"`
	DiscountPrice   *float64 `json:"discountPrice,omitempty"`
	Allocated       bool     `json:"allocated"`
	OnlineOrderable bool     `json:"onlineOrderable"`
	New             bool     `json:"new"`
	UPC             []string `json:"upc,omitempty"`
	URL             string   `json:"url,omitempty"`
}

// Store is a Virginia ABC retail store. StoreNumber is the small integer ABC
// store number (the locator's "ABC Store 088" → 88), which is also the value the
// inventory API expects. Distance is miles from a query point, when relevant.
type Store struct {
	StoreNumber    int     `json:"storeNumber"`
	Name           string  `json:"name,omitempty"`
	Address        string  `json:"address,omitempty"`
	Address1       string  `json:"address1,omitempty"`
	Address2       string  `json:"address2,omitempty"`
	City           string  `json:"city,omitempty"`
	State          string  `json:"state,omitempty"`
	Zip            string  `json:"zip,omitempty"`
	Phone          string  `json:"phone,omitempty"`
	Lat            float64 `json:"lat,omitempty"`
	Lng            float64 `json:"lng,omitempty"`
	Hours          string   `json:"hours,omitempty"`
	ShoppingCenter string   `json:"shoppingCenter,omitempty"`
	URL            string   `json:"url,omitempty"`
	// Distance in miles from a query point, when relevant. A pointer so a genuine
	// 0.0 (the nearest store) renders, while "not computed" is omitted (not null=0).
	Distance *float64 `json:"distance,omitempty"`
}

// StoreStock is a store plus the on-hand quantity of a specific product.
type StoreStock struct {
	Store
	Quantity int `json:"quantity"`
}

// InventoryResult is the per-store availability of a product, with nearby stores
// that also stock it, ranked by distance (the /webapi/inventory/storeNearby shape).
type InventoryResult struct {
	ProductCode  string       `json:"productCode"`
	Store        StoreStock   `json:"store"`
	NearbyStores []StoreStock `json:"nearbyStores"`
}

// WarehouseResult is the statewide central-warehouse stock for a product.
type WarehouseResult struct {
	ProductCode        string `json:"productCode"`
	WarehouseInventory int    `json:"warehouseInventory"`
}

// LotteryEvent is one limited-availability event link. Its text/URL are
// CMS-authored free text — treat as untrusted (the CLI fences it in agent mode).
type LotteryEvent struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

// LotteryResult is the limited-availability ("lottery") status for a product.
// Allocated comes from the catalog flag; Active/EventLinks from the live hook.
type LotteryResult struct {
	ProductCode string         `json:"productCode"`
	Allocated   bool           `json:"allocated"`
	Active      bool           `json:"active"`
	EventLinks  []LotteryEvent `json:"eventLinks"`
}

// Envelope is the stable response wrapper for in-band scope/version metadata.
// SchemaVersion lets agents detect contract changes; Scope declares partial or
// cached results (e.g. "catalog snapshot 2026-06-01; live inventory") so a caller
// never mistakes a cached catalog for live stock.
type Envelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Scope         string `json:"scope,omitempty"`
	Data          any    `json:"data"`
	NextCursor    string `json:"nextCursor,omitempty"`
}
