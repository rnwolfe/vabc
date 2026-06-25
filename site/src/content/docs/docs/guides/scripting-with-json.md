---
title: Scripting with JSON
description: Use vabc's machine-readable output, field projection, and exit codes to build shell pipelines and automation scripts.
sidebar:
  order: 4
---

Every `vabc` command follows the same output contract: **data goes to stdout, human
notes and errors go to stderr**. This split makes shell pipelines safe — you can
pipe stdout into `jq` without worrying about progress notes polluting the JSON.

## Requesting JSON output

Two equivalent ways to get JSON:

```bash
vabc --json product search bourbon
vabc --format json product search bourbon
```

Both produce 2-space indented JSON with HTML escaping disabled (so URLs survive
intact). Plain text and TSV are the other options:

```bash
vabc --format tsv store near 22182
```

For scripting, always use `--json` or `--format json`. The plain and TSV renderers
are for human-readable terminal output and their column selection is not guaranteed
to be stable across versions.

## Field projection with --select

`--select` takes a comma-separated list of dot-path fields and keeps only those
keys in each object. Nested paths use dot notation.

```bash
# Keep only storeNumber and distance from nearby-store results
vabc --json store near 22182 --select storeNumber,distance

# Pull name and retailPrice from a product search
vabc --json product search bourbon --select name,retailPrice
```

Projection happens before output, so it works with any command. If a path does not
exist in an object it is silently omitted (no error).

## Limiting results with --limit

All list operations default to 50 results. Override with `--limit`:

```bash
# The three nearest stores to ZIP 22182
vabc --json store near 22182 --limit 3

# Top 10 bourbon results
vabc --json product search bourbon --limit 10
```

When the output is truncated a note is written to stderr:

```
note: output truncated to 10 of 94 items (use --limit to change)
```

That note never lands on stdout, so your `jq` pipeline is unaffected.

## The structured error envelope

When a command fails, vabc writes a JSON object to **stderr** (when `--json` is
active) and exits with a non-zero code:

```json
{
  "error": "product 999999 not found",
  "code": "NOT_FOUND",
  "remediation": "list available products to find a valid id"
}
```

Fields:

| Field | Type | Description |
|---|---|---|
| `error` | string | Human-readable message |
| `code` | string | Machine-readable symbol, stable across versions |
| `remediation` | string | Suggested fix, may be empty |

## Exit codes

Exit codes are stable and documented. The ones you'll encounter most in scripts:

| Code | Value | Meaning |
|---|---|---|
| `ok` | 0 | Success |
| `generic_error` | 1 | Unexpected error |
| `usage` | 2 | Bad flag or argument |
| `empty_results` | 3 | Query succeeded but returned nothing |
| `not_found` | 5 | Specific resource (product, store) not found |
| `rate_limited` | 7 | Throttle block — retry after a moment |
| `retryable` | 8 | Transient backend error — safe to retry |
| `config_error` | 10 | Environment or state-file problem |
| `input_required` | 13 | Interactive prompt needed but `--no-input` is set |
| `cancelled` | 130 | SIGINT |

Check codes in shell with `$?` immediately after the command:

```bash
vabc --json inventory check 010807 --near 22182
rc=$?

case $rc in
  0)   echo "got results" ;;
  3)   echo "product not found in inventory (but query worked)" ;;
  7)   echo "rate limited — wait and retry" ;;
  *)   echo "unexpected error ($rc)" ;;
esac
```

For non-interactive scripts, pass `--no-input` so prompts become a clean exit 13
instead of hanging:

```bash
vabc --json --no-input inventory check 010807 --store 219
```

## jq recipes

The recipes below assume `jq` is installed. All commands are real, runnable
invocations — adjust product codes and locations for your own use.

### Cheapest in-stock product near a ZIP

Search for bourbon, get inventory near ZIP 22182 for each result, keep only
products where the anchor store has stock, then sort by retail price:

```bash
vabc --json product search bourbon --limit 20 \
  | jq -r '.[].productCode' \
  | while read code; do
      vabc --json inventory check "$code" --near 22182 2>/dev/null \
        | jq --arg c "$code" '
            select(.store.quantity > 0)
            | {productCode: $c, quantity: .store.quantity, storeNumber: .store.storeNumber}
          '
    done \
  | jq -s 'sort_by(.quantity) | reverse'
```

### All nearby stores that stock a product, sorted by distance

`inventory check` returns the anchor store plus `nearbyStores` ranked by
distance. Merge them and filter for stock > 0:

```bash
vabc --json inventory check 953714 --near 22182 \
  | jq '[.store, .nearbyStores[]] | map(select(.quantity > 0)) | sort_by(.distance)'
```

### Allocated-only product search

The `--allocated` flag filters on the server side, but you can also post-filter
in jq to be explicit:

```bash
vabc --json product search --allocated \
  | jq '[.[] | select(.allocated == true) | {productCode, name, retailPrice}]'
```

Or combine it with `--select` to skip the jq projection entirely:

```bash
vabc --json product search --allocated --select productCode,name,retailPrice,allocated
```

### Store numbers nearest a location

```bash
vabc --json store near 22182 --limit 5 \
  | jq -r '.[].storeNumber'
```

### Extract warehouse stock for a product

```bash
vabc --json inventory warehouse 953714 \
  | jq '{code: .productCode, warehouse: .warehouseInventory}'
```

### Check if a lottery product is allocated and has active events

```bash
vabc --json lottery check 953714 \
  | jq '{allocated: .allocated, active: .active, events: (.eventLinks | length)}'
```

The `allocated` field comes from the live product catalog record; `active` and
`eventLinks` come from the live limited-availability hook. Both are present in the
same response object.

### Handle errors in a pipeline

Redirect stderr to capture the structured error, check the exit code:

```bash
result=$(vabc --json inventory check 010807 --store 219 2>/tmp/vabc_err)
rc=$?
if [ $rc -ne 0 ]; then
  code=$(jq -r '.code' /tmp/vabc_err 2>/dev/null)
  echo "failed: $code (exit $rc)"
  exit $rc
fi
echo "$result" | jq '.store.quantity'
```

## Throttle and retry

vabc maintains a persistent cross-process throttle. If a call is blocked, the
default behavior is to **fail fast** with exit 7 rather than wait. For scripts
that can afford to wait, use `--wait`:

```bash
vabc --json --wait inventory check 010807 --near 22182
```

`--max-wait` sets the ceiling (default 30 seconds):

```bash
vabc --json --wait --max-wait 60s inventory check 010807 --near 22182
```

In a loop, check for exit 7 and sleep before retrying rather than hammering the
endpoint:

```bash
for code in 010807 953714; do
  while true; do
    vabc --json inventory check "$code" --store 219
    rc=$?
    [ $rc -ne 7 ] && break
    echo "rate limited, sleeping 5s" >&2
    sleep 5
  done
done
```

Or simply pass `--wait` and let vabc handle the backoff for you.

## Machine-readable schema

`vabc schema --json` prints the full command tree, all flags, the exit-code
table, and the live throttle state. Useful for programmatic introspection:

```bash
vabc schema --json | jq '.exitCodes'
```

See [Commands reference](/docs/reference/commands/) and
[Flags and environment variables](/docs/reference/flags-and-env/) for the full
reference. For agent-oriented usage (tool definitions, prompt-injection fencing),
see [For agents](/docs/guides/for-agents/).
