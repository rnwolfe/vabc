// Package vabc is a small, dependency-light client for Virginia ABC (Virginia
// Alcoholic Beverage Control) — product catalog, live per-store inventory, the
// statewide warehouse, the limited-availability ("lottery") hook, and the store
// locator.
//
// It is the importable core of the vabc CLI: the command-line tool in
// cmd/vabc is one consumer of this package, so anything the CLI can do is also
// available to Go programs via the Client interface and the catalog subpackage.
//
// Design notes:
//   - The live surface (inventory, stores, lottery) is plain HTTP+JSON against
//     undocumented but public, unauthenticated endpoints. No auth, no secrets.
//   - The product catalog has no live search API; it is served from a periodically
//     refreshed snapshot (see the catalog subpackage), keyed by 6-digit product code.
//   - Catalog generation (XLSX parsing) lives under internal/harvest so importers of
//     this package never inherit a heavy dependency graph — `go get` pulls HTTP+JSON only.
//
// The endpoints are reverse-engineered and may change without notice; pin behavior
// to this package's typed contract.
package vabc

// SchemaVersion is the output-contract version reported by `vabc schema --json`.
// Bump only on a breaking change to the JSON field contracts in this package.
const SchemaVersion = 1
