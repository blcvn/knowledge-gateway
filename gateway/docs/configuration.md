---
id: DOC-S05
service: vnp-gateway
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# vnp-gateway — Configuration Reference

## Environment Variables

### Server

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `REST_PORT` | int | `8080` | Yes | REST HTTP server port |
| `GRPC_PORT` | int | `8081` | Yes | gRPC server port |
| `MCP_PORT` | int | `8082` | Yes | MCP server port (SSE/HTTP Streamable) |
| `HEALTH_PORT` | int | `11080` | Yes | Health check HTTP port |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | No | Graceful shutdown timeout |
| `LOG_LEVEL` | string | `info` | No | Log level: debug, info, warn, error |
| `LOG_FORMAT` | string | `json` | No | Log format: json, text |

### Authentication

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `AUTH_JWT_PUBLIC_KEY` | string | — | Yes | RS256 public key (PEM or base64) |
| `AUTH_JWT_ISSUER` | string | `vnp-memory` | No | Expected JWT issuer |
| `AUTH_JWT_AUDIENCE` | string | `vnp-api` | No | Expected JWT audience |
| `AUTH_API_KEY_PREFIX` | string | `vnp_` | No | API key prefix for identification |
| `AUTH_DEV_MODE` | bool | `false` | No | Skip auth (development only!) |

### Database

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `POSTGRES_DSN` | string | — | Yes | PostgreSQL connection string |
| `POSTGRES_MAX_CONNS` | int | `20` | No | Max connection pool size |
| `POSTGRES_MIN_CONNS` | int | `5` | No | Min idle connections |

### Redis

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `REDIS_ADDR` | string | `redis:6379` | Yes | Redis server address |
| `REDIS_PASSWORD` | string | — | No | Redis password |
| `REDIS_DB` | int | `0` | No | Redis database number |

### Rate Limiting

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `RATELIMIT_ENABLED` | bool | `true` | No | Enable/disable rate limiting |
| `RATELIMIT_FREE_RPM` | int | `60` | No | Free tier: requests/minute |
| `RATELIMIT_PRO_RPM` | int | `600` | No | Pro tier: requests/minute |
| `RATELIMIT_ENTERPRISE_RPM` | int | `6000` | No | Enterprise tier: requests/minute |

### Circuit Breaker

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `CB_MAX_FAILURES` | int | `5` | No | Failures before opening circuit |
| `CB_TIMEOUT` | duration | `60s` | No | Time before half-open retry |
| `CB_MAX_REQUESTS` | int | `3` | No | Max requests in half-open state |

### Timeouts (Per-Route)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `TIMEOUT_DEFAULT` | duration | `30s` | No | Default request timeout |
| `TIMEOUT_INGESTION` | duration | `120s` | No | Timeout for ingestion routes |
| `TIMEOUT_SEARCH` | duration | `10s` | No | Timeout for search routes |
| `TIMEOUT_MCP` | duration | `300s` | No | Timeout for MCP tool calls |

### CORS

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `CORS_ALLOWED_ORIGINS` | string | `*` | No | Comma-separated allowed origins |
| `CORS_ALLOW_CREDENTIALS` | bool | `true` | No | Allow credentials |
| `CORS_MAX_AGE` | int | `86400` | No | Preflight cache seconds |

### Observability

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `OTEL_SERVICE_NAME` | string | `vnp-gateway` | No | Service name in traces |
| `METRICS_ENABLED` | bool | `true` | No | Enable Prometheus metrics |

### NATS

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `NATS_CREDS_FILE` | string | — | No | NATS credentials file path |

### Service Discovery

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `SVC_COGNEE_INGESTION_ADDR` | string | `cognee-ingestion:9011` | Yes | cognee-ingestion gRPC address |
| `SVC_COGNEE_COGNIFY_ADDR` | string | `cognee-cognify:9012` | Yes | cognee-cognify gRPC address |
| `SVC_COGNEE_SEARCH_ADDR` | string | `cognee-search:9013` | Yes | cognee-search gRPC address |
| `SVC_GRAPHITI_*_ADDR` | string | See defaults | Yes | Graphiti services |
| `SVC_MEMOBASE_*_ADDR` | string | See defaults | Yes | Memobase services |
| `SVC_OV_*_ADDR` | string | See defaults | Yes | OpenViking services |
| `SVC_ZEP_*_ADDR` | string | See defaults | Yes | Zep services |
| `SVC_SM_*_ADDR` | string | See defaults | Yes | Supermemory services |
| `SVC_VNP_*_ADDR` | string | See defaults | Yes | Platform services |

## Example .env

```env
# Server
REST_PORT=8080
GRPC_PORT=8081
MCP_PORT=8082
HEALTH_PORT=11080
LOG_LEVEL=info

# Auth
AUTH_JWT_PUBLIC_KEY=base64://...
AUTH_DEV_MODE=false

# Database
POSTGRES_DSN=postgres://gateway:pass@postgres:5432/vnp_gateway?sslmode=disable

# Redis
REDIS_ADDR=redis:6379

# NATS
NATS_URL=nats://nats:4222

# Timeouts
TIMEOUT_DEFAULT=30s
TIMEOUT_INGESTION=120s
TIMEOUT_SEARCH=10s

# Service Discovery (examples)
SVC_COGNEE_INGESTION_ADDR=cognee-ingestion:9011
SVC_VNP_SEARCH_HUB_ADDR=vnp-search-hub:9042
SVC_VNP_ADMIN_ADDR=vnp-admin:9050
```
