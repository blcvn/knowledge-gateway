---
id: DOC-S06
service: graphiti-pipeline
version: 1.0.0
status: Active
created: 2026-05-10
updated: 2026-05-10
---

# graphiti-pipeline — Runbook (Operations Guide)

## Startup / Shutdown

### Startup Sequence

```bash
# 1. Verify dependencies
pg_isready -h localhost -p 5432
grpcurl -plaintext graphiti-store:9024 grpc.health.v1.Health/Check
nats server check connection --server nats://localhost:4222

# 2. Run migrations
go run cmd/migrate/main.go up

# 3. Start service
go run cmd/server/main.go
# Or: docker compose up graphiti-pipeline
```

**Startup checks** (in order):
1. PostgreSQL connection → fail-fast if unavailable
2. graphiti-store gRPC health → warn + circuit breaker (will retry)
3. NATS connection → warn + async retry
4. Bifrost LLM health → warn (graceful degradation)

### Shutdown Procedure

1. Stop accepting new gRPC requests (drain listener)
2. Wait for in-flight sagas to complete (up to `SHUTDOWN_TIMEOUT`)
3. Persist any queued episodes to PostgreSQL for recovery
4. Close database connections
5. Flush OTel traces and Prometheus metrics

## Health Check Endpoints

| Endpoint | Port | Expected Response | Checks |
|----------|------|-------------------|--------|
| `grpc.health.v1.Health/Check` | 9021 | `SERVING` | gRPC server alive |
| `GET /healthz` | 9094 | `200 OK` | Liveness (process alive) |
| `GET /readyz` | 9094 | `200 OK` | Readiness (all deps connected) |
| `GET /metrics` | 9094 | Prometheus format | Metrics endpoint |

### Readiness Checks Detail

| Check | Timeout | Failure Impact |
|-------|---------|---------------|
| PostgreSQL ping | 5s | NOT READY |
| graphiti-store gRPC | 3s | NOT READY |
| NATS connection | 3s | DEGRADED (still READY) |

## Common Errors & Resolution

### Error: `ErrDuplicateEpisode`
- **Symptom**: `ALREADY_EXISTS` gRPC status code
- **Cause**: Episode with same (name, group_id, valid_at) hash already ingested
- **Resolution**: Expected behavior (idempotent). Client should treat as success.

### Error: `ErrGroupLocked`
- **Symptom**: `RESOURCE_EXHAUSTED` gRPC status code
- **Cause**: Per-group queue full, another saga still processing for this group_id
- **Resolution**: Client retry with exponential backoff. Check if saga is stuck (`saga_state.status = 'PROCESSING'` for >10min).

### Error: `circuit breaker is open [graphiti-store]`
- **Symptom**: `UNAVAILABLE` gRPC status code
- **Cause**: graphiti-store service unresponsive (>5 consecutive failures)
- **Resolution**: Check graphiti-store health. Circuit breaker auto-recovers after `STORE_CB_TIMEOUT`.

### Error: `LLM timeout`
- **Symptom**: Saga stuck at `EXTRACTING_ENTITIES` or `RESOLVING_ENTITIES`
- **Cause**: Bifrost/LLM provider response time >60s
- **Resolution**: Check Bifrost dashboard. Verify LLM API key validity. Consider switching to smaller model.

### Error: `saga compensation failed`
- **Symptom**: Saga in `COMPENSATING` state permanently
- **Cause**: graphiti-store rollback failed during compensation
- **Resolution**: Manual intervention required. Check Neo4j for orphaned nodes. Run cleanup query.

## Deployment & Rollback

### Deployment

```bash
# Build image
docker build -t graphiti-pipeline:v1.0.0 .

# Deploy to K8s
kubectl set image deployment/graphiti-pipeline \
  pipeline=ghcr.io/vnp/graphiti-pipeline:v1.0.0 \
  -n graphiti-system

# Verify rollout
kubectl rollout status deployment/graphiti-pipeline -n graphiti-system
```

### Rollback

```bash
kubectl rollout undo deployment/graphiti-pipeline -n graphiti-system
```

## Monitoring Dashboards

- **Grafana**: `http://grafana:3000/d/graphiti-pipeline`
- **Jaeger**: `http://jaeger:16686` (filter by `graphiti-pipeline`)
- **Prometheus**: `http://prometheus:9090`

### Key Metrics to Watch

| Metric | Alert Threshold | Action |
|--------|----------------|--------|
| `graphiti_pipeline_saga_duration_seconds` (p95) | >120s | Check LLM latency |
| `graphiti_pipeline_saga_total{status="FAILED"}` | >10/min | Check downstream services |
| `graphiti_pipeline_circuit_breaker_state{state="open"}` | Any | Check target service |
| `graphiti_pipeline_llm_tokens_total` | Budget threshold | Review prompt efficiency |
| `graphiti_pipeline_group_queue_length` | >50 | Scale replicas or increase workers |

## Escalation

1. **L1 (On-call)**: Check dashboards, restart pod if unhealthy
2. **L2 (Service owner)**: Investigate saga state, check circuit breakers
3. **L3 (Architecture)**: Cross-service issues, Neo4j consistency problems
