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
./vabc product search bourbon        # offline (embedded catalog snapshot)
```

## Layout (library-first — keep the API importable & dependency-light)
- **`/` (package `vabc`)** — the public, importable API: `Client` interface (live inventory /
  stores / lottery), domain types, `NewClient`. **HTTP + JSON deps only.** Do not import
  `catalog`, `internal/*`, or anything heavy here.
- **`catalog/`** — `Catalog` interface + embedded-snapshot provider. Imports `vabc` only.
- **`internal/harvest/`** — XLSX → catalog generation (gets `excelize` in cli-implement).
  Quarantined so importers of `vabc`/`catalog` never inherit it.
- **`internal/cli/`** — the thin kong CLI (parse → call library → format). No business logic.
- **`internal/{output,errs,version,skill}/`** — the agent-CLI contract surface. Treat as
  stable; `output` (stdout/stderr split, `--format/--select/--limit`) and `errs` (exit-code
  table) are load-bearing.
- **`cmd/vabc/`** — `main()` is `os.Exit(cli.Run(...))` only.
- **`cmd/vabc-catalog-gen/`** — maintainer tool that regenerates `catalog/data/catalog.json`.

## Conventions / invariants
- **Read-only.** vabc never mutates Virginia ABC. The `--allow-mutations`/`--dry-run` flags are
  inert (present for contract uniformity). The `Guard` gate stays default-deny so a future
  mutation is protected automatically.
- **Output contract**: data → stdout, everything else (scope notes, warnings) → stderr. JSON is
  2-space, `SetEscapeHTML(false)`. Never print to stdout outside `output.Writer`.
- **Exit codes** are append-only (`internal/errs`). vabc adds `11 catalog_unavailable`,
  `14 catalog_stale`.
- **Catalog vs live**: product search/get read the snapshot; inventory/store/lottery are live.
  Surface freshness via the `scope:` stderr note + `catalog status`.
- **Backend etiquette**: the inventory endpoints are undocumented and Cloudflare-exempt by
  courtesy. cli-implement must add persistent cross-process throttle/backoff under
  `os.UserCacheDir()/vabc`. Never add evasion (UA spoofing, proxy rotation, CAPTCHA solving).

## Status
Implemented and verified. The live HTTP client (inventory/stores/lottery), ArcGIS store locator,
persistent cross-process throttle/circuit-breaker, ZIP/lat-lng resolution, XLSX catalog harvester,
prompt-injection fencing, and the full agent-CLI contract all work; validated against the real API
(read-only) and with httptest unit tests. Next (optional): **cli-publish** for the landing page +
docs site. See `spec.md`.

<!-- DOCS-FRESHNESS: when commands/flags/output fields change, update internal/skill/SKILL.md and the schema snapshot. -->
