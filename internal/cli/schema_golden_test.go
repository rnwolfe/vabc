package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSchemaSnapshot is the contract-stability gate (contract §10): any change to
// the command tree, flags, or exit-code table is a reviewed diff, not a silent
// break. Regenerate intentionally with VABC_UPDATE_GOLDEN=1.
func TestSchemaSnapshot(t *testing.T) {
	setup(t)
	out, _, code := run(t, "schema")
	if code != 0 {
		t.Fatalf("schema exit = %d", code)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("schema not JSON: %v", err)
	}
	delete(m, "version") // volatile; not part of the contract surface

	norm, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "schema.golden.json")

	if os.Getenv("VABC_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, append(norm, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("missing golden (regenerate with VABC_UPDATE_GOLDEN=1): %v", err)
	}
	if strings.TrimSpace(string(want)) != strings.TrimSpace(string(norm)) {
		t.Fatalf("schema drift — if intentional, regenerate with VABC_UPDATE_GOLDEN=1\n--- got ---\n%s", norm)
	}
}
