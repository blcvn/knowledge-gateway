---
id: DOC-S06
service: vnp-gateway
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-gateway — Runbook

## Startup / Shutdown

```bash
make run-vnp-gateway
docker compose up vnp-gateway redis
# Graceful shutdown: SIGTERM → drain HTTP connections (30s) → close gRPC pools → exit
```

## Health Check

| Endpoint | Port | Expected |
|----------|------|----------|
| HTTP /healthz | 8083 | `200 OK {"status":"healthy"}` |
| HTTP /readyz | 8083 | `200 OK` (Redis + downstream services reachable) |
| HTTP /metrics | 8083 | Prometheus metrics |

```bash
curl http://localhost:8083/healthz
curl http://localhost:8083/readyz
```

## Common Issues

### Memobase routes returning 502

**Symptom**: `POST /v1/memobase/users/{uid}/blobs` returns 502 Bad Gateway.

**Diagnosis**:
```bash
# Check circuit breaker state
curl http://localhost:8083/internal/circuit-breakers
# Check memobase-ingestion health
grpcurl -plaintext memobase-ingestion:9031 grpc.health.v1.Health/Check
```

**Resolution**:
- Verify `MEMOBASE_INGESTION_ADDR` and `MEMOBASE_CONTEXT_ADDR` are correct
- Check if circuit breaker is OPEN (wait for `CB_TIMEOUT` or restart)
- Verify memobase services are running: `docker compose ps | grep memobase`

### Rate limiting false positives

**Symptom**: Legitimate requests rejected with 429.

**Diagnosis**:
```bash
# Check Redis rate limit counters
redis-cli KEYS "rl::*"
redis-cli GET "rl::{tenant_id}::{endpoint}::*"
```

**Resolution**: Increase `RATE_LIMIT_PER_MINUTE` or `RATE_LIMIT_BURST`. Verify tenant_id extraction is correct.

### JWT validation failures

**Symptom**: Valid JWTs rejected with 401.

**Diagnosis**: Check JWT public key, issuer, and clock skew.

**Resolution**: Verify `JWT_PUBLIC_KEY_PATH` points to correct PEM file. Check server clock synchronization.

### Auto-routing misclassification

**Symptom**: `memory.store` routes data to wrong engine.

**Diagnosis**: Check gateway logs for classification decision:
```bash
kubectl logs -f deployment/vnp-gateway -n vnp-memory | grep "route.classify"
```

**Resolution**: Use explicit `type` field instead of `auto` classification.

## Deployment

```bash
kubectl rollout restart deployment/vnp-gateway -n vnp-memory
kubectl rollout undo deployment/vnp-gateway -n vnp-memory
```

## Monitoring

- **Key Metrics**: requests_total (by route, method, status), request_latency_ms, circuit_breaker_state (by service), rate_limit_rejected_total
- **Alerts**: 5xx rate > 1%, p95 latency > 2s, circuit breaker OPEN for any service > 5min

## Escalation

1. **L1**: Check health endpoints, verify Redis connectivity, review circuit breaker states
2. **L2**: Check OTel traces for routing decisions, review downstream service health
3. **L3**: Contact VNP Memory — Platform team
