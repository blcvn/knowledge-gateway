---
id: DOC-S06
service: graphiti-store
version: 2.0.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# graphiti-store — Runbook (Operations Guide)

## Startup / Shutdown

### Startup Sequence

```bash
# 1. Verify Neo4j is available
cypher-shell -u neo4j -p secret "RETURN 1"

# 2. Start service
go run cmd/server/main.go
# Or: docker compose up graphiti-store
```

**Startup checks:**
1. Neo4j Bolt connection → fail-fast if unavailable
2. Index verification → auto-create missing indexes via BuildIndices
3. gRPC server bind on :9024 + health HTTP on :9097

### Shutdown Procedure

1. Stop accepting new gRPC requests
2. Wait for in-flight transactions to complete (up to `SHUTDOWN_TIMEOUT`)
3. Close Neo4j driver (flush connection pool)
4. Flush OTel traces

## Health Check Endpoints

| Endpoint | Port | Expected | Checks |
|----------|------|----------|--------|
| `grpc.health.v1.Health/Check` | 9024 | `SERVING` | gRPC alive |
| `GET /healthz` | 9097 | `200 OK` | Liveness |
| `GET /readyz` | 9097 | `200 OK` | Readiness (Neo4j connected) |
| `GET /metrics` | 9097 | Prometheus | Metrics |

## Common Errors & Resolution

### Error: `Neo4j connection refused`
- **Cause**: Neo4j not running or wrong URI
- **Resolution**: Verify `NEO4J_URI`, check Neo4j container logs

### Error: `vector index not found`
- **Cause**: Indexes not built
- **Resolution**: Call `BuildIndices` RPC or run `grpcurl ... /BuildIndices`

### Error: `TransactionTimedOut`
- **Cause**: Long-running transaction exceeds `NEO4J_MAX_TRANSACTION_RETRY_TIME`
- **Resolution**: Increase timeout or optimize Cypher query. Check for Neo4j lock contention.

### Error: `connection pool exhausted`
- **Cause**: More concurrent requests than `NEO4J_MAX_CONN_POOL`
- **Resolution**: Increase pool size or add backpressure at gateway level

### Error: `ErrNodeNotFound`
- **Expected behavior**: Normal 404 for missing entities
- **Resolution**: Client should handle gracefully

## Monitoring

### Key Metrics

| Metric | Alert Threshold | Action |
|--------|----------------|--------|
| `graphiti_store_operation_duration_seconds` (p95) | >500ms | Check Neo4j query plans |
| `graphiti_store_neo4j_pool_active` | >80% of MAX_CONN_POOL | Scale or increase pool |
| `graphiti_store_bulk_size` | >1000 items/batch | Monitor transaction time |
| `graphiti_store_errors_total{code="INTERNAL"}` | >5/min | Check Neo4j health |

### Dashboards

- **Grafana**: `http://grafana:3000/d/graphiti-store`
- **Neo4j Browser**: `http://localhost:7474` (direct Cypher queries)
- **Jaeger**: Filter by `graphiti-store`

## Deployment

```bash
# Build
docker build -t graphiti-store:v1.0.0 .

# Deploy
kubectl set image deployment/graphiti-store \
  store=ghcr.io/vnp/graphiti-store:v1.0.0 -n graphiti-system

# Rollback
kubectl rollout undo deployment/graphiti-store -n graphiti-system
```

## Escalation

1. **L1**: Check health, restart pod, verify Neo4j connectivity
2. **L2**: Review Cypher query performance, check connection pool metrics
3. **L3**: Neo4j cluster issues, data corruption, driver bugs
