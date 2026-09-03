---
id: DOC-S06
service: vnp-gateway
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
audience: SRE, DevOps, On-call engineers
---

# vnp-gateway — Runbook

---

## 1. Startup

```bash
# From monorepo root
make run-gateway

# Or with Docker
docker compose up vnp-gateway

# Expected startup logs:
# {"level":"info","msg":"loading config","service":"vnp-gateway"}
# {"level":"info","msg":"connected to PostgreSQL","pool_size":20}
# {"level":"info","msg":"connected to Redis","addr":"redis:6379"}
# {"level":"info","msg":"connected to NATS","url":"nats://nats:4222"}
# {"level":"info","msg":"REST server started","port":8080}
# {"level":"info","msg":"gRPC server started","port":8081}
# {"level":"info","msg":"MCP server started","port":8082}
# {"level":"info","msg":"health server started","port":11080}
```

## 2. Shutdown

Graceful shutdown via `SIGTERM` or `SIGINT`:
1. Stop accepting new connections
2. Drain in-flight requests (30s timeout)
3. Close gRPC client connections
4. Close Redis + PostgreSQL pools
5. Flush OTel spans
6. Exit 0

```bash
# Graceful
kill -SIGTERM <pid>

# Force (after 30s)
kill -SIGKILL <pid>
```

## 3. Health Checks

### HTTP Health

```bash
# Liveness (is the process alive?)
curl http://localhost:11080/healthz
# → {"status": "serving"}

# Readiness (can it accept traffic?)
curl http://localhost:11080/readyz
# → {"status": "ready", "checks": {"postgres": "ok", "redis": "ok", "nats": "ok"}}

# Deep health (all downstream services)
curl http://localhost:11080/healthz/deep
# → {"status": "degraded", "services": {"cognee-ingestion": "ok", "graphiti-search": "circuit_open", ...}}
```

### gRPC Health

```bash
grpcurl -plaintext localhost:8081 grpc.health.v1.Health/Check
# → {"status": "SERVING"}
```

## 4. Common Issues

| Symptom | Diagnosis | Resolution |
|---------|-----------|-----------|
| **503 on all routes** | Gateway can't reach any downstream service | Check NATS/Redis connectivity, verify service addresses |
| **503 on specific route** | Circuit breaker open for that service | Wait for half-open retry (60s default), check target service health |
| **401 Unauthorized** | JWT expired or API key revoked | Verify token expiry, check api_keys table for revocation |
| **429 Rate Limited** | Tenant exceeded rate limit | Check `RATELIMIT_*` config, upgrade tenant tier |
| **504 Timeout** | Downstream service slow | Check target service logs/metrics, adjust `TIMEOUT_*` |
| **High latency (>500ms)** | Redis slow or DNS resolution delay | Check Redis latency, verify service discovery addresses |
| **MCP tools not responding** | SSE connection dropped | Client should reconnect; check MCP_PORT accessibility |
| **WebDAV 502** | ov-fs service unreachable | Verify SVC_OV_FS_ADDR, check ov-fs health |
| **Console 403** | User missing admin role | Verify JWT `roles` claim includes `admin` or `super_admin` |
| **Dashboard slow (>2s)** | Fan-out timeout to multiple engines | Check DASHBOARD_FAN_OUT_TIMEOUT, verify engine health |
| **WebSocket disconnects** | Connection limit or network issue | Check WS_MAX_CONNECTIONS, verify client reconnect logic |
| **Debugger empty trace** | search-hub or memobase-context down | Check vnp-search-hub and memobase-context health |
| **Audit log missing** | vnp-admin PostgreSQL unavailable | Verify vnp-admin DB connection, check audit_logs table |

## 5. Monitoring

### Prometheus Metrics

Available at `:11080/metrics`

| Metric | Type | Description |
|--------|------|-------------|
| `gateway_requests_total` | Counter | Total requests by route, method, status |
| `gateway_request_duration_seconds` | Histogram | Request latency by route |
| `gateway_active_connections` | Gauge | Active HTTP connections |
| `gateway_circuit_breaker_state` | Gauge | 0=closed, 1=half-open, 2=open per service |
| `gateway_ratelimit_rejected_total` | Counter | Rate limited requests |
| `gateway_downstream_errors_total` | Counter | Errors from downstream services |

### Grafana Dashboards

- **Gateway Overview**: Request rate, error rate, p50/p95/p99 latency
- **Service Health**: Circuit breaker states, per-service error rates
- **Rate Limiting**: Rejected requests per tenant/tier

### OTel Traces

- Every request gets a trace with spans for: auth → ratelimit → route → downstream_call
- Trace ID propagated via `X-Request-ID` header and gRPC metadata

## 6. Alert Runbook

| Alert | Severity | Action |
|-------|----------|--------|
| `GatewayErrorRate > 5%` | Warning | Check downstream service health |
| `GatewayErrorRate > 20%` | Critical | Escalate immediately, check all services |
| `GatewayP99Latency > 2s` | Warning | Check slow downstream services |
| `CircuitBreakerOpen` | Warning | Check target service, wait for recovery |
| `RedisConnectionFailed` | Critical | Rate limiting disabled (fail-open), fix Redis |
| `PostgresConnectionFailed` | Critical | API key auth broken, fix PostgreSQL |
| `NATSDisconnected` | Warning | Events not published, reconnect NATS |

## 7. Deployment

### Rolling Update (Zero Downtime)

```bash
# Kubernetes
kubectl rollout restart deployment/vnp-gateway

# Docker Compose
docker compose up -d --no-deps vnp-gateway
```

### Rollback

```bash
# Kubernetes
kubectl rollout undo deployment/vnp-gateway

# Docker Compose
docker compose up -d --no-deps vnp-gateway:previous-tag
```

## 8. Escalation

1. **L1**: On-call engineer — check logs, restart if needed (< 15min)
2. **L2**: Platform team lead — investigate circuit breaker patterns, config issues
3. **L3**: Architecture team — cross-service failures, NATS/Redis infrastructure
