---
title: Installation
description: Every way to install vabc — Go, Homebrew, prebuilt binaries, or the install script.
sidebar:
  order: 2
---

`vabc` is a single static binary with no runtime dependencies. Pick the method that fits
your workflow; all four install the same binary.

## Go

```bash
go install github.com/rnwolfe/vabc/cmd/vabc@latest
```

Requires Go 1.25+. The binary lands in `$(go env GOPATH)/bin` — make sure that directory is on
your `PATH`. To pin a version, replace `@latest` with `@vX.Y.Z`.

## Homebrew

```bash
brew install rnwolfe/tap/vabc
```

Homebrew manages upgrades (`brew upgrade vabc`) and handles macOS Gatekeeper automatically.

## Install script

```bash
curl -fsSL https://vabc-cli.vercel.app/install.sh | sh
```

The script:

1. Detects your OS and CPU architecture (Linux/macOS, amd64/arm64).
2. Downloads the matching release archive and `checksums.txt` from GitHub Releases.
3. Verifies the SHA-256 checksum before touching anything.
4. Installs the binary to `~/.local/bin` (or `$VABC_INSTALL_DIR` if set).

Two environment variables let you control the install:

| Variable | Default | Purpose |
|---|---|---|
| `VABC_VERSION` | latest release | Version tag to install, e.g. `v0.1.0` |
| `VABC_INSTALL_DIR` | `~/.local/bin` | Directory to place the `vabc` binary |

```bash
# Pin to a specific version and install to /usr/local/bin
VABC_VERSION=v0.1.0 VABC_INSTALL_DIR=/usr/local/bin \
  curl -fsSL https://vabc-cli.vercel.app/install.sh | sh
```

If `~/.local/bin` is not on your `PATH`, the script prints a reminder:

```console
! /home/you/.local/bin is not on your PATH — add it:
    export PATH="/home/you/.local/bin:$PATH"
```

Add that line to your shell's rc file (`~/.bashrc`, `~/.zshrc`, etc.).

## Prebuilt binaries

Download an archive directly from the
[GitHub Releases page](https://github.com/rnwolfe/vabc/releases) and verify it yourself
before installing.

Supported platforms:

| OS | Arch | Archive |
|---|---|---|
| Linux | amd64, arm64 | `.tar.gz` |
| macOS | amd64, arm64 | `.tar.gz` |
| Windows | amd64, arm64 | `.zip` |

```bash
# Example: Linux amd64, version v0.1.0
VERSION=v0.1.0
ARCHIVE="vabc_linux_amd64.tar.gz"
BASE="https://github.com/rnwolfe/vabc/releases/download/$VERSION"

curl -fsSL -o "$ARCHIVE"       "$BASE/$ARCHIVE"
curl -fsSL -o checksums.txt    "$BASE/checksums.txt"

# Verify
grep " $ARCHIVE$" checksums.txt | sha256sum --check

# Extract and install
tar -xzf "$ARCHIVE"
install -m 0755 vabc ~/.local/bin/vabc
```

On macOS, use `shasum -a 256` instead of `sha256sum`:

```bash
grep " $ARCHIVE$" checksums.txt | shasum -a 256 --check
```

On Windows, extract the `.zip`, place `vabc.exe` somewhere on your `%PATH%`, and verify with
PowerShell:

```console
> $hash = (Get-FileHash vabc_windows_amd64.zip -Algorithm SHA256).Hash.ToLower()
> Select-String "vabc_windows_amd64.zip" checksums.txt
```

## Verify the installation

Run these two commands after any install method:

```bash
vabc version
```

```json
{
  "version": "v0.1.0"
}
```

```bash
vabc doctor
```

```json
{
  "ok": true,
  "checks": [
    { "name": "auth",               "ok": true, "detail": "no authentication required" },
    { "name": "inventory_endpoint", "ok": true, "detail": "configured: https://www.abc.virginia.gov (pass --online to probe)" },
    { "name": "stores_endpoint",    "ok": true, "detail": "configured: ArcGIS FeatureServer (pass --online to probe)" },
    { "name": "product_search",     "ok": true, "detail": "configured: Coveo web catalog (pass --online to probe)" }
  ]
}
```

`vabc doctor` runs offline checks only by default. Pass `--online` to make live network
requests to the inventory API, the ArcGIS store locator, and the Coveo product search:

```bash
vabc doctor --online
```

A check that fails with `unreachable` or `throttled/blocked` means the endpoint isn't
reachable from your machine right now. The endpoints are public and unauthenticated; no
credentials are involved.

Once `vabc version` returns cleanly, head to [Getting started](/docs/start/getting-started/)
to run your first search.
