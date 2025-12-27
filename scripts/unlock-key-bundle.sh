#!/usr/bin/env bash
set -euo pipefail

# Downloads + decrypts an encrypted bootstrap bundle (SOPS + age) so Anvil
# can load your Git credentials, tokens, and lockbox age key before provisioning.
#
# Usage:
#   ./scripts/unlock-key-bundle.sh \
#     --bundle-url https://raw.githubusercontent.com/your/key/bundles/default.sops.yaml \
#     --output ~/.config/anvil/key-bundle.yml
#
# Environment overrides:
#   KEY_BUNDLE_URL, KEY_OUTPUT_FILE, KEY_AGE_KEY_FILE, SOPS_BINARY
#   SOPS_AGE_KEY (inline private key) or SOPS_AGE_KEY_FILE (existing path)

BUNDLE_URL="${KEY_BUNDLE_URL:-https://raw.githubusercontent.com/Beta-Techno/key/main/bundles/default.sops.yaml}"
OUTPUT_FILE="${KEY_OUTPUT_FILE:-$HOME/.config/anvil/key-bundle.yml}"
AGE_KEY_FILE="${KEY_AGE_KEY_FILE:-$HOME/.config/anvil/age.key}"
SOPS_BIN="${SOPS_BINARY:-sops}"

usage() {
  cat <<'USAGE'
Options:
  --bundle-url URL     Source of the encrypted .sops.yaml bundle
  --output FILE        Where to write decrypted YAML (default ~/.config/anvil/key-bundle.yml)
  --age-key-file FILE  Where to store the age private key (default ~/.config/anvil/age.key)
  -h, --help           Show this message
USAGE
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bundle-url)
      BUNDLE_URL="$2"; shift 2 ;;
    --output)
      OUTPUT_FILE="$2"; shift 2 ;;
    --age-key-file)
      AGE_KEY_FILE="$2"; shift 2 ;;
    -h|--help)
      usage ;;
    *)
      echo "Unknown option: $1" >&2
      usage ;;
  esac
 done

ensure_sops() {
  if command -v "$SOPS_BIN" >/dev/null 2>&1; then
    return
  fi
  echo "[unlock] sops not found; attempting to install via apt..."
  if ! command -v sudo >/dev/null 2>&1; then
    echo "sudo not available; please install sops manually" >&2
    exit 1
  fi
  sudo apt-get update -y
  sudo apt-get install -y sops
}

prompt_age_key() {
  mkdir -p "$(dirname "$AGE_KEY_FILE")"
  chmod 700 "$(dirname "$AGE_KEY_FILE")"

  if [[ -n "${SOPS_AGE_KEY:-}" ]]; then
    printf '%s' "$SOPS_AGE_KEY" > "$AGE_KEY_FILE"
  elif [[ -n "${SOPS_AGE_KEY_FILE:-}" && -f "$SOPS_AGE_KEY_FILE" ]]; then
    cp "$SOPS_AGE_KEY_FILE" "$AGE_KEY_FILE"
  elif [[ -f "$AGE_KEY_FILE" ]]; then
    echo "[unlock] Using existing age key at $AGE_KEY_FILE"
  else
    echo "Paste your age secret key (end with EOF/Ctrl+D):"
    cat > "$AGE_KEY_FILE"
  fi
  chmod 600 "$AGE_KEY_FILE"
  export SOPS_AGE_KEY_FILE="$AGE_KEY_FILE"
}

main() {
  ensure_sops
  prompt_age_key

  tmp_bundle="$(mktemp)"
  echo "[unlock] Downloading bundle from $BUNDLE_URL"
  curl -fsSL "$BUNDLE_URL" -o "$tmp_bundle"

  mkdir -p "$(dirname "$OUTPUT_FILE")"
  chmod 700 "$(dirname "$OUTPUT_FILE")"

  echo "[unlock] Decrypting to $OUTPUT_FILE"
  if ! "$SOPS_BIN" --decrypt "$tmp_bundle" > "$OUTPUT_FILE"; then
    echo "[unlock] failed to decrypt bundle" >&2
    rm -f "$tmp_bundle"
    exit 1
  fi
  rm -f "$tmp_bundle"
  chmod 600 "$OUTPUT_FILE"
  echo "[unlock] Success. Rerun ./run.sh (set KEY_BUNDLE_VARS_FILE=$OUTPUT_FILE if you used a custom path)."
}

main "$@"
