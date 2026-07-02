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
| `EMBEDDING_MODEL` | Model identifier sent to the HTTP embedding provider. | none | Required when `EMBEDDING_PROVIDER=http` |
| `EMBEDDING_API_KEY` | API key for the HTTP embedding provider. | none | Required when `EMBEDDING_PROVIDER=http` |
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

## CodeGraph Compose Validation Notes

Use these values together when running `make deploy-compose-codegraph-runtime` or `make validate-codegraph-runtime`:

| Variable | Purpose | Default | Required |
| --- | --- | --- | --- |
| `EMBEDDING_PROVIDER` | Must be `http` for the dedicated CodeGraph Compose path. | `http` | Required by the CodeGraph Compose path |
| `EMBEDDING_URL` | HTTP embedding endpoint used by the CodeGraph stack. See `tests/llm/embedding-vnp.txt` for local test reference values. | none | Required |
| `EMBEDDING_MODEL` | HTTP embedding model used by the CodeGraph stack. See `tests/llm/embedding-vnp.txt` for local test reference values. | none | Required |
| `EMBEDDING_API_KEY` | Local secret for the HTTP embedding provider. Use a placeholder in repo-owned examples and inject the real secret from your shell or external env file. | none | Required |
| `KG_API_KEY` | Existing tenant admin key to reuse for the bootstrap, sync, and query validation steps. When omitted, the script can create and persist a tenant admin app using `KG_PLATFORM_API_KEY`. | none | Optional |
| `KG_BASE_URL` | Base URL targeted by the CodeGraph validation script after the Compose stack boots. | `http://127.0.0.1:8082` | Optional |
| `KG_PLATFORM_API_KEY` | Platform admin key used to create a tenant/app on the first CodeGraph validation run. | `kgsk_platform_admin` | Optional |
| `KG_RUNTIME_STATE_FILE` | Local state file where the validation script stores the generated tenant/app identity for reruns. | `codegraph-sync/.state/codegraph-runtime-bootstrap.json` | Optional |

## Notes

- `make deploy-compose-integration` uses its own fixed runtime profile in the Compose manifest and does not require `KG_RUNTIME_PROFILE`.
- `make deploy-compose-codegraph-runtime` fixes `KG_RUNTIME_PROFILE=qdrant-memgraph` and expects Postgres, Memgraph, and Qdrant to be started by the dedicated Compose manifest.
- `scripts/validate-codegraph-runtime.sh` accepts `--skip-compose`, `--skip-tenant-bootstrap`, `--skip-ontology-bootstrap`, `--skip-sync`, and `--skip-verify` for reruns.
- Invalid integer and duration values fail startup with configuration errors instead of panicking the process.
- `GET /healthz` is public, but it is limited to safe operational metadata and does not expose raw DSNs or secrets.
