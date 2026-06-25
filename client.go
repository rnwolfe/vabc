package vabc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Endpoint defaults. The inventory routes are undocumented but public,
// unauthenticated, and (today) exempt from the site's Cloudflare challenge.
const (
	// DefaultBaseURL hosts the /webapi/inventory/* and /webapi/limitedavailability/* routes.
	DefaultBaseURL = "https://www.abc.virginia.gov"
	// DefaultStoresURL is the Virginia VGIN ArcGIS FeatureServer for the store locator.
	DefaultStoresURL = "https://services9.arcgis.com/6EuFgO4fLTqfNOhu/arcgis/rest/services/Virginia_ABC_Stores/FeatureServer/0/query"
	// DefaultUserAgent identifies the tool politely to an undocumented backend.
	DefaultUserAgent = "vabc (+https://github.com/rnwolfe/vabc)"
	// defaultMinInterval spaces requests from a fresh-process-per-call agent.
	defaultMinInterval = 250 * time.Millisecond
	// maxRetries for transient (5xx / network) errors.
	maxRetries = 2
)

// Client is the live Virginia ABC API surface. All methods are reads; there are
// no mutations. Implementations are safe for an agent's fresh-process-per-call
// usage (see the throttle).
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
	// SearchProducts runs a live product search against the site's Coveo index,
	// which covers the full web catalog (more complete and current than the
	// downloadable price list). Results carry the inventory product code.
	SearchProducts(ctx context.Context, query string, limit int) ([]Product, error)
}

// Option configures the HTTP client.
type Option func(*httpClient)

// WithBaseURL overrides the inventory/lottery host.
func WithBaseURL(u string) Option { return func(c *httpClient) { c.baseURL = strings.TrimRight(u, "/") } }

// WithStoresURL overrides the ArcGIS store-locator endpoint.
func WithStoresURL(u string) Option { return func(c *httpClient) { c.storesURL = u } }

// WithHTTPClient injects a custom *http.Client (e.g. for tests or a tuned transport).
func WithHTTPClient(h *http.Client) Option { return func(c *httpClient) { c.http = h } }

// WithUserAgent overrides the request User-Agent.
func WithUserAgent(ua string) Option { return func(c *httpClient) { c.userAgent = ua } }

// WithMinInterval sets the minimum spacing between requests (politeness throttle).
func WithMinInterval(d time.Duration) Option { return func(c *httpClient) { c.minInterval = d } }

// WithWait makes the client wait out an open circuit breaker (up to maxWait)
// instead of failing fast. Wired from the CLI's --wait/--max-wait flags.
func WithWait(wait bool, maxWait time.Duration) Option {
	return func(c *httpClient) { c.wait, c.maxWait = wait, maxWait }
}

// WithStatePath overrides the throttle state file (mainly for tests).
func WithStatePath(p string) Option { return func(c *httpClient) { c.statePath = p } }

// NewClient builds the default HTTP-backed Client, including the persistent
// cross-process throttle/circuit-breaker (contract §12).
func NewClient(opts ...Option) Client {
	c := &httpClient{
		baseURL:     DefaultBaseURL,
		storesURL:   DefaultStoresURL,
		userAgent:   DefaultUserAgent,
		http:        &http.Client{Timeout: 15 * time.Second},
		minInterval: defaultMinInterval,
		statePath:   defaultStatePath(),
		maxWait:     30 * time.Second,
	}
	for _, o := range opts {
		o(c)
	}
	c.throttle = newThrottle(c.statePath, c.minInterval, c.wait, c.maxWait)
	return c
}

type httpClient struct {
	baseURL     string
	storesURL   string
	userAgent   string
	http        *http.Client
	minInterval time.Duration
	statePath   string
	wait        bool
	maxWait     time.Duration
	throttle    *throttle
}

