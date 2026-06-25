# spec.md — vabc

> The build spec for an agent-focused CLI. Written by `cli-plan`; consumed by `cli-scaffold`,
> `cli-implement`, and `cli-publish`. Keep it current — it is the single source of truth.

**Tool name:** `vabc` (binary + command root). Brand/tagline: "vabc — search Virginia ABC
products and check store inventory from the command line." (The hot use case is bourbon hunters
tracking allocated/limited drops, but the tool covers the full catalog: spirits, wine, etc.)

## Target
- **Service**: Virginia ABC (Virginia Alcoholic Beverage Control) — the state liquor retail
  system. Public site `virginiaabc.com` → `abc.virginia.gov`; shop at `shop.abc.virginia.gov`.
- **Surface**: a mix, all **verified live 2026-06-24**:
  - **Inventory (core, undocumented JSON, no auth, Cloudflare-EXEMPT)** on `https://www.abc.virginia.gov`:
    - `GET /webapi/inventory/storeNearby?storeNumber={n}&productCode={6digit}` — **primary**. Returns
      the anchor store's quantity **plus a `nearbyStores[]` array ranked by `distance`** with full
      address/coords/hours per store. *This single endpoint is "what's in stock near me."* Both
      params required (omit either → `400 Missing required parameter`).
    - `GET /webapi/inventory/mystore?storeNumbers={n}&productCodes={6digit}` — leaner; params are
      plural-named but **singular server-side** (comma lists → 400). Fan out one call per (store,product).
    - `GET /webapi/inventory/store?productId={6digit}` — statewide warehouse stock → `{"warehouseInventory":"100"}`.
    - `GET /webapi/limitedavailability/eventLinks?productCode={6digit}` — lottery/allocated hook;
      `{}` when no active drop, event links when live. (ABC hides actual limited-availability counts.)
  - **Store locator (official open data, no auth)**: Virginia VGIN ArcGIS FeatureServer —
    `https://services9.arcgis.com/6EuFgO4fLTqfNOhu/arcgis/rest/services/Virginia_ABC_Stores/FeatureServer/0/query?where=1=1&outFields=*&f=json`.
    **394 stores**, one call (`maxRecordCount=2000`). Fields: `VAPID, LandmkName` ("ABC Store 088"),
    `Address, City, State, Zip, Phone, URL, X, Y (lng/lat), LOC, FIPScode, FIPSname, LastCheck`.
  - **Product search / catalog — LIVE via Coveo (CORRECTED).** The earlier finding that search was
    Cloudflare-blocked was **wrong**: `POST /coveo/rest/search/v2` is **publicly queryable,
    anonymous, and not CF-challenged** (verified live 2026-06-25). `GET /coveo/rest/token` mints an
    anonymous token but isn't even required. It indexes the **full web catalog** — more complete and
    current than the downloadable price list (e.g. "Planteray O.f.t.d Overproof Rum", sku `953714`,
    is on the site but absent from the quarterly XLSX). Key Coveo raw fields (special chars encoded:
    z32x=space, z95x=`_`, z120x=`x`, z122x=`z`): `z95xproductz32xskuz32xids` (= the inventory
    `productCode`, confirmed: Planteray Original Dark → `042395`), `productz32xlabelz32xname`,
    `hierarchyz32xcategory/type`, `z95xproductz32xsiz122xes`, `z95xproductz32xpricez32xsort`,
    `proofmin`, `z95xproductz32xlimitedz32xavailability`, `clickUri`.
    → **Implemented:** `product search`/`get` query Coveo LIVE by default (full coverage, current).
    The downloadable **quarterly price-list XLSX** still feeds an **embedded snapshot** (~4,900
    products) used as the `--offline` path and the automatic fallback when the live catalog is
    unreachable. (Original snapshot-only plan retained as the offline tier.)
- **Rate limits / pagination**: No documented rate limit on `/webapi/*`; **no pagination** (one
  product/store per call → per-pair fan-out). **The endpoints are undocumented and CF-exempt only
  because the site's own JS calls them** → treat as a courtesy surface. Because an agent spawns a
  **fresh process per call**, the tool MUST carry **persistent cross-process throttle/backoff state**
  (token bucket / last-call timestamps in XDG cache) — an in-process timer is a no-op. Modest
  concurrency, polite delay, descriptive User-Agent. ArcGIS: one call returns all 394 stores.
