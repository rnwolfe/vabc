// Package vabc is a small, dependency-light client for Virginia ABC (Virginia
// Alcoholic Beverage Control) — live product search, per-store inventory, the
// statewide warehouse, the limited-availability ("lottery") hook, and the store
// locator.
//
// It is the importable core of the vabc CLI: the command-line tool in cmd/vabc is
// one consumer of this package, so anything the CLI can do is also available to Go
// programs via the Client interface.
//
// Design notes:
//   - Everything is plain HTTP+JSON against undocumented but public, unauthenticated
//     endpoints. No auth, no secrets, no heavy dependencies — `go get` pulls a tiny
//     HTTP+JSON client only.
//   - Product search (SearchProducts) queries the site's Coveo index, which covers the
//     full web catalog; results carry the 6-digit inventory product code.
//
// The endpoints are reverse-engineered and may change without notice; pin behavior
// to this package's typed contract.
package vabc

// SchemaVersion is the output-contract version reported by `vabc schema --json`.
// Bump only on a breaking change to the JSON field contracts in this package.
const SchemaVersion = 1
