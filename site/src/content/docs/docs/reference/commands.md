---
title: Command reference
description: Every vabc command, its arguments, and key flags.
sidebar:
  order: 1
---

Global flags apply to every command: `--json` / `--format json|plain|tsv`, `--limit N`
(default 50), `--select a,b.c`, `--no-color`, `--no-input`, `--wait` / `--max-wait`. The full,
machine-readable grammar is always available via `vabc schema --json`.

## product

| Command | Description |
| --- | --- |
| `product search <query>` | Live keyword search over the web catalog. Flags: `--type`, `--allocated`. |
| `product get <productCode>` | Look up one product by 6-digit code. |

```bash
vabc product search oftd --select productCode,name
vabc product get 010807
```

## inventory

| Command | Description |
| --- | --- |
| `inventory check <code>` | Per-store availability + nearby stores by distance. Requires `--store <n>` **or** `--near <ZIP\|address\|lat,lng>`. |
| `inventory warehouse <code>` | Statewide central-warehouse stock. |

```bash
vabc inventory check 010807 --near 22182
vabc inventory warehouse 010807
```

## store

| Command | Description |
| --- | --- |
| `store list` | All ~394 Virginia ABC stores. |
| `store get <storeNumber>` | One store's details. |
| `store near <ZIP\|address\|lat,lng>` | Nearest stores, ranked by distance. |

## lottery

| Command | Description |
| --- | --- |
| `lottery check <code>` | Active limited-availability / allocated events for a product. |

## Utility

| Command | Description |
| --- | --- |
| `auth status` | Reports that no authentication is required. |
| `doctor [--online]` | Diagnose setup; `--online` probes live endpoints. |
| `schema --json` | Machine-readable command tree + exit codes. |
| `agent` | Print the embedded agent guide (SKILL.md). |
| `version` | Print the version. |
