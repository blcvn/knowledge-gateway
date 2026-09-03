# Configuration — Graphiti App

All configuration is done via **environment variables**. A reference file is at `config.yaml`.

## Server

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `REST_PORT` | int | `8080` | Gateway REST API port |
| `MCP_PORT` | int | `8082` | MCP server port |
| `HEALTH_PORT` | int | `9090` | Aggregated health probes port |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | Graceful shutdown timeout |
| `LOG_LEVEL` | string | `info` | Log level: debug, info, warn, error |

## Service gRPC Ports

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `INGESTION_GRPC_PORT` | int | `9021` | graphiti-ingestion gRPC port |
| `SEARCH_GRPC_PORT` | int | `9022` | graphiti-search gRPC port |
| `KNOWLEDGE_GRPC_PORT` | int | `9023` | graphiti-knowledge gRPC port |
| `STORE_GRPC_PORT` | int | `9024` | graphiti-store gRPC port |
| `PIPELINE_GRPC_PORT` | int | `9025` | graphiti-pipeline gRPC port |

## Authentication

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `AUTH_DEV_MODE` | bool | `false` | Skip JWT validation (dev only) |
| `AUTH_JWT_PUBLIC_KEY` | string | `""` | JWT RS256 public key (required in prod) |
| `AUTH_JWT_ISSUER` | string | `vnp-memory` | Expected JWT issuer |
| `AUTH_JWT_AUDIENCE` | string | `vnp-api` | Expected JWT audience |

## Neo4j (Primary Data Store)

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `NEO4J_URI` | string | `bolt://localhost:7687` | Neo4j Bolt URI |
| `NEO4J_USERNAME` | string | `neo4j` | Neo4j username |
| `NEO4J_PASSWORD` | string | `""` | Neo4j password |
| `NEO4J_DATABASE` | string | `neo4j` | Neo4j database name |

## LLM Provider

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `LLM_PROVIDER` | string | `openai` | LLM provider: openai, anthropic, ollama |
| `LLM_API_KEY` | string | `""` | API key (required in production) |
| `LLM_MODEL` | string | `gpt-4o` | Primary LLM model |
| `LLM_SMALL_MODEL` | string | `gpt-4o-mini` | Small/fast model for simple tasks |
| `LLM_TEMPERATURE` | float | `0.0` | Sampling temperature |
| `LLM_BASE_URL` | string | `""` | Override API base URL (for ollama/vLLM) |

## Embedder

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `EMBEDDER_PROVIDER` | string | `openai` | Embedding provider |
| `EMBEDDER_MODEL` | string | `text-embedding-3-small` | Embedding model |
| `EMBEDDER_DIMENSIONS` | int | `1536` | Embedding vector dimensions |

## Reranker

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `RERANKER_PROVIDER` | string | `openai` | Reranker provider |
| `RERANKER_MODEL` | string | `gpt-4o-mini` | Reranker model |

## Redis

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `REDIS_ADDR` | string | `localhost:6379` | Redis address |
| `REDIS_PASSWORD` | string | `""` | Redis password |
| `REDIS_DB` | int | `0` | Redis database number |

## NATS

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `NATS_URL` | string | `nats://localhost:4222` | NATS server URL |
| `NATS_CREDS_FILE` | string | `""` | NATS credentials file path |
| `NATS_STREAM` | string | `graphiti` | JetStream stream name |

## PostgreSQL (Optional)

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `POSTGRES_DSN` | string | `""` | Gateway key store DSN |
| `POSTGRES_MAX_CONNS` | int | `20` | Max pool connections |
| `POSTGRES_MIN_CONNS` | int | `5` | Min pool connections |

## CORS

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `CORS_ALLOWED_ORIGINS` | string | `*` | Allowed origins |
| `CORS_ALLOW_CREDENTIALS` | bool | `true` | Allow credentials |

## Timeouts

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `TIMEOUT_DEFAULT` | duration | `30s` | Default request timeout |
| `TIMEOUT_INGESTION` | duration | `300s` | Ingestion timeout (5 min) |
| `TIMEOUT_SEARCH` | duration | `30s` | Search timeout |
| `TIMEOUT_PIPELINE` | duration | `600s` | Pipeline timeout (10 min) |

## Observability

| ENV Variable | Type | Default | Description |
|---|---|---|---|
| `OTEL_ENDPOINT` | string | `""` | OpenTelemetry collector endpoint |

## Config Injection Flow

```
Load config.yaml defaults
        │
        ▼
ENV vars override (os.LookupEnv)
        │
        ▼
cfg.Validate() — check required values
        │
        ▼
cfg.SetServiceEnvVars() — export to env for embedded services
        │
        ▼
cfg.GatewayServicesMap() — generate localhost:PORT map for gateway
```

## Production Checklist

- [ ] Set `AUTH_DEV_MODE=false`
- [ ] Set `AUTH_JWT_PUBLIC_KEY` to your RS256 public key
- [ ] Set `LLM_API_KEY` to your provider API key
- [ ] Set `NEO4J_PASSWORD` to a strong password
- [ ] Verify `NEO4J_URI` points to production cluster
- [ ] Set `CORS_ALLOWED_ORIGINS` to specific domains
- [ ] Configure `OTEL_ENDPOINT` for observability
