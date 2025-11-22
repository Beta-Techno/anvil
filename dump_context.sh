#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COPY=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    -c|--copy) COPY=1 ;;
    -h|--help)
      cat <<'EOF'
Usage: ./dump_context.sh [--copy]
  --copy  Also copy the output to the system clipboard (requires wl-copy, xclip, xsel, or pbcopy).
EOF
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
  shift
done

detect_clip() {
  if [[ -n "${WAYLAND_DISPLAY:-}" ]] && command -v wl-copy >/dev/null 2>&1; then
    echo "wl-copy"
    return 0
  fi
  if [[ -n "${DISPLAY:-}" ]]; then
    if command -v xclip >/dev/null 2>&1; then
      echo "xclip -selection clipboard"
      return 0
    fi
    if command -v xsel >/dev/null 2>&1; then
      echo "xsel -ib"
      return 0
    fi
  fi
  if command -v pbcopy >/dev/null 2>&1; then
    echo "pbcopy"
    return 0
  fi
  return 1
}

emit_context() {
  find "$ROOT_DIR" -type f ! -path '*/.git/*' -print0 \
    | sort -z \
    | while IFS= read -r -d '' file; do
        [[ "$file" == *.txt ]] && continue
        rel_path="${file#"$ROOT_DIR/"}"
        echo "===== ${rel_path} ====="
        cat "$file"
        echo
      done
}

if [[ $COPY -eq 1 ]]; then
  tmp="$(mktemp)"
  emit_context | tee "$tmp"
  if clip_cmd="$(detect_clip)"; then
    # shellcheck disable=SC2086
    if $clip_cmd < "$tmp"; then
      echo "(copied to clipboard via $clip_cmd)"
    else
      echo "Failed to copy to clipboard using: $clip_cmd" >&2
      exit 1
    fi
  else
    echo "Clipboard tool not found (install wl-copy, xclip, xsel, or pbcopy)." >&2
    exit 1
  fi
  rm -f "$tmp"
else
  emit_context
fi
