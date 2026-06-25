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
    → **Implemented (live-only):** `product search`/`get` query Coveo LIVE. The earlier offline
    snapshot/XLSX-harvest/`catalog` subsystem was **removed** — it was incomplete (missed
    online-only SKUs like OFTD) and bloated the binary (~4 MB, incl. the `excelize` dep). The
    catalog has no offline tier; everything is live.
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
- **SDK/library used**: direct HTTP (`net/http` + `encoding/json`) for the whole surface — product
  search (Coveo), inventory, stores, lottery. **No heavy deps, no headless browser.**
- **Blueprint**: references/research/blueprint-go.md
- **Language-specific gotchas to honor**:
  - Keep the public library tiny: HTTP + JSON only. The ZIP/address geocoder + embedded ZCTA
    centroid table live under `internal/geocode` so importers never inherit them.
  - Pin a thin `webapi` client to the verified contract; tolerate the plural-but-singular param quirk
    and per-pair fan-out (no comma lists).
  - `productCode` is **NOT validated server-side** — a bogus code returns `200 {quantity:0}`. So
    inventory cannot confirm a SKU exists; existence is confirmed via Coveo `product search`/`get`.
  - `storeNumber` IS validated (bad number → `400 "No Store exists…"`) — map to exit 5.
  - The inventory `storeId` (small int) = the ABC store number embedded in ArcGIS `LandmkName`
    ("ABC Store 088" → 88), **not** the long `VAPID`. Join on the store-number suffix.

## Architecture & packaging (library-first — decoupled API)
The reusable API is the product; the CLI is one consumer of it. Importers must get clean typed
interfaces and a **lightweight dependency footprint** (HTTP + JSON only — no headless browser, no
kong).

**Module layout** (`github.com/rnwolfe/vabc`):
```
github.com/rnwolfe/vabc                  (root = the public library; tiny deps — HTTP+JSON only)
├── client.go            // Client struct + functional options; persistent throttle/circuit-breaker
├── coveo.go             // SearchProducts(ctx, query, limit) — live web-catalog search
├── inventory.go         // StoreNearby(ctx, store, code), MyStore(...), Warehouse(ctx, code)
├── stores.go            // Stores(ctx) ArcGIS locator; StoreNear(ctx, lat, lng, limit)
├── lottery.go           // LimitedAvailability(ctx, code)
├── throttle.go errors.go types.go
├── internal/geocode/    // ZIP/address → coords (embedded ZCTA centroids + Census) — not public
├── internal/cli/        // thin kong CLI: parse → call library → format. No business logic.
└── cmd/vabc/            // main() = os.Exit(cli.Run(...)) only
```

**Decoupling rules:**
- **Public interface, not concretes.** `Client` is an interface; the CLI and any third party depend
  on it, so the HTTP implementation can be swapped or mocked.
- **Politeness lives in the library, not the CLI.** The persistent cross-process throttle/circuit-
  breaker is part of the `Client`, so *every* importer is a polite citizen of the undocumented API.
- **Tiny import graph.** `go get github.com/rnwolfe/vabc` pulls an HTTP+JSON client only — no
  embedded data, no excelize, no browser. The geocoder + ZCTA table are quarantined in
  `internal/geocode`.

### Product catalog (live, no offline tier)
`product search`/`get` query the site's **Coveo index** (`POST /coveo/rest/search/v2`, anonymous) —
the full web catalog, current, including online-only SKUs. Coveo results carry the 6-digit inventory
`productCode`, so a search hit feeds straight into inventory. There is **no embedded snapshot and no
`catalog` command** (the original XLSX-snapshot design was removed as incomplete + bloating).

## Auth
- **Model**: **None.** All surfaces (`/webapi/*` inventory, ArcGIS stores, the Coveo search proxy)
  are **public and unauthenticated**. No API key, bearer, cookie/session, or CSRF.
- **Provider constraints**: n/a — nothing to authenticate.
- **Feasible path to usability (end-to-end)**: trivially headless — the tool works on first run with
  no setup. No tokens, no onboarding, no secret storage required.
