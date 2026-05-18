# Configuration Reference — Cognee App

All configuration is loaded from **environment variables**, falling back to built-in defaults.
A `config.yaml` example is provided for reference.

## Config Precedence

```
ENV variable → built-in default
```

## Environment Variables

### App / Server

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `REST_PORT` | int | `8080` | Gateway HTTP REST API port |
| `HEALTH_PORT` | int | `11080` | Aggregated health probe port |
| `LOG_LEVEL` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | Graceful shutdown timeout |

### Embedded Service Ports

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `INGESTION_GRPC_PORT` | int | `9011` | cognee-ingestion gRPC port |
| `COGNIFY_GRPC_PORT` | int | `9012` | cognee-cognify gRPC port |
| `SEARCH_GRPC_PORT` | int | `9013` | cognee-search gRPC port |

### Authentication

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AUTH_JWT_PUBLIC_KEY` | string | _(required in prod)_ | JWT RS256 public key PEM |
| `AUTH_JWT_ISSUER` | string | `vnp-memory` | Expected JWT issuer |
| `AUTH_JWT_AUDIENCE` | string | `vnp-api` | Expected JWT audience |
| `AUTH_DEV_MODE` | bool | `false` | Skip JWT validation in dev |

### PostgreSQL

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `POSTGRES_DSN` | string | _(required)_ | PostgreSQL connection string |
| `POSTGRES_MAX_CONNS` | int | `20` | Max connection pool size |

### NATS JetStream

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `NATS_URL` | string | `nats://nats:4222` | NATS server URL |
| `NATS_CREDS_FILE` | string | _(empty)_ | NATS credentials file path |
| `NATS_STREAM` | string | `cognee` | JetStream stream name |

### Redis

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `REDIS_ADDR` | string | `redis:6379` | Redis address |
| `REDIS_PASSWORD` | string | _(empty)_ | Redis password |

### Neo4j

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `NEO4J_URI` | string | `bolt://neo4j:7687` | Neo4j Bolt URI |
| `NEO4J_USERNAME` | string | `neo4j` | Neo4j username |
| `NEO4J_PASSWORD` | string | _(empty)_ | Neo4j password |

### Qdrant

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `QDRANT_URL` | string | `http://qdrant:6333` | Qdrant HTTP URL |
| `QDRANT_COLLECTION` | string | `cognee` | Default collection name |

### MinIO / S3

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `MINIO_ENDPOINT` | string | `minio:9000` | MinIO endpoint |
| `MINIO_ACCESS_KEY` | string | _(empty)_ | Access key |
| `MINIO_SECRET_KEY` | string | _(empty)_ | Secret key |
| `MINIO_BUCKET` | string | `cognee` | Bucket name |

### LLM

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `LLM_PROVIDER` | string | `openai` | LLM provider |
| `LLM_API_KEY` | string | _(required in prod)_ | API key |
| `LLM_MODEL` | string | `gpt-4o-mini` | Model for text generation |
| `LLM_EMBED_MODEL` | string | `text-embedding-3-small` | Model for embeddings |
| `LLM_BASE_URL` | string | _(empty)_ | Custom base URL |

### Timeouts

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `TIMEOUT_DEFAULT` | duration | `30s` | Default request timeout |
| `TIMEOUT_INGESTION` | duration | `120s` | Ingestion request timeout |
| `TIMEOUT_SEARCH` | duration | `10s` | Search request timeout |

### Observability (Optional)

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | string | _(empty)_ | OTel collector gRPC endpoint |
| `ENV` | string | `production` | Environment name for OTel |

## ENV Injection for Embedded Services

The monolith calls `cfg.SetServiceEnvVars()` at startup to inject unified config values as environment variables. This ensures embedded services (which read config via `os.Getenv()`) receive correct values:

```
Unified Config → SetServiceEnvVars() → os.Setenv("DATABASE_URL", dsn)
                                      → os.Setenv("NATS_URL", natsUrl)
                                      → os.Setenv("NEO4J_URI", neo4jUri)
                                      → ...
```
