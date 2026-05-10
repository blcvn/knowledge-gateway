---
id: DOC-S05
service: vnp-gateway
version: 1.3.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# vnp-gateway — Configuration Reference

> **Source**: `gateway/internal/infra/config/config.go`

## Server

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `REST_PORT` | int | `8080` | REST API port (external) |
| `GRPC_PORT` | int | `8081` | gRPC-Web port |
| `MCP_PORT` | int | `8082` | MCP (SSE/JSON-RPC) port |
| `HEALTH_PORT` | int | `11080` | Health check port |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | Graceful shutdown timeout |
| `LOG_LEVEL` | string | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | string | `json` | Log format (json, text) |

## Authentication

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `AUTH_JWT_PUBLIC_KEY` | string | — | RS256 public key (PEM) **required in prod** |
| `AUTH_JWT_ISSUER` | string | `vnp-memory` | Expected JWT issuer |
| `AUTH_JWT_AUDIENCE` | string | `vnp-api` | Expected JWT audience |
| `AUTH_API_KEY_PREFIX` | string | `vnp_` | Prefix for API key recognition |
| `AUTH_DEV_MODE` | bool | `false` | Skip auth validation (dev only!) |

## PostgreSQL

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `POSTGRES_DSN` | string | — | PostgreSQL connection string |
| `POSTGRES_MAX_CONNS` | int | `20` | Max pool connections |
| `POSTGRES_MIN_CONNS` | int | `5` | Min pool connections |

## Redis

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `REDIS_ADDR` | string | `redis:6379` | Redis address |
| `REDIS_PASSWORD` | string | — | Redis password |
| `REDIS_DB` | int | `0` | Redis database number |

## Rate Limiting

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `RATELIMIT_ENABLED` | bool | `true` | Enable rate limiting |
| `RATELIMIT_FREE_RPM` | int | `60` | Free tier requests/minute |
| `RATELIMIT_PRO_RPM` | int | `600` | Pro tier requests/minute |
| `RATELIMIT_ENTERPRISE_RPM` | int | `6000` | Enterprise requests/minute |

## Circuit Breaker

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `CB_MAX_FAILURES` | int | `5` | Failures before opening circuit |
| `CB_TIMEOUT` | duration | `60s` | Time before half-open |
| `CB_MAX_REQUESTS` | int | `3` | Half-open test requests |

## Timeouts

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `TIMEOUT_DEFAULT` | duration | `30s` | Default request timeout |
| `TIMEOUT_INGESTION` | duration | `120s` | Ingestion endpoints |
| `TIMEOUT_SEARCH` | duration | `10s` | Search endpoints |
| `TIMEOUT_MCP` | duration | `300s` | MCP tool calls |

## CORS

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `CORS_ALLOWED_ORIGINS` | string | `*` | Allowed origins |
| `CORS_ALLOW_CREDENTIALS` | bool | `true` | Allow credentials |
| `CORS_MAX_AGE` | int | `86400` | Preflight cache (seconds) |

## OpenTelemetry

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | OTLP endpoint |
| `OTEL_SERVICE_NAME` | string | `vnp-gateway` | Service name in traces |
| `METRICS_ENABLED` | bool | `true` | Enable Prometheus metrics |

## NATS

| Env Variable | Type | Default | Description |
|-------------|------|---------|-------------|
| `NATS_URL` | string | `nats://nats:4222` | NATS server URL |
| `NATS_CREDS_FILE` | string | — | NATS credentials file path |

## Downstream Services

The gateway routes to 35 downstream services via gRPC. Default addresses are configured in `config.go:defaultServiceAddresses()`. Each service can be overridden via environment variables:

```env
# Override individual service addresses
SERVICE_COGNEE_INGESTION=cognee-ingestion:9011
SERVICE_VNP_ADMIN=vnp-admin:9050
# ... etc
```

See `gateway/internal/infra/config/config.go` for the full 35-service address map.
