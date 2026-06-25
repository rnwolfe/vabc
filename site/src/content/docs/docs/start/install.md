---
title: Installation
description: Every way to install vabc — Go, Homebrew, prebuilt binaries, or the install script.
sidebar:
  order: 2
---

`vabc` is a single static binary with no runtime dependencies.

## Go

```bash
go install github.com/rnwolfe/vabc/cmd/vabc@latest
```

Installs to `$(go env GOPATH)/bin`. Requires Go 1.25+. Version is reported via build info, so
`vabc version` works for `@vX.Y.Z` installs.

## Homebrew

```bash
brew install rnwolfe/tap/vabc
```

## Prebuilt binaries

Download a checksummed archive for your OS/arch from the
[Releases page](https://github.com/rnwolfe/vabc/releases), verify it against `checksums.txt`, and
put `vabc` on your `PATH`.

## Install script

```bash
curl -fsSL https://vabc-cli.vercel.app/install.sh | sh
```

The script detects your OS/arch, downloads the matching release archive **and** `checksums.txt`,
**verifies the SHA-256 before installing**, and installs to `~/.local/bin` (override with
`VABC_INSTALL_DIR`). Pin a version with `VABC_VERSION=v0.1.0`.

## Verify

```bash
vabc version
vabc doctor          # offline checks
vabc doctor --online # also probes the live endpoints
```
