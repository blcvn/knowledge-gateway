#!/usr/bin/env bash

set -euo pipefail

base_url="${KG_BASE_URL:-}"
api_key="${KG_API_KEY:-}"
domain_id="${KG_SMOKE_DOMAIN_ID:-sample-policy}"
template_name="${KG_SMOKE_TEMPLATE_NAME:-action-guide}"
template_params="${KG_SMOKE_TEMPLATE_PARAMS:-{\"topic_key\":\"returns\"}}"

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

step "health check"
health_body="$(http_get "${base_url}/healthz")"
if [[ -z "${health_body}" ]]; then
  echo "health check returned an empty body" >&2
  exit 1
fi

step "access resolution"
resolve_body="$(http_get "${base_url}/v1/access/resolve" -H "Authorization: Bearer ${api_key}")"
if [[ "${resolve_body}" != *'"tenant_id"'* || "${resolve_body}" != *'"app_id"'* ]]; then
  echo "access resolve response did not include tenant_id/app_id" >&2
  echo "${resolve_body}" >&2
  exit 1
fi

step "template listing"
templates_body="$(http_get "${base_url}/v1/kg/read/templates?domain_id=${domain_id}" -H "Authorization: Bearer ${api_key}")"
if [[ -z "${templates_body}" ]]; then
  echo "template listing returned an empty body" >&2
  exit 1
fi

step "template execution"
template_body="$(curl -fsS \
  -X POST \
  -H "Authorization: Bearer ${api_key}" \
  -H "Content-Type: application/json" \
  -d "{\"params\":${template_params}}" \
  "${base_url}/v1/kg/read/template/${domain_id}/${template_name}")"

if [[ -z "${template_body}" ]]; then
  echo "template execution returned an empty body" >&2
  exit 1
fi

echo "integration validation passed"
