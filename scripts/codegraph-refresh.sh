#!/usr/bin/env bash

set -euo pipefail

background=false

while [ "$#" -gt 0 ]; do
  case "$1" in
    -b|--background)
      background=true
      shift
      ;;
    -h|--help)
      cat <<'EOF'
Usage: scripts/codegraph-refresh.sh [--background]

Refresh the local CodeGraph index from the repo root.
EOF
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
cd "$repo_root"

if ! command -v codegraph >/dev/null 2>&1; then
  echo "CodeGraph CLI is not installed; refresh cannot run."
  exit 0
fi

refresh_index() {
  codegraph index --incremental
}

if [ "$background" = true ]; then
  refresh_index >/dev/null 2>&1 &
  printf 'CodeGraph incremental refresh started in the background.\n'
  exit 0
fi

refresh_index
