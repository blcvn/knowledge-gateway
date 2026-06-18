#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="${repo_root}/deploy/compose/docker-compose.yml"

# shellcheck source=runtime-profile.sh
source "${repo_root}/scripts/runtime-profile.sh"

if [[ -z "${KG_RUNTIME_PROFILE:-}" ]]; then
  echo "KG_RUNTIME_PROFILE is required for the Compose deployment path" >&2
  exit 1
fi

if ! kg_runtime_profile_defaults "${KG_RUNTIME_PROFILE}"; then
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for the Compose deployment path" >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose v2 is required for the Compose deployment path" >&2
  exit 1
fi

echo "starting Compose stack from ${compose_file}"
echo "using runtime profile ${KG_RUNTIME_PROFILE} (${GRAPH_ADAPTER}/${VECTOR_ADAPTER})"
docker compose -f "${compose_file}" up -d --build
