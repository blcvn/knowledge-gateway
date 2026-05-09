---
id: DOC-S05
service: vnp-search-hub
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-search-hub — Configuration Reference

## Environment Variables

### Core

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | 9042 | Yes | gRPC server listen port |
| `HEALTH_PORT` | int | 9102 | Yes | Health/metrics HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | string | `json` | No | Log output format (json/text) |

### Redis

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `REDIS_URL` | string | `redis://localhost:6379` | Yes | Redis for result caching |
| `CACHE_TTL` | int | 60 | No | Recall result cache TTL (seconds) |

### Engine Addresses

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `COGNEE_SEARCH_ADDR` | string | `cognee-search:9023` | Yes | Cognee search gRPC address |
| `GRAPHITI_SEARCH_ADDR` | string | `graphiti-search:9013` | Yes | Graphiti search gRPC address |
| `MEMOBASE_CONTEXT_ADDR` | string | `memobase-context:9033` | Yes | Memobase context gRPC address |
| `OV_SEARCH_ADDR` | string | `ov-search:9043` | Yes | OpenViking search gRPC address |
| `ZEP_SEARCH_ADDR` | string | `zep-search:9053` | Yes | Zep search gRPC address |
| `SM_SEARCH_ADDR` | string | `sm-search:9063` | Yes | Supermemory search gRPC address |
| `VNP_EVENT_ADDR` | string | `vnp-event:9041` | Yes | Event timeline gRPC address |

### Search Configuration

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `DEFAULT_RERANKER` | string | `rrf` | No | Default reranking strategy (`rrf`, `cross_encoder`) |
| `RRF_K` | int | 60 | No | RRF constant (higher = less weight to top ranks) |
| `FAN_OUT_TIMEOUT` | duration | `10s` | No | Per-engine search timeout |
| `DEFAULT_LIMIT` | int | 20 | No | Default result limit per recall |

### Circuit Breaker (Per Engine)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `CB_MAX_FAILURES` | int | 3 | No | Failures before circuit opens |
| `CB_TIMEOUT` | duration | `15s` | No | Time before half-open retry |

### Observability

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_EXPORTER_ENDPOINT` | string | `localhost:4317` | No | OTel collector gRPC endpoint |
| `OTEL_SERVICE_NAME` | string | `vnp-search-hub` | No | OTel service name |

## Example `.env`

```env
GRPC_PORT=9042
HEALTH_PORT=9102
REDIS_URL=redis://redis:6379
COGNEE_SEARCH_ADDR=cognee-search:9023
GRAPHITI_SEARCH_ADDR=graphiti-search:9013
MEMOBASE_CONTEXT_ADDR=memobase-context:9033
OV_SEARCH_ADDR=ov-search:9043
ZEP_SEARCH_ADDR=zep-search:9053
SM_SEARCH_ADDR=sm-search:9063
VNP_EVENT_ADDR=vnp-event:9041
DEFAULT_RERANKER=rrf
FAN_OUT_TIMEOUT=10s
OTEL_EXPORTER_ENDPOINT=otel-collector:4317
```
