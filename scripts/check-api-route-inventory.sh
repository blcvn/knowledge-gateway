#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
app_file="${repo_root}/internal/bootstrap/app.go"
spec_file="${repo_root}/docs/api/openapi.yaml"

runtime_routes_file="$(mktemp)"
spec_routes_file="$(mktemp)"
cleanup() {
  rm -f "${runtime_routes_file}" "${spec_routes_file}"
}
trap cleanup EXIT

rg -o 'Handle(Func)?\("[A-Z]+ [^"]+' "${app_file}" \
  | sed -E 's/.*"//' \
  | sort > "${runtime_routes_file}"

awk '
  $0 == "paths:" { in_paths = 1; next }
  in_paths && /^components:/ { in_paths = 0 }
  in_paths && /^  \// {
    path = $1
    sub(":$", "", path)
    next
  }
  in_paths && /^    (get|post|put|delete|patch):/ {
    method = toupper($1)
    sub(":$", "", method)
    print method " " path
  }
' "${spec_file}" | sort > "${spec_routes_file}"

missing_in_spec="$(comm -23 "${runtime_routes_file}" "${spec_routes_file}" || true)"
extra_in_spec="$(comm -13 "${runtime_routes_file}" "${spec_routes_file}" || true)"

if [[ -n "${missing_in_spec}" ]]; then
  echo "Routes present in runtime but missing from docs/api/openapi.yaml:"
  echo "${missing_in_spec}"
fi

if [[ -n "${extra_in_spec}" ]]; then
  echo "Routes present in docs/api/openapi.yaml but missing from runtime:"
  echo "${extra_in_spec}"
fi

if [[ -n "${missing_in_spec}" || -n "${extra_in_spec}" ]]; then
  exit 1
fi

echo "Route inventory matches between internal/bootstrap/app.go and docs/api/openapi.yaml."
