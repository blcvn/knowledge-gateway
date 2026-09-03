---
id: DOC-S05
service: ov-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-admin — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9056` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9109` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_MAX_CONNECTIONS` | int | `10` | No | Connection pool max size |
| `AUTH_MODE` | string | `api_key` | Yes | Auth mode: dev / trusted / api_key |
| `DEV_ACCOUNT_ID` | string | `default` | No | Default account for dev mode |
| `DEV_USER_ID` | string | `default` | No | Default user for dev mode |
| `OV_CRYPTO_ADDR` | string | `ov-crypto:9055` | No | ov-crypto for API key hashing |
| `HEALTH_CHECK_TIMEOUT_MS` | int | `5000` | No | Health aggregation timeout per service |
| `OV_FS_ADDR` | string | `ov-fs:9051` | No | Health check target |
| `OV_SEARCH_ADDR` | string | `ov-search:9052` | No | Health check target |
| `OV_SESSION_ADDR` | string | `ov-session:9053` | No | Health check target |
| `OV_RESOURCE_ADDR` | string | `ov-resource:9054` | No | Health check target |

## Example .env

```env
GRPC_PORT=9056
HEALTH_PORT=9109
LOG_LEVEL=info
DB_DSN=postgres://vnp:secret@localhost:5432/ov_admin?sslmode=disable
AUTH_MODE=api_key
OV_CRYPTO_ADDR=ov-crypto:9055
HEALTH_CHECK_TIMEOUT_MS=5000
```

## Development Example

```env
AUTH_MODE=dev
DEV_ACCOUNT_ID=dev-account
DEV_USER_ID=dev-user
```
