---
id: DOC-S05
service: cognee-ingestion
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
---

# cognee-ingestion — Configuration Reference

## Environment Variables

### Server

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `GRPC_PORT` | int | `9011` | Yes | gRPC server port |
| `HEALTH_PORT` | int | `9091` | Yes | Health check HTTP port |
| `LOG_LEVEL` | string | `info` | No | Log level (debug/info/warn/error) |
| `SHUTDOWN_TIMEOUT` | duration | `30s` | No | Graceful shutdown timeout |

### Database (PostgreSQL)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `PG_HOST` | string | `localhost` | Yes | PostgreSQL host |
| `PG_PORT` | int | `5432` | Yes | PostgreSQL port |
| `PG_DATABASE` | string | `cognee_ingestion` | Yes | Database name |
| `PG_USER` | string | `cognee` | Yes | Database user |
| `PG_PASSWORD` | string | — | Yes | Database password |
| `PG_POOL_MAX` | int | `20` | No | Max connection pool size |
| `PG_POOL_MIN` | int | `5` | No | Min connection pool size |
| `PG_SSL_MODE` | string | `prefer` | No | SSL mode (disable/prefer/require) |

### Object Storage (MinIO/S3)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `S3_ENDPOINT` | string | `minio:9000` | Yes | MinIO/S3 endpoint |
| `S3_ACCESS_KEY` | string | — | Yes | Access key ID |
| `S3_SECRET_KEY` | string | — | Yes | Secret access key |
| `S3_BUCKET` | string | `cognee-data` | Yes | Default bucket name |
| `S3_REGION` | string | `us-east-1` | No | S3 region |
| `S3_USE_SSL` | bool | `false` | No | Use HTTPS for S3 |

### Cache (Redis)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `REDIS_URL` | string | `redis://redis:6379/0` | Yes | Redis connection URL |
| `REDIS_UPLOAD_TTL` | duration | `1h` | No | Upload progress cache TTL |

### Messaging (NATS)

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `NATS_URL` | string | `nats://nats:4222` | Yes | NATS server URL |
| `NATS_STREAM` | string | `cognee` | No | NATS JetStream stream name |

### Observability

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `OTEL_ENDPOINT` | string | `otel-collector:4317` | No | OTel collector gRPC endpoint |
| `OTEL_SERVICE_NAME` | string | `cognee-ingestion` | No | Service name for traces |

### Upload Limits

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `MAX_FILE_SIZE` | int | `104857600` | No | Max file size in bytes (100MB) |
| `MAX_TEXT_SIZE` | int | `1048576` | No | Max text content size (1MB) |
| `MAX_CONCURRENT_SCRAPES` | int | `10` | No | Max concurrent URL scrapes per tenant |

## Example .env

```env
GRPC_PORT=9011
HEALTH_PORT=9091
LOG_LEVEL=info

PG_HOST=localhost
PG_PORT=5432
PG_DATABASE=cognee_ingestion
PG_USER=cognee
PG_PASSWORD=secret
PG_POOL_MAX=20

S3_ENDPOINT=minio:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=cognee-data

REDIS_URL=redis://redis:6379/0
NATS_URL=nats://nats:4222
OTEL_ENDPOINT=otel-collector:4317
```
