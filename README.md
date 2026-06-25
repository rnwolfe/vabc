# vabc

> Search Virginia ABC products and check store inventory from the command line.

`vabc` is a small, **read-only** CLI (and Go library) for [Virginia ABC](https://www.abc.virginia.gov/)
— the Virginia Alcoholic Beverage Control retail system. Search the product catalog, check
**live per-store and warehouse inventory**, locate stores, and check limited-availability
("lottery" / allocated) releases. **No login, no API key** — the data endpoints are public.

Built for humans *and* agents: structured JSON output, a machine-readable `schema`, an embedded
`agent` usage doc, stable exit codes, and token-bounded results.

> **Status:** scaffolded. Catalog search and the full agent-CLI contract work today; the live
> inventory/store/lottery calls are placeholders until the implementation stage (see `spec.md`).

## Install

```bash
go install github.com/rnwolfe/vabc/cmd/vabc@latest
# or (after release): brew install rnwolfe/tap/vabc
```

## Usage

```bash
vabc product search bourbon --allocated      # search the catalog (allocated only)
vabc product get 010807                       # one product by 6-digit code
vabc inventory check 010807 --store 219       # live availability + nearby stores
vabc inventory warehouse 010807               # statewide warehouse stock
vabc store near 38.91,-77.23                  # nearest stores
vabc lottery check 010807                     # limited-availability events
vabc catalog status                           # snapshot freshness

vabc --json product search rye                # JSON for scripts/agents
vabc schema --json                            # machine-readable command tree + exit codes
vabc agent                                    # print the embedded agent guide
```

`--json` / `--format`, `--limit`, and `--select a,b` work on every read. Data goes to stdout;
notes and errors go to stderr.

## Catalog data

Virginia ABC has no live product-search API (search sits behind a bot challenge), so `vabc`
serves product data from a **snapshot** keyed by 6-digit product code:

- A snapshot ships **embedded in the binary** (works offline).
- It is refreshed from ABC's **quarterly XLSX price list** and re-released.
- Run `vabc catalog refresh --from-xlsx <price-list.xlsx>` to update locally between releases.

Live inventory, store, and lottery data are fetched fresh on each call.

## Library

The CLI is a thin wrapper over the importable `github.com/rnwolfe/vabc` package
(`Client` interface + typed models) and the `vabc/catalog` package. `go get` pulls an
HTTP+JSON client only — the XLSX/catalog-generation code is quarantined under `internal/`.

## Disclaimer

`vabc` is an unofficial tool, not affiliated with or endorsed by Virginia ABC. It uses
undocumented public endpoints that may change at any time, and is intended for personal-scale,
courteous use. No credentials are involved.

## License

[MIT](./LICENSE)
