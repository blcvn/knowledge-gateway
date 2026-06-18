#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bin_dir="${repo_root}/bin"
binary="${bin_dir}/kg-service"

required_vars=(
  KG_POSTGRES_HOST
  KG_POSTGRES_PASSWORD
  KG_REDIS_HOST
)

for var in "${required_vars[@]}"; do
  if [[ -z "${!var:-}" ]]; then
    echo "${var} is required for the VM deployment path" >&2
    exit 1
  fi
done

mkdir -p "${bin_dir}"

if [[ ! -x "${binary}" || "${KG_REBUILD_BINARY:-0}" == "1" ]]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "go is required to build the VM deployment binary" >&2
    exit 1
  fi

  echo "building ${binary}"
  (cd "${repo_root}" && go build -o "${binary}" .)
fi

echo "starting kg-service from ${binary}"
exec "${binary}"
