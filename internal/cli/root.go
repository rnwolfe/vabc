// Package cli wires the kong grammar, the runtime, and the exit-code mapping.
// main() does nothing but os.Exit(cli.Run(...)) so every path is testable in-process.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"

	"github.com/rnwolfe/vabc"
	"github.com/rnwolfe/vabc/catalog"
	"github.com/rnwolfe/vabc/internal/errs"
	"github.com/rnwolfe/vabc/internal/output"
)

// CLI is the kong grammar. Global flags are the universal agent-CLI contract surface;
// subcommands follow noun-verb grammar. vabc is read-only: the mutation flags below are
// present for contract uniformity but inert (no command mutates Virginia ABC).
type CLI struct {
	// Output (contract §1, §6)
	Format   string `enum:"json,plain,tsv" default:"plain" help:"Output format: json, plain, or tsv."`
	JSON     bool   `help:"Shorthand for --format=json."`
	NoColor  bool   `help:"Disable colored output."`
	Limit    int    `default:"50" help:"Maximum items to return for list operations."`
	Select   string `help:"Comma-separated dot-path field projection, e.g. productCode,name."`
	Concise  bool   `help:"Terser output (default)."`
	Detailed bool   `help:"Richer output."`

	// Safety (contract §2). Inert here — present for uniformity; vabc is read-only.
	AllowMutations bool `help:"Permit state-changing operations (no-op: vabc is read-only)."`
	DryRun         bool `help:"Print intended mutations without performing them (no-op: vabc is read-only)."`
	Yes            bool `help:"Assume yes for confirmations (no-op: vabc is read-only)."`
	Force          bool `help:"Bypass safety checks (no-op: vabc is read-only)."`
	NoInput        bool `help:"Never prompt; fail with exit 13 instead."`

	// Backend etiquette (contract §12). Live calls fail fast on a throttle block by
	// default so an agent loop never deadlocks; --wait opts into waiting it out.
	Wait    bool          `help:"Wait out throttle/backoff instead of failing fast."`
	MaxWait time.Duration `default:"30s" help:"Maximum time to wait when --wait is set."`

	// Commands
	Product   ProductCmd   `cmd:"" help:"Search and look up products (catalog snapshot)."`
	Inventory InventoryCmd `cmd:"" help:"Check live per-store and warehouse inventory."`
	Store     StoreCmd     `cmd:"" help:"List and locate Virginia ABC stores."`
	Lottery   LotteryCmd   `cmd:"" help:"Check limited-availability / allocated releases."`
	Catalog   CatalogCmd   `cmd:"" help:"Inspect or refresh the local product catalog snapshot."`
	Auth      AuthCmd      `cmd:"" help:"Authentication status (none required for Virginia ABC)."`
	Doctor    DoctorCmd    `cmd:"" help:"Diagnose setup and report fixes."`
	Schema    SchemaCmd    `cmd:"" help:"Print the machine-readable command schema (JSON)."`
	Agent     AgentCmd     `cmd:"" help:"Print the bundled agent SKILL.md."`
	Version   VersionCmd   `cmd:"" help:"Print the version."`
}

// topCommands lists the top-level command names for "did you mean" suggestions.
var topCommands = []string{
	"product", "inventory", "store", "lottery", "catalog",
	"auth", "doctor", "schema", "agent", "version",
}

// Runtime is the per-invocation context bound into every command's Run method.
type Runtime struct {
	Cfg     *CLI
	Out     *output.Writer
	Client  vabc.Client     // live Virginia ABC API
	Catalog catalog.Catalog // product snapshot (may be nil if it failed to load)
	Stdin   io.Reader
}

// Guard enforces the read-only-by-default mutation gate (contract §2). vabc exposes
// no mutating commands, so nothing calls this today; it is kept for contract
// uniformity and tested directly, so adding a mutation later is gated automatically.
func (rt *Runtime) Guard(op string) error {
	if rt.Cfg.AllowMutations {
		return nil
	}
	return errs.MutationBlocked(op)
}

