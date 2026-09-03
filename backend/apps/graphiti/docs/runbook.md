# Runbook — Graphiti App

## 1. Local Development

### Prerequisites
- Go 1.25+
- Docker + Docker Compose
- Infrastructure: Neo4j, Redis, NATS

### Quick Start with Docker Compose

```bash
cd apps/graphiti
docker compose up -d    # Starts app + all infra
docker compose logs -f graphiti-app   # Follow app logs
```

### Quick Start without Docker

```bash
# Start infrastructure manually (or use existing instances)
# Then build and run the app:
cd apps/graphiti
make run   # Sets AUTH_DEV_MODE=true + local connection strings
```

### Verify Startup

```bash
# 1. Aggregated health (dedicated port)
curl http://localhost:9090/healthz   # → {"status":"alive"}
curl http://localhost:9090/readyz    # → {"status":"ready","services":{...}}

# 2. Gateway health
curl http://localhost:8080/healthz   # → {"status":"alive"}
curl http://localhost:8080/readyz    # → {"status":"ready","services":{...}}
```

**Expected startup logs:**
```json
{"level":"INFO","msg":"graphiti-app starting","rest_port":8080,"mcp_port":8082,"health_port":9090,"store_port":9024,"knowledge_port":9023,"ingestion_port":9021,"search_port":9022,"pipeline_port":9025}
{"level":"INFO","msg":"starting phase","phase":0,"services":1,"names":["graphiti-store"]}
{"level":"INFO","msg":"service ready","service":"graphiti-store","port":9024}
{"level":"INFO","msg":"starting phase","phase":1,"services":1,"names":["graphiti-knowledge"]}
{"level":"INFO","msg":"service ready","service":"graphiti-knowledge","port":9023}
{"level":"INFO","msg":"starting phase","phase":2,"services":3,"names":["graphiti-ingestion","graphiti-search","graphiti-pipeline"]}
{"level":"INFO","msg":"service ready","service":"graphiti-ingestion","port":9021}
{"level":"INFO","msg":"service ready","service":"graphiti-search","port":9022}
{"level":"INFO","msg":"service ready","service":"graphiti-pipeline","port":9025}
{"level":"INFO","msg":"starting phase","phase":3,"services":1,"names":["vnp-gateway"]}
{"level":"INFO","msg":"gateway HTTP server listening","addr":":8080"}
{"level":"INFO","msg":"all services started"}
```

## 2. Health Checks

| Endpoint | Port | Purpose | Response |
|----------|------|---------|----------|
| `GET /healthz` | 9090 | Liveness (process alive) | `200 {"status":"alive"}` |
| `GET /readyz` | 9090 | Readiness (all services serving) | `200 {"status":"ready","services":{...}}` |
| `GET /status` | 9090 | Detailed per-service status | `200 {"app":"graphiti-app","services":{...}}` |
| `GET /healthz` | 8080 | Gateway alive | `200 {"status":"alive"}` |
| `GET /readyz` | 8080 | Gateway + gRPC services | `200 {"status":"ready","services":{...}}` |

### Kubernetes Probe Config

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 9090
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /readyz
    port: 9090
  initialDelaySeconds: 15
  periodSeconds: 5
```

## 3. Graceful Shutdown

On `SIGTERM` or `SIGINT`:

1. **Phase 3 shutdown**: Gateway stops accepting new HTTP connections, drains in-flight (30s)
2. **Phase 2 shutdown**: Ingestion, Search, Pipeline — gRPC GracefulStop
3. **Phase 1 shutdown**: Knowledge — gRPC GracefulStop
4. **Phase 0 shutdown**: Store — gRPC GracefulStop, Neo4j connections closed
5. Health server stops
6. Process exits 0

## 4. Building Docker Image

```bash
cd apps/graphiti
make docker         # Builds from Dockerfile
docker images graphiti-app:latest   # Check size (target: <50MB)
```

## 5. Troubleshooting

### App won't start

| Error | Cause | Fix |
|-------|-------|-----|
| `LLM_API_KEY is required` | Production mode | Set `AUTH_DEV_MODE=true` for dev |
| `AUTH_JWT_PUBLIC_KEY is required` | Production mode | Set `AUTH_DEV_MODE=true` for dev |
| `port already in use` | Port conflict | Change `REST_PORT` / `HEALTH_PORT` |
| `service X port not ready` | Service failed to start | Check logs for panic in service goroutine |

### Service fails to start

```bash
# Check if port is already in use
lsof -i :9024   # Store
lsof -i :9023   # Knowledge
lsof -i :9021   # Ingestion
lsof -i :9022   # Search
lsof -i :9025   # Pipeline
lsof -i :8080   # Gateway
```

### Gateway returns 501 Not Implemented

This is expected for v1. API routes are defined but gRPC-JSON transcoding
is pending protobuf definitions. The gRPC connections to services are active.

### Infrastructure connectivity

```bash
# Neo4j
curl http://localhost:7474

# Redis
redis-cli -h localhost ping

# NATS
curl http://localhost:8222/varz   # NATS monitoring
```

## 6. Rollback to Microservices

If you need to switch back to separate microservice deployment:

1. Stop the monolith: `docker compose down`
2. Deploy each service individually:
   - `services/graphiti-store` → port 9024
   - `services/graphiti-knowledge` → port 9023
   - `services/graphiti-ingestion` → port 9021
   - `services/graphiti-search` → port 9022
   - `services/graphiti-pipeline` → port 9025
   - `gateway/` → port 8080
3. Update DNS/service discovery from `localhost` to service names
4. No code changes needed — services are unchanged

## 7. Log Analysis

All logs are structured JSON. Key fields:

| Field | Source | Description |
|-------|--------|-------------|
| `service` | supervisor | Which embedded service |
| `phase` | supervisor | Startup phase (0=data, 1=intel, 2=app, 3=gateway) |
| `method`, `path`, `status` | gateway | HTTP request log |
| `duration_ms` | gateway | Request latency |
| `error` | any | Error details |
| `port`, `grpc_port` | embed | gRPC listen port |
