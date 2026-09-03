---
id: DOC-S05
service: sm-profile
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-profile — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9074` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9119` | Yes | Health check port |
| `LOG_LEVEL` | string | `info` | No | Log level |
| `DB_DSN` | string | — | Yes | PostgreSQL connection |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS for memory events |
| `REDIS_URL` | string | `redis://redis:6379` | No | Profile cache |
| `CACHE_TTL_MINUTES` | int | `15` | No | Profile cache TTL |
| `TRAIT_UPDATE_BATCH_SIZE` | int | `10` | No | Dynamic trait batch size |

## Example .env

```env
GRPC_PORT=9074
HEALTH_PORT=9119
DB_DSN=postgres://user:pass@postgresql:5432/sm_profile?sslmode=disable
NATS_URL=nats://nats:4222
REDIS_URL=redis://redis:6379
CACHE_TTL_MINUTES=15
```
