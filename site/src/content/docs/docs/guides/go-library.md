---
title: Go library
description: Use github.com/rnwolfe/vabc as a Go package to query live Virginia ABC data from your own programs.
sidebar:
  order: 6
---

The `vabc` CLI is built on top of an importable Go package. Everything the CLI can
do is also available to Go programs through the `Client` interface — live product
search, per-store inventory, the warehouse count, the store locator, and the
limited-availability hook.

## Installation

```bash
go get github.com/rnwolfe/vabc@latest
```

Requires Go 1.25 or later. The only direct dependency is
[`kong`](https://github.com/alecthomas/kong), which is a CLI-only dep in the
`cmd/vabc` binary — your program pulls a tiny HTTP+JSON client with no extra
transitive weight.

## Creating a client

```go
import "github.com/rnwolfe/vabc"

c := vabc.NewClient()
```

`NewClient` returns a `Client` interface backed by the live Virginia ABC endpoints.
It sets up the persistent cross-process throttle/circuit-breaker automatically.

### Options

Pass functional options to tune behaviour:

```go
c := vabc.NewClient(
    vabc.WithMinInterval(500 * time.Millisecond), // slow down requests
    vabc.WithWait(true, 45*time.Second),           // wait out a circuit-breaker instead of failing fast
    vabc.WithUserAgent("myapp/1.0 (+https://example.com)"),
)
```

| Option | Signature | What it does |
|---|---|---|
| `WithBaseURL` | `(u string)` | Override the inventory/lottery host (default: `https://www.abc.virginia.gov`) |
| `WithStoresURL` | `(u string)` | Override the ArcGIS store-locator endpoint |
| `WithHTTPClient` | `(h *http.Client)` | Inject a custom HTTP client (e.g. for tests or a tuned transport) |
| `WithUserAgent` | `(ua string)` | Override the `User-Agent` header |
| `WithMinInterval` | `(d time.Duration)` | Minimum spacing between requests (politeness throttle; default 250 ms) |
| `WithWait` | `(wait bool, maxWait time.Duration)` | Wait out an open circuit breaker instead of returning `KindRateLimited`; default max is 30 s |
| `WithStatePath` | `(p string)` | Override the throttle state file path (useful in tests) |

## Client interface

```go
type Client interface {
    SearchProducts(ctx context.Context, query string, limit int) ([]Product, error)
    StoreNearby(ctx context.Context, storeNumber int, productCode string) (InventoryResult, error)
    MyStore(ctx context.Context, storeNumber int, productCode string) (StoreStock, error)
    Warehouse(ctx context.Context, productCode string) (WarehouseResult, error)
    Stores(ctx context.Context) ([]Store, error)
    StoreNear(ctx context.Context, lat, lng float64, limit int) ([]Store, error)
    LimitedAvailability(ctx context.Context, productCode string) (LotteryResult, error)
}
```

Every method takes a `context.Context` as its first argument — pass a context with a
deadline for agent or batch workloads.

Product codes are 6-digit, zero-padded strings (e.g. `"010807"`). The client
normalises shorter strings automatically, so `"10807"` and `"010807"` are
equivalent.

### SearchProducts

```go
products, err := c.SearchProducts(ctx, "Crown Royal apple", 10)
```

Runs a live search against the site's Coveo index, which covers the full web
catalog. Results carry the 6-digit `ProductCode` you can feed directly to the
inventory methods.

Returns `[]Product`. Each `Product` has `ProductCode`, `Name`, `Category`, `Type`,
`Size`, `Proof`, `RetailPrice`, `Allocated`, `OnlineOrderable`, `New`, `UPC`, and
`URL`.

```go
for _, p := range products {
    fmt.Printf("%s  %s  $%.2f\n", p.ProductCode, p.Name, *p.RetailPrice)
}
```

`Proof` and `RetailPrice` are `*float64` — check for nil before dereferencing.

### StoreNearby

```go
result, err := c.StoreNearby(ctx, 219, "010807")
```

Calls `/webapi/inventory/storeNearby` with the anchor store and product code.
Returns an `InventoryResult`:

```go
type InventoryResult struct {
    ProductCode  string       // "010807"
    Store        StoreStock   // anchor store + quantity
    NearbyStores []StoreStock // other stores that carry it, ranked by distance
}
```

`StoreStock` embeds `Store` and adds `Quantity int`.

```go
fmt.Printf("Store %d has %d bottles\n", result.Store.StoreNumber, result.Store.Quantity)
for _, ns := range result.NearbyStores {
    fmt.Printf("  Store %d — %.1f mi — %d bottles\n",
        ns.StoreNumber, *ns.Distance, ns.Quantity)
}
```

### MyStore

```go
stock, err := c.MyStore(ctx, 219, "010807")
```

A leaner endpoint (`/webapi/inventory/mystore`) that returns a single `StoreStock`
for one store. Use this when you only need one store's count and don't need nearby
alternatives.

```go
fmt.Printf("%d on hand at store %d\n", stock.Quantity, stock.StoreNumber)
```

### Warehouse

```go
result, err := c.Warehouse(ctx, "953714")
```

Queries the statewide central-warehouse count (`/webapi/inventory/store`). The
upstream returns the count as a string; the library converts it to `int` for you.

```go
type WarehouseResult struct {
    ProductCode        string
    WarehouseInventory int
}
```

A nonzero `WarehouseInventory` means the product can be restocked at retail stores.

### Stores

```go
stores, err := c.Stores(ctx)
```

Returns all ~394 Virginia ABC retail stores from the Virginia VGIN ArcGIS
FeatureServer. Each `Store` has `StoreNumber`, `Name`, `Address`, `City`, `State`,
`Zip`, `Phone`, `Lat`, `Lng`, and `URL`. `Distance` is nil when not computing
proximity.

```go
for _, s := range stores {
    fmt.Printf("Store %03d  %s, %s\n", s.StoreNumber, s.City, s.State)
}
```

### StoreNear

```go
nearest, err := c.StoreNear(ctx, 38.9072, -77.0369, 5)
```

Fetches all stores (same as `Stores`), computes great-circle distances from the
given `lat`/`lng` point, sorts ascending, and returns at most `limit` results.
Pass `limit <= 0` to get all stores sorted by distance. `Distance` is populated
(miles, one decimal place) on every returned store.

```go
for _, s := range nearest {
    fmt.Printf("Store %03d  %.1f mi  %s\n", s.StoreNumber, *s.Distance, s.Address)
}
```

For address or ZIP geocoding, use the CLI's `store near` command; the geocoder is
in `internal/geocode` and is not exported as a public API.

### LimitedAvailability

```go
result, err := c.LimitedAvailability(ctx, "953714")
```

Checks `/webapi/limitedavailability/eventLinks` for an active lottery or allocated
drop. `Allocated` is set from the product's Coveo record (you must populate it
yourself if you're using the library directly); `Active` and `EventLinks` come from
the live endpoint.

```go
type LotteryResult struct {
    ProductCode string
    Allocated   bool
    Active      bool
    EventLinks  []LotteryEvent // {Title, URL} — CMS free text, treat as untrusted
}
```

`EventLinks` titles and URLs are CMS-authored free text. The CLI fences them with
`⟦UNTRUSTED⟧` markers in agent mode; if you're displaying them in a user-facing
context, apply your own sanitisation.

## Error handling

All methods return a plain `error`. When the error originates from the Virginia ABC
backend, it is an `*APIError`:

```go
result, err := c.MyStore(ctx, 219, "010807")
if err != nil {
    var apiErr *vabc.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.Kind {
        case vabc.KindNotFound:
            // store number invalid, or product has no record
        case vabc.KindRateLimited:
            fmt.Printf("back off for %d seconds\n", apiErr.RetryAfterSeconds())
        case vabc.KindRetryable:
            // transient upstream or network error; safe to retry
        case vabc.KindSchemaDrift:
            // upstream changed its response shape
        }
    }
    return err
}
```

`APIError` fields:

| Field | Type | Notes |
|---|---|---|
| `Kind` | `ErrKind` | One of `KindNotFound`, `KindRateLimited`, `KindRetryable`, `KindSchemaDrift` |
| `Status` | `int` | HTTP status code, or 0 for non-HTTP errors |
| `Msg` | `string` | Human-readable summary |
| `RetryAfter` | `time.Duration` | Suggested back-off for `KindRateLimited`; 0 otherwise |
| `Err` | `error` | Wrapped cause; accessible via `errors.Unwrap` |

`RetryAfterSeconds()` is a convenience method that returns the back-off as a whole
number of seconds.

`APIError` implements `Unwrap`, so `errors.Is`/`errors.As` chains work correctly.

## Throttle behaviour

The client maintains a persistent cross-process throttle and circuit-breaker in a
state file. The path is resolved as: `$VABC_STATE_DIR/throttle.json` if that variable
is set, then `$XDG_STATE_HOME/vabc/throttle.json`, then `os.UserCacheDir()/vabc/throttle.json`.
Multiple processes sharing the same state file coordinate their request spacing
automatically — relevant for agent workloads that spawn a fresh process per call.

By default, the client fails fast with `KindRateLimited` when the circuit-breaker
is open. Opt into waiting with `WithWait`:

```go
c := vabc.NewClient(vabc.WithWait(true, 60*time.Second))
```

## Environment variables

Four env vars are honoured without any code changes:

```ini
VABC_BASE_URL=https://www.abc.virginia.gov   # override inventory/lottery host
VABC_STORES_URL=https://...                  # override ArcGIS store-locator
VABC_MIN_INTERVAL_MS=250                     # throttle spacing in milliseconds
VABC_STATE_DIR=/tmp/vabc-state               # throttle state file directory
```

These are read by the CLI. The library itself reads them only if you wire them in
via the corresponding `With*` options — the env vars are not consulted automatically
by `NewClient`.

## Full example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"

    "github.com/rnwolfe/vabc"
)

