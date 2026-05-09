---
id: DOC-S05
service: vnp-admin
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-admin — Configuration Reference

## Environment Variables

### Core

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | 9050 | Yes | gRPC server listen port |
| `HEALTH_PORT` | int | 9103 | Yes | Health/metrics HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | string | `json` | No | Log output format (json/text) |

### Database

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `DB_POOL_SIZE` | int | 25 | No | Connection pool size |
| `DB_MAX_OVERFLOW` | int | 10 | No | Max overflow connections |

### NATS

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NATS_URL` | string | `nats://localhost:4222` | Yes | NATS JetStream server URL |

### Observability

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_EXPORTER_ENDPOINT` | string | `localhost:4317` | No | OTel collector gRPC endpoint |
| `OTEL_SERVICE_NAME` | string | `vnp-admin` | No | OTel service name |

### Service-Specific

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `HEALTH_CHECK_TIMEOUT` | duration | `5s` | No | Timeout for fan-out health checks |
| `HEALTH_CHECK_CONCURRENCY` | int | 10 | No | Max concurrent health check goroutines |
| `API_KEY_HASH_ALGORITHM` | string | `sha256` | No | Key hashing algorithm |

## Example `.env`

```env
GRPC_PORT=9050
HEALTH_PORT=9103
DB_DSN=postgres://admin:secret@postgresql:5432/vnp_admin?sslmode=disable
NATS_URL=nats://nats:4222
LOG_LEVEL=info
OTEL_EXPORTER_ENDPOINT=otel-collector:4317
HEALTH_CHECK_TIMEOUT=5s
```
