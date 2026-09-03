---
id: DOC-S06
service: graphiti-ingestion
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-ingestion — Runbook

## Startup / Shutdown

### Startup
```bash
# Prerequisites: PostgreSQL, NATS, graphiti-knowledge, graphiti-store must be running
docker compose up -d postgresql nats graphiti-knowledge graphiti-store
docker compose up graphiti-ingestion
```

### Graceful Shutdown
- Send `SIGTERM` → service drains in-flight sagas (up to 30s) → exits
- In-progress sagas are persisted to PostgreSQL and resumed on restart

## Health Check

| Endpoint | Expected Response |
|----------|------------------|
| `grpcurl localhost:9094 grpc.health.v1.Health/Check` | `SERVING` |
| `GET http://localhost:9094/healthz` | `200 OK` |
| `GET http://localhost:9094/readyz` | `200 OK` (all dependencies connected) |
| `GET http://localhost:9094/livez` | `200 OK` (process alive) |

## Common Issues

### Issue: Episodes stuck in QUEUED status
**Diagnosis**: Check if graphiti-knowledge/store are reachable
```bash
grpcurl -plaintext localhost:9023 grpc.health.v1.Health/Check
grpcurl -plaintext localhost:9024 grpc.health.v1.Health/Check
```
**Resolution**: Restart downstream services, circuit breaker will reset after timeout

### Issue: High latency on ingestion
**Diagnosis**: Check OTEL traces for slow saga steps
**Resolution**: 
- Entity extraction (LLM-bound): check graphiti-knowledge Bifrost connection
- SaveBulk (DB-bound): check Neo4j performance via graphiti-store metrics

### Issue: NATS connection refused
**Diagnosis**: `nats-cli server check connection`
**Resolution**: Verify `NATS_URL` env var, ensure NATS cluster is healthy

## Deployment

### Rolling Update
```bash
kubectl rollout restart deployment/graphiti-ingestion -n vnp-memory
```

### Rollback
```bash
kubectl rollout undo deployment/graphiti-ingestion -n vnp-memory
```

## Monitoring

- **Prometheus metrics**: `http://localhost:9094/metrics`
- Key metrics:
  - `graphiti_ingestion_episodes_total{status}` — Counter by status
  - `graphiti_ingestion_saga_duration_seconds` — Pipeline latency histogram
  - `graphiti_ingestion_saga_step_duration_seconds{step}` — Per-step latency
  - `graphiti_ingestion_circuit_breaker_state{service}` — CB state gauge

## Escalation

1. **L1**: Check health endpoints, restart service
2. **L2**: Check OTEL traces, review saga state in PostgreSQL
3. **L3**: Contact VNP Memory — Graphiti team lead
