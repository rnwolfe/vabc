package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, strings.NewReader(""), &out, &errb)
	return out.String(), errb.String(), code
}

func setup(t *testing.T) {
	t.Helper()
	t.Setenv("NO_COLOR", "1")
	// Isolate throttle state and disable spacing so tests are fast and hermetic.
	t.Setenv("VABC_STATE_DIR", t.TempDir())
	t.Setenv("VABC_MIN_INTERVAL_MS", "0")
}

// --- product reads (live Coveo web catalog, faked via httptest) --------------

// coveoServer stands in for the site's Coveo product search. It matches a small
// fixture by name-substring or exact code and returns Coveo-shaped JSON, so the
// product commands can be exercised hermetically (no real network).
func coveoServer(t *testing.T) {
	t.Helper()
	type fx struct{ sku, name, cat string }
	fixtures := []fx{
		{"010807", "Crown Royal Regal Apple", "Whisky"},
		{"010646", "Crown Royal Blackberry", "Whisky"},
		{"042395", "Planteray Original Dark Rum", "Rum"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/coveo/rest/search") {
			http.NotFound(w, r)
			return
		}
		var body struct {
			Q string `json:"q"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		q := strings.ToLower(strings.TrimSpace(body.Q))
		var results []map[string]any
		for _, f := range fixtures {
			if q == "" || strings.Contains(strings.ToLower(f.name), q) || f.sku == q {
				results = append(results, map[string]any{
					"clickUri": "https://x/products/" + f.sku,
					"raw": map[string]any{
						"z95xproductz32xskuz32xids": []string{f.sku},
						"productz32xlabelz32xname":  f.name,
						"hierarchyz32xcategory":     f.cat,
					},
				})
			}
		}
		resp := map[string]any{"totalCount": len(results), "results": results}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("VABC_BASE_URL", srv.URL)
}

func TestProductSearchJSON(t *testing.T) {
	setup(t)
	coveoServer(t)
	out, _, code := run(t, "product", "search", "crown", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var products []map[string]any
	if err := json.Unmarshal([]byte(out), &products); err != nil {
		t.Fatalf("stdout not valid JSON array: %v\n%s", err, out)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 crown matches, got %d", len(products))
	}
	for _, p := range products {
		if !strings.Contains(strings.ToLower(p["name"].(string)), "crown") {
			t.Fatalf("unexpected non-matching result: %v", p["name"])
		}
	}
}

func TestProductSearchEmptyIsArray(t *testing.T) {
	setup(t)
	coveoServer(t)
	out, _, code := run(t, "product", "search", "zzzznotathing", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty search should emit [], got: %s", out)
	}
}

func TestProductGetJSON(t *testing.T) {
	setup(t)
	coveoServer(t)
	out, _, code := run(t, "product", "get", "010807", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	var p map[string]any
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		t.Fatalf("stdout not valid JSON: %v\n%s", err, out)
	}
	if p["productCode"] != "010807" {
		t.Fatalf("wrong product: %v", p["productCode"])
	}
}

func TestProductGetNotFound(t *testing.T) {
	setup(t)
	coveoServer(t)
	_, errb, code := run(t, "product", "get", "000000", "--json")
	if code != 5 {
		t.Fatalf("exit = %d, want 5 (not found)", code)
	}
	if !strings.Contains(errb, "NOT_FOUND") {
		t.Fatalf("stderr missing NOT_FOUND: %s", errb)
	}
}

func TestSelectProjection(t *testing.T) {
	setup(t)
	coveoServer(t)
	out, _, code := run(t, "product", "search", "", "--select", "productCode", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if strings.Contains(out, "\"name\"") {
		t.Fatalf("--select should drop name field: %s", out)
	}
}

// --- contract surface --------------------------------------------------------

func TestSchemaHasSafetyAndExitCodes(t *testing.T) {
	setup(t)
	out, _, code := run(t, "schema")
	if code != 0 {
		t.Fatalf("schema exit = %d, want 0", code)
	}
	var s map[string]any
	if err := json.Unmarshal([]byte(out), &s); err != nil {
		t.Fatalf("schema not valid JSON: %v", err)
	}
	if _, ok := s["safety"]; !ok {
		t.Fatalf("schema missing safety state")
	}
	codes, ok := s["exit_codes"].(map[string]any)
	if !ok {
		t.Fatalf("schema missing exit_codes")
	}
	if _, ok := codes["mutation_blocked"]; !ok {
		t.Fatalf("exit_codes missing mutation_blocked")
	}
}

func TestAgentPrintsSkill(t *testing.T) {
	setup(t)
	out, _, code := run(t, "agent")
	if code != 0 {
		t.Fatalf("agent exit = %d, want 0", code)
	}
	if !strings.Contains(out, "vabc") || !strings.Contains(out, "Exit codes") {
		n := 120
		if len(out) < n {
			n = len(out)
		}
		t.Fatalf("agent output does not look like SKILL.md: %s", out[:n])
	}
}

func TestDidYouMean(t *testing.T) {
	setup(t)
	_, errb, code := run(t, "prodcut", "search", "x")
	if code != 2 {
		t.Fatalf("exit = %d, want 2 (usage)", code)
	}
	if !strings.Contains(errb, "did you mean") || !strings.Contains(errb, "product") {
		t.Fatalf("missing suggestion: %s", errb)
	}
}

func TestNoSuggestionForValidCommand(t *testing.T) {
	setup(t)
	// A valid command with a bad subcommand/flag must NOT suggest itself.
	_, errb, code := run(t, "product", "nonsense")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if strings.Contains(errb, "did you mean") {
		t.Fatalf("should not suggest for a valid command: %s", errb)
	}
}

func TestAuthStatusNoAuth(t *testing.T) {
	setup(t)
	out, _, code := run(t, "auth", "status", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if !strings.Contains(out, "\"authRequired\": false") {
		t.Fatalf("auth status should report no auth: %s", out)
	}
}

// --- read-only posture (the mutation gate is present but inert) ---------------

func TestGuardBlocksByDefault(t *testing.T) {
	setup(t)
	// vabc has no mutating command, but the gate must still default-deny so that
	// adding one later is automatically protected (contract §2).
	rt := &Runtime{Cfg: &CLI{}}
	if err := rt.Guard("hypothetical mutation"); err == nil {
		t.Fatalf("Guard should block by default")
	}
	rt.Cfg.AllowMutations = true
	if err := rt.Guard("hypothetical mutation"); err != nil {
		t.Fatalf("Guard should allow with --allow-mutations, got %v", err)
	}
}

// --- live commands are wired end-to-end (httptest, no real network) ----------

func TestLiveWarehouseSuccess(t *testing.T) {
	setup(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"warehouseInventory":"42"}`))
	}))
	defer srv.Close()
	t.Setenv("VABC_BASE_URL", srv.URL)

	out, _, code := run(t, "inventory", "warehouse", "010807", "--json")
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	var res map[string]any
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	if res["warehouseInventory"].(float64) != 42 {
		t.Fatalf("want 42, got %v", res["warehouseInventory"])
	}
}

func TestLiveCommandStructuredError(t *testing.T) {
	setup(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("VABC_BASE_URL", srv.URL)

	out, errb, code := run(t, "inventory", "warehouse", "010807", "--json")
	if code != 8 {
		t.Fatalf("exit = %d, want 8 (retryable)", code)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("stdout should be empty on error, got: %s", out)
	}
	var e map[string]any
	if err := json.Unmarshal([]byte(errb), &e); err != nil {
		t.Fatalf("error not valid JSON on stderr: %v\n%s", err, errb)
	}
	if e["code"] == nil || e["code"] == "" {
		t.Fatalf("error envelope missing code: %s", errb)
	}
}
