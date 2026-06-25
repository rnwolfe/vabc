#!/bin/sh
# vabc installer — downloads a release archive, verifies its SHA-256, installs the binary.
#
#   curl -fsSL https://vabc.sh/install.sh | sh
#
# Env:
#   VABC_VERSION       version tag to install (default: latest)
#   VABC_INSTALL_DIR   install dir (default: $HOME/.local/bin)
set -eu

REPO="rnwolfe/vabc"
BIN="vabc"

err() { printf 'error: %s\n' "$1" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

main() {
  # --- platform ---
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  case "$os" in
    linux|darwin) ;;
    *) err "unsupported OS: $os (try a prebuilt binary from https://github.com/$REPO/releases)" ;;
  esac
  case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) err "unsupported arch: $arch" ;;
  esac

  # --- downloader ---
  if have curl; then DL="curl -fsSL"; DLO="curl -fsSL -o";
  elif have wget; then DL="wget -qO-"; DLO="wget -qO";
  else err "need curl or wget"; fi

  # --- version ---
  version="${VABC_VERSION:-}"
  if [ -z "$version" ]; then
    version=$($DL "https://api.github.com/repos/$REPO/releases/latest" \
      | grep '"tag_name"' | head -1 | cut -d'"' -f4)
    [ -n "$version" ] || err "could not resolve latest version; set VABC_VERSION"
  fi

  archive="${BIN}_${os}_${arch}.tar.gz"
  base="https://github.com/$REPO/releases/download/$version"
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  printf 'Downloading %s %s (%s/%s)...\n' "$BIN" "$version" "$os" "$arch" >&2
  $DLO "$tmp/$archive" "$base/$archive" || err "download failed: $base/$archive"
  $DLO "$tmp/checksums.txt" "$base/checksums.txt" || err "could not fetch checksums.txt"

  # --- verify SHA-256 ---
  expected=$(grep " $archive\$" "$tmp/checksums.txt" | awk '{print $1}')
  [ -n "$expected" ] || err "no checksum for $archive"
  if have sha256sum; then actual=$(sha256sum "$tmp/$archive" | awk '{print $1}');
  elif have shasum; then actual=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}');
  else err "need sha256sum or shasum to verify the download"; fi
  [ "$actual" = "$expected" ] || err "checksum mismatch (expected $expected, got $actual)"

  # --- install ---
  tar -xzf "$tmp/$archive" -C "$tmp" || err "extract failed"
  [ -f "$tmp/$BIN" ] || err "binary not found in archive"
  dir="${VABC_INSTALL_DIR:-$HOME/.local/bin}"
  mkdir -p "$dir"
  install -m 0755 "$tmp/$BIN" "$dir/$BIN" 2>/dev/null || { cp "$tmp/$BIN" "$dir/$BIN"; chmod 0755 "$dir/$BIN"; }

  printf '\n✓ installed %s to %s/%s\n' "$BIN" "$dir" "$BIN" >&2
  case ":$PATH:" in
    *":$dir:"*) ;;
    *) printf '! %s is not on your PATH — add it:\n    export PATH="%s:$PATH"\n' "$dir" "$dir" >&2 ;;
  esac
  "$dir/$BIN" version 2>/dev/null || true
}

main "$@"
