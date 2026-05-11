---
id: DOC-S05
service: graphiti-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
---

# graphiti-pipeline — Configuration Reference

## Environment Variables

### Core Service

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9021` | No | gRPC server port |
| `HEALTH_PORT` | int | `9094` | No | Health/metrics HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level: debug, info, warn, error |
| `LOG_FORMAT` | string | `json` | No | Log format: json, text |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | No | Graceful shutdown timeout |

### PostgreSQL

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `POSTGRES_URI` | string | — | **Yes** | PostgreSQL connection string |
| `POSTGRES_MAX_CONNS` | int | `20` | No | Max connection pool size |
| `POSTGRES_MIN_CONNS` | int | `5` | No | Min idle connections |
| `POSTGRES_CONN_MAX_LIFETIME` | duration | `1h` | No | Connection max lifetime |

### graphiti-store (gRPC Client)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `STORE_ADDR` | string | `graphiti-store:9024` | **Yes** | Store service gRPC address |
| `STORE_TIMEOUT` | duration | `30s` | No | Per-call timeout |
| `STORE_CB_THRESHOLD` | int | `5` | No | Circuit breaker failure threshold |
| `STORE_CB_TIMEOUT` | duration | `60s` | No | Circuit breaker recovery timeout |

### LLM (Bifrost)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `LLM_PROVIDER` | string | `openai` | No | LLM provider (openai, anthropic, openrouter) |
| `LLM_MODEL` | string | `gpt-4o` | No | Default LLM model for extraction |
| `LLM_SMALL_MODEL` | string | `gpt-4o-mini` | No | Smaller model for resolution/classification |
| `LLM_API_KEY` | string | — | **Yes** | API key for LLM provider |
| `LLM_BASE_URL` | string | — | No | Custom LLM endpoint (for Bifrost gateway) |
| `LLM_TIMEOUT` | duration | `60s` | No | LLM request timeout |
| `LLM_MAX_CONCURRENT` | int | `20` | No | Max concurrent LLM requests (bulkhead) |
| `LLM_MAX_RETRIES` | int | `3` | No | LLM retry attempts |
| `LLM_TEMPERATURE` | float | `0.0` | No | LLM temperature for extraction |

### Embedder

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `EMBEDDER_PROVIDER` | string | `openai` | No | Embedding provider |
| `EMBEDDER_MODEL` | string | `text-embedding-3-small` | No | Embedding model |
| `EMBEDDER_API_KEY` | string | — | **Yes** | Embedding API key (can be same as LLM) |
| `EMBEDDER_DIMENSIONS` | int | `1536` | No | Embedding vector dimensions |
| `EMBEDDER_BATCH_SIZE` | int | `100` | No | Max batch size per embedding request |

### NATS

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NATS_URL` | string | `nats://localhost:4222` | No | NATS server URL |
| `NATS_STREAM` | string | `graphiti` | No | JetStream stream name |

### Observability (OTel)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_ENDPOINT` | string | `localhost:4317` | No | OTel collector gRPC endpoint |
| `OTEL_SERVICE_NAME` | string | `graphiti-pipeline` | No | Service name for tracing |
| `OTEL_ENABLED` | bool | `true` | No | Enable/disable OTel tracing |

### Pipeline Tuning

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `PIPELINE_TIMEOUT` | duration | `300s` | No | Max time for full saga pipeline |
| `GROUP_QUEUE_SIZE` | int | `100` | No | Max queued episodes per group_id |
| `DEDUP_ENABLED` | bool | `true` | No | Enable episode deduplication |
| `COMMUNITY_ASYNC` | bool | `true` | No | Community update async (non-blocking) |

## Example `.env`

```env
# Core
GRPC_PORT=9021
HEALTH_PORT=9094
LOG_LEVEL=debug

# PostgreSQL
POSTGRES_URI=postgres://graphiti:secret@localhost:5432/graphiti?sslmode=disable

# graphiti-store
STORE_ADDR=localhost:9024

# LLM
LLM_PROVIDER=openai
LLM_MODEL=gpt-4o
LLM_API_KEY=sk-xxx
LLM_BASE_URL=http://bifrost:8002/v1

# Embedder
EMBEDDER_MODEL=text-embedding-3-small
EMBEDDER_API_KEY=sk-xxx

# NATS
NATS_URL=nats://localhost:4222

# OTel
OTEL_ENDPOINT=localhost:4317
```
