---
title: Exit codes
description: vabc's stable exit-code table and the structured error envelope emitted on stderr.
sidebar:
  order: 3
---

Exit codes are a first-class contract: distinct, documented, and append-only. Scripts and agents
can rely on them without parsing human-readable text.

## Code table

| Code | Name | Notes |
| ---: | :--- | :---- |
| `0` | `ok` | Success. |
| `1` | `generic_error` | Unclassified error (internal or unrecognised upstream response). |
| `2` | `usage` | Bad arguments or parse error — check the command syntax. |
| `3` | `empty_results` | Request succeeded but matched nothing. |
| `4` | `auth_required` | **Reserved, never emitted.** vabc requires no authentication. |
| `5` | `not_found` | Unknown product code, store number, or geocode input. |
| `6` | `permission` | **Reserved, never emitted.** |
| `7` | `rate_limited` | Throttled or blocked by the upstream or the local circuit-breaker. |
| `8` | `retryable` | Transient upstream or network error — retry shortly. |
| `10` | `config_error` | Local configuration problem (e.g. bad env var value). |
| `12` | `mutation_blocked` | **Reserved, never emitted.** vabc is read-only; no mutations exist. |
| `13` | `input_required` | `--no-input` is set but interactive input is required. |
| `130` | `cancelled` | Interrupted by SIGINT. |

Codes 9, 11, and 14 are intentionally unassigned.

## Error envelope

Every error that vabc can classify is emitted to **stderr** only. Data always goes to **stdout**.

### Plain format (default)

```console
error: product 999999 not found
  code: NOT_FOUND
  fix:  list available products to find a valid id
```

### JSON format (`--json` or `--format json`)

```json
{
  "error": "product 999999 not found",
  "code": "NOT_FOUND",
  "remediation": "list available products to find a valid id"
}
```

The three fields are always present (remediation may be an empty string). JSON output has HTML
escaping disabled and uses two-space indentation.

## How codes are triggered

### `7` rate_limited

Emitted when the local throttle or circuit-breaker fires before a live call is made, and also
when the upstream itself returns a 429 or equivalent block. The error message includes a
retry-after hint when the upstream provides one.

By default vabc **fails fast** (exit 7) so agent loops never deadlock. Pass `--wait` to wait
out the backoff instead; `--max-wait` sets the ceiling (default `30s`).

```bash
vabc inventory check 010807 --near 22182 --wait --max-wait 60s
```

### `8` retryable

A transient upstream or network error that is not classified as rate limiting. Retry the same
command; if errors persist, run `vabc doctor --online` to check connectivity.

### `5` not_found

Returned for an unrecognised product code after a Coveo product lookup, an invalid store
number, or a geocode failure. Example:

```bash
vabc product get 000000   # exit 5 — no product with that code
vabc store get 9999       # exit 5 — no store with that number
```

### `13` input_required

Pass `--no-input` in scripts or agent contexts to guarantee vabc never blocks on a prompt. Any
path that would otherwise prompt the user exits 13 instead.

```bash
vabc inventory check 010807 --no-input   # exits 13 if location prompt is needed
```

### `130` cancelled

Standard POSIX convention: 128 + signal 2 (SIGINT). Emitted when the user presses Ctrl-C or
the process receives SIGINT.

## Machine-readable table

The live code table is also embedded in `vabc schema --json` under `exit_codes`. It is generated
from the same `errs` package the runtime uses, so it cannot drift from actual behaviour.

```bash
vabc schema --json | jq '.exit_codes'
```

See [Commands](/docs/reference/commands/) for full schema output, and
[Flags and environment variables](/docs/reference/flags-and-env/) for `--wait`, `--max-wait`,
and `--no-input`.
