---
id: DOC-S06
service: vnp-admin
version: 1.2.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# vnp-admin — Runbook

## Startup / Shutdown

```bash
make run-vnp-admin
docker compose up vnp-admin postgresql
# Graceful shutdown: SIGTERM → drain in-flight gRPC requests (30s) → exit
```

## Health Check

| Endpoint | Port | Expected |
|----------|------|----------|
| gRPC health | 9050 | `SERVING` |
| HTTP /healthz | 9103 | `200 OK {"status":"healthy"}` |
| HTTP /readyz | 9103 | `200 OK` (DB + NATS connected) |
| HTTP /metrics | 9103 | Prometheus metrics |

```bash
grpcurl -plaintext localhost:9050 grpc.health.v1.Health/Check
curl http://localhost:9103/healthz
```

## Common Issues

### Health aggregation slow or timing out

**Symptom**: `GetAggregatedHealth` RPC exceeds 30s or returns partial results.

**Diagnosis**: Check which downstream services are unhealthy:
```bash
grpcurl -plaintext localhost:9050 vnp.admin.v1.VNPAdminService/GetAggregatedHealth
```

**Resolution**: 
- Verify `HEALTH_CHECK_TIMEOUT` setting (default 5s per service)
- Check circuit breaker state for failing services
- Ensure all 35 service addresses are resolvable in DNS

### API key validation failures

**Symptom**: Valid API keys rejected with `UNAUTHENTICATED`.

**Diagnosis**:
```sql
SELECT id, name, key_hash, revoked_at, expires_at 
FROM api_keys WHERE tenant_id = 'xxx';
```

**Resolution**: Check if key is revoked or expired. Verify SHA-256 hash matches.

### Tenant creation fails

**Symptom**: `CreateTenant` returns `ALREADY_EXISTS`.

**Diagnosis**: Check for duplicate tenant name in database.

**Resolution**: Use unique tenant name or check existing tenants first.

## Deployment

```bash
kubectl rollout restart deployment/vnp-admin -n vnp-memory
kubectl rollout undo deployment/vnp-admin -n vnp-memory
```

## Monitoring

- **Key Metrics**: tenant_created_total, key_issued_total, health_check_latency_ms, health_aggregation_failures
- **Alerts**: Health aggregation > 50% services failing, key validation error rate > 5%

## Escalation

1. **L1**: Check health endpoints, verify DB connectivity, restart service
2. **L2**: Check OTel traces, review structured logs, verify NATS stream health
3. **L3**: Contact VNP Memory — Platform team
