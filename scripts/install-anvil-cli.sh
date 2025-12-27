#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_URL="${BINARY_URL:-https://github.com/Beta-Techno/anvil/releases/latest/download/anvil-linux-amd64}"
TARGET="$INSTALL_DIR/anvil"

if [[ $EUID -ne 0 ]]; then
  sudo="sudo"
else
  sudo=""
fi

$sudo mkdir -p "$INSTALL_DIR"

curl -fsSL "$BINARY_URL" -o /tmp/anvil-cli
chmod +x /tmp/anvil-cli
$sudo mv /tmp/anvil-cli "$TARGET"

$sudo chown root:root "$TARGET"

cat <<'EOS'
Anvil CLI installed. Run:
  anvil up
EOS
