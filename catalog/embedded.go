package catalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed data/catalog.json
var embeddedJSON []byte

// Default returns the catalog snapshot embedded in the binary at build time.
// It always works offline. The embedded snapshot is a small seed; the CI catalog
// refresh regenerates it from Virginia ABC's quarterly price list.
func Default() (Catalog, error) {
	var snap snapshot
	if err := json.Unmarshal(embeddedJSON, &snap); err != nil {
		return nil, fmt.Errorf("decode embedded catalog: %w", err)
	}
	return newStore(snap, "embedded"), nil
}

// Load reads a catalog snapshot from a file (e.g. the XDG cache written by
// `vabc catalog refresh`). Used to overlay fresher data over the embedded seed.
func Load(path string) (Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, fmt.Errorf("decode catalog %s: %w", path, err)
	}
	return newStore(snap, "cache:"+path), nil
}
