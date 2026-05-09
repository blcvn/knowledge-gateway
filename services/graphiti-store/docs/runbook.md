---
id: DOC-S06
service: graphiti-store
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# graphiti-store — Runbook

## Startup / Shutdown

```bash
docker compose up graphiti-store
# Graceful shutdown: SIGTERM → drain in-flight requests (30s) → exit
```

## Health Check

| Endpoint | Expected |
|----------|----------|
| gRPC Health Check on port 9097 | `SERVING` |
| `GET http://localhost:9097/healthz` | `200 OK` |
| `GET http://localhost:9097/readyz` | `200 OK` |
| `GET http://localhost:9097/livez` | `200 OK` |

## Monitoring

- **Prometheus metrics**: `http://localhost:9097/metrics`
- **OTel traces**: Exported to collector at `OTEL_EXPORTER_ENDPOINT`
- **Structured logs**: JSON format via slog with `request_id`, `tenant_id`

## Deployment

```bash
kubectl rollout restart deployment/graphiti-store -n vnp-memory
kubectl rollout undo deployment/graphiti-store -n vnp-memory  # rollback
```

## Escalation

1. **L1**: Check health endpoints, restart service
2. **L2**: Check OTEL traces, review logs
3. **L3**: Contact VNP Memory — Graphiti team
