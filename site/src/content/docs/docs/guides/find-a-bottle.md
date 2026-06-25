---
title: Find a bottle
description: Walk the search → product code → inventory loop to locate any Virginia ABC product near you.
sidebar:
  order: 2
---

Everything in vabc follows the same three-step loop:

1. **Search** the live catalog to get a product code.
2. **Check inventory** at a store near you (or statewide).
3. **Investigate further** — warehouse stock, nearby alternatives, lottery events.

Every command hits a live Virginia ABC endpoint. There is no local cache; results reflect the current state of the website.

---

## Step 1 — Search the catalog

`product search` queries the site's Coveo index. It covers the full catalog, including new and online-only SKUs that the downloadable price list omits.

```bash
vabc product search "crown royal"
vabc product search oftd
```

Narrow by product category or type with `--type`:

```bash
vabc product search bourbon --type rye
vabc product search "" --type gin          # browse all gins
```

Show only allocated / limited-availability products:

```bash
vabc product search "" --allocated
vabc product search bourbon --allocated
```

Both filters can be combined. `--type` matches against the product name, category, and type fields (case-insensitive).

### Reading the results

Every result row carries a **`productCode`** — a six-digit, zero-padded string:

```
productCode   name                            retailPrice
953714        Planteray O.F.T.D.              49.99
010807        Crown Royal Regal Apple         29.99
```

That `productCode` is the key for every other command. Copy it exactly as shown (with leading zeros if present).

### Fetch one product by code

If you already know the code, skip the search:

```bash
vabc product get 010807
vabc product get 953714
```

---

## Step 2 — Check inventory near you

`inventory check` takes a product code and a location. It returns the anchor store's quantity plus nearby stores that carry the product, ranked by distance.

### Using --near

`--near` accepts a **5-digit ZIP**, a **street address**, or a **`lat,lng`** pair. vabc resolves the point, finds the nearest store, and uses that as the anchor.

```bash
vabc inventory check 953714 --near 22182
vabc inventory check 953714 --near "1100 Bank St, Richmond VA"
vabc inventory check 953714 --near "38.9072,-77.0369"
```

vabc prints the resolved location and the anchor store number to stderr so you know what it picked:

```console
resolved "22182" to ZIP 22182 centroid; nearest store is 219
```

### Using --store

If you already know your preferred store number, pass it directly:

```bash
vabc inventory check 010807 --store 219
```

Store 219 is Vienna. You can find other store numbers with `store list` or `store near` (see below).

### Reading the inventory result

```json
{
  "productCode": "953714",
  "store": {
    "storeNumber": 219,
    "address1": "...",
    "city": "Vienna",
    "quantity": 3,
    "distance": 0.4
  },
  "nearbyStores": [
    { "storeNumber": 230, "quantity": 1, "distance": 2.1 },
    { "storeNumber": 410, "quantity": 0, "distance": 3.7 }
  ]
}
```

`quantity: 0` means the store does not have the product in stock; it still appears in `nearbyStores` because the endpoint returned it.

---

## Warehouse stock

`inventory warehouse` checks the statewide central warehouse — useful for gauging whether a product is being restocked at all.

```bash
vabc inventory warehouse 953714
vabc inventory warehouse 010807
```

The API returns the count as a string internally; vabc normalizes it to a number. A zero means the warehouse has none on hand; it does not confirm whether the product is discontinued.

---

## Finding stores

### Nearest stores to a location

```bash
vabc store near 22182
vabc store near "1100 Bank St, Richmond VA"
vabc store near "38.9072,-77.0369" --limit 10
```

`--limit` controls how many stores come back (default 50). Results are sorted by great-circle distance from the resolved point.

### Get one store by number

```bash
vabc store get 219
```

### List all stores

```bash
vabc store list
```

Returns all ~394 Virginia ABC stores. Combine with `--json` and `jq` to filter by city or region.

---

## Lottery and allocated releases

`lottery check` hits the limited-availability events endpoint and returns any active lottery or allocated events for a product. It also sets an `allocated` flag from the product's catalog record.

```bash
vabc lottery check 953714
vabc lottery check 010807
```

Event titles are CMS-authored free text. By default vabc wraps them in untrusted-content markers so a downstream agent cannot be prompted by embedded instructions:

```
⟦UNTRUSTED⟧ Fall Allocated Bourbon Release 2026 ⟦/UNTRUSTED⟧
```

To suppress the markers (for human-only use):

```bash
vabc lottery check 953714 --no-wrap-untrusted
```

---

## Scripting with JSON and jq

Pass `--json` (or `--format json`) to get machine-readable output on stdout. Errors go to stderr in the same structured format.

Find the closest in-stock store for a product:

```bash
vabc --json inventory check 953714 --near 22182 \
  | jq '[.store, .nearbyStores[]]
        | map(select(.quantity > 0))
        | sort_by(.distance)
        | first'
```

List store numbers where warehouse stock is nonzero (illustrative — warehouse returns a single count, not per-store):

```bash
vabc --json inventory warehouse 010807 | jq '.warehouseInventory'
```

Get product codes for all allocated bourbons:

```bash
vabc --json product search bourbon --allocated \
  | jq -r '.[].productCode'
```

For a more complete scripting reference, see [Scripting with JSON](/docs/guides/scripting-with-json/).

---

## Throttle behavior

vabc keeps a cross-process throttle so it does not hammer Virginia ABC's endpoints. If you hit the rate limit, the default behavior is to fail fast with exit code 7 and print a retry hint:

```console
error: rate limited — try again in 4s
  code: RATE_LIMITED
  fix:  re-run with --wait to wait it out, or wait 4s and retry
```

Pass `--wait` to let vabc wait instead of failing (up to `--max-wait`, default 30 s):

```bash
vabc --wait inventory check 953714 --near 22182
```
