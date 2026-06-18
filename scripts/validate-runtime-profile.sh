#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

base_url="${KG_BASE_URL:-}"
api_key="${KG_API_KEY:-}"
tenant_id="${KG_SMOKE_TENANT_ID:-11111111-1111-1111-1111-111111111111}"
domain_id="${KG_SMOKE_DOMAIN_ID:-sample-policy}"
profile_node_key="${KG_SMOKE_NODE_KEY:-runtime-profile-smoke}"

if [[ -z "${base_url}" ]]; then
  echo "KG_BASE_URL is required" >&2
  exit 1
fi
if [[ -z "${api_key}" ]]; then
  echo "KG_API_KEY is required" >&2
  exit 1
fi

base_url="${base_url%/}"

step() {
  echo "==> $1"
}

http_get() {
  local url="$1"
  shift
  curl -fsS "$@" "${url}"
}

http_json() {
  local method="$1"
  local url="$2"
  local body="$3"
  curl -fsS -X "${method}" \
    -H "Authorization: Bearer ${api_key}" \
    -H "Content-Type: application/json" \
    -d "${body}" \
    "${url}"
}

step "health check"
http_get "${base_url}/healthz" >/dev/null

step "access resolution"
http_get "${base_url}/v1/access/resolve" -H "Authorization: Bearer ${api_key}" >/dev/null

step "write node"
create_resp="$(http_json POST "${base_url}/v1/kg/write/nodes" "{\"domain_id\":\"${domain_id}\",\"node_type\":\"Topic\",\"properties\":{\"topic_key\":\"${profile_node_key}\",\"title\":\"Runtime Profile Smoke\"}}")"
node_id="$(printf '%s' "${create_resp}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["node_id"])')"

step "read node"
http_get "${base_url}/v1/kg/read/nodes/${node_id}" -H "Authorization: Bearer ${api_key}" >/dev/null

step "semantic search"
http_json POST "${base_url}/v1/kg/search/semantic" "{\"query\":\"Runtime Profile Smoke\",\"domain_ids\":[\"${domain_id}\"],\"top_k\":3}" >/dev/null

step "integrity"
http_get "${base_url}/v1/kg/integrity/tenant/${tenant_id}" -H "Authorization: Bearer ${api_key}" >/dev/null

step "reconciliation"
(cd "${repo_root}" && go test ./internal/workers -run TestRuntimeReconcileReportsStaleProjectionVersion -count=1)

echo "runtime profile validation passed"
