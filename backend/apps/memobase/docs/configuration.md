# Memobase App — Configuration Reference

> All environment variables and configuration options.

## Configuration Loading Order

1. `config.yaml` (YAML file defaults)
2. Environment variables (override YAML)
3. Built-in defaults (fallback)

## Server Configuration

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `REST_PORT` | Gateway REST API port | int | `8080` | No |
| `MCP_PORT` | MCP Server SSE port | int | `8082` | No |
| `HEALTH_PORT` | Health/metrics server port | int | `9090` | No |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | string | `info` | No |
| `SHUTDOWN_TIMEOUT` | Graceful shutdown timeout | duration | `30s` | No |

## Service Ports

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `INGESTION_GRPC_PORT` | memobase-ingestion gRPC port | int | `9041` | No |
| `ENGINE_GRPC_PORT` | memobase-engine gRPC port | int | `9042` | No |
| `CONTEXT_GRPC_PORT` | memobase-context gRPC port | int | `9043` | No |
| `PIPELINE_GRPC_PORT` | memobase-pipeline gRPC port | int | `9044` | No |

## Database

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | string | - | ✅ |
| `DB_POOL_SIZE` | Connection pool size | int | `25` | No |
| `DB_MAX_OVERFLOW` | Max overflow connections | int | `15` | No |

## Redis

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `REDIS_ADDR` | Redis server address | string | `localhost:6379` | ✅ |
| `REDIS_PASSWORD` | Redis password | string | - | No |
| `REDIS_DB` | Redis database number | int | `0` | No |

## NATS

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `NATS_URL` | NATS server URL | string | `nats://localhost:4222` | ✅ |
| `NATS_CREDS_FILE` | NATS credentials file path | string | - | No |
| `NATS_STREAM` | JetStream stream name | string | `memobase` | No |

## LLM

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `LLM_BASE_URL` | LLM API base URL | string | - | ✅ (for engine) |
| `LLM_API_KEY` | LLM API key | string | - | ✅ (for engine) |
| `LLM_MODEL` | LLM model name | string | `gpt-4o-mini` | No |
| `LLM_PROVIDER` | LLM provider (bifrost/openai/ollama) | string | `bifrost` | No |

## Embedding

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `EMBEDDING_PROVIDER` | Embedding provider | string | `openai` | No |
| `EMBEDDING_MODEL` | Embedding model name | string | `text-embedding-3-small` | No |
| `EMBEDDING_DIMENSION` | Vector dimension | int | `1536` | No |
| `EMBEDDING_ENABLED` | Enable embeddings | bool | `true` | No |

## Auth

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `AUTH_DEV_MODE` | Enable dev mode (skip auth) | bool | `false` | No |
| `ROOT_ACCESS_TOKEN` | Root admin token | string | - | No |
| `JWT_PUBLIC_KEY` | JWT public key for verification | string | - | No |
| `JWT_ISSUER` | JWT issuer | string | `memobase` | No |

## Observability

| Variable | Description | Type | Default | Required |
|----------|-------------|------|---------|----------|
| `OTEL_ENDPOINT` | OpenTelemetry collector endpoint | string | - | No |
| `OTEL_SERVICE_NAME` | OTel service name | string | `memobase-app` | No |

## Example `.env`

```bash
DATABASE_URL=postgres://memobase:memobase@localhost:5432/memobase?sslmode=disable
REDIS_ADDR=localhost:6379
NATS_URL=nats://localhost:4222
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=sk-...
AUTH_DEV_MODE=true
LOG_LEVEL=info
```
