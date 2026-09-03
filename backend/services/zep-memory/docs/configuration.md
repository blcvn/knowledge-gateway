---
id: DOC-S05
service: zep-memory
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-memory — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9063` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `12063` | Yes | Health check HTTP port |
| `PG_DSN` | string | — | Yes | PostgreSQL connection string |
| `PG_MAX_OPEN_CONNS` | int | `15` | No | Max open connections |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS JetStream URL |
| `THREAD_CLIENT_ADDR` | string | `zep-thread:9062` | Yes | zep-thread gRPC address |
| `SEARCH_CLIENT_ADDR` | string | `zep-search:9065` | Yes | zep-search gRPC address |
| `DEFAULT_LAST_N` | int | `10` | No | Default messages to return |
| `MAX_FACTS` | int | `5` | No | Max facts for context assembly |
| `MIN_MESSAGES_FOR_FACTS` | int | `4` | No | Min messages to use as search context |

## YAML Configuration

```yaml
memory:
  grpc:
    port: 9063
  health:
    port: 12063
  postgres:
    dsn: "postgres://postgres:postgres@db:5432/zep?sslmode=disable"
    max_open_connections: 15
  nats:
    url: "nats://nats:4222"
    stream: "zep"
  clients:
    thread: "zep-thread:9062"
    search: "zep-search:9065"
  context:
    default_last_n: 10
    max_facts: 5
    min_messages_for_facts: 4
  telemetry:
    service_name: "zep-memory"
    otel_endpoint: "otel-collector:4317"
```
