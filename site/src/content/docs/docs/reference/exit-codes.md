---
title: Exit codes
description: vabc's stable, documented exit codes for scripting and agents.
sidebar:
  order: 2
---

Exit codes are a first-class contract: distinct, documented, and append-only. On error, a JSON
envelope `{ "error", "code", "remediation" }` is printed to **stderr** (under `--json`) and the
process exits with the mapped code.

| Code | Name | Meaning |
| ---: | --- | --- |
| `0` | ok | Success. |
| `1` | generic_error | Unclassified error. |
| `2` | usage | Bad arguments / parse error. |
| `3` | empty_results | The request succeeded but matched nothing. |
| `4` | auth_required | Reserved (vabc needs no auth). |
| `5` | not_found | Unknown product code, or an invalid store number. |
| `6` | permission | Reserved. |
| `7` | rate_limited | Throttled or blocked by the upstream / circuit breaker. |
| `8` | retryable | Transient upstream/network error — retry. |
| `10` | config_error | Local configuration problem. |
| `12` | mutation_blocked | Reserved (vabc is read-only). |
| `13` | input_required | `--no-input` was set but input was needed. |
| `130` | cancelled | Interrupted (SIGINT). |

The live table is also emitted by `vabc schema --json` under `exit_codes`, generated from the same
source the runtime uses — so it can never drift.
