// Package catalog provides product search/lookup over a periodically refreshed
// snapshot of the Virginia ABC catalog, keyed by 6-digit product code.
//
// Virginia ABC exposes no live, agent-usable product-search API (search sits behind
// a Cloudflare challenge), so catalog data is served from a snapshot: one embedded
// in the binary at build time, optionally overlaid by a fresher local file written
// by `vabc catalog refresh`. The Catalog interface lets callers swap the data source.
package catalog

import (
	"sort"
	"strings"

	"github.com/rnwolfe/vabc"
)

// SearchOpts filters a catalog search. An empty Query matches everything (use with
// Type/Allocated to browse). Allocated is a tri-state: nil = no filter.
type SearchOpts struct {
	Query     string
	Type      string
	Allocated *bool
}

// Catalog is a searchable product snapshot.
type Catalog interface {
	// Search returns products matching opts. Results are not length-bounded here;
	// the CLI applies --limit. Ordering is deterministic (by product code).
	Search(opts SearchOpts) ([]vabc.Product, error)
	// Get returns one product by 6-digit code; ok is false if absent.
	Get(productCode string) (vabc.Product, bool, error)
	// SnapshotDate is the snapshot's build date (YYYY-MM-DD), for freshness reporting.
	SnapshotDate() string
	// Source describes where the snapshot came from, e.g. "embedded" or "cache:<path>".
	Source() string
	// Count is the number of products in the snapshot.
	Count() int
}

// snapshot is the on-disk/embedded catalog file shape.
type snapshot struct {
	SchemaVersion int            `json:"schemaVersion"`
	SnapshotDate  string         `json:"snapshotDate"`
	Note          string         `json:"note,omitempty"`
	Products      []vabc.Product `json:"products"`
}

// store is the in-memory Catalog implementation shared by the embedded and
// file-backed providers.
type store struct {
	snap   snapshot
	source string
	byCode map[string]vabc.Product
}

func newStore(snap snapshot, source string) *store {
	idx := make(map[string]vabc.Product, len(snap.Products))
	for _, p := range snap.Products {
		idx[p.ProductCode] = p
	}
	return &store{snap: snap, source: source, byCode: idx}
}

func (s *store) SnapshotDate() string { return s.snap.SnapshotDate }
func (s *store) Source() string       { return s.source }
func (s *store) Count() int           { return len(s.snap.Products) }

func (s *store) Get(productCode string) (vabc.Product, bool, error) {
	p, ok := s.byCode[productCode]
	return p, ok, nil
}

func (s *store) Search(opts SearchOpts) ([]vabc.Product, error) {
	q := strings.ToLower(strings.TrimSpace(opts.Query))
	typ := strings.ToLower(strings.TrimSpace(opts.Type))
	out := make([]vabc.Product, 0) // non-nil so empty search emits [] not null
	for _, p := range s.snap.Products {
		if q != "" && !strings.Contains(strings.ToLower(p.Name), q) &&
			!strings.Contains(strings.ToLower(p.ProductCode), q) {
			continue
		}
		if typ != "" && !strings.Contains(strings.ToLower(p.Type), typ) &&
			!strings.Contains(strings.ToLower(p.Category), typ) {
			continue
		}
		if opts.Allocated != nil && p.Allocated != *opts.Allocated {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductCode < out[j].ProductCode })
	return out, nil
}
