---
id: DOC-S05
service: cognee-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
---

# cognee-pipeline — Configuration Reference

## Environment Variables

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `SERVICE_NAME` | string | `cognee-pipeline` | No | Service identifier |
| `SERVICE_VERSION` | string | `0.1.0` | No | Service version |
| `GRPC_PORT` | int | `9011` | No | gRPC server port |
| `HEALTH_PORT` | int | `9091` | No | HTTP health check port |
| `POSTGRES_DSN` | string | — | **Yes** | PostgreSQL connection string |
| `POSTGRES_MAX_CONNS` | int | `20` | No | Connection pool max |
| `NEO4J_URI` | string | — | **Yes** | Neo4j bolt URI |
| `NEO4J_USER` | string | `neo4j` | No | Neo4j username |
| `NEO4J_PASSWORD` | string | — | **Yes** | Neo4j password |
| `MINIO_ENDPOINT` | string | — | **Yes** | MinIO/S3 endpoint |
| `MINIO_ACCESS_KEY` | string | — | **Yes** | MinIO access key |
| `MINIO_SECRET_KEY` | string | — | **Yes** | MinIO secret key |
| `MINIO_BUCKET` | string | `cognee-data` | No | Default bucket |
| `MINIO_USE_SSL` | bool | `false` | No | SSL for MinIO |
| `NATS_URL` | string | — | **Yes** | NATS server URL |
| `NATS_STREAM` | string | `cognee` | No | JetStream stream name |
| `BIFROST_ENDPOINT` | string | — | **Yes** | Bifrost LLM gateway URL |
| `BIFROST_API_KEY` | string | — | **Yes** | Bifrost API key |
| `BIFROST_DEFAULT_MODEL` | string | `gpt-4o` | No | Default LLM model |
| `BIFROST_EMBEDDING_MODEL` | string | `text-embedding-3-large` | No | Embedding model |
| `PIPELINE_MAX_CONCURRENT_LLM` | int | `5` | No | LLM concurrency limit |
| `PIPELINE_STAGE_TIMEOUT` | duration | `5m` | No | Per-stage timeout |
| `PIPELINE_CHUNK_SIZE` | int | `512` | No | Default chunk size (tokens) |
| `PIPELINE_CHUNK_OVERLAP` | int | `50` | No | Chunk overlap (tokens) |
| `OTEL_EXPORTER_ENDPOINT` | string | — | No | OTel collector endpoint |
| `OTEL_SERVICE_NAME` | string | `cognee-pipeline` | No | OTel service name |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | string | `json` | No | Log format (json/text) |

## Example: Docker Compose

```yaml
cognee-pipeline:
  image: vnp-memory/cognee-pipeline:latest
  ports:
    - "9011:9011"
    - "9091:9091"
  environment:
    POSTGRES_DSN: "postgres://cognee:secret@postgres:5432/cognee?sslmode=disable"
    NEO4J_URI: "bolt://neo4j:7687"
    NEO4J_PASSWORD: "password"
    MINIO_ENDPOINT: "minio:9000"
    MINIO_ACCESS_KEY: "minioadmin"
    MINIO_SECRET_KEY: "minioadmin"
    NATS_URL: "nats://nats:4222"
    BIFROST_ENDPOINT: "http://bifrost:8080"
    BIFROST_API_KEY: "key"
```