- **Secret storage**: n/a (no secrets).
- **Subcommands**: no `login/logout/refresh`. Keep a single **`vabc auth status`** for contract
  uniformity that reports `{"authRequired": false, "ok": true}` and exits 0. `doctor` (below) covers
  real readiness (endpoint reachability + catalog freshness).

## Command surface (noun-verb)
| Command | Read/Mutation | Description | Key output fields |
|---|---|---|---|
| `product search <query>` | read | Live keyword search over the Coveo web catalog. `--allocated` filter, `--type`, `--limit`, `--select`. | `productCode, name, category, type, proof, size, retailPrice, allocated, onlineOrderable, new, url` |
| `product get <productCode>` | read | Live lookup of one 6-digit SKU (via Coveo). | (all product fields above) |
| `inventory check <productCode>` | read | Live per-store availability + nearby stores ranked by distance. `--store <n>` (anchor) **or** `--near <ZIP\|address\|lat,lng>` (geocode → nearest store, then `storeNearby`). | `productCode, store{storeNumber,quantity,address,city,zip,distance,lat,lng,phone,hours,url}, nearbyStores[]` |
| `inventory warehouse <productCode>` | read | Statewide central-warehouse stock. | `productCode, warehouseInventory` |
| `lottery check <productCode>` | read | Active limited-availability / allocated event links for a SKU. | `productCode, allocated, active, eventLinks[]` (fenced untrusted) |
| `store list` | read | All ~394 stores from the ArcGIS dataset. `--limit`, `--select`. | `storeNumber, name, address, city, state, zip, phone, lat, lng, url` |
| `store get <storeNumber>` | read | One store's details. | (store fields above) |
| `store near <ZIP\|address\|lat,lng>` | read | Nearest stores to a geocoded location, by distance. `--limit`. | `storeNumber, name, address, distance, lat, lng, …` |
| `auth status` | read | Reports no-auth (contract uniformity). | `authRequired:false, ok` |
| `doctor [--online]` | read | Offline summary; `--online` probes inventory/ArcGIS/Coveo reachability. | `checks[]{name,ok,detail}` |

No state-changing operations against the target exist anywhere in scope (VA ABC exposes no
cart/order API we wrap) → **the tool is wholly read-only**. `--allow-mutations` is present (scaffold)
but gates nothing.

## Exit codes
Base from contract §4:
```
0   ok                         5  not found (unknown 6-digit SKU; or 400 "No Store exists…")
1   generic error              7  rate limited / WAF challenge
2   usage/parse                8  retryable/transient (network, upstream 5xx)
3   empty results              10 config error
130 cancelled (SIGINT)
```
Reserved but unused: `4 auth required` (no auth), `6 permission denied`, `12 mutation blocked` (no mutations).

## Output schema
`schemaVersion: 1` (reported by `schema --json`). Reads emit the domain data *directly* (arrays for
lists, objects for single records) so `--limit`/`--select` operate naturally; scope/context notes go
to **stderr**. Field contracts (append-only):
- **product**: `productCode` (string, 6-digit zero-padded), `name`, `category`, `type`, `proof`
  (number|null), `size`, `retailPrice` (number|null), `allocated` (bool), `onlineOrderable` (bool),
  `new` (bool), `url`.
- **inventory check**: `productCode`, `store` { `storeNumber` (int), `quantity` (int), `distance`
  (number, miles, pointer — present incl. 0.0 when computed), `address`, `address1`, `address2`,
  `city`, `state`, `zip`, `lat`, `lng`, `phone`, `hours`, `shoppingCenter`, `url` }, `nearbyStores`
  (same store shape, ranked by `distance`).
- **inventory warehouse**: `productCode`, `warehouseInventory` (int).
- **lottery check**: `productCode`, `allocated` (bool, from the product's web-catalog record),
  `active` (bool), `eventLinks` (array of `{title,url}` — **fenced untrusted**).
- **store**: `storeNumber` (int), `name`, `address`, `city`, `state`, `zip`, `phone`, `lat`, `lng`,
  `url`; `distance` added on `store near`.
- **doctor**: `checks[]{name,ok,detail}`.

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
  immediately (no auth, no setup).
- **Agent hot-loop path**: prebuilt static Go binary — lowest cold start, embedded SKILL.md, no
  setup. (All product/inventory data is live.)

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
