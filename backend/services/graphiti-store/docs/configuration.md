---
id: DOC-S05
service: graphiti-store
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-store — Configuration Reference

## Environment Variables

### Core Service

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9024` | No | gRPC server port |
| `HEALTH_PORT` | int | `9097` | No | Health/metrics HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level: debug, info, warn, error |
| `LOG_FORMAT` | string | `json` | No | Log format: json, text |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | No | Graceful shutdown timeout |

### Driver Selection

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `DRIVER_PROVIDER` | string | `neo4j` | No | Graph driver: neo4j, falkordb, kuzu, neptune |

### Neo4j

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NEO4J_URI` | string | — | **Yes** | Neo4j Bolt URI (e.g. `bolt://localhost:7687`) |
| `NEO4J_USERNAME` | string | `neo4j` | No | Neo4j username |
| `NEO4J_PASSWORD` | string | — | **Yes** | Neo4j password |
| `NEO4J_DATABASE` | string | `neo4j` | No | Neo4j database name |
| `NEO4J_MAX_CONN_POOL` | int | `50` | No | Max connection pool size |
| `NEO4J_CONN_ACQUISITION_TIMEOUT` | duration | `60s` | No | Connection acquisition timeout |
| `NEO4J_MAX_TRANSACTION_RETRY_TIME` | duration | `30s` | No | Transaction retry timeout |

### Vector Index

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `VECTOR_DIMENSIONS` | int | `1536` | No | Embedding vector dimensions |
| `VECTOR_SIMILARITY_FUNCTION` | string | `cosine` | No | Similarity function: cosine, euclidean |

### Observability (OTel)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_ENDPOINT` | string | `localhost:4317` | No | OTel collector gRPC endpoint |
| `OTEL_SERVICE_NAME` | string | `graphiti-store` | No | Service name for tracing |
| `OTEL_ENABLED` | bool | `true` | No | Enable/disable OTel tracing |

## Example `.env`

```env
# Core
GRPC_PORT=9024
HEALTH_PORT=9097
LOG_LEVEL=debug

# Driver
DRIVER_PROVIDER=neo4j

# Neo4j
NEO4J_URI=bolt://localhost:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=secret
NEO4J_DATABASE=neo4j
NEO4J_MAX_CONN_POOL=50

# Vector
VECTOR_DIMENSIONS=1536

# OTel
OTEL_ENDPOINT=localhost:4317
```
