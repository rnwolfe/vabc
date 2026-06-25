# Contributing to vabc

Thanks for your interest! `vabc` is a small, dependency-light Go CLI + library for Virginia ABC.

## Development

```bash
git clone https://github.com/rnwolfe/vabc
cd vabc
go build ./...        # build
go vet ./...          # vet
go test ./...         # unit + contract tests
go build -o vabc ./cmd/vabc && ./vabc --help
```

Go 1.25+ is required. The only dependency is [`kong`](https://github.com/alecthomas/kong).

## Architecture (where things live)

- **`/` (package `vabc`)** — the importable HTTP+JSON client (`Client` interface, typed models).
  Keep this dependency-light; do not import `internal/*` here.
- **`internal/cli/`** — the thin kong CLI (parse → call library → format). No business logic.
- **`internal/geocode/`** — ZIP/address → coordinates (embedded centroids + Census geocoder).
- **`internal/{output,errs,version,skill}/`** — the agent-CLI contract surface. Treat as stable.

See [`AGENTS.md`](./AGENTS.md) for the full map and invariants.

## Conventions

- **[Conventional Commits](https://www.conventionalcommits.org/)** (`feat:`, `fix:`, `docs:`, …).
  The changelog and version bumps derive from them.
- **Sign off your commits** (`git commit -s`) — we use the
  [DCO](https://developercertificate.org/). No CLA.
- **The output contract is load-bearing**: data → stdout, everything else → stderr; JSON is
  2-space with HTML escaping off. Never print to stdout outside `output.Writer`.
- **Exit codes and output fields are append-only.** If you change the command grammar, run the
  schema-snapshot gate and regenerate the golden: `VABC_UPDATE_GOLDEN=1 go test ./internal/cli/`.
- **Keep it read-only and polite.** No mutating commands; no scraping evasion.

## Pull requests

1. Fork, branch, and make focused changes with tests.
2. `go build ./... && go vet ./... && go test ./...` must pass.
3. Update docs in the **same PR** when commands/flags/output change (see the freshness note in
   `AGENTS.md`).
4. Open the PR with a Conventional-Commit title and a clear description.