// Run parses args and dispatches, returning the process exit code.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var cfg CLI
	helpShown := false
	parser, err := kong.New(&cfg,
		kong.Name("vabc"),
		kong.Description("Search Virginia ABC products and check store inventory from the command line."),
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) { helpShown = true }), // --help/--version: we control exit
	)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return errs.ExitGeneric
	}

	kctx, perr := parser.Parse(args)
	if helpShown {
		return errs.ExitOK
	}
	if perr != nil {
		return handleParseError(stderr, args, perr)
	}

	if cfg.JSON {
		cfg.Format = "json"
	}
	rt := newRuntime(&cfg, stdin, stdout, stderr)

	if err := kctx.Run(rt); err != nil {
		return emitError(rt, err)
	}
	return errs.ExitOK
}

func newRuntime(cfg *CLI, stdin io.Reader, stdout, stderr io.Writer) *Runtime {
	format := output.Format(cfg.Format)
	color := !cfg.NoColor && os.Getenv("NO_COLOR") == "" && isTTY(stdout) && format == output.FormatPlain
	var sel []string
	if cfg.Select != "" {
		sel = strings.Split(cfg.Select, ",")
	}
	w := &output.Writer{
		Stdout: stdout, Stderr: stderr,
		Format: format, Color: color, Limit: cfg.Limit, Select: sel,
	}
	return &Runtime{
		Cfg:     cfg,
		Out:     w,
		Client:  vabc.NewClient(clientOptions(cfg)...),
		Catalog: loadCatalog(w),
		Stdin:   stdin,
	}
}

// clientOptions builds the live-client options from flags + env overrides.
// VABC_BASE_URL / VABC_STORES_URL / VABC_MIN_INTERVAL_MS are mainly for testing
// and advanced use; VABC_STATE_DIR (read by the library) relocates throttle state.
func clientOptions(cfg *CLI) []vabc.Option {
	opts := []vabc.Option{vabc.WithWait(cfg.Wait, cfg.MaxWait)}
	if u := os.Getenv("VABC_BASE_URL"); u != "" {
		opts = append(opts, vabc.WithBaseURL(u))
	}
	if u := os.Getenv("VABC_STORES_URL"); u != "" {
		opts = append(opts, vabc.WithStoresURL(u))
	}
	if ms := os.Getenv("VABC_MIN_INTERVAL_MS"); ms != "" {
		if n, err := strconv.Atoi(ms); err == nil {
			opts = append(opts, vabc.WithMinInterval(time.Duration(n)*time.Millisecond))
		}
	}
	return opts
}

// loadCatalog resolves the catalog snapshot: a fresher local cache (if present)
// overrides the embedded seed. Returns nil only if both fail (commands then emit
// a structured CATALOG_UNAVAILABLE error).
func loadCatalog(w *output.Writer) catalog.Catalog {
	if p := cachePath(); p != "" {
		if c, err := catalog.Load(p); err == nil {
			return c
		}
	}
	c, err := catalog.Default()
	if err != nil {
		w.Info("warning: embedded catalog failed to load: %v", err)
		return nil
	}
	return c
}

// cachePath returns the runtime-refreshed catalog location, or "" if none.
// Override with VABC_CATALOG; otherwise $XDG_CACHE_HOME/vabc/catalog.json.
func cachePath() string {
	if p := os.Getenv("VABC_CATALOG"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		return ""
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(dir, "vabc", "catalog.json")
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// emitError prints a structured error to stderr and returns its exit code (contract §3).
func emitError(rt *Runtime, err error) int {
	var ce *errs.CLIError
	if !errors.As(err, &ce) {
		ce = errs.New(errs.ExitGeneric, "INTERNAL", err.Error(), "")
	}
	if rt.Out.Format == output.FormatJSON {
		enc := json.NewEncoder(rt.Out.Stderr)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"error":       ce.Message,
			"code":        ce.Code,
			"remediation": ce.Remediation,
		})
	} else {
		fmt.Fprintf(rt.Out.Stderr, "error: %s\n", ce.Message)
		if ce.Code != "" {
			fmt.Fprintf(rt.Out.Stderr, "  code: %s\n", ce.Code)
		}
		if ce.Remediation != "" {
			fmt.Fprintf(rt.Out.Stderr, "  fix:  %s\n", ce.Remediation)
		}
	}
	return ce.Exit
}

// handleParseError reports usage errors and offers a "did you mean" suggestion.
func handleParseError(stderr io.Writer, args []string, err error) int {
	fmt.Fprintf(stderr, "error: %s\n", err)
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if s, ok := closest(a, topCommands); ok {
			fmt.Fprintf(stderr, "  did you mean %q?\n", s)
		}
		break
	}
	return errs.ExitUsage
}
