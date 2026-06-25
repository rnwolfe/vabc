# AGENTS.md — vabc

Agent-focused CLI + Go library for Virginia ABC (product search, store inventory, store
locator, limited-availability). This file is for agents editing the code. The bundled,
end-user-facing usage contract lives in `internal/skill/SKILL.md` (printed by `vabc agent`).

## Build / test / run
```bash
go build ./...                       # build everything
go vet ./...                         # vet
go test ./...                        # unit + contract tests (internal/cli)
go build -o vabc ./cmd/vabc          # build the CLI
./vabc schema --json                 # inspect the command tree
./vabc product search bourbon        # live Coveo web-catalog search
```

## Layout (library-first — keep the API importable & dependency-light)
- **`/` (package `vabc`)** — the public, importable API: `Client` interface (product search +
  inventory / stores / lottery), domain types, `NewClient`. **HTTP + JSON deps only.** Do not
  import `internal/*` or anything heavy here. Files: `client.go` (transport + throttle),
  `coveo.go` (live product search), `inventory.go`, `stores.go`, `lottery.go`, `errors.go`,
  `throttle.go`, `types.go`.
- **`internal/cli/`** — the thin kong CLI (parse → call library → format). No business logic.
- **`internal/geocode/`** — ZIP/address → coordinates (embedded ZCTA centroids + Census
  geocoder). Internal so the embedded table never reaches importers.
- **`internal/{output,errs,version,skill}/`** — the agent-CLI contract surface. Treat as
  stable; `output` (stdout/stderr split, `--format/--select/--limit`) and `errs` (exit-code
  table) are load-bearing.
- **`cmd/vabc/`** — `main()` is `os.Exit(cli.Run(...))` only.

## Conventions / invariants
- **Read-only.** vabc never mutates Virginia ABC. The `--allow-mutations`/`--dry-run` flags are
  inert (present for contract uniformity). The `Guard` gate stays default-deny so a future
  mutation is protected automatically.
- **Everything is live.** No embedded catalog. `product search`/`get` go through Coveo
  (`SearchProducts`); results carry the inventory product code. There is no offline mode.
- **Output contract**: data → stdout, everything else (scope notes, warnings) → stderr. JSON is
  2-space, `SetEscapeHTML(false)`. Never print to stdout outside `output.Writer`.
- **Exit codes** are append-only (`internal/errs`).
- **Backend etiquette**: the endpoints are undocumented and reachable by courtesy. The client
  carries a persistent cross-process throttle/circuit-breaker (`throttle.go`) under
  `os.UserCacheDir()/vabc`. Never add evasion (UA spoofing, proxy rotation, CAPTCHA solving).

## Status
Implemented and verified. Live product search (Coveo), inventory/warehouse, ArcGIS store locator,
limited-availability, persistent throttle/circuit-breaker, ZIP/address geocoding, prompt-injection
fencing, and the full agent-CLI contract all work; validated against the real API (read-only) and
with httptest unit tests. Next (optional): **cli-publish** for the landing page + docs site.

## Web presence & freshness

The site lives in `site/` (Astro + Starlight). The landing (`src/pages/index.astro`) and docs
share ONE token source: `site/src/styles/tokens.css`. Canonical URL is the live Vercel alias
`https://vabc-cli.vercel.app` (set in `astro.config.mjs`); cut over to `https://vabc.sh` once the
domain is bought (update `astro.config.mjs` default + `SITE_URL`, re-deploy, re-alias, regenerate
OG cards, and find/replace `vabc-cli.vercel.app` in README/docs/install.sh).

**FRESHNESS — in the SAME PR as a change to the value proposition, command surface, flags, exit
codes, or brand, update all of:**
1. the affected docs page under `site/src/content/docs/` **and** `internal/skill/SKILL.md`;
2. the **landing** copy/examples (`site/src/pages/index.astro`);
3. regenerate the **OG cards** (`cd site && node scripts/gen-og.mjs`) if titles/headline changed;
4. the **README** quickstart/badges; and run `VABC_UPDATE_GOLDEN=1 go test ./internal/cli/` if the
   grammar changed. Rebuild the site so `llms.txt`/sitemap regenerate. Docs *and* landing are "done".
