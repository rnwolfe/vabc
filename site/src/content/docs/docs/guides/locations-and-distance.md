---
title: Locations & distance
description: How vabc resolves a ZIP, address, or lat/lng to coordinates and measures distance to stores.
sidebar:
  order: 3
---

Any command that accepts a location — `store near`, `inventory check --near` — takes the same flexible input: a 5-digit ZIP code, a full street address, or a `"lat,lng"` pair. vabc resolves whichever form you provide to WGS84 coordinates, then computes great-circle (haversine) distances to all ~394 Virginia ABC stores.

## Three ways to express a location

### 1. 5-digit ZIP code

```bash
vabc store near 22182
```

ZIP codes are resolved **offline** using an embedded US Census ZCTA centroid table (~34 000 entries, compiled from public-domain Census data). No network call is made. The resolved point is the centroid of the ZIP code tabulation area — good enough for finding the nearest store, though it may be several miles from your actual address.

### 2. Street address

```bash
vabc store near "8300 Boone Blvd, Vienna, VA 22182"
```

A free-text address is forwarded to the **US Census geocoder** (`geocoding.geo.census.gov`) — no API key required. vabc prints the matched address on stderr so you can confirm it resolved correctly:

```console
nearest stores to 8300 BOONE BLVD, VIENNA, VA 22182
```

The geocoder requires an internet connection and has a 15-second timeout. If it returns no match, vabc exits with code `5` (`NOT_FOUND`) and suggests passing a ZIP or `"lat,lng"` instead.

### 3. Explicit coordinates

```bash
vabc store near "38.9007,-77.2614"
```

A `"lat,lng"` string is parsed directly — no geocoding step at all. Use this when you already have coordinates (from a GPS reading, another API, etc.) or when you want a precise, deterministic input that won't change if Census data updates.

## How distance is computed

Once the input resolves to a point, vabc computes the **haversine (great-circle) distance** in miles from that point to every store. Earth radius used: **3958.8 miles**. Results are rounded to one decimal place.

```
distance = 2 × R × atan2( √a, √(1−a) )
  where a = sin²(Δlat/2) + cos(lat1)·cos(lat2)·sin²(Δlng/2)
```

Store coordinates come from the Virginia VGIN ArcGIS FeatureServer (WGS84, `outSR=4326`). The distance column in JSON output is labeled `distance`.

## `store near` — list stores by distance

```bash
vabc store near 22182
vabc store near 22182 --limit 5
vabc store near 22182 --json
```

Returns stores sorted nearest-first. `--limit` caps the list (default 50). Pass `--json` for structured output:

```bash
vabc store near 22182 --limit 3 --json
```

```json
[
  {
    "storeNumber": 219,
    "name": "ABC Store 219",
    "address": "...",
    "city": "Vienna",
    "state": "VA",
    "zip": "22180",
    "distance": 1.4
  },
  ...
]
```

## `inventory check --near` — auto-select the nearest anchor store

`inventory check` requires an anchor store (the `/webapi/inventory/storeNearby` endpoint is keyed by store number). `--near` resolves the location and picks the single closest store automatically:

```bash
vabc inventory check 010807 --near 22182
vabc inventory check 010807 --near "38.9007,-77.2614"
vabc inventory check 010807 --near "8300 Boone Blvd, Vienna, VA 22182"
```

vabc prints which store it resolved to on stderr before emitting results:

```console
resolved "22182" to ZIP 22182; nearest store is 219
scope: live inventory for product 010807 anchored at store 219
```

If you already know the store number, skip geocoding entirely:

```bash
vabc inventory check 010807 --store 219
```

`--near` and `--store` are mutually exclusive; passing both is a usage error (exit 2).

## Error cases

| Situation | Exit code | Code string |
|---|---|---|
| Unknown ZIP (not in ZCTA table) | 5 | `GEOCODE_FAILED` |
| Address not matched by Census geocoder | 5 | `GEOCODE_FAILED` |
| Census geocoder unreachable | 5 | `GEOCODE_FAILED` |
| No store found near the resolved point | 3 | `NO_NEARBY_STORE` |
| `--near` omitted and `--store` not set | 2 | `USAGE` |

Errors are written to stderr as structured JSON:

```json
{
  "error": "no geocoding match for \"not a real place\"",
  "code": "GEOCODE_FAILED",
  "remediation": "pass a 5-digit ZIP, a full street address, or \"lat,lng\""
}
```

## Tips

- **Offline by preference.** If you're scripting and want zero network dependencies for location resolution, pass coordinates or a ZIP. Address geocoding is the only step that requires the Census geocoder to be reachable.
- **ZIP centroids can be far from a store.** A large rural ZIP may have its centroid several miles from the nearest town. If results seem off, pass an exact address or coordinates instead.
- **Distance is as-the-crow-flies.** Haversine measures the shortest path over the Earth's surface, not driving distance. A store 1.4 miles away might be a 10-minute drive on surface roads.

## Related pages

- [Find a bottle](/docs/guides/find-a-bottle/) — combine product search with inventory and distance
- [Commands reference](/docs/reference/commands/) — full flag list for `store near` and `inventory check`
- [Flags & env vars](/docs/reference/flags-and-env/) — `--limit`, `--json`, `--wait`, and others
