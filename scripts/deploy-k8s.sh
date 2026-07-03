#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
k8s_dir="${repo_root}/deploy/k8s"
namespace="${KG_NAMESPACE:-kg-service}"
image="${KG_IMAGE:-kg-service:local}"

# shellcheck source=runtime-profile.sh
source "${repo_root}/scripts/runtime-profile.sh"

if [[ -z "${KG_RUNTIME_PROFILE:-}" ]]; then
  echo "KG_RUNTIME_PROFILE is required for the Kubernetes deployment path" >&2
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
    echo "${var} is required for the Kubernetes deployment path" >&2
    exit 1
  fi
done

kg_postgres_port="${KG_POSTGRES_PORT:-5432}"
kg_postgres_user="${KG_POSTGRES_USER:-postgres}"
kg_postgres_database="${KG_POSTGRES_DATABASE:-kg_service}"
kg_postgres_sslmode="${KG_POSTGRES_SSLMODE:-disable}"
kg_redis_port="${KG_REDIS_PORT:-6379}"
kg_redis_db="${KG_REDIS_DB:-0}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl is required for the Kubernetes deployment path" >&2
  exit 1
fi

escape_sed_replacement() {
  printf '%s' "$1" | sed -e 's/[\\&|]/\\&/g'
}

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

rendered="${tmp_dir}/kg-service.yaml"
rendered_namespace="${tmp_dir}/namespace.yaml"
rendered_service="${tmp_dir}/service.yaml"
sed \
  -e "s|__KG_NAMESPACE__|$(escape_sed_replacement "${namespace}")|g" \
  -e "s|__KG_IMAGE__|$(escape_sed_replacement "${image}")|g" \
  -e "s|__KG_POSTGRES_HOST__|$(escape_sed_replacement "${KG_POSTGRES_HOST}")|g" \
  -e "s|__KG_POSTGRES_PORT__|$(escape_sed_replacement "${kg_postgres_port}")|g" \
  -e "s|__KG_POSTGRES_USER__|$(escape_sed_replacement "${kg_postgres_user}")|g" \
  -e "s|__KG_POSTGRES_PASSWORD__|$(escape_sed_replacement "${KG_POSTGRES_PASSWORD}")|g" \
  -e "s|__KG_POSTGRES_DATABASE__|$(escape_sed_replacement "${kg_postgres_database}")|g" \
  -e "s|__KG_POSTGRES_SSLMODE__|$(escape_sed_replacement "${kg_postgres_sslmode}")|g" \
  -e "s|__KG_REDIS_HOST__|$(escape_sed_replacement "${KG_REDIS_HOST}")|g" \
  -e "s|__KG_REDIS_PORT__|$(escape_sed_replacement "${kg_redis_port}")|g" \
  -e "s|__KG_REDIS_DB__|$(escape_sed_replacement "${kg_redis_db}")|g" \
  -e "s|__KG_RUNTIME_PROFILE__|$(escape_sed_replacement "${KG_RUNTIME_PROFILE}")|g" \
  -e "s|__KG_GRAPH_ADAPTER__|$(escape_sed_replacement "${GRAPH_ADAPTER}")|g" \
  -e "s|__KG_GRAPH_ENDPOINT__|$(escape_sed_replacement "${KG_GRAPH_ENDPOINT:-}")|g" \
  -e "s|__KG_GRAPH_DATABASE__|$(escape_sed_replacement "${KG_GRAPH_DATABASE:-}")|g" \
  -e "s|__KG_VECTOR_ADAPTER__|$(escape_sed_replacement "${VECTOR_ADAPTER}")|g" \
  -e "s|__KG_VECTOR_ENDPOINT__|$(escape_sed_replacement "${KG_VECTOR_ENDPOINT:-}")|g" \
  -e "s|__KG_VECTOR_COLLECTION__|$(escape_sed_replacement "${KG_VECTOR_COLLECTION:-kg_vectors}")|g" \
  -e "s|__KG_FTS_ADAPTER__|$(escape_sed_replacement "${FTS_ADAPTER}")|g" \
  -e "s|__KG_EMBEDDING_PROVIDER__|$(escape_sed_replacement "${EMBEDDING_PROVIDER}")|g" \
  "${k8s_dir}/deployment.yaml" > "${rendered}"

sed \
  -e "s|__KG_NAMESPACE__|$(escape_sed_replacement "${namespace}")|g" \
  "${k8s_dir}/namespace.yaml" > "${rendered_namespace}"

sed \
  -e "s|__KG_NAMESPACE__|$(escape_sed_replacement "${namespace}")|g" \
  "${k8s_dir}/service.yaml" > "${rendered_service}"

kubectl apply -f "${rendered_namespace}"
kubectl apply -f "${rendered}"
kubectl apply -f "${rendered_service}"

kubectl -n "${namespace}" rollout status deployment/kg-service --timeout=180s
