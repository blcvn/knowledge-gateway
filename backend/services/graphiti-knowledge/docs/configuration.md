---
id: DOC-S05
service: graphiti-knowledge
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-knowledge — Configuration Reference

## Environment Variables

### Core Service

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9023` | No | gRPC server port |
| `HEALTH_PORT` | int | `9096` | No | Health/metrics HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | No | Graceful shutdown timeout |

### LLM (Bifrost Gateway)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `LLM_BASE_URL` | string | — | **Yes** | Bifrost /v1/chat/completions base URL |
| `LLM_API_KEY` | string | — | **Yes** | Bifrost API key |
| `LLM_MODEL` | string | `gpt-4o` | No | Default extraction model |
| `LLM_MODEL_MINI` | string | `gpt-4o-mini` | No | Resolution/classification model |
| `LLM_TIMEOUT` | duration | `60s` | No | Per-call timeout |
| `LLM_MAX_CONCURRENT` | int | `10` | No | Bulkhead: max concurrent LLM calls |
| `LLM_MAX_TOKENS` | int | `4096` | No | Max response tokens |
| `LLM_TEMPERATURE` | float | `0.0` | No | LLM temperature (0 = deterministic) |

### Embedder (Bifrost)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `EMBEDDER_BASE_URL` | string | — | **Yes** | Bifrost /v1/embeddings base URL |
| `EMBEDDER_API_KEY` | string | — | **Yes** | Bifrost API key |
| `EMBEDDER_MODEL` | string | `text-embedding-3-small` | No | Embedding model |
| `EMBEDDER_DIMENSIONS` | int | `1536` | No | Output vector dimensions |
| `EMBEDDER_BATCH_SIZE` | int | `100` | No | Max texts per batch request |

### graphiti-store (gRPC Client — Read-Only)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `STORE_ADDR` | string | `graphiti-store:9024` | **Yes** | Store service gRPC address |
| `STORE_TIMEOUT` | duration | `10s` | No | Per-call timeout |

### NATS

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NATS_URL` | string | `nats://localhost:4222` | No | NATS server for event publishing |

### Resolution Tuning

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `ENTITY_SIMILARITY_THRESHOLD` | float | `0.85` | No | Cosine threshold for entity dedup |
| `EDGE_CONTRADICTION_CHECK` | bool | `true` | No | Enable edge contradiction detection |

### Observability (OTel)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_ENDPOINT` | string | `localhost:4317` | No | OTel collector endpoint |
| `OTEL_SERVICE_NAME` | string | `graphiti-knowledge` | No | Service name |

## Example `.env`

```env
GRPC_PORT=9023
HEALTH_PORT=9096
LOG_LEVEL=debug

LLM_BASE_URL=http://bifrost:8080/v1
LLM_API_KEY=sk-bifrost-key
LLM_MODEL=gpt-4o
LLM_MAX_CONCURRENT=10

EMBEDDER_BASE_URL=http://bifrost:8080/v1
EMBEDDER_API_KEY=sk-bifrost-key
EMBEDDER_MODEL=text-embedding-3-small
EMBEDDER_DIMENSIONS=1536

STORE_ADDR=localhost:9024
NATS_URL=nats://localhost:4222

ENTITY_SIMILARITY_THRESHOLD=0.85
OTEL_ENDPOINT=localhost:4317
```
