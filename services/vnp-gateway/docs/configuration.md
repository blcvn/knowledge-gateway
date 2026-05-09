---
id: DOC-S05
service: vnp-gateway
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-gateway — Configuration Reference

## Environment Variables

### Core

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `REST_PORT` | int | 8080 | Yes | REST/WebDAV HTTP port |
| `GRPC_PORT` | int | 8081 | Yes | gRPC-Web proxy port |
| `MCP_PORT` | int | 8082 | Yes | MCP SSE/HTTP Streamable port |
| `HEALTH_PORT` | int | 8083 | Yes | Health/metrics HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | string | `json` | No | Log output format (json/text) |

### Authentication

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `JWT_PUBLIC_KEY_PATH` | string | — | Yes | Path to RS256 public key PEM |
| `JWT_ISSUER` | string | `vnp-memory` | No | Expected JWT issuer |
| `VNP_ADMIN_ADDR` | string | `vnp-admin:9050` | Yes | vnp-admin gRPC address for key validation |

### Rate Limiting

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `REDIS_URL` | string | `redis://localhost:6379` | Yes | Redis for rate limiting + caching |
| `RATE_LIMIT_PER_MINUTE` | int | 60 | No | Default per-tenant per-endpoint rate |
| `RATE_LIMIT_BURST` | int | 10 | No | Burst allowance |

### Circuit Breaker

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `CB_MAX_FAILURES` | int | 5 | No | Failures before circuit opens |
| `CB_TIMEOUT` | duration | `30s` | No | Time before half-open retry |
| `CB_MAX_REQUESTS` | int | 1 | No | Requests in half-open state |

### Downstream Services

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `MEMOBASE_INGESTION_ADDR` | string | `memobase-ingestion:9031` | Yes | Memobase ingestion gRPC address |
| `MEMOBASE_CONTEXT_ADDR` | string | `memobase-context:9033` | Yes | Memobase context gRPC address |
| `VNP_EVENT_ADDR` | string | `vnp-event:9041` | Yes | Event timeline gRPC address |
| `VNP_SEARCH_HUB_ADDR` | string | `vnp-search-hub:9042` | Yes | Search hub gRPC address |
| `NATS_URL` | string | `nats://localhost:4222` | Yes | NATS JetStream server URL |

### Timeouts

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `DEFAULT_TIMEOUT` | duration | `30s` | No | Default request timeout |
| `INGESTION_TIMEOUT` | duration | `120s` | No | Timeout for ingestion routes |
| `SEARCH_TIMEOUT` | duration | `10s` | No | Timeout for search routes |

### Observability

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_EXPORTER_ENDPOINT` | string | `localhost:4317` | No | OTel collector gRPC endpoint |
| `OTEL_SERVICE_NAME` | string | `vnp-gateway` | No | OTel service name |

### CORS

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `CORS_ORIGINS` | string | `*` | No | Comma-separated allowed origins |
| `CORS_MAX_AGE` | int | 86400 | No | Preflight cache duration (seconds) |

## Example `.env`

```env
REST_PORT=8080
GRPC_PORT=8081
MCP_PORT=8082
HEALTH_PORT=8083
JWT_PUBLIC_KEY_PATH=/etc/vnp/jwt-pub.pem
REDIS_URL=redis://redis:6379
VNP_ADMIN_ADDR=vnp-admin:9050
MEMOBASE_INGESTION_ADDR=memobase-ingestion:9031
MEMOBASE_CONTEXT_ADDR=memobase-context:9033
VNP_EVENT_ADDR=vnp-event:9041
VNP_SEARCH_HUB_ADDR=vnp-search-hub:9042
NATS_URL=nats://nats:4222
OTEL_EXPORTER_ENDPOINT=otel-collector:4317
```
