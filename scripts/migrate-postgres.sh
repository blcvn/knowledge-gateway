#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
migrations_dir="${repo_root}/migrations"

dsn="${KG_MIGRATION_DSN:-}"
if [[ -z "${dsn}" ]]; then
  if [[ -z "${KG_POSTGRES_HOST:-}" || -z "${KG_POSTGRES_PASSWORD:-}" ]]; then
    echo "set KG_MIGRATION_DSN or KG_POSTGRES_HOST and KG_POSTGRES_PASSWORD" >&2
    exit 1
  fi

  kg_postgres_port="${KG_POSTGRES_PORT:-5432}"
  kg_postgres_user="${KG_POSTGRES_USER:-postgres}"
  kg_postgres_database="${KG_POSTGRES_DATABASE:-kg_service}"
  kg_postgres_sslmode="${KG_POSTGRES_SSLMODE:-disable}"
  dsn="postgres://${kg_postgres_user}:${KG_POSTGRES_PASSWORD}@${KG_POSTGRES_HOST}:${kg_postgres_port}/${kg_postgres_database}?sslmode=${kg_postgres_sslmode}"
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required to run the migration helper" >&2
  exit 1
fi

echo "applying migrations to ${dsn}"
docker run --rm \
  -v "${migrations_dir}:/migrations:ro" \
  migrate/migrate:v4.17.1 \
  -path /migrations \
  -database "${dsn}" \
  up
