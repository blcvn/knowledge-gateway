---
id: DOC-S05
service: zep-graph
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# zep-graph — Configuration Reference

## YAML Configuration

```yaml
graph:
  grpc:
    port: 9064
  health:
    port: 12064
  graphiti:
    service_url: "http://graphiti:8003"
    timeout: 60s                    # longer timeout for LLM extraction
    max_retries: 3
  neo4j:
    uri: "bolt://neo4j:7687"
    username: "neo4j"
    password: "${NEO4J_PASSWORD}"
    max_connection_pool_size: 50
  nats:
    url: "nats://nats:4222"
    stream: "zep"
    consumer_group: "zep-graph"
    max_deliver: 3                 # retry failed extractions
    ack_wait: 120s                 # long ack wait for LLM processing
  telemetry:
    service_name: "zep-graph"
    otel_endpoint: "otel-collector:4317"
```

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9064` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `12064` | Yes | Health check port |
| `GRAPHITI_URL` | string | `http://graphiti:8003` | Yes | Graphiti service URL |
| `GRAPHITI_TIMEOUT` | duration | `60s` | No | Graphiti request timeout |
| `NEO4J_URI` | string | `bolt://neo4j:7687` | Yes | Neo4j connection URI |
| `NEO4J_USERNAME` | string | `neo4j` | Yes | Neo4j username |
| `NEO4J_PASSWORD` | string | — | Yes | Neo4j password |
| `NEO4J_MAX_POOL` | int | `50` | No | Neo4j connection pool size |
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS JetStream URL |
| `NATS_ACK_WAIT` | duration | `120s` | No | NATS ack wait (for LLM) |
| `NATS_MAX_DELIVER` | int | `3` | No | Max delivery retries |