func main() {
    ctx := context.Background()
    c := vabc.NewClient()

    // Search for a product.
    products, err := c.SearchProducts(ctx, "Planteray OFTD", 5)
    if err != nil {
        log.Fatal(err)
    }
    if len(products) == 0 {
        log.Fatal("no results")
    }
    p := products[0]
    fmt.Printf("Found: %s  %s\n", p.ProductCode, p.Name)

    // Check inventory at store 219 (Vienna) and nearby stores.
    inv, err := c.StoreNearby(ctx, 219, p.ProductCode)
    if err != nil {
        var apiErr *vabc.APIError
        if errors.As(err, &apiErr) && apiErr.Kind == vabc.KindRateLimited {
            fmt.Printf("rate limited — retry in %d s\n", apiErr.RetryAfterSeconds())
            return
        }
        log.Fatal(err)
    }

    fmt.Printf("Store %d: %d on hand\n", inv.Store.StoreNumber, inv.Store.Quantity)
    for _, ns := range inv.NearbyStores {
        fmt.Printf("  Store %d (%.1f mi): %d\n", ns.StoreNumber, *ns.Distance, ns.Quantity)
    }

    // Check warehouse stock.
    wh, err := c.Warehouse(ctx, p.ProductCode)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Warehouse: %d units\n", wh.WarehouseInventory)
}
```

## Notes and caveats

`vabc` is an unofficial project and is not affiliated with Virginia ABC. The
inventory and lottery endpoints are undocumented and may change without notice. Pin
to a specific module version in production and watch releases for schema-drift
notices.

For the command-line interface, see [Commands](/docs/reference/commands/) and
[Flags and environment variables](/docs/reference/flags-and-env/).
