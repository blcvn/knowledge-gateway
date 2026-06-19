# Docker Compose Deployment

Use Docker Compose when you want the fastest self-contained deployment path for local evaluation or a single host.

## What This Path Starts

- Postgres for the current bootstrap runtime
- Redis for the current bootstrap runtime
- `kg-service` built from the repository Dockerfile
- a one-shot migration container that applies the SQL schema before the app starts
- a selected runtime profile via `KG_RUNTIME_PROFILE`

There are now two dedicated Compose entrypoints:

- `deploy/compose/integration-test/docker-compose.yml` for the repeatable smoke path behind `make deploy-compose-integration`
- `deploy/compose/runtime-validation/docker-compose.yml` for the multi-backend validation path behind `make deploy-compose-runtime-validation`

## Prerequisites

- Docker with Compose v2
- Access to the repository files on the machine that runs Compose
- `KG_RUNTIME_PROFILE` exported before running `make deploy-compose`

## Start The Stack

```bash
KG_RUNTIME_PROFILE=pgvector-memgraph \
make deploy-compose
```

The script runs the Compose file under `deploy/compose/` with `docker compose up -d --build`.

To switch runtime profiles, set `KG_RUNTIME_PROFILE` before running the script. Supported profiles and related variables are documented in [Environment Variables](./environment.md).

For the integration smoke stack, run `make deploy-compose-integration`.

For the runtime validation stack with Neo4j, Memgraph, Nebula, Qdrant, Milvus, and Postgres available together, run `make deploy-compose-runtime-validation`.

## Verify The Deployment

```bash
make integration-test
```

For the default bootstrap smoke path, keep the sample environment variables described in the integration validation guide.

To validate a deployed profile end to end, use `scripts/validate-runtime-profile.sh` with `KG_BASE_URL` and `KG_API_KEY`.

## Notes

- The Compose path now selects a named runtime profile instead of silently booting memory adapters.
- The migration container applies the repository SQL schema before the service starts.
- The integration smoke stack under `make deploy-compose-integration` uses its own fixed profile and does not require you to export `KG_RUNTIME_PROFILE`.
