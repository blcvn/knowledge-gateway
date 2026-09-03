# Runbook — Cognee App

## 1. Local Development

### Prerequisites
- Go 1.23+
- Docker + Docker Compose
- Infrastructure: PostgreSQL, NATS, Neo4j, Qdrant, Redis, MinIO

### Quick Start with Docker Compose

```bash
cd apps/cognee
docker compose up -d    # Starts app + all infra
docker compose logs -f cognee-app   # Follow app logs
```

### Quick Start without Docker

```bash
# Start infrastructure manually (or use existing instances)
# Then build and run the app:
cd apps/cognee
make run   # Sets AUTH_DEV_MODE=true + local connection strings
```

### Verify Startup

```bash
# 1. Aggregated health (dedicated port)
curl http://localhost:11080/healthz   # → {"status":"alive"}
curl http://localhost:11080/readyz    # → {"status":"ready","services":{...}}

# 2. Gateway health
curl http://localhost:8080/healthz    # → {"status":"alive"}
curl http://localhost:8080/readyz     # → {"status":"ready","services":{...}}
```

**Expected startup logs:**
```json
{"level":"INFO","msg":"cognee-app starting","rest_port":8080,"health_port":11080,"ingestion_port":9011,"cognify_port":9012,"search_port":9013}
{"level":"INFO","msg":"starting phase","phase":0,"services":3}
{"level":"INFO","msg":"starting service","service":"cognee-ingestion","port":9011}
{"level":"INFO","msg":"starting service","service":"cognee-cognify","port":9012}
{"level":"INFO","msg":"starting service","service":"cognee-search","port":9013}
{"level":"INFO","msg":"service ready","service":"cognee-ingestion","port":9011}
{"level":"INFO","msg":"service ready","service":"cognee-cognify","port":9012}
{"level":"INFO","msg":"service ready","service":"cognee-search","port":9013}
{"level":"INFO","msg":"starting phase","phase":1,"services":1}
{"level":"INFO","msg":"gateway HTTP server listening","addr":":8080"}
{"level":"INFO","msg":"all services started"}
```

## 2. Health Checks

| Endpoint | Port | Purpose | Response |
|----------|------|---------|----------|
| `GET /healthz` | 11080 | Liveness (process alive) | `200 {"status":"alive"}` |
| `GET /readyz` | 11080 | Readiness (all services serving) | `200 {"status":"ready","services":{...}}` |
| `GET /healthz` | 8080 | Gateway alive | `200 {"status":"alive"}` |
| `GET /readyz` | 8080 | Gateway + gRPC services | `200 {"status":"ready","services":{...}}` |

### Kubernetes Probe Config

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 11080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 11080
  initialDelaySeconds: 15
  periodSeconds: 5
```

## 3. Graceful Shutdown

On `SIGTERM` or `SIGINT`:

1. **Phase 1 shutdown**: Gateway stops accepting new HTTP connections, drains in-flight (30s)
2. **Phase 0 shutdown**: Each gRPC service calls GracefulStop (drains RPCs)
3. Health server stops
4. Process exits 0

## 4. Building Docker Image

```bash
cd apps/cognee
make docker         # Builds from monorepo root context
docker images cognee-app:latest   # Check size (target: <30MB)
```

## 5. Troubleshooting

### App won't start

| Error | Cause | Fix |
|-------|-------|-----|
| `POSTGRES_DSN is required` | Missing config | Set `POSTGRES_DSN` env var |
| `LLM_API_KEY is required` | Production mode | Set `AUTH_DEV_MODE=true` for dev |
| `port already in use` | Port conflict | Change `REST_PORT` / `HEALTH_PORT` |
| `service X port not ready` | Service failed to start | Check logs for panic in service goroutine |

### Service fails to start

```bash
# Check if port is already in use
lsof -i :9011   # Ingestion
lsof -i :9012   # Cognify
lsof -i :9013   # Search
lsof -i :8080   # Gateway
```

### Gateway returns 501 Not Implemented

This is expected for v1. API routes are defined but gRPC-JSON transcoding
is pending protobuf definitions. The gRPC connections to services are active.

### Infrastructure connectivity

```bash
# PostgreSQL
psql "postgresql://postgres:password@localhost:5432/cognee" -c "SELECT 1"

# NATS
curl http://localhost:8222/varz   # NATS monitoring

# Neo4j
curl http://localhost:7474

# Qdrant
curl http://localhost:6333/healthz

# MinIO
curl http://localhost:9000/minio/health/live

# Redis
redis-cli -h localhost ping
```

## 6. Rollback to Microservices

If you need to switch back to separate microservice deployment:

1. Stop the monolith: `docker compose down`
2. Deploy each service individually:
   - `services/cognee-ingestion` → port 9011
   - `services/cognee-cognify` → port 9012
   - `services/cognee-search` → port 9013
   - `gateway/` → port 8080
3. Update DNS/service discovery from `localhost` to service names
4. No code changes needed — services are unchanged

## 7. Log Analysis

All logs are structured JSON. Key fields:

| Field | Source | Description |
|-------|--------|-------------|
| `service` | supervisor | Which embedded service |
| `phase` | supervisor | Startup phase (0=infra, 1=gateway) |
| `method`, `path`, `status` | gateway | HTTP request log |
| `duration_ms` | gateway | Request latency |
| `error` | any | Error details |
| `port` | embed | gRPC listen port |
