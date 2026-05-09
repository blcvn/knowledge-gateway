---
id: DOC-S05
service: zep-search
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-search — Configuration Reference

## YAML Configuration

```yaml
search:
  grpc:
    port: 9065
  health:
    port: 12065
  graphiti:
    service_url: "http://graphiti:8003"
    timeout: 30s
  redis:
    url: "redis://redis:6379/1"
    cache_ttl: 30s
  reranker:
    default: "rrf"
    rrf:
      k: 60
    mmr:
      default_lambda: 0.5
    cross_encoder:
      model: "cross-encoder/ms-marco-MiniLM-L-12-v2"
      batch_size: 32
    node_distance:
      max_depth: 3
    episode_mentions:
      time_decay: 0.95
  nats:
    url: "nats://nats:4222"
    stream: "zep"
    consumer_group: "zep-search"
  telemetry:
    service_name: "zep-search"
    otel_endpoint: "otel-collector:4317"
```

## Environment Variables

| Variable | Type | Default | Required |
|----------|------|---------|----------|
| `GRPC_PORT` | int | `9065` | Yes |
| `HEALTH_PORT` | int | `12065` | Yes |
| `GRAPHITI_URL` | string | `http://graphiti:8003` | Yes |
| `REDIS_URL` | string | `redis://redis:6379/1` | Yes |
| `REDIS_CACHE_TTL` | duration | `30s` | No |
| `DEFAULT_RERANKER` | string | `rrf` | No |
| `NATS_URL` | string | `nats://nats:4222` | Yes |
