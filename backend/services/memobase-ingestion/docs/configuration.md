---
id: DOC-S05
service: memobase-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-ingestion — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | 9031 | Yes | gRPC server port |
| `HEALTH_PORT` | int | 9098 | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_POOL_SIZE` | int | 75 | No | Base connection pool size |
| `DB_MAX_OVERFLOW` | int | 50 | No | Burst connection capacity |
| `DB_POOL_RECYCLE` | int | 300 | No | Connection recycle interval (seconds) |
| `MAX_BUFFER_TOKEN_SIZE` | int | 1024 | No | Token threshold to trigger buffer flush |
| `BUFFER_IDLE_TIMEOUT` | duration | `1h` | No | Max idle time before force flush |
| `BLOB_PERSISTENT` | bool | false | No | Keep raw blobs after processing |

## Example .env

```env
GRPC_PORT=9031
HEALTH_PORT=9098
LOG_LEVEL=info
NATS_URL=nats://nats:4222
DB_DSN=postgres://user:pass@postgresql:5432/memobase?sslmode=disable
MAX_BUFFER_TOKEN_SIZE=1024
BUFFER_IDLE_TIMEOUT=1h
BLOB_PERSISTENT=false
```
