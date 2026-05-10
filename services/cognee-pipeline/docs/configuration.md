# cognee-pipeline — Configuration Reference

> **Service**: `cognee-pipeline`  
> **Status**: Draft — To be completed during implementation

---

## Environment Variables

| Variable | Description | Type | Default | Required |
|----------|------------|------|---------|----------|
| GRPC_PORT | gRPC server port | int | varies | Yes |
| HEALTH_PORT | Health check port | int | varies | Yes |
| DATABASE_URL | PostgreSQL connection string | string | — | Yes |
| REDIS_URL | Redis connection string | string | — | Yes |
| NATS_URL | NATS JetStream URL | string | nats://nats:4222 | Yes |
| LOG_LEVEL | Logging level | string | info | No |
| OTEL_ENDPOINT | OpenTelemetry collector | string | — | No |

_Full configuration to be documented during implementation._
