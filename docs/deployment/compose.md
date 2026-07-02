# Docker Compose Deployment

Use Docker Compose when you want the fastest self-contained deployment path for local evaluation or a single host.

## What This Path Starts

- Postgres for the current bootstrap runtime
- Redis for the current bootstrap runtime
- `kg-service` built from the repository Dockerfile
- a one-shot migration container that applies the SQL schema before the app starts
- a selected runtime profile via `KG_RUNTIME_PROFILE`

There are now three dedicated Compose entrypoints:

- `deploy/compose/integration-test/docker-compose.yml` for the repeatable smoke path behind `make deploy-compose-integration`
- `deploy/compose/runtime-validation/docker-compose.yml` for the multi-backend validation path behind `make deploy-compose-runtime-validation`
- `deploy/compose/codegraph-runtime/docker-compose.yml` for local CodeGraph validation behind `make deploy-compose-codegraph-runtime`

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

## CodeGraph Validation Stack

Use the dedicated CodeGraph stack when you want a single repeatable path for the `code-graph` domain:

- Postgres stays `pgvector`-compatible for migrations
- Memgraph is the graph runtime backend
- Qdrant is the vector runtime backend
- `KG_RUNTIME_PROFILE=qdrant-memgraph` is fixed by the repo-owned deploy script

Start it with HTTP embeddings enabled:

```bash
make deploy-compose-codegraph-runtime
```

The deploy script reads [`deploy/compose/codegraph-runtime/.env`](/Users/anhdt/vnpay/knowledge/kg-service/deploy/compose/codegraph-runtime/.env) directly, so that file is the single source of truth for the CodeGraph embedding and rate-limit settings. Treat the values there as local test configuration.

To run the full stack validation flow, including health, ontology bootstrap, sync, and query checks:

```bash
make validate-codegraph-runtime
```

The validation entrypoint can also reuse earlier bootstrap work:

```bash
make validate-codegraph-runtime
make validate-codegraph-runtime ARGS="--skip-compose"
make validate-codegraph-runtime ARGS="--skip-compose --skip-sync"
```

On the first successful run, the script stores the generated tenant/app identity in
`codegraph-sync/.state/codegraph-runtime-bootstrap.json` so later runs can reuse it without creating a new tenant.

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
- `make deploy-compose-codegraph-runtime` and `make validate-codegraph-runtime` both read `deploy/compose/codegraph-runtime/.env` directly for the CodeGraph embedding settings.
