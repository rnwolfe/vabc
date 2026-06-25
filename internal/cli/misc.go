package cli

import (
	"context"
	"errors"
	"time"

	"github.com/alecthomas/kong"

	"github.com/rnwolfe/vabc"
	"github.com/rnwolfe/vabc/internal/errs"
	"github.com/rnwolfe/vabc/internal/skill"
	"github.com/rnwolfe/vabc/internal/version"
)

// --- auth -------------------------------------------------------------------
// Virginia ABC's data endpoints are public and unauthenticated, so there is no
// login/logout. `auth status` exists only for contract uniformity.

type AuthCmd struct {
	Status AuthStatusCmd `cmd:"" help:"Show authentication status (none required)."`
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(rt *Runtime) error {
	return rt.Out.Emit(map[string]any{
		"authRequired": false,
		"ok":           true,
		"note":         "Virginia ABC's endpoints are public; no authentication is needed",
	})
}

// --- doctor -----------------------------------------------------------------
// Offline-safe checks only (no network) so doctor is deterministic in CI and for
// agents. cli-implement adds live reachability probes for the inventory + ArcGIS
// endpoints behind a --network/--online flag.

type DoctorCmd struct {
	Online bool `help:"Also probe live endpoint reachability (makes network requests)."`
}

func (c *DoctorCmd) Run(rt *Runtime) error {
	checks := []map[string]any{
		{"name": "auth", "ok": true, "detail": "no authentication required"},
	}

	if c.Online {
		ctx, cancel := context.WithTimeout(rt.Ctx, 12*time.Second)
		defer cancel()
		_, invErr := rt.Client.Warehouse(ctx, "010807")
		checks = append(checks, probe("inventory_endpoint", vabc.DefaultBaseURL, invErr))
		_, stErr := rt.Client.Stores(ctx)
		checks = append(checks, probe("stores_endpoint", "ArcGIS FeatureServer", stErr))
		_, searchErr := rt.Client.SearchProducts(ctx, "bourbon", 1)
		checks = append(checks, probe("product_search", "Coveo web catalog", searchErr))
	} else {
		checks = append(checks,
			map[string]any{"name": "inventory_endpoint", "ok": true, "detail": "configured: " + vabc.DefaultBaseURL + " (pass --online to probe)"},
			map[string]any{"name": "stores_endpoint", "ok": true, "detail": "configured: ArcGIS FeatureServer (pass --online to probe)"},
			map[string]any{"name": "product_search", "ok": true, "detail": "configured: Coveo web catalog (pass --online to probe)"},
		)
	}

	allOK := true
	for _, ch := range checks {
		if ok, _ := ch["ok"].(bool); !ok {
			allOK = false
		}
	}
	if !allOK {
		return errs.New(errs.ExitConfig, "DOCTOR_FAILED", "one or more checks failed", "see the failing check's detail")
	}
	return rt.Out.Emit(map[string]any{"ok": true, "checks": checks})
}

// probe classifies a live reachability result for doctor.
func probe(name, target string, err error) map[string]any {
	if err == nil {
		return map[string]any{"name": name, "ok": true, "detail": "reachable: " + target}
	}
	var ae *vabc.APIError
	if errors.As(err, &ae) && ae.Kind == vabc.KindRateLimited {
		return map[string]any{"name": name, "ok": false, "detail": "throttled/blocked (try later or --wait): " + target}
	}
	return map[string]any{"name": name, "ok": false, "detail": "unreachable: " + err.Error()}
}

// --- schema -----------------------------------------------------------------

type SchemaCmd struct{}

func (c *SchemaCmd) Run(rt *Runtime) error {
	k, err := kong.New(&CLI{}, kong.Name("vabc"))
	if err != nil {
		return errs.New(errs.ExitGeneric, "SCHEMA_ERROR", err.Error(), "")
	}
	out := map[string]any{
		"tool":       "vabc",
		"version":    version.String(),
		"commands":   nodeToMap(k.Model.Node),
		"exit_codes": errs.Table(),
		"safety": map[string]any{
			"allow_mutations": rt.Cfg.AllowMutations,
			"dry_run":         rt.Cfg.DryRun,
			"no_input":        rt.Cfg.NoInput,
		},
	}
	return rt.Out.EmitJSON(out) // schema is always JSON
}

func nodeToMap(n *kong.Node) map[string]any {
	m := map[string]any{"name": n.Name}
	if n.Help != "" {
		m["help"] = n.Help
	}
	var flags []map[string]any
	for _, f := range n.Flags {
		if f.Name == "help" {
			continue
		}
		fm := map[string]any{"name": f.Name}
		if f.Help != "" {
			fm["help"] = f.Help
		}
		if f.Default != "" {
			fm["default"] = f.Default
		}
		flags = append(flags, fm)
	}
	if len(flags) > 0 {
		m["flags"] = flags
	}
	var args []map[string]any
	for _, p := range n.Positional {
		args = append(args, map[string]any{"name": p.Name, "help": p.Help})
	}
	if len(args) > 0 {
		m["args"] = args
	}
	var subs []any
	for _, ch := range n.Children {
		subs = append(subs, nodeToMap(ch))
	}
	if len(subs) > 0 {
		m["subcommands"] = subs
	}
	return m
}

// --- agent ------------------------------------------------------------------

type AgentCmd struct{}

func (c *AgentCmd) Run(rt *Runtime) error {
	_, err := rt.Out.Stdout.Write([]byte(skill.Content))
	return err
}

// --- version ----------------------------------------------------------------

type VersionCmd struct{}

func (c *VersionCmd) Run(rt *Runtime) error {
	return rt.Out.Emit(map[string]any{"version": version.String()})
}