// getJSON performs a throttled, retried GET and decodes JSON into out.
// status classification: 400 → NotFound, 429/403-challenge → RateLimited (circuit
// breaker tripped), 5xx/network → Retryable (with bounded backoff).
func (c *httpClient) getJSON(ctx context.Context, url string, out any) error {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := c.throttle.acquire(ctx); err != nil {
			return err
		}
		body, status, err := c.do(ctx, url)
		if err != nil {
			lastErr = retryable(0, "request failed: "+err.Error(), err)
			if attempt < maxRetries {
				if werr := sleepCtx(ctx, backoff(attempt)); werr != nil {
					return werr
				}
				continue
			}
			return lastErr
		}

		switch {
		case status == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return schemaDrift("could not decode "+url, err)
			}
			return nil
		case status == http.StatusBadRequest:
			return notFound(status, strings.TrimSpace(firstLine(body)))
		case status == http.StatusTooManyRequests || isChallenge(status, body):
			ra := retryAfterFrom(body)
			c.throttle.observe(ra)
			return rateLimited(status, "blocked or rate-limited by the upstream", ra)
		case status >= 500:
			lastErr = retryable(status, "upstream error", nil)
			if attempt < maxRetries {
				if werr := sleepCtx(ctx, backoff(attempt)); werr != nil {
					return werr
				}
				continue
			}
			return lastErr
		default:
			return retryable(status, fmt.Sprintf("unexpected status %d", status), nil)
		}
	}
	return lastErr
}

func (c *httpClient) do(ctx context.Context, url string) ([]byte, int, error) {
	return c.doAccept(ctx, url, "application/json")
}

// postJSON performs a throttled, retried POST of a JSON body and decodes the JSON
// response into out, classifying failures like getJSON.
func (c *httpClient) postJSON(ctx context.Context, url string, reqBody, out any) error {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := c.throttle.acquire(ctx); err != nil {
			return err
		}
		body, status, err := c.doPost(ctx, url, payload)
		if err != nil {
			lastErr = retryable(0, "request failed: "+err.Error(), err)
			if attempt < maxRetries {
				if werr := sleepCtx(ctx, backoff(attempt)); werr != nil {
					return werr
				}
				continue
			}
			return lastErr
		}
		switch {
		case status == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return schemaDrift("could not decode "+url, err)
			}
			return nil
		case status == http.StatusTooManyRequests || isChallenge(status, body):
			ra := retryAfterFrom(body)
			c.throttle.observe(ra)
			return rateLimited(status, "blocked or rate-limited by the upstream", ra)
		case status >= 500:
			lastErr = retryable(status, "upstream error", nil)
			if attempt < maxRetries {
				if werr := sleepCtx(ctx, backoff(attempt)); werr != nil {
					return werr
				}
				continue
			}
			return lastErr
		default:
			return retryable(status, fmt.Sprintf("unexpected status %d", status), nil)
		}
	}
	return lastErr
}

func (c *httpClient) doPost(ctx context.Context, url string, payload []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func (c *httpClient) doAccept(ctx context.Context, url, accept string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func backoff(attempt int) time.Duration {
	return time.Duration(200*(1<<attempt)) * time.Millisecond
}

// isChallenge detects a Cloudflare/WAF interstitial returned with a non-OK status.
func isChallenge(status int, body []byte) bool {
	if status != http.StatusForbidden && status != http.StatusServiceUnavailable {
		return false
	}
	b := strings.ToLower(string(body))
	return strings.Contains(b, "just a moment") || strings.Contains(b, "cf-challenge") ||
		strings.Contains(b, "/cdn-cgi/") || strings.Contains(b, "attention required")
}

func retryAfterFrom(body []byte) time.Duration {
	// The JSON endpoints rarely set a parseable Retry-After in-body; default to a
	// conservative cooldown. (HTTP header parsing happens in do() callers if needed.)
	return 60 * time.Second
}

func firstLine(body []byte) string {
	s := string(body)
	// Some 400s are JSON {"message":"..."}; surface the message if present.
	var m map[string]any
	if json.Unmarshal(body, &m) == nil {
		for _, k := range []string{"message", "error", "Message"} {
			if v, ok := m[k].(string); ok && v != "" {
				return html.UnescapeString(v)
			}
		}
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return html.UnescapeString(s)
}

// pad6 normalizes a product code to the 6-digit zero-padded form the API expects.
func pad6(code string) string {
	code = strings.TrimSpace(code)
	if len(code) < 6 {
		return strings.Repeat("0", 6-len(code)) + code
	}
	return code
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
