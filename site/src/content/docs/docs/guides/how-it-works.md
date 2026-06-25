---
title: How it works
description: The four live data sources behind vabc, the persistent throttle/circuit-breaker, and the legitimacy boundary.
sidebar:
  order: 1
---

vabc is a thin, read-only shell over four public, unauthenticated HTTP endpoints. There is no
local catalog, no cache, and no embedded data (except the ZIP centroid table described below).
Every command makes a live network request and returns what the server says right now.

## Mental model

```
you
 │
 ├── product search / get  →  POST /coveo/rest/search/v2          (www.abc.virginia.gov)
 ├── inventory check       →  GET  /webapi/inventory/storeNearby  (www.abc.virginia.gov)
 ├── inventory warehouse   →  GET  /webapi/inventory/store        (www.abc.virginia.gov)
 ├── lottery check         →  GET  /webapi/limitedavailability/eventLinks  (same)
 └── store list / near     →  GET  FeatureServer/0/query          (services9.arcgis.com)
      └── --near <ZIP|address|lat,lng>
           ├── ZIP    →  embedded ZCTA centroid table (Census, no network)
           └── address→  GET geocoding.geo.census.gov  (free, no key)
```

No authentication. No API keys. No secrets. The tool works on first run.

## Data source reference

| Source | What it provides | Network |
|--------|-----------------|---------|
| **Coveo index** `POST /coveo/rest/search/v2` | Full web catalog — name, category, proof, retail price, allocated flag, 6-digit product code | Live, always |
| **`/webapi/inventory/storeNearby`** | Anchor store quantity + nearby stores ranked by distance | Live, always |
| **`/webapi/inventory/mystore`** | Single store quantity (leaner endpoint) | Live, always |
| **`/webapi/inventory/store`** | Statewide central-warehouse count | Live, always |
| **`/webapi/limitedavailability/eventLinks`** | Active lottery/allocated event links for a SKU | Live, always |
| **ArcGIS FeatureServer** (VGIN) | ~394 store locations, addresses, coordinates (WGS84) | Live, always |
| **Embedded ZCTA centroid table** | ZIP → lat/lng (US Census public-domain data) | Offline |
| **US Census geocoder** `geocoding.geo.census.gov` | Street address → lat/lng (no key required) | Live, on demand |

## Product search: Coveo

`product search` and `product get` both hit the site's Coveo search index — the same index
the Virginia ABC website uses. It covers the full web catalog, including products that appear
online but are absent from the downloadable price list.

```bash
vabc product search "crown royal apple"
vabc product get 010807
```

Coveo results carry the 6-digit `productCode`, which is the same identifier used by every
inventory endpoint. A search hit feeds straight into an inventory check with no extra lookup.

Product codes are zero-padded to six digits (`010807`, not `10807`). The inventory API does not
validate codes — a bogus code returns quantity 0 rather than an error. Existence is confirmed
only through a successful Coveo search or `product get`.

## Inventory: three endpoints

The `/webapi/inventory/` family is the core of what makes this tool useful.

**`storeNearby`** is the primary endpoint. Given an anchor store number and a product code, it
returns the anchor store's on-hand quantity plus a ranked list of nearby stores that stock the
product, each with distance, address, hours, and coordinates.

```bash
# inventory check uses storeNearby under the hood
vabc inventory check 010807 --store 219
vabc inventory check 953714 --near 22182
```

**`mystore`** is a leaner variant used when you only need a single store's count. The parameter
names are plural (`storeNumbers`, `productCodes`) but the server only accepts one value each —
comma-separated lists return a 400 error.

**`store`** (warehouse) returns the statewide central-warehouse count as a plain integer. The
API returns the count as a string internally; vabc converts it to a number in the JSON output.

```bash
vabc inventory warehouse 010807
```

## Store locator: ArcGIS

