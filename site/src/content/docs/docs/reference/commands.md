---
title: Command reference
description: Every vabc command with its arguments, per-command flags, and example invocations.
sidebar:
  order: 1
---

All commands stream data to **stdout** and notes/errors to **stderr**. The machine-readable
grammar — including the full flag table, exit-code table, and live safety state — is always
available via [`vabc schema --json`](#schema).

## Global flags

These flags apply to every command.

| Flag | Default | Description |
|------|---------|-------------|
| `--format json\|plain\|tsv` | `plain` | Output format. |
| `--json` | — | Shorthand for `--format json`. |
| `--limit N` | `50` | Maximum items for list operations. |
| `--select a,b.c` | — | Comma-separated dot-path field projection. |
| `--no-color` | — | Disable colored output. |
| `--no-input` | — | Never prompt; fail with exit 13 instead. |
| `--wait` | — | Wait out a throttle block instead of failing fast (exit 7). |
| `--max-wait DURATION` | `30s` | Maximum time to wait when `--wait` is set. |
| `--wrap-untrusted` / `--no-wrap-untrusted` | on | Fence CMS-originated free text with `⟦UNTRUSTED⟧` markers (see [lottery check](#lottery-check)). |
| `--allow-mutations` | — | No-op; vabc is read-only. Present for contract uniformity. |
| `--dry-run` | — | No-op; vabc is read-only. |

---

## product

Searches and looks up products via the site's live Coveo index. Every call is a live
network request — there is no embedded catalog.

### product search

```
vabc product search [<query>] [flags]
```

Keyword search over the live web catalog. `<query>` is matched against product name.
Omit it to browse broadly.

| Flag | Description |
|------|-------------|
| `--type <string>` | Filter by product category or type (e.g. `bourbon`, `gin`). Matched case-insensitively against the product's category, type, and name fields. |
| `--allocated` | Return only allocated / limited-availability products. |

```bash
# Search for Crown Royal Regal Apple by keyword
vabc product search "crown royal regal apple"

# Browse allocated releases, return only code and name
vabc product search --allocated --json --select productCode,name

# Find allocated bourbons, cap at 10 results
vabc product search bourbon --allocated --limit 10
```

### product get

```
vabc product get <code>
```

Look up one product by its 6-digit zero-padded product code. Returns exit 5 (`not_found`)
if the code does not appear in the live catalog.

```bash
vabc product get 010807   # Crown Royal Regal Apple
vabc product get 953714   # Planteray O.F.T.D
vabc --json product get 010807
```

---

## inventory

All inventory data is live and unauthenticated. The inventory API does not validate product
codes — a bogus code returns quantity 0 rather than an error.

### inventory check

```
vabc inventory check <code> (--store <n> | --near <location>)
```

Per-store availability for a product anchored at one store, with nearby stores ranked by
distance. Requires either `--store` or `--near`. When `--near` is given it is resolved
to the nearest store number, which becomes the anchor; `--store` provides the anchor
directly by number.

| Argument / Flag | Description |
|-----------------|-------------|
| `<code>` | 6-digit product code. |
| `--store <n>` | Anchor store number (e.g. `219` for Vienna). |
| `--near <location>` | Resolve the nearest store from a 5-digit ZIP, street address, or `"lat,lng"` string. |

```bash
# Check Crown Royal Regal Apple inventory anchored at store 219 (Vienna)
vabc inventory check 010807 --store 219

# Same, but resolve the nearest store from a ZIP
vabc inventory check 010807 --near 22182

# Resolve from a street address
vabc inventory check 953714 --near "8100 Lee Highway, Fairfax, VA"

# Resolve from coordinates
vabc inventory check 010807 --near "38.9,-77.2"

# JSON output, projected to store number and quantity
vabc --json inventory check 010807 --near 22182 --select storeNumber,quantity
```

### inventory warehouse

```
vabc inventory warehouse <code>
```

Statewide central-warehouse stock for a product. The API returns the count as a string;
`vabc` surfaces it as-is.

```bash
vabc inventory warehouse 010807
vabc inventory warehouse 953714
vabc --json inventory warehouse 010807
```

---

## store

Store data comes from the Virginia VGIN ArcGIS FeatureServer (~394 stores). Store numbers
are parsed from the `"ABC Store NNN"` landmark name. Coordinates are in WGS84 (lat/lng).

### store list

```
vabc store list
```

List all Virginia ABC stores. Honors `--limit` (default 50) and `--select`.

```bash
vabc store list
vabc --json store list --limit 400
vabc store list --select storeNumber,name,city
```

### store get

```
vabc store get <number>
```

Retrieve one store by store number. Returns exit 5 (`not_found`) if the number does not
match any store.

```bash
vabc store get 219    # Vienna store
vabc --json store get 219
```

### store near

```
vabc store near <location>
```

Find stores nearest to a location, ranked by great-circle distance in miles. `<location>`
accepts a 5-digit ZIP (resolved from an embedded Census ZCTA centroid table, no network),
a street address (resolved via the free US Census geocoder), or a `"lat,lng"` string.

Honors `--limit` (default 50) to control how many stores are returned.

```bash
# Nearest stores to a ZIP code
vabc store near 22182

# Nearest 3 stores to a street address, JSON
vabc --json store near "8100 Lee Highway, Fairfax, VA" --limit 3

# Nearest stores to coordinates
vabc store near "38.9,-77.2" --limit 5

# Pipe into jq to extract just the store numbers
vabc --json store near 22182 --limit 5 | jq '.[].storeNumber'
```

---

## lottery

Checks the Virginia ABC limited-availability event hook for a product. Event titles are
CMS-authored free text and are fenced as untrusted by default to prevent prompt injection
when used in agent pipelines (see `--wrap-untrusted` in [Global flags](#global-flags)).

The `allocated` flag in the response comes from the product's live Coveo record, not the
event-link endpoint. If the Coveo lookup fails the field is omitted rather than errored.

### lottery check

```
vabc lottery check <code>
```

Check active limited-availability events for a product.

```bash
vabc lottery check 010807

# Disable untrusted-text fencing (trusted pipeline only)
vabc lottery check 010807 --no-wrap-untrusted

# JSON, all fields
vabc --json lottery check 953714
```

Event titles appear wrapped by default:

```
⟦UNTRUSTED⟧ Spring Allocated Release — Enter by April 30 ⟦/UNTRUSTED⟧
```

Pass `--no-wrap-untrusted` only in pipelines where the source text is already sanitized.

---

## Utility commands

### auth status

```
vabc auth status
```

Reports authentication state. Virginia ABC's endpoints are public and unauthenticated, so
this always returns `authRequired: false`. Exists for contract uniformity.

```bash
vabc auth status
vabc --json auth status
```

```json
{
  "authRequired": false,
  "ok": true,
  "note": "Virginia ABC's endpoints are public; no authentication is needed"
}
```

### doctor

```
vabc doctor [--online]
```

Diagnose setup and report the health of each component. Offline by default (deterministic
in CI); `--online` makes live network requests to the inventory endpoint, ArcGIS
FeatureServer, and Coveo product search.

| Flag | Description |
|------|-------------|
| `--online` | Also probe live endpoint reachability. |

```bash
# Offline checks only
vabc doctor

# Include live endpoint probes
vabc doctor --online
vabc --json doctor --online
```

Returns exit 10 (`config_error`) if any check fails.

### schema

```
vabc schema
```

Print the machine-readable command tree, flag definitions, exit-code table, and live
safety state as JSON. Always emits JSON regardless of `--format`. This is the authoritative
grammar source for agent tool-call generation and CI validation.

```bash
vabc schema
```

```bash
# Pretty-print with jq
vabc schema | jq '.commands.subcommands[].name'

# Extract the exit-code table
vabc schema | jq '.exit_codes'
```

The output shape:

```json
{
  "tool": "vabc",
  "version": "...",
  "commands": { ... },
  "exit_codes": { "ok": 0, "rate_limited": 7, ... },
  "safety": {
    "allow_mutations": false,
    "dry_run": false,
    "no_input": false
  }
}
```

For the full agent guide, see [For agents](/docs/guides/for-agents/).

### agent

```
vabc agent
```

Print the embedded `SKILL.md` agent guide to stdout. Intended for agents that need to
self-describe vabc's capabilities and safe-use contract.

```bash
vabc agent
vabc agent | head -40
```

### version

```
vabc version
```

Print the current version.

```bash
vabc version
vabc --json version
```

---

## Related

- [Flags and environment variables](/docs/reference/flags-and-env/) — full flag reference and env var overrides
- [Exit codes](/docs/reference/exit-codes/) — stable exit-code table
- [For agents](/docs/guides/for-agents/) — safe use in automated pipelines
- [Scripting with JSON](/docs/guides/scripting-with-json/) — combining `--json` with `jq`
