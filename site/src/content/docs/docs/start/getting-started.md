---
title: Getting started
description: Install vabc and run your first product search and inventory check in under a minute.
sidebar:
  order: 1
---

`vabc` is a read-only command-line tool (and Go library) for Virginia ABC — the state's liquor
retail system. Search the live product catalog, check per-store and warehouse inventory, locate
stores, and track limited-availability ("lottery") releases. **No login, no API key, no
configuration file required.**

It works equally well for humans at a terminal and for AI agents: every response can be JSON,
exit codes are stable and documented, and untrusted text from remote sources is fenced
automatically.

## Install

Pick any one of these:

```bash
# Go (any platform, requires Go 1.25+)
go install github.com/rnwolfe/vabc/cmd/vabc@latest

# Homebrew
brew install rnwolfe/tap/vabc

# Prebuilt binary (verifies SHA-256)
curl -fsSL https://vabc-cli.vercel.app/install.sh | sh
```

Or grab a release archive directly from the
[Releases page](https://github.com/rnwolfe/vabc/releases). See
[Installation](/docs/start/install/) for more detail and platform notes.

Verify the install:

```bash
vabc version
vabc doctor          # checks your setup without hitting any live endpoints
```

## What "live" means

Everything `vabc` returns comes from Virginia ABC's own public endpoints — the same ones the
website uses. There is no embedded catalog to refresh and no offline mode. A `product search`
queries the site's Coveo search index in real time, which means new SKUs and online-only bottles
show up immediately. Inventory numbers reflect current shelf stock.

The flip side: calls require a network connection, and the endpoints are undocumented. `vabc`
self-throttles to stay polite; if it hits a rate limit it exits with code 7 and prints a retry
hint.

## 60-second first run

### 1. Search the catalog

```bash
vabc product search bourbon
```

The results include each product's 6-digit code (e.g. `010807` for Crown Royal Regal Apple).
That code is what you pass to every other command.

### 2. Get one product by code

```bash
vabc product get 010807
```

Returns name, type, size, proof, price, and whether the product is allocated (lottery-only).

### 3. Check inventory near you

```bash
vabc inventory check 953714 --near 22182
```

`953714` is Planteray O.F.T.D. `22182` is a ZIP code (Vienna, VA). You can also pass a street
address or a `lat,lng` pair — `vabc` geocodes it and ranks stores by distance.

The output shows the anchor store's quantity and a ranked list of nearby stores with their own
counts.

### 4. Grab JSON for scripts or agents

Add `--json` to any command. Data goes to **stdout**; progress notes and errors go to
**stderr**, so pipes stay clean.

```bash
# quantities only
vabc --json inventory check 953714 --near 22182 \
  | jq '.store.quantity, .nearbyStores[].quantity'

# search with field projection
vabc --json product search bourbon --select productCode,name
```

`--select a,b.c` lets you project specific fields (dot-path notation) without reaching for `jq`
at all.

## Common flags

These work on every command:

| Flag | Default | Effect |
|---|---|---|
| `--json` | off | Shorthand for `--format json` |
| `--format` | `plain` | `json`, `plain`, or `tsv` |
| `--limit N` | 50 | Cap list results |
| `--select a,b` | (all) | Dot-path field projection |
| `--no-color` | off | Disable color in plain output |
| `--wait` | off | Wait out a throttle block instead of failing fast (exit 7) |
| `--no-input` | off | Never prompt; fail with exit 13 instead |

## For agents

`vabc` is designed to be called autonomously:

- `vabc agent` prints the embedded usage contract (no network needed).
- `vabc schema --json` dumps the full command tree, all flags, the exit-code table, and the live safety state.
- Errors are structured on stderr: `{ "error", "code", "remediation" }` with stable exit codes.
- Lottery event titles — free text from a CMS — are wrapped in `⟦UNTRUSTED⟧ … ⟦/UNTRUSTED⟧`
  fences by default to stop prompt-injection attacks. Use `--no-wrap-untrusted` to disable.

See [Using vabc with agents](/docs/guides/for-agents/) for the full contract.

## Next steps

- [Find a bottle](/docs/guides/find-a-bottle/) — the end-to-end search and inventory workflow.
- [Locations and distance](/docs/guides/locations-and-distance/) — ZIP, address, and `lat,lng`
  geocoding explained.
- [Scripting with JSON](/docs/guides/scripting-with-json/) — `--json`, `--select`, and shell
  pipelines.
- [Using vabc with agents](/docs/guides/for-agents/) — exit codes, fencing, and the agent
  contract.
