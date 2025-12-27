#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VARS_FILE="${VARS_FILE:-$ROOT_DIR/vars/all.yml}"
PROFILE="${PROFILE:-devheavy}"
TAGS="${TAGS:-${ANSIBLE_TAGS:-all}}"
PERSONA="${PERSONA:-dev}"
PERSONA_FILE="${PERSONA_FILE:-$ROOT_DIR/vars/personas/${PERSONA}.yml}"
KEY_VARS_FILE="${KEY_BUNDLE_VARS_FILE:-$HOME/.config/anvil/key-bundle.yml}"
BUNDLE_FETCH_POLICY="${KEY_BUNDLE_FETCH:-auto}"
BUNDLE_URL_DEFAULT="${KEY_BUNDLE_URL:-https://raw.githubusercontent.com/Beta-Techno/key/main/bundles/default.sops.yaml}"
BUNDLE_AGE_FILE="${KEY_AGE_KEY_FILE:-$HOME/.config/anvil/age.key}"

if [[ ! -f "$VARS_FILE" ]]; then
  echo "Vars file not found at $VARS_FILE"
  echo "Copy and edit $ROOT_DIR/vars/all.example.yml → $VARS_FILE first."
  exit 1
fi

if [[ ! -f "$PERSONA_FILE" ]]; then
  echo "Persona file not found for PERSONA=$PERSONA"
  echo "Expected file at $PERSONA_FILE or override via PERSONA_FILE env var."
  exit 1
fi

export PERSONA

ensure_key_bundle() {
  if [[ -f "$KEY_VARS_FILE" ]]; then
    return
  fi
  if [[ "$BUNDLE_FETCH_POLICY" == "skip" ]]; then
    echo "[key] KEY_BUNDLE_FETCH=skip set and no bundle present; continuing without key bundle vars."
    return
  fi
  if [[ ! -x "$ROOT_DIR/scripts/unlock-key-bundle.sh" ]]; then
    echo "[key] unlock helper missing at scripts/unlock-key-bundle.sh; continuing without key bundle vars."
    return
  fi
  echo "[key] No decrypted bundle found at $KEY_VARS_FILE"
  echo "[key] Running unlock helper (Ctrl+C to abort)..."
  KEY_BUNDLE_URL="$BUNDLE_URL_DEFAULT" \
  KEY_OUTPUT_FILE="$KEY_VARS_FILE" \
  KEY_AGE_KEY_FILE="$BUNDLE_AGE_FILE" \
    "$ROOT_DIR/scripts/unlock-key-bundle.sh"
}

ensure_key_bundle

export ANSIBLE_TAGS="$TAGS"
# Always include vars file, then append any user-provided args
KEY_ARG=""
if [[ -f "$KEY_VARS_FILE" ]]; then
  KEY_ARG="-e @$KEY_VARS_FILE"
fi

export ANSIBLE_ARGS="-e @$VARS_FILE -e @$PERSONA_FILE $KEY_ARG ${ANSIBLE_ARGS:-}"

cd "$ROOT_DIR"
./bootstrap.sh "$PROFILE"
