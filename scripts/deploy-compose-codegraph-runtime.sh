#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="${repo_root}/deploy/compose/codegraph-runtime/docker-compose.yml"
env_file="${repo_root}/deploy/compose/codegraph-runtime/.env"

if [[ -f "${env_file}" ]]; then
  # shellcheck disable=SC1090
  set -a
  source "${env_file}"
  set +a
fi

require_var() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "${name} is required for the CodeGraph Compose path" >&2
    exit 1
  fi
}

export KG_RUNTIME_PROFILE="qdrant-memgraph"

if [[ "${EMBEDDING_PROVIDER:-}" != "http" ]]; then
  echo "EMBEDDING_PROVIDER must be http for the CodeGraph Compose path" >&2
  exit 1
fi

require_var EMBEDDING_URL
require_var EMBEDDING_MODEL
require_var EMBEDDING_API_KEY

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for the Compose deployment path" >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose v2 is required for the Compose deployment path" >&2
  exit 1
fi

echo "starting CodeGraph Compose stack from ${compose_file}"
echo "using runtime profile ${KG_RUNTIME_PROFILE} (memgraph/qdrant)"
docker compose -f "${compose_file}" up -d --build
