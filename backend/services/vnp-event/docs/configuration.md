---
id: DOC-S05
service: vnp-event
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-event — Configuration Reference

## Environment Variables

### Core

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | 9041 | Yes | gRPC server listen port |
| `HEALTH_PORT` | int | 9101 | Yes | Health/metrics HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | string | `json` | No | Log output format (json/text) |

### Database

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `DB_DSN` | string | — | Yes | PostgreSQL + pgvector connection string |
| `DB_POOL_SIZE` | int | 50 | No | Connection pool size |
| `DB_MAX_OVERFLOW` | int | 25 | No | Max overflow connections |

### Redis

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `REDIS_URL` | string | `redis://localhost:6379` | Yes | Redis connection URL |
| `REDIS_RECENT_EVENTS_TTL` | int | 300 | No | Recent events cache TTL (seconds) |

### NATS

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NATS_URL` | string | `nats://localhost:4222` | Yes | NATS JetStream server URL |
| `NATS_STREAM_NAME` | string | `vnp-events` | No | JetStream stream name |
| `NATS_CONSUMER_GROUP` | string | `vnp-event-svc` | No | Consumer group for load balancing |

### Embedding

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `EMBEDDING_PROVIDER` | string | `openai` | No | Embedding provider for semantic search |
| `EMBEDDING_DIM` | int | 1536 | No | Vector dimension |

### Observability

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_EXPORTER_ENDPOINT` | string | `localhost:4317` | No | OTel collector gRPC endpoint |
| `OTEL_SERVICE_NAME` | string | `vnp-event` | No | OTel service name |

## Example `.env`

```env
GRPC_PORT=9041
HEALTH_PORT=9101
DB_DSN=postgres://event:secret@postgresql:5432/vnp_event?sslmode=disable
REDIS_URL=redis://redis:6379
NATS_URL=nats://nats:4222
LOG_LEVEL=info
OTEL_EXPORTER_ENDPOINT=otel-collector:4317
EMBEDDING_PROVIDER=openai
EMBEDDING_DIM=1536
```
