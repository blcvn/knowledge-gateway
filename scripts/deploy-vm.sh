#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
bin_dir="${repo_root}/bin"
binary="${bin_dir}/kg-service"

# shellcheck source=runtime-profile.sh
source "${repo_root}/scripts/runtime-profile.sh"

if [[ -z "${KG_RUNTIME_PROFILE:-}" ]]; then
  echo "KG_RUNTIME_PROFILE is required for the VM deployment path" >&2
  exit 1
fi

if ! kg_runtime_profile_defaults "${KG_RUNTIME_PROFILE}"; then
  exit 1
fi

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
echo "using runtime profile ${KG_RUNTIME_PROFILE} (${GRAPH_ADAPTER}/${VECTOR_ADAPTER})"
exec "${binary}"
