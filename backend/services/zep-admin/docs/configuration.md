---
id: DOC-S05
service: zep-admin
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-admin — Configuration Reference

## YAML Configuration

```yaml
admin:
  grpc:
    port: 9066
  health:
    port: 12066
  postgres:
    dsn: "postgres://postgres:postgres@db:5432/zep?sslmode=disable"
    max_open_connections: 5
  backends:
    user: "zep-user:9061"
    thread: "zep-thread:9062"
    memory: "zep-memory:9063"
    graph: "zep-graph:9064"
    search: "zep-search:9065"
  health_check:
    timeout: 5s
    interval: 30s
  api_key:
    key_length: 32              # bytes
    default_scopes: ["read", "write"]
  nats:
    url: "nats://nats:4222"
    stream: "zep"
  telemetry:
    service_name: "zep-admin"
    otel_endpoint: "otel-collector:4317"
```

## Environment Variables

| Variable | Type | Default | Required |
|----------|------|---------|----------|
| `GRPC_PORT` | int | `9066` | Yes |
| `HEALTH_PORT` | int | `12066` | Yes |
| `PG_DSN` | string | — | Yes |
| `BACKEND_USER` | string | `zep-user:9061` | Yes |
| `BACKEND_THREAD` | string | `zep-thread:9062` | Yes |
| `BACKEND_MEMORY` | string | `zep-memory:9063` | Yes |
| `BACKEND_GRAPH` | string | `zep-graph:9064` | Yes |
| `BACKEND_SEARCH` | string | `zep-search:9065` | Yes |
| `HEALTH_CHECK_TIMEOUT` | duration | `5s` | No |
| `API_KEY_LENGTH` | int | `32` | No |
| `NATS_URL` | string | `nats://nats:4222` | Yes |
