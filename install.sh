#!/usr/bin/env bash
# Install or update the Anvil CLI binary.
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"
TARGET="$INSTALL_DIR/anvil"

uname_s="$(uname -s)"
uname_m="$(uname -m)"
case "$uname_s" in
  Linux)   goos=linux ;;
  Darwin)  goos=darwin ;;
  *)
    echo "[install] Unsupported OS: $uname_s" >&2
    exit 1
    ;;
esac

case "$uname_m" in
  x86_64|amd64) goarch=amd64 ;;
  arm64|aarch64)
    goarch=arm64
    ;;
  *)
    echo "[install] Unsupported architecture: $uname_m" >&2
    exit 1
    ;;
esac

asset="anvil-${goos}-${goarch}"
if [[ "$VERSION" == "latest" ]]; then
  download_url="https://github.com/Beta-Techno/anvil/releases/latest/download/${asset}"
  version_label="latest"
else
  version_tag="$VERSION"
  if [[ "$version_tag" != v* ]]; then
    version_tag="v${version_tag}"
  fi
  download_url="https://github.com/Beta-Techno/anvil/releases/download/${version_tag}/${asset}"
  version_label="$version_tag"
fi

echo "[install] Installing Anvil CLI (${version_label}) for ${goos}/${goarch} -> ${TARGET}"

if [[ $EUID -ne 0 && "$INSTALL_DIR" == /usr/local/* ]]; then
  if ! command -v sudo >/dev/null 2>&1; then
    echo "[install] sudo required to write ${INSTALL_DIR}" >&2
    exit 1
  fi
  sudo_prefix="sudo"
else
  sudo_prefix=""
fi

$sudo_prefix mkdir -p "$INSTALL_DIR"

tmpfile=$(mktemp)
trap 'rm -f "$tmpfile"' EXIT

if ! curl -fsSL "$download_url" -o "$tmpfile"; then
  echo "[install] Failed to download $download_url" >&2
  exit 1
fi
chmod +x "$tmpfile"

$sudo_prefix mv "$tmpfile" "$TARGET"
trap - EXIT
$sudo_prefix chown root:root "$TARGET" 2>/dev/null || true

if command -v "$TARGET" >/dev/null 2>&1; then
  echo "[install] Installed version: $($TARGET version 2>/dev/null || echo unknown)"
fi

echo "[install] Done. Run: anvil up"
