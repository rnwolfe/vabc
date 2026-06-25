---
title: Getting started
description: Install vabc and run your first product search and inventory check in under a minute.
sidebar:
  order: 1
---

`vabc` is a read-only command-line tool (and Go library) for Virginia ABC. Search the product
catalog, check live per-store and warehouse inventory, locate stores, and check
limited-availability ("lottery") releases. **No login, no API key.**

## Install

```bash
# Go (any platform)
go install github.com/rnwolfe/vabc/cmd/vabc@latest

# Homebrew
brew install rnwolfe/tap/vabc
```

Or download a prebuilt binary from the [Releases page](https://github.com/rnwolfe/vabc/releases).
See [Installation](/docs/start/install/) for all options.

## First commands

```bash
# Search the live catalog
vabc product search bourbon

# Find a specific bottle (even allocated / online-only ones)
vabc product search oftd

# Where can I get it near me? (ZIP, address, or lat,lng)
vabc inventory check 953714 --near 22182

# Statewide warehouse stock
vabc inventory warehouse 010807
```

Product codes are 6-digit (e.g. `010807`). A `product search` result includes the code, so you
can feed it straight into `inventory check`.

## JSON for scripts and agents

Add `--json` to any read. Data goes to **stdout**; notes and errors go to **stderr**.

```bash
vabc --json product search rum --select productCode,name | jq '.[0]'
vabc --json inventory check 010807 --near 22182 | jq '.store.quantity'
```

## Next steps

- [Find a bottle](/docs/guides/find-a-bottle/) — the full hunt workflow.
- [Using vabc with agents](/docs/guides/for-agents/) — the contract that makes it LLM-safe.
- [Command reference](/docs/reference/commands/) and [exit codes](/docs/reference/exit-codes/).
