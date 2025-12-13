#!/usr/bin/env bash
set -euo pipefail

# Usage (example):
#   curl -fsSL https://raw.githubusercontent.com/you/your-repo/main/install.sh | REPO_URL=https://github.com/you/your-repo.git bash
#
# Env overrides:
#   REPO_URL   (required) git URL of this repo
#   BRANCH     (default: main)
#   PROFILE    (default: devheavy)
#   TAGS       (default: all)
#   VARS_FILE  (default: vars/all.yml)
#   TARGET_DIR (default: /tmp/<repo-name>-ansible)

REPO_URL="${REPO_URL:-https://github.com/Beta-Techno/anvil.git}"
BRANCH="${BRANCH:-main}"
PROFILE="${PROFILE:-devheavy}"
TAGS="${TAGS:-all}"
VARS_FILE="${VARS_FILE:-vars/all.yml}"

repo_name="$(basename "$REPO_URL" .git)"
TARGET_DIR="${TARGET_DIR:-$HOME/.cache/anvil-bootstrap}"

if ! command -v git >/dev/null 2>&1; then
  echo "[install] git not found; installing via apt…"
  sudo apt update
  sudo apt install -y git
fi

# Prime sudo credentials early if needed
if ! sudo -n true >/dev/null 2>&1; then
  echo "[install] sudo password required to proceed…"
  sudo -v
fi

# Ensure .cache directory exists
mkdir -p "$HOME/.cache"

# Clean up previous bootstrap directory (use sudo in case Ansible created root files)
if [[ -d "$TARGET_DIR" ]]; then
  sudo rm -rf "$TARGET_DIR"
fi

git clone --depth 1 --branch "$BRANCH" "$REPO_URL" "$TARGET_DIR"

cd "$TARGET_DIR"
if [[ ! -f "$VARS_FILE" && -f vars/all.example.yml ]]; then
  cp vars/all.example.yml "$VARS_FILE"
  echo "[install] Created $VARS_FILE from template. Edit it if needed before rerunning."
fi

TAGS="$TAGS" VARS_FILE="$VARS_FILE" PROFILE="$PROFILE" ./run.sh
