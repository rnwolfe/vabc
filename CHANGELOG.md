# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-25

Initial release.

### Added

- **Live product search** over Virginia ABC's web catalog (`product search`, `product get`) —
  full coverage including new/online-only SKUs; results carry the inventory product code.
- **Live inventory** — `inventory check` (per-store availability + nearby stores ranked by
  distance) and `inventory warehouse` (statewide stock).
- **Store locator** — `store list`, `store get`, `store near` (by ZIP, street address, or
  lat,lng; distances measured from the resolved point).
- **Limited-availability** — `lottery check` for allocated/lottery releases, with untrusted-text
  fencing.
- **Agent-CLI contract** — `--json`/`--format`, `--limit`, `--select`, structured errors with
  stable exit codes, an embedded `agent` guide (`SKILL.md`), `schema --json`, and `doctor`.
- **Backend etiquette** — a persistent, cross-process throttle/circuit-breaker; no evasion.
- Read-only by design; no authentication required.

[Unreleased]: https://github.com/rnwolfe/vabc/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/rnwolfe/vabc/releases/tag/v0.1.0
