#!/usr/bin/env bash
set -euo pipefail

# One-command runner: installs Ansible (pipx preferred), maps a profile to tags, and runs the local play.
# Usage:
#   ./bootstrap.sh                # defaults to "server" profile
#   ./bootstrap.sh workstation    # server + GUI bits
#   ./bootstrap.sh devheavy       # workstation + language stacks + LazyVim
#   ANSIBLE_TAGS="base,docker" ./bootstrap.sh  # override tags directly
# Env knobs:
#   ANSIBLE_TAGS        Comma list to override profile tags
#   ANSIBLE_EXTRA_VARS  Extra vars string, e.g. 'git_user_email=you@example.com'
#   ANSIBLE_ARGS        Additional flags, e.g. "-vv"

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
profile="${1:-server}"
tags="${ANSIBLE_TAGS:-}"
ansible_args="${ANSIBLE_ARGS:-}"

profile_tags() {
  case "$1" in
    server) echo "base,docker,tailscale,git" ;;
    workstation) echo "base,docker,tailscale,git,terminals,fonts,flatpak_snap" ;;
    devheavy) echo "base,docker,tailscale,git,langs,lazyvim,terminals,fonts,flatpak_snap" ;;
    minimal) echo "base,docker" ;;
    *) echo "" ;;
  esac
}

if [[ -z "$tags" ]]; then
  tags="$(profile_tags "$profile")"
  if [[ -z "$tags" ]]; then
    # Treat unknown profile as literal tags (e.g., "base,docker")
    tags="$profile"
  fi
fi

ensure_ansible() {
  if command -v ansible-playbook >/dev/null 2>&1; then
    return
  fi

  echo "[bootstrap] Installing ansible prerequisites via apt…"
  sudo apt update
  sudo apt install -y python3 python3-venv python3-pip python3-apt pipx

  python3 -m pipx ensurepath || true
  export PATH="$HOME/.local/bin:$PATH"

  if ! command -v ansible-playbook >/dev/null 2>&1; then
    echo "[bootstrap] Installing ansible-core via pipx…"
    if ! python3 -m pipx install --include-deps ansible-core; then
      echo "[bootstrap] pipx install failed; falling back to apt ansible package."
      sudo apt install -y ansible
    fi
  fi
}

run_play() {
  cd "$here"
  local cmd=(ansible-playbook -i localhost, -c local playbook.yml --tags "$tags")
  if [[ -n "${ANSIBLE_EXTRA_VARS:-}" ]]; then
    cmd+=(--extra-vars "$ANSIBLE_EXTRA_VARS")
  fi
  if [[ -n "$ansible_args" ]]; then
    # shellcheck disable=SC2206
    extra_args=($ansible_args)
    cmd+=("${extra_args[@]}")
  fi

  echo "[bootstrap] Running: ${cmd[*]}"
  "${cmd[@]}"
}

ensure_ansible
run_play
