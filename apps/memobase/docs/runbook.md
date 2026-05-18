# Memobase App — Runbook

> Operations guide for startup, shutdown, troubleshooting, and monitoring.

## Prerequisites

- PostgreSQL 17+ with pgvector extension
- Redis 7+
- NATS Server 2+ with JetStream enabled
- LLM API access (OpenAI / Bifrost)

## Startup Procedure

### Local Development

```bash
# 1. Start infrastructure
make compose-up

# 2. Build binary
make build

# 3. Run with env
export DATABASE_URL="postgres://memobase:memobase@localhost:5432/memobase?sslmode=disable"
export REDIS_ADDR="localhost:6379"
export NATS_URL="nats://localhost:4222"
export AUTH_DEV_MODE=true
./bin/memobase
```

### Production

```bash
docker run -d \
  -e DATABASE_URL="..." \
  -e REDIS_ADDR="..." \
  -e NATS_URL="..." \
  -e LLM_API_KEY="..." \
  -p 8080:8080 -p 8082:8082 -p 9090:9090 \
  memobase-app:latest
```

### Startup Sequence

```
1. Config loaded & validated
2. ENV vars set for embedded services
3. Phase 0: memobase-ingestion starts (gRPC :9041)
4. Phase 1: memobase-engine starts (gRPC :9042)
5. Phase 2: memobase-context + memobase-pipeline start (:9043, :9044)
6. Phase 3: vnp-gateway starts (REST :8080, MCP :8082)
7. Health server starts (:9090)
8. All services ready → /readyz returns 200
```

## Shutdown Procedure

Signal: `SIGINT` or `SIGTERM`

```
1. Gateway stops accepting new requests (drain)
2. memobase-context + memobase-pipeline stop
3. memobase-engine finishes in-flight LLM calls
4. memobase-ingestion stops (flush pending buffers)
5. Process exits
```

Timeout: 30s (configurable via `SHUTDOWN_TIMEOUT`)

## Health Check Endpoints

| Endpoint | Port | Purpose |
|----------|------|---------|
| `GET /healthz` | 9090 | Liveness — process alive |
| `GET /readyz` | 9090 | Readiness — all services serving |
| `GET /status` | 9090 | Per-service status detail |
| `GET /api/v1/healthcheck` | 8080 | Gateway health (no auth) |

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 9090
  initialDelaySeconds: 10
  periodSeconds: 15

readinessProbe:
  httpGet:
    path: /readyz
    port: 9090
  initialDelaySeconds: 30
  periodSeconds: 10
```

## Common Issues

### Service fails to start

**Symptom**: `/readyz` returns 503, `/status` shows service as "failed"

**Diagnosis**:
```bash
curl http://localhost:9090/status | jq
# Check which service is not "serving"
```

**Resolution**:
- Check port conflicts (all ports must be unique)
- Check database connectivity
- Check NATS server availability
- Check logs for specific error messages

### Gateway can't reach services

**Symptom**: REST API returns `{"status":"not_connected"}`

**Resolution**:
- Verify services started before gateway (check startup logs)
- Check gRPC ports are correct in config
- Verify no firewall blocking localhost ports

### Buffer pipeline stuck

**Symptom**: Blobs inserted but profiles not updating

**Resolution**:
- Check NATS JetStream is running and `memobase` stream exists
- Check engine service is healthy (LLM API accessible)
- Check engine logs for LLM timeout errors

## Monitoring

### Key Metrics
- Service status via `/status` endpoint
- Gateway request latency via access logs
- NATS consumer lag for pipeline backpressure
- PostgreSQL connection pool utilization

### Log Format
Structured JSON logging via `slog`:
```json
{"time":"2026-05-12T10:00:00Z","level":"INFO","msg":"service ready","service":"memobase-ingestion","port":9041}
```

## Escalation

| Severity | Condition | Action |
|----------|-----------|--------|
| P0 | All services down | Restart process, check infrastructure |
| P1 | Engine fails (LLM) | Check LLM API key/quota, fallback mode |
| P2 | Cache miss rate high | Check Redis connectivity, increase TTL |
| P3 | Slow context assembly | Check PostgreSQL query performance |
