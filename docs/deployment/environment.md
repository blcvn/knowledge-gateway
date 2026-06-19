# Environment Variables

This page is the operator-facing inventory of supported environment variables for `kg-service` startup and deployment.

## Shared Runtime Variables

| Variable | Purpose | Default | Required |
| --- | --- | --- | --- |
| `KG_RUNTIME_PROFILE` | Selects the supported graph/vector backend pairing used by deploy scripts. | none | Required for `make deploy-compose`, `make deploy-k8s`, and `make deploy-vm` |
| `KG_HTTP_HOST` | HTTP bind host for the service listener. | `0.0.0.0` | Optional |
| `KG_HTTP_PORT` | HTTP bind port for the service listener. Must be an integer. | `8082` | Optional |
| `EMBEDDING_PROVIDER` | Embedding provider implementation. Supported values: `deterministic`, `http`. | `deterministic` | Optional |
| `EMBEDDING_URL` | Base URL for the HTTP embedding provider. | none | Required when `EMBEDDING_PROVIDER=http` |
| `EMBEDDING_MODEL` | Model identifier sent to the HTTP embedding provider. | none | Optional |
| `EMBEDDING_API_KEY` | API key for the HTTP embedding provider. | none | Optional |
| `EMBEDDING_PROXY_URL` | Proxy endpoint layered in front of the embedding provider. | none | Optional |
| `EMBEDDING_CACHE_TTL_S` | Embedding cache TTL in seconds. Must be an integer. | `0` | Optional |

## Postgres Variables

| Variable | Purpose | Default | Required |
| --- | --- | --- | --- |
| `KG_POSTGRES_HOST` | Postgres host used by runtime bootstrap. | `127.0.0.1` | Required for deployed environments |
| `KG_POSTGRES_PORT` | Postgres port. Must be an integer. | `5432` | Optional |
| `KG_POSTGRES_USER` | Postgres username. | `postgres` | Optional |
| `KG_POSTGRES_PASSWORD` | Postgres password. | `postgres` | Required for deployed environments unless local bootstrap uses the default |
| `KG_POSTGRES_DATABASE` | Postgres database name. | `kg_service` | Optional |
| `KG_POSTGRES_SSLMODE` | Postgres SSL mode in the runtime DSN. | `disable` | Optional |
| `KG_POSTGRES_MAX_OPEN_CONNS` | Max open Postgres connections. Must be an integer. | `20` | Optional |
| `KG_POSTGRES_MAX_IDLE_CONNS` | Max idle Postgres connections. Must be an integer. | `5` | Optional |
| `KG_POSTGRES_CONN_MAX_LIFETIME` | Max connection lifetime. Must be a Go duration string such as `30m` or `1h`. | `30m` | Optional |
| `KG_MIGRATION_DSN` | DSN used by `make migrate` to apply schema changes. | none | Required when running migrations manually |

## Redis Variables

| Variable | Purpose | Default | Required |
| --- | --- | --- | --- |
| `KG_REDIS_HOST` | Redis host used by runtime bootstrap. | `127.0.0.1` | Required for deployed environments |
| `KG_REDIS_PORT` | Redis port. Must be an integer. | `6379` | Optional |
| `KG_REDIS_USERNAME` | Redis username. | empty | Optional |
| `KG_REDIS_PASSWORD` | Redis password. | empty | Optional |
| `KG_REDIS_DB` | Redis logical database. Must be an integer. | `0` | Optional |

## Profile-Driven Backend Variables

The deploy scripts derive adapter kinds from `KG_RUNTIME_PROFILE` and may also set defaults for backend endpoints.

| Variable | Purpose | Default Source | Required |
| --- | --- | --- | --- |
| `GRAPH_ADAPTER` | Graph backend kind. Supported values include `memory`, `neo4j`, `memgraph`, `nebula`. | Derived by `scripts/runtime-profile.sh` for deploy scripts | Optional when using deploy scripts, otherwise required if overriding manually |
| `KG_GRAPH_ENDPOINT` | Graph backend endpoint. | Derived by `scripts/runtime-profile.sh` for supported profiles | Required when `GRAPH_ADAPTER` is `neo4j`, `memgraph`, or `nebula` |
| `KG_GRAPH_DATABASE` | Graph database or graph space selection. | Derived by `scripts/runtime-profile.sh` for supported profiles | Required when `GRAPH_ADAPTER=nebula`; recommended to keep set with profile defaults for other profile-managed graph backends |
| `VECTOR_ADAPTER` | Vector backend kind. Supported values include `memory`, `pgvector`, `qdrant`, `milvus`. | Derived by `scripts/runtime-profile.sh` for deploy scripts | Optional when using deploy scripts, otherwise required if overriding manually |
| `KG_VECTOR_ENDPOINT` | Vector backend endpoint. | Derived by `scripts/runtime-profile.sh` for `qdrant` and `milvus` profiles | Required when `VECTOR_ADAPTER` is `qdrant` or `milvus` |
| `KG_VECTOR_COLLECTION` | Vector collection name. | `kg_vectors` | Optional |
| `FTS_ADAPTER` | Full-text search backend kind. Supported values: `memory`, `postgres`. | Derived by `scripts/runtime-profile.sh` for deploy scripts | Optional |

## Deployment-Specific Helper Variables

| Variable | Purpose | Default | Required |
| --- | --- | --- | --- |
| `KG_IMAGE` | Image reference used by the Kubernetes deploy script. | `kg-service:local` | Optional |
| `KG_NAMESPACE` | Namespace used by the Kubernetes deploy script. | `kg-service` | Optional |
| `KG_REBUILD_BINARY` | Forces the VM deploy script to rebuild the binary before launch. | `0` | Optional |
| `KG_BASE_URL` | Base URL used by validation scripts such as `make integration-test` and `make validate-runtime-profile`. | none | Required for validation against a deployed instance |
| `KG_API_KEY` | API key used by authenticated validation flows. | none | Required for authenticated validation |

## Notes

- `make deploy-compose-integration` uses its own fixed runtime profile in the Compose manifest and does not require `KG_RUNTIME_PROFILE`.
- Invalid integer and duration values fail startup with configuration errors instead of panicking the process.
- `GET /healthz` is public, but it is limited to safe operational metadata and does not expose raw DSNs or secrets.
