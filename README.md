# vabc

> Search Virginia ABC products and check store inventory from the command line.

`vabc` is a small, **read-only** CLI (and Go library) for [Virginia ABC](https://www.abc.virginia.gov/)
— the Virginia Alcoholic Beverage Control retail system. Search the product catalog, check
**live per-store and warehouse inventory**, locate stores, and check limited-availability
("lottery" / allocated) releases. **No login, no API key** — the data endpoints are public.

Built for humans *and* agents: structured JSON output, a machine-readable `schema`, an embedded
`agent` usage doc, stable exit codes, and token-bounded results.

> **Status:** functional. Live product search, inventory/store/lottery, the store locator, and the
> full agent-CLI contract all work and are validated against the real API. Next up (optional): the
> published landing page + docs site (see `spec.md`).

## Install

```bash
go install github.com/rnwolfe/vabc/cmd/vabc@latest
# or (after release): brew install rnwolfe/tap/vabc
```

## Usage

```bash
vabc product search bourbon --allocated      # search the catalog (allocated only)
vabc product get 010807                       # one product by 6-digit code
vabc inventory check 010807 --near 22182      # nearest store to a ZIP, then check stock
vabc inventory warehouse 010807               # statewide warehouse stock
vabc store near "1100 Bank St, Richmond VA"   # nearest stores to a ZIP / address / lat,lng
vabc lottery check 010807                     # limited-availability events

vabc --json product search rye                # JSON for scripts/agents
vabc schema --json                            # machine-readable command tree + exit codes
vabc agent                                    # print the embedded agent guide
```

`--json` / `--format`, `--limit`, and `--select a,b` work on every read. Data goes to stdout;
notes and errors go to stderr.

## How it works

Everything is live. `product search`/`get` query the site's **Coveo web catalog** (full coverage
and current, including new/online-only SKUs); results carry the 6-digit inventory product code, so
a search hit feeds straight into `inventory check`. Inventory, store, and lottery data are fetched
fresh on each call. Location inputs (`--near`, `store near`) accept a ZIP, a street address, or
`lat,lng`, and distances are measured from that point.

No login, no API key, no embedded data — `go get github.com/rnwolfe/vabc` pulls a tiny HTTP+JSON
client.

## Library

The CLI is a thin wrapper over the importable `github.com/rnwolfe/vabc` package — a `Client`
interface plus typed models. `go get github.com/rnwolfe/vabc` pulls an HTTP+JSON client only.

## Disclaimer

`vabc` is an unofficial tool, not affiliated with or endorsed by Virginia ABC. It uses
undocumented public endpoints that may change at any time, and is intended for personal-scale,
courteous use. No credentials are involved.

## License

[MIT](./LICENSE)