- **ToS / risk**: ⚠️ **The `/webapi/*` inventory endpoints are reverse-engineered and undocumented**
  — no published developer terms (even `robots.txt` is behind the CF challenge). Param names
  (`storeNumbers` plural-but-singular) and the route table can change without notice → wrap behind a
  thin client pinned to a known contract; breakage risk is real and expected. The **aggressive
  Cloudflare WAF on all HTML/static paths signals ABC discourages bulk scraping**; the inventory XHR
  endpoints are exempt today but could be locked down. ArcGIS store data is genuinely official/open
  (low risk). **No credentials are involved** (nothing to leak), which materially lowers the risk
  profile vs. most unofficial-API tools.
- **Prior art / competitive landscape**:
  - `austinkeeley/bourbon-finder` (Go, 2022, unmaintained) — the reference impl for the inventory
    half; hits `/webapi/inventory/mystore`, no auth, concurrent. **Mine for mechanics.** Agentic gaps:
    no read-only gate (it's read-only by nature but no contract), no `schema`/`agent`, no structured
    errors/exit codes, no bounded output, config-file-only (not a general CLI).
  - `awunderground/Virginia-ABC-Data` (2016) — full catalog dump + the `{"Results":[...]}` schema
    (still current). Source for the catalog snapshot shape.
  - `ViennaMike` gist (Python+Selenium+Twilio) — the fragile "don't do it this way" contrast.
  - Closed-source SaaS: **VABourbon** (vabourbon.com, Patreon) and **WhiskyDB** (whiskydb.org) — same
    data source; their value-add is *history + allocated alerts*. Good competitive reference.
  - **Registries: nothing on npm, PyPI, or crates.io** (`virginia-abc`/`va-abc`/`vaabc` all free).
- **Build verdict**: **BUILD.** No agent-engineered tool exists in any ecosystem; the only working
  prior art is a 4-year-old unmaintained Go config-runner. Differentiators: **(1)** unifies the four
  data surfaces (catalog search + live per-store inventory + nearby-store fan-out + allocated/lottery
  + store locator) behind one noun-verb CLI; **(2)** full agent contract — `schema --json`, embedded
  `agent`/SKILL.md, structured errors + stable exit codes, bounded/`--select` output; **(3)**
  persistent cross-process throttle so an agent hot-loop stays a polite citizen of an undocumented
  WAF-fronted API; **(4)** first-class **allocated/limited-availability** awareness (the actual
  reason people hunt VA ABC). Mine `bourbon-finder` for the exact inventory request mechanics.

## Language & framework
- **Language**: **Go**
- **Rationale (SDK gravity > distribution > performance)**: no SDK forces any language (everything is
  plain HTTP GET → JSON), so distribution decides — Go gives the best single-binary + lowest cold
  start for an agent hot-loop, and the only working prior art is already Go.
- **Framework**: **kong**
- **SDK/library used**: direct HTTP (`net/http` + `encoding/json`) for the whole live surface. XLSX
  parsing (`excelize`) for catalog generation only, confined to `internal/harvest` so it never reaches
  library importers. **No headless browser anywhere.**
- **Blueprint**: references/research/blueprint-go.md
- **Language-specific gotchas to honor**:
  - **Catalog is a bundled snapshot; importers must never inherit catalog-gen deps.** Ship the
    snapshot embedded via `go:embed` so `product search`/`product get` and all inventory commands work
    offline with zero network-to-CF. Catalog generation parses ABC's quarterly XLSX and lives under
    `internal/harvest` (unimportable externally) so the public library's dependency graph stays tiny —
    just HTTP + JSON (see **Architecture & packaging**). No headless browser is used anywhere.
  - Pin a thin `webapi` client to the verified contract; tolerate the plural-but-singular param quirk
    and per-pair fan-out (no comma lists).
  - `productCode` is **NOT validated server-side** — a bogus code returns `200 {quantity:0}`. So
    inventory cannot confirm a SKU exists; existence checks MUST go through the catalog snapshot.
  - `storeNumber` IS validated (bad number → `400 "No Store exists…"`) — map to exit 5.
  - The inventory `storeId` (small int) = the ABC store number embedded in ArcGIS `LandmkName`
    ("ABC Store 088" → 88), **not** the long `VAPID`. Join on the store-number suffix.

## Architecture & packaging (library-first — decoupled API)
The reusable API is the product; the CLI is one consumer of it. Importers must get clean typed
interfaces and a **lightweight dependency footprint** (HTTP + JSON only — no headless browser, no
kong).

**Module layout** (`github.com/rnwolfe/vabc`):
```
github.com/rnwolfe/vabc                  (root = the public library; tiny deps)
├── client.go            // Client struct + functional options; throttling http.RoundTripper
├── inventory.go         // StoreNearby(ctx, store, code), MyStore(...), Warehouse(ctx, code)
├── stores.go            // Stores(ctx) ArcGIS locator; StoreNear(ctx, loc)
├── lottery.go           // LimitedAvailability(ctx, code)
├── types.go             // Product, Store, Inventory, NearbyStore, LotteryEvent, Envelope
├── catalog/             // catalog as a SWAPPABLE provider, not a hardcoded file
│   ├── catalog.go       //   Catalog interface { Search(q, opts) ; Get(code) } + errors
│   ├── embedded.go      //   //go:embed data/catalog.json  → default provider
│   └── data/catalog.json//   committed snapshot = source of truth (diffable, CI-refreshed)
├── internal/harvest/    // XLSX → catalog.json parser (excelize) — NOT part of the public API surface
├── cmd/vabc/            // thin kong CLI: parse → call library → format. No business logic.
└── cmd/vabc-catalog-gen/// generator binary: internal/harvest over a quarterly XLSX → catalog.json
```

**Decoupling rules:**
- **Public interfaces, not concretes.** `Client` and `Catalog` are interfaces; the CLI and any third
  party depend on those, so the HTTP/embedded implementations can be swapped or mocked.
- **Politeness lives in the library, not the CLI.** The persistent cross-process throttle/backoff is a
  composable `http.RoundTripper` on the `Client`, so *every* importer is a polite citizen of the
  undocumented WAF-fronted API for free.
- **Catalog-gen is quarantined.** XLSX parsing (`excelize`) is reachable only via `internal/harvest`
  (Go forbids external imports of `internal/`), used by both `cmd/vabc-catalog-gen` and the CLI's
  `catalog refresh` command. `go get github.com/rnwolfe/vabc` pulls a minimal HTTP+JSON client only —
  no excelize, no browser.
- **Stable envelope shared by both consumers.** `types.Envelope{ schemaVersion, scope?, data,
  nextCursor? }` is defined in the library; the CLI just serializes it. One schema, no drift.

### Catalog data model & refresh (resolves the "search has no live API" gap)
- **Source of truth**: committed `catalog/data/catalog.json` (keyed by 6-digit `productCode`),
  **embedded into every release binary** via `go:embed` → `product search`/`get` are offline,
  browser-free, deterministic.
- **Repo refresh (the CLI cycle)**: ABC publishes a **quarterly XLSX price list**. The maintainer
  downloads it (a once-a-quarter manual click — the file URL is CF-gated so it isn't auto-fetched),
  runs `cmd/vabc-catalog-gen --from-xlsx <file>` to regenerate `catalog.json`, and commits → the next
  release ships current data. A scheduled GitHub Action can *check* for a newer quarter's file and
  open a reminder issue, but does not fetch through Cloudflare. Worst-case staleness = release cadence
  (acceptable: the catalog itself only changes quarterly).
- **Runtime refresh (between releases)**: `vabc catalog refresh --from-xlsx <file>` writes a fresh
  snapshot into the XDG cache from a price list you already downloaded; the `Catalog` provider resolves
  **local cache (if present & newer) → embedded snapshot**, so an agent never blocks on the
  network/browser for search.
- Every response carries `scope` (e.g. `"catalog snapshot 2026-06-01; live inventory"`) so a caller
  never mistakes cached catalog for live stock.

## Auth
- **Model**: **None.** All four data surfaces (`/webapi/*` inventory + ArcGIS stores) are fully
  **public, unauthenticated, and Cloudflare-exempt**. No API key, bearer, cookie/session, or CSRF.
- **Provider constraints**: n/a — nothing to authenticate. (The CF challenge only fronts HTML/static
  paths, which the runtime never touches; only `catalog refresh` deals with it, browser-side.)
- **Feasible path to usability (end-to-end)**: trivially headless — the tool works on first run with
  no setup. No tokens, no onboarding, no secret storage required.
- **Secret storage**: n/a (no secrets).
- **Subcommands**: no `login/logout/refresh`. Keep a single **`vabc auth status`** for contract
  uniformity that reports `{"authRequired": false, "ok": true}` and exits 0. `doctor` (below) covers
  real readiness (endpoint reachability + catalog freshness).

## Command surface (noun-verb)
| Command | Read/Mutation | Description | Key output fields |
|---|---|---|---|
| `product search <query>` | read | Keyword search over the cached catalog snapshot. `--allocated` filter, `--type`, `--limit`, `--select`. | `productCode, name, category, type, proof, size, retailPrice, discountPrice, allocated, onlineOrderable, new, upc, url` |
| `product get <productCode>` | read | Full catalog record for one 6-digit SKU. | (all catalog fields above) |
| `inventory check <productCode>` | read | Live per-store availability + nearby stores ranked by distance. `--store <n>` (anchor) **or** `--near <zip>` (resolve nearest store via ArcGIS, then `storeNearby`). `--limit`/`--radius` bound the nearby list. | `productCode, store{storeId,quantity,address,city,zip,distance,lat,lng,phone,hours,url}, nearbyStores[]` |
| `inventory warehouse <productCode>` | read | Statewide central-warehouse stock. | `productCode, warehouseInventory` |
| `lottery check <productCode>` | read | Active limited-availability / allocated event links for a SKU. | `productCode, allocated, active, eventLinks[]` (fenced untrusted) |
| `store list` | read | All ~394 stores from the ArcGIS dataset. `--limit`, `--select`. | `storeNumber, name, address, city, state, zip, phone, lat, lng, url` |
| `store get <storeNumber>` | read | One store's details. | (store fields above) |
| `store near <zip\|lat,lng>` | read | Nearest stores to a location, by distance. `--limit`, `--radius`. | `storeNumber, name, address, distance, lat, lng, …` |
| `catalog status` | read | Snapshot version, build date, product count, staleness. | `schemaVersion, snapshotDate, productCount, stale` |
| `catalog refresh --from-xlsx <path>` | local maintenance | Rebuild the local catalog snapshot into XDG cache from a downloaded ABC quarterly price list (shares `internal/harvest`). Not a target mutation. | `snapshotDate, productCount, source` |
| `auth status` | read | Reports no-auth (contract uniformity). | `authRequired:false, ok` |
| `doctor` | read | Probe `/webapi/*` + ArcGIS reachability and catalog freshness; actionable diagnostics. | `checks[]{name,ok,detail}` |

No state-changing operations against the target exist anywhere in scope (VA ABC exposes no
cart/order API we wrap) → **the tool is wholly read-only**. `--allow-mutations` is present (scaffold)
but gates nothing; `catalog refresh` writes only the local snapshot cache.

## Exit codes
Base from contract §4, plus target-specific:
```
0   ok                         5  not found (unknown 6-digit SKU in catalog; or 400 "No Store exists…")
1   generic error              7  rate limited / WAF challenge hit on a normally-exempt endpoint
2   usage/parse                8  retryable/transient (network, upstream 5xx, transient CF challenge)
3   empty results              10 config error
                               11 CATALOG_UNAVAILABLE  [target-specific] — snapshot missing/unreadable; run `catalog refresh`
                               14 CATALOG_STALE         [target-specific] — snapshot older than threshold (hard-fail only under --strict; otherwise warn on stderr)
130 cancelled (SIGINT)
```
Reserved but unused: `4 auth required` (no auth), `6 permission denied`, `12 mutation blocked` (no mutations).

## Output schema
`schemaVersion: 1`. **Realized in scaffold:** reads emit the domain data *directly* (arrays for
lists, objects for single records) so the token-economy flags (`--limit`, `--select`) operate
naturally on the result; freshness/`scope` is surfaced on **stderr** (`scope: …` note) and in-band
via `catalog status`, and `schemaVersion` is reported by `schema --json` + `catalog status`. The
typed `vabc.Envelope{schemaVersion, scope?, data, nextCursor?}` is defined in the library for
callers that want the full wrapper; cli-implement may promote list/object reads onto it if an
in-band `scope` per response proves necessary. Field contracts (append-only):
- **product**: `productCode` (string, 6-digit zero-padded), `name`, `category`, `type`, `proof`
  (number|null), `size`, `retailPrice` (number|null), `discountPrice` (number|null), `allocated`
  (bool), `onlineOrderable` (bool), `new` (bool), `upc` (string[]), `url`.
- **inventory check**: `productCode`, `store` { `storeId` (int), `quantity` (int), `distance` (number,
  miles), `address`, `address1`, `address2`, `city`, `state`, `zip`, `lat`, `lng`, `phone`, `hours`,
  `shoppingCenter`, `url` }, `nearbyStores` (same store shape, ranked by `distance`).
- **inventory warehouse**: `productCode`, `warehouseInventory` (int).
- **lottery check**: `productCode`, `allocated` (bool, from catalog), `active` (bool), `eventLinks`
  (array of `{title,url}` — **fenced untrusted**).
- **store**: `storeNumber` (int), `name`, `address`, `city`, `state`, `zip`, `phone`, `lat`, `lng`,
  `url`; `distance` added on `store near`.
- **catalog status / doctor**: as in the table.

## Universal contract surface (provided by scaffold — confirm no conflicts)
`--format json|plain|tsv` · `--allow-mutations` (no-op here — no target mutations) · `--dry-run` ·
`--yes`/`--force` · `--no-input` · `--limit` · `--select` · `--concise`/`--detailed` ·
`schema --json` · `agent`. No conflicts. Note `--allow-mutations`/`--dry-run` are inert given the
read-only surface; keep them for uniformity but `schema --json` should reflect that no command is
gated.

## Distribution
- **Targets**: `go install` · Homebrew tap (`brew install rnwolfe/tap/vabc`) · GoReleaser release
  binaries (darwin/linux/windows × amd64/arm64), checksummed.
- **Trial path**: `brew install` or a one-line install script downloading the prebuilt binary; works
  immediately (no auth, bundled catalog snapshot).
- **Agent hot-loop path**: prebuilt static Go binary — lowest cold start, embedded SKILL.md + catalog
  snapshot, no network needed for catalog/help.

## Publish
- **Flag**: **full** (portfolio-bound — operator confirmed).
- **License**: **MIT** (single `LICENSE`).
- **If full**: docs site (starlight-docs) · doc content (harvest-docs) · release (release skill) ·
  README + VHS demo · hygiene files · discoverability (Homebrew tap, Show HN, r/bourbon &
  r/VirginiaABC, awesome-cli lists, the bourbon-hunting community).
  - **Web presence**: bold custom landing page + Starlight docs sharing **ONE design-token source**;
    per-page **OG/social cards**. Visual spirit: a "store-shelf / bottle-hunt" motif; a live demo
    block showing `vabc inventory check` output.
  - **Deploy target**: **Vercel** (git-connected, CI deploy). **Start on the auto `*.vercel.app`
    subdomain**; cut over to the custom domain once it's secured.
  - **Custom domain (target)**: **`vabc.sh`** — *publish confirms availability via the Vercel domain
    tool; operator buys, then points the project at it.* Until then the canonical URL is the
    `vabc.vercel.app` (or assigned) subdomain.
  - **Canonical docs URL**: `https://vabc.sh` (landing) with `/docs` (Starlight) once the domain is
    live — README, landing, and CITATION point here; **wire to the `*.vercel.app` URL initially** so
    nothing is asserted-but-dead, and swap to `vabc.sh` at cutover.

## Prompt-injection surface
Low but non-zero. Free text returned from the target: product names, store addresses / hours /
shopping-center names, and **`lottery check` event link titles + URLs**. The lottery event links
(arbitrary CMS-authored text + URLs) are the realistic injection vector → **fence `eventLinks` and
product/store free-text fields as untrusted by default in agent mode** (contract §8). No tokens or
mutating actions exist for an injection to abuse, which caps the blast radius.
