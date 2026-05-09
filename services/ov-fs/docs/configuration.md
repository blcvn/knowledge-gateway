---
id: DOC-S05
service: ov-fs
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-fs — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9051` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9104` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_BACKEND` | string | `postgres` | No | Backend: `postgres` or `surrealdb` |
| `DB_MAX_CONNECTIONS` | int | `20` | No | Connection pool max size |
| `CRYPTO_SERVICE_ADDR` | string | `ov-crypto:9055` | Yes | ov-crypto gRPC address |
| `CRYPTO_ENABLED` | bool | `true` | No | Enable/disable transparent encryption |
| `PATHLOCK_TIMEOUT_MS` | int | `5000` | No | PathLock acquisition timeout |
| `MAX_FILE_SIZE_MB` | int | `50` | No | Maximum single file size |
| `ABSTRACT_GENERATION` | string | `async` | No | L0/L1 generation: `async`, `sync`, `disabled` |
| `LLM_ENDPOINT` | string | — | No | LLM endpoint for abstract generation |

## Database-Specific Config

### PostgreSQL

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `PG_HOST` | string | `localhost` | PostgreSQL host |
| `PG_PORT` | int | `5432` | PostgreSQL port |
| `PG_DATABASE` | string | `ov_fs` | Database name |
| `PG_SSLMODE` | string | `prefer` | SSL mode |

### SurrealDB

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `SURREAL_URL` | string | `ws://surrealdb:8000` | SurrealDB endpoint |
| `SURREAL_NS` | string | `vnp_memory` | SurrealDB namespace |
| `SURREAL_DB` | string | `ov_fs` | SurrealDB database |

## Example .env

```env
GRPC_PORT=9051
HEALTH_PORT=9104
LOG_LEVEL=info
NATS_URL=nats://nats:4222
DB_DSN=postgres://vnp:secret@localhost:5432/ov_fs?sslmode=disable
DB_BACKEND=postgres
CRYPTO_SERVICE_ADDR=ov-crypto:9055
CRYPTO_ENABLED=true
PATHLOCK_TIMEOUT_MS=5000
MAX_FILE_SIZE_MB=50
ABSTRACT_GENERATION=async
```
