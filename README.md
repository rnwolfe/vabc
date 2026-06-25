<div align="center">

# vabc

**Virginia ABC product search and store inventory — from your terminal.**

Search the catalog, check live per-store and warehouse stock, find the nearest store, and track
allocated/lottery bottles. Built for humans *and* AI agents: structured JSON, stable exit codes,
an embedded usage contract. No login, no API key.

[![CI](https://github.com/rnwolfe/vabc/actions/workflows/ci.yml/badge.svg)](https://github.com/rnwolfe/vabc/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/rnwolfe/vabc?sort=semver)](https://github.com/rnwolfe/vabc/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/rnwolfe/vabc.svg)](https://pkg.go.dev/github.com/rnwolfe/vabc)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](./LICENSE)
[![Agent CLI Guidelines: Full](https://aclig.dev/badge/agent-cli-guidelines-full.svg)](https://aclig.dev/conformance/)

<img src="./demo/vabc.gif" alt="vabc demo" width="720">

**[Documentation →](https://vabc-cli.vercel.app)**

</div>

## Why

The Virginia ABC website is the only way to check what's on the shelf — and it's a slow,
JavaScript-heavy click-through. `vabc` turns it into one-liners:

```console
$ vabc inventory check 953714 --near 22182
```
> Where can I get Planteray O.F.T.D. near me? → store 219 (Vienna) has 21, store 76 (Falls Church) has 33…

Everything is **live** and **read-only**. Product search runs against the same catalog the website
uses (so new and online-only bottles show up), and every result carries the code you need to check
inventory.

## Install

```bash
# Go
go install github.com/rnwolfe/vabc/cmd/vabc@latest

# Homebrew (after first release)
brew install rnwolfe/tap/vabc

# Or grab a prebuilt binary from the Releases page.
```

## Quickstart

```bash
vabc product search bourbon              # search the live catalog
vabc product search oftd --json          # JSON for scripts/agents
vabc product get 010807                  # one product by 6-digit code

vabc inventory check 010807 --near 22182 # nearest store + nearby stock (ZIP, address, or lat,lng)
vabc inventory warehouse 010807          # statewide warehouse stock

vabc store near "1100 Bank St, Richmond VA"   # nearest stores to an address
vabc lottery check 010807                # limited-availability releases

vabc --help                              # example-led help
```

`--json` / `--format`, `--limit`, and `--select a,b` work on every read. Data goes to stdout;
notes and errors go to stderr.

## For agents

`vabc` is engineered for autonomous callers, not just humans:

- **`vabc agent`** prints a complete, embedded usage contract (no repo or network needed).
- **`vabc schema --json`** dumps the full command tree, flags, and the exit-code table.
- **Structured errors** on stderr: `{ "error", "code", "remediation" }` with stable exit codes.
- **Read-only by default** with a mutation gate (inert here — there's nothing to mutate).
- **Bounded output** (`--limit`, default 50) and field projection (`--select`).
- **Prompt-injection fencing**: untrusted target text (lottery event titles) is wrapped by default.

```bash
vabc --json inventory check 010807 --near 22182 | jq '.store.quantity, .nearbyStores[].quantity'
```

## How it works

Everything is a live read against Virginia ABC's public, unauthenticated endpoints — product
search (the site's Coveo index), inventory and warehouse (`/webapi/inventory/*`), the store
locator (Virginia VGIN's ArcGIS service), and the limited-availability hook. Locations are
geocoded (an embedded ZIP-centroid table; the free US Census geocoder for street addresses) so
distances are measured from where you actually are.

`go get github.com/rnwolfe/vabc` pulls a tiny HTTP+JSON client — the CLI is a thin wrapper over
the same importable `Client` interface.

## Disclaimer

`vabc` is an unofficial tool, **not affiliated with or endorsed by Virginia ABC**. It uses
undocumented public endpoints that may change at any time, and is intended for personal-scale,
courteous use — it self-throttles and adds no scraping evasion. No credentials are involved.

## License

[MIT](./LICENSE)
