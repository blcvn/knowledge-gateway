---
id: DOC-S05
service: memobase-context
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# memobase-context — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | 9033 | Yes | gRPC server port |
| `HEALTH_PORT` | int | 9100 | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `DB_DSN` | string | — | Yes | PostgreSQL connection string |
| `REDIS_URL` | string | `redis://redis:6379` | Yes | Redis connection URL |
| `PROFILE_CACHE_TTL` | int | 1200 | No | Profile cache TTL in seconds (20 min) |
| `DEFAULT_MAX_TOKEN_SIZE` | int | 500 | No | Default token budget for context assembly |
| `PROFILE_EVENT_RATIO` | float | 0.7 | No | Token budget split (profiles / events) |
| `EVENT_SEARCH_THRESHOLD` | float | 0.2 | No | Minimum cosine similarity for event gist search |
| `EVENT_SEARCH_WINDOW_DAYS` | int | 21 | No | Time window for event gist search |
| `EVENT_SEARCH_TOPK` | int | 10 | No | Max event gists per search |

## Example .env

```env
GRPC_PORT=9033
HEALTH_PORT=9100
LOG_LEVEL=info
NATS_URL=nats://nats:4222
DB_DSN=postgres://user:pass@postgresql:5432/memobase?sslmode=disable
REDIS_URL=redis://redis:6379
PROFILE_CACHE_TTL=1200
DEFAULT_MAX_TOKEN_SIZE=500
PROFILE_EVENT_RATIO=0.7
```