Store data comes from the Virginia VGIN ArcGIS FeatureServer — official state open data, not a
scrape. A single query returns all ~394 retail locations with addresses, phone numbers, and
WGS84 coordinates. vabc parses the store number from the `LandmkName` field (e.g.
`"ABC Store 219"` → store number 219).

```bash
vabc store list
vabc store get 219
vabc store near 22182
```

Distances (from `store near` and from `inventory check --near`) are great-circle miles computed
with the haversine formula, rounded to one decimal place.

## Geocoding: ZIP, address, or coordinates

`--near` and `store near` accept three input forms:

- **`lat,lng`** — used directly, no network call.
- **5-digit ZIP** — resolved from an embedded US Census ZCTA centroid table. This is the only
  offline data bundled in the binary; it makes ZIP lookups instant and not subject to rate
  limits or network errors.
- **Street address** — sent to the free US Census geocoder
  (`geocoding.geo.census.gov/geocoder`). No API key is required. vabc uses the first match
  returned.

```bash
vabc store near 22182              # ZIP
vabc store near "7700 Leesburg Pike, Falls Church, VA"   # address
vabc store near "38.9,-77.3"       # lat,lng
```

The geocoder and ZCTA table live in `internal/geocode` and are not part of the public Go
library. Importers of `github.com/rnwolfe/vabc` do not inherit them.

## The persistent throttle and circuit-breaker

Because the `/webapi/` endpoints are undocumented and reachable only by courtesy, vabc enforces
a minimum interval between requests and backs off automatically when the server signals a block.

The throttle state is written to a small JSON file. The location is resolved in order:
`$VABC_STATE_DIR/throttle.json` if that variable is set, then `$XDG_STATE_HOME/vabc/throttle.json`
if `XDG_STATE_HOME` is set, and finally `os.UserCacheDir()/vabc/throttle.json` as the default.
Persisting state to disk is deliberate: an agent hot-loop spawns a fresh process for each call, so
an in-process timer would be a no-op. The file records the timestamp of the last request and, when
applicable, a `blockedUntil` timestamp for the circuit breaker.

**Normal flow.** vabc reads the state file, sleeps for the remainder of the minimum interval if
the last request was too recent, then updates the file and makes the request.

**Circuit-breaker flow.** When an HTTP 429 or a WAF challenge is detected, vabc writes a
`blockedUntil` time (honoring `Retry-After`, defaulting to 60 seconds). Subsequent processes
read this and fail fast with exit code 7, so an agent loop never deadlocks waiting out a block
it cannot clear.

```console
$ vabc inventory check 010807 --store 219
error: rate limited — circuit breaker open for 47s; retry after 2026-06-25T14:32:00Z
hint: pass --wait to wait it out (up to --max-wait, default 30s)
```

Passing `--wait` opts into waiting. If the remaining block exceeds `--max-wait`, vabc still
fails fast.

There is no evasion anywhere in the client: no User-Agent spoofing, no proxy rotation, no
CAPTCHA solving. The throttle is a courtesy gate, not a cat-and-mouse measure.

## The legitimacy boundary

| | Status |
|---|---|
| Authentication required | None |
| Official developer API | No — endpoints are reverse-engineered from the site's own JavaScript |
| Official open data | ArcGIS store layer only |
| Published terms for `/webapi/*` | None found |
| Credentials at risk | None (no auth anywhere) |
| Mutations possible | None — vabc is strictly read-only |

The ArcGIS store layer is genuinely official (VGIN open data, low risk). The `/webapi/`
inventory and Coveo endpoints are undocumented; they are the same XHR calls the Virginia ABC
website makes from a browser, currently unauthenticated and not challenged by Cloudflare, but
that could change without notice.

vabc is not affiliated with or endorsed by Virginia Alcoholic Beverage Control. The endpoint
contract is reverse-engineered and may change at any time. Use at personal scale and with
courtesy — which the built-in throttle enforces automatically.

For further details on the command surface, see [Commands](/docs/reference/commands/) and
[Flags and environment variables](/docs/reference/flags-and-env/).
