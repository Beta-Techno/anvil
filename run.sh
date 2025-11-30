#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VARS_FILE="${VARS_FILE:-$ROOT_DIR/vars/all.yml}"
PROFILE="${PROFILE:-devheavy}"
TAGS="${TAGS:-${ANSIBLE_TAGS:-all}}"

if [[ ! -f "$VARS_FILE" ]]; then
  echo "Vars file not found at $VARS_FILE"
  echo "Copy and edit $ROOT_DIR/vars/all.example.yml → $VARS_FILE first."
  exit 1
fi

export ANSIBLE_TAGS="$TAGS"
# Always include vars file, then append any user-provided args
export ANSIBLE_ARGS="-e @$VARS_FILE ${ANSIBLE_ARGS:-}"

cd "$ROOT_DIR"
./bootstrap.sh "$PROFILE"
