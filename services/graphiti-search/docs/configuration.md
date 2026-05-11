---
id: DOC-S05
service: graphiti-search
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-search — Configuration Reference

## Environment Variables

### Core Service

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9022` | No | gRPC server port |
| `HEALTH_PORT` | int | `9095` | No | Health/metrics HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level: debug, info, warn, error |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | No | Graceful shutdown timeout |

### graphiti-store (gRPC Client)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `STORE_ADDR` | string | `graphiti-store:9024` | **Yes** | Store service gRPC address |
| `STORE_TIMEOUT` | duration | `30s` | No | Per-call timeout |
| `STORE_CB_THRESHOLD` | int | `5` | No | Circuit breaker failure threshold |

### graphiti-pipeline (gRPC Client — for Cross-Encoder)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `PIPELINE_ADDR` | string | `graphiti-pipeline:9021` | No | Pipeline service for cross-encoder reranking |
| `PIPELINE_TIMEOUT` | duration | `10s` | No | Rerank call timeout |

### Redis (Cache)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `REDIS_URL` | string | `redis://localhost:6379` | No | Redis connection URL |
| `CACHE_TTL` | duration | `5m` | No | Search result cache TTL |
| `CACHE_ENABLED` | bool | `true` | No | Enable/disable caching |

### NATS (Cache Invalidation)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NATS_URL` | string | `nats://localhost:4222` | No | NATS server URL |
| `NATS_STREAM` | string | `graphiti` | No | JetStream stream name |

### Search Tuning

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `DEFAULT_SEARCH_LIMIT` | int | `10` | No | Default result limit |
| `MAX_SEARCH_LIMIT` | int | `100` | No | Max allowed result limit |
| `DEFAULT_RERANKER` | string | `rrf` | No | Default reranking strategy |
| `RRF_K` | int | `60` | No | RRF fusion constant |
| `MMR_LAMBDA` | float | `0.7` | No | MMR diversity parameter |
| `BFS_MAX_DEPTH` | int | `3` | No | Max BFS traversal depth |

### Observability (OTel)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_ENDPOINT` | string | `localhost:4317` | No | OTel collector endpoint |
| `OTEL_SERVICE_NAME` | string | `graphiti-search` | No | Service name |

## Example `.env`

```env
GRPC_PORT=9022
HEALTH_PORT=9095
LOG_LEVEL=debug

STORE_ADDR=localhost:9024
PIPELINE_ADDR=localhost:9021

REDIS_URL=redis://localhost:6379
CACHE_TTL=5m

NATS_URL=nats://localhost:4222

DEFAULT_RERANKER=rrf
RRF_K=60
MMR_LAMBDA=0.7

OTEL_ENDPOINT=localhost:4317
```
