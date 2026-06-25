---
title: Flags & environment
description: Every global flag and environment variable that vabc accepts, with defaults and effects.
sidebar:
  order: 2
---

Global flags are accepted before any subcommand and apply to all commands. Environment variables are read at startup and override built-in defaults; they cannot be set per-invocation via flag.

## Global flags

All flags below are valid in every invocation:

```bash
vabc [GLOBAL FLAGS] <command> [COMMAND FLAGS] [ARGS]
```

### Output

| Flag | Default | Meaning |
|---|---|---|
| `--format json\|plain\|tsv` | `plain` | Output format. `plain` renders a human-readable aligned table; `tsv` emits tab-separated values suitable for `cut` or `awk`; `json` emits 2-space-indented JSON with HTML escaping disabled. |
| `--json` | off | Shorthand for `--format json`. When both are set, `--json` wins. |
| `--no-color` | off | Disable ANSI colour. Colour is already suppressed when stdout is not a TTY, when `NO_COLOR` is set, or when `--format` is not `plain`. |
| `--limit N` | `50` | Maximum items returned from list operations. Truncation is noted on stderr. Set to `0` for no limit. |
| `--select a,b.c` | (none) | Comma-separated dot-path field projection applied after `--limit`. Nested fields use dot notation. Unmatched paths are silently omitted. |
| `--concise` | off | Terser output. |
| `--detailed` | off | Richer output where commands support it. |

### Prompt-injection hardening

| Flag | Default | Meaning |
|---|---|---|
| `--wrap-untrusted` / `--no-wrap-untrusted` | on | Fence free-text content fetched from the target (lottery event titles) with `⟦UNTRUSTED⟧ … ⟦/UNTRUSTED⟧` markers. Keeps an agent from acting on instructions embedded in CMS content. Disable only when you are post-processing the text yourself. |

### Safety

| Flag | Default | Meaning |
|---|---|---|
| `--allow-mutations` | off | Would permit state-changing operations. No-op — vabc is read-only. |
| `--dry-run` | off | Would print intended mutations without executing them. No-op — vabc is read-only. |
| `--yes` | off | Would assume yes for confirmations. No-op — vabc is read-only. |
| `--force` | off | Would bypass safety checks. No-op — vabc is read-only. |
| `--no-input` | off | Never prompt for interactive input; fail with exit 13 (`input_required`) instead. Useful in scripts and CI. |

`--allow-mutations`, `--dry-run`, `--yes`, and `--force` are inert (present for contract uniformity; vabc makes no mutations). `--no-input` is active: it prevents any interactive prompt and exits 13 instead.

### Backend etiquette / throttle

| Flag | Default | Meaning |
|---|---|---|
| `--wait` | off | Wait out an open throttle circuit-breaker instead of failing fast with exit 7. |
| `--max-wait DURATION` | `30s` | Upper bound on how long `--wait` will block. Accepts Go duration strings (`15s`, `2m`). Has no effect unless `--wait` is also set. |

By default, when the persistent cross-process throttle detects a recent block, the command **fails immediately** with exit 7 and prints a retry hint. This prevents an agent loop from deadlocking on a long backoff. Pass `--wait` to opt in to waiting.

```bash
# fail fast (default) — agent gets exit 7 and can schedule a retry
vabc inventory check 010807 --near 22182

# wait up to 45 s for the circuit breaker to clear
vabc --wait --max-wait 45s inventory check 010807 --near 22182
```

## Environment variables

| Variable | What it overrides | Default |
|---|---|---|
| `VABC_BASE_URL` | Base URL for all inventory and lottery endpoints (`/webapi/inventory/*`, `/webapi/limitedavailability/*`). | `https://www.abc.virginia.gov` |
| `VABC_STORES_URL` | Full URL for the ArcGIS FeatureServer store-locator query. | `https://services9.arcgis.com/…/Virginia_ABC_Stores/FeatureServer/0/query` |
| `VABC_MIN_INTERVAL_MS` | Minimum milliseconds between requests (politeness throttle). Parsed as an integer; non-numeric values are ignored. | `250` |
| `VABC_STATE_DIR` | Directory for the throttle state file (`throttle.json`). Falls back to `$XDG_STATE_HOME/vabc` then `os.UserCacheDir()/vabc`. | (platform cache dir) |

`VABC_BASE_URL` and `VABC_STORES_URL` are primarily for testing or pointing at a local proxy. In normal use you should leave them unset.

`VABC_STATE_DIR` is useful when you want to isolate throttle state per project or CI job:

```bash
export VABC_STATE_DIR=/tmp/vabc-ci
vabc inventory check 953714 --near 22182
```

`VABC_MIN_INTERVAL_MS` lets you increase the spacing between requests if you are running a batch script that calls vabc in a loop:

```bash
export VABC_MIN_INTERVAL_MS=500
for code in 010807 953714; do
  vabc --json inventory check "$code" --near 22182
done
```

## Combining flags and env vars

Flags take precedence over environment variables for anything that has both a flag and an env var counterpart (`--wait`, `--max-wait`). Environment variables are the only way to override endpoints and throttle timing — there are no equivalent flags.

The output-routing contract is always in effect regardless of flags: data goes to **stdout**, diagnostic notes and errors go to **stderr**. JSON errors on stderr follow the `{ "error", "code", "remediation" }` shape even when `--format` is `plain`.

```bash
# pipe only the JSON data; errors still appear in the terminal
vabc --json product search "apple bourbon" --limit 5 2>/dev/null | jq '.[].productCode'
```

## Related

- [Commands](/docs/reference/commands/) — full subcommand and argument reference
- [Exit codes](/docs/reference/exit-codes/) — numeric exit codes and their meanings
- [Scripting with JSON](/docs/guides/scripting-with-json/) — practical piping and projection patterns
- [For agents](/docs/guides/for-agents/) — throttle behaviour and the agent contract
