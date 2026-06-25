---
title: Using vabc with agents
description: The agent-CLI contract — JSON, stable exit codes, embedded self-description, read-only safety, and injection fencing.
sidebar:
  order: 2
---

`vabc` is built to be driven by LLM agents, not just typed by humans. It follows the
[Agent CLI Guidelines](https://aclig.dev/) at the Full conformance level.

## Self-description (no docs needed)

The binary describes itself, so an agent that has `vabc` has everything:

```bash
vabc agent          # prints the embedded usage contract (SKILL.md)
vabc schema --json  # full command tree, flags, and the exit-code table
```

## Structured output

```bash
vabc --json product search bourbon --limit 5 --select productCode,name
```

- JSON to **stdout**, human notes/errors to **stderr** — pipe stdout straight into `jq`.
- `--limit` (default 50) bounds output; `--select a,b.c` projects fields.
- `--format json|plain|tsv` for other shapes.

## Errors are structured

On failure, stderr carries a JSON envelope and the process exits with a stable, mapped code:

```json
{ "error": "product 000000 not found", "code": "NOT_FOUND", "remediation": "check the product code" }
```

See the [exit-code table](/docs/reference/exit-codes/).

## Safe by default

- **Read-only.** No command changes Virginia ABC state. `--allow-mutations` exists for contract
  uniformity but gates nothing.
- **No credentials.** Nothing to leak.
- **Prompt-injection fenced.** Free text from the target (lottery event titles) is wrapped as
  `⟦UNTRUSTED⟧ … ⟦/UNTRUSTED⟧` by default. Disable with `--no-wrap-untrusted`.
- **Polite.** A persistent, cross-process throttle/circuit-breaker keeps a fresh-process-per-call
  agent from hammering the undocumented endpoints. On a block, live calls fail fast (exit `7`)
  with a retry hint; pass `--wait` to wait it out.

## As a Go library

```go
import "github.com/rnwolfe/vabc"

c := vabc.NewClient()
res, err := c.StoreNearby(ctx, 219, "010807")
```

`go get github.com/rnwolfe/vabc` pulls a tiny HTTP+JSON client — the same `Client` the CLI uses.
