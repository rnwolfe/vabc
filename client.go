package vabc

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// ErrNotImplemented is returned by the scaffold's placeholder client methods.
// cli-implement replaces the method bodies with real HTTP calls and removes it.
var ErrNotImplemented = errors.New("vabc: not implemented yet (wired by cli-implement)")

// Endpoint defaults. The inventory routes are undocumented but public,
// unauthenticated, and (today) exempt from the site's Cloudflare challenge.
const (
	// DefaultBaseURL hosts the /webapi/inventory/* and /webapi/limitedavailability/* routes.
	DefaultBaseURL = "https://www.abc.virginia.gov"
	// DefaultStoresURL is the Virginia VGIN ArcGIS FeatureServer for the store locator.
	DefaultStoresURL = "https://services9.arcgis.com/6EuFgO4fLTqfNOhu/arcgis/rest/services/Virginia_ABC_Stores/FeatureServer/0/query"
	// DefaultUserAgent identifies the tool politely to an undocumented backend.
	DefaultUserAgent = "vabc (+https://github.com/rnwolfe/vabc)"
)

// Client is the live Virginia ABC API surface. All methods are reads; there are
// no mutations. Implementations must be safe for an agent's fresh-process-per-call
// usage (see the throttle note in NewClient).
type Client interface {
	// StoreNearby returns the anchor store's stock of a product plus other nearby
	// stores that carry it, ranked by distance (/webapi/inventory/storeNearby).
	StoreNearby(ctx context.Context, storeNumber int, productCode string) (InventoryResult, error)
	// MyStore returns just one store's stock of a product (/webapi/inventory/mystore).
	MyStore(ctx context.Context, storeNumber int, productCode string) (StoreStock, error)
	// Warehouse returns statewide central-warehouse stock (/webapi/inventory/store).
	Warehouse(ctx context.Context, productCode string) (WarehouseResult, error)
	// Stores returns all Virginia ABC retail stores (ArcGIS FeatureServer).
	Stores(ctx context.Context) ([]Store, error)
	// StoreNear returns stores nearest a point, ranked by distance.
	StoreNear(ctx context.Context, lat, lng float64, limit int) ([]Store, error)
	// LimitedAvailability returns the lottery/allocated event hook for a product
	// (/webapi/limitedavailability/eventLinks).
	LimitedAvailability(ctx context.Context, productCode string) (LotteryResult, error)
}

// Option configures the HTTP client.
type Option func(*httpClient)

// WithBaseURL overrides the inventory/lottery host.
func WithBaseURL(u string) Option { return func(c *httpClient) { c.baseURL = u } }

// WithStoresURL overrides the ArcGIS store-locator endpoint.
func WithStoresURL(u string) Option { return func(c *httpClient) { c.storesURL = u } }

// WithHTTPClient injects a custom *http.Client (e.g. for tests or a tuned transport).
func WithHTTPClient(h *http.Client) Option { return func(c *httpClient) { c.http = h } }

// WithUserAgent overrides the request User-Agent.
func WithUserAgent(ua string) Option { return func(c *httpClient) { c.userAgent = ua } }

// NewClient builds the default HTTP-backed Client.
//
// TODO(cli-implement): the real implementation must carry PERSISTENT, cross-process
// throttle/backoff state (an agent spawns a fresh process per call, so an in-memory
// timer is a no-op). Persist a token-bucket / last-call timestamp under
// os.UserCacheDir()/vabc, honor Retry-After, and circuit-break on a block. Never add
// evasion (UA spoofing, proxy rotation, CAPTCHA solving) — this is a courtesy surface.
func NewClient(opts ...Option) Client {
	c := &httpClient{
		baseURL:   DefaultBaseURL,
		storesURL: DefaultStoresURL,
		userAgent: DefaultUserAgent,
		http:      &http.Client{Timeout: 15 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// httpClient is the placeholder live implementation. Methods return
// ErrNotImplemented until cli-implement wires the real requests.
type httpClient struct {
	baseURL   string
	storesURL string
	userAgent string
	http      *http.Client
}

func (c *httpClient) StoreNearby(ctx context.Context, storeNumber int, productCode string) (InventoryResult, error) {
	return InventoryResult{}, ErrNotImplemented
}

func (c *httpClient) MyStore(ctx context.Context, storeNumber int, productCode string) (StoreStock, error) {
	return StoreStock{}, ErrNotImplemented
}

func (c *httpClient) Warehouse(ctx context.Context, productCode string) (WarehouseResult, error) {
	return WarehouseResult{}, ErrNotImplemented
}

func (c *httpClient) Stores(ctx context.Context) ([]Store, error) {
	return nil, ErrNotImplemented
}

func (c *httpClient) StoreNear(ctx context.Context, lat, lng float64, limit int) ([]Store, error) {
	return nil, ErrNotImplemented
}

func (c *httpClient) LimitedAvailability(ctx context.Context, productCode string) (LotteryResult, error) {
	return LotteryResult{}, ErrNotImplemented
}
